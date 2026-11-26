package device

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/waneock/app-lora/device/decoder"
)

const (
	// decodeEvery is how often the worker looks for pending uplinks. The
	// query reads a partial index, so an idle check costs little.
	decodeEvery = 2 * time.Second
	// decodeBatch bounds the uplinks of one transaction.
	decodeBatch = 1000
)

// decodeWorker decodes the pending uplinks and keeps the alarms of their
// devices. A new uplink, an attach, a device model change, and a decoder
// change all mark uplinks pending, so one path decodes a live uplink and
// years of history alike.
type decodeWorker struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	// batch bounds the uplinks of one transaction; a test makes it small.
	batch int
	stop  chan struct{}
	done  chan struct{}
}

func newDecodeWorker(pool *pgxpool.Pool, logger *slog.Logger) *decodeWorker {
	return &decodeWorker{pool: pool, logger: logger, batch: decodeBatch}
}

// runDecodeWorker ties the worker loop to the lifecycle of the process.
func runDecodeWorker(lc fx.Lifecycle, w *decodeWorker) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			w.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return w.close(ctx)
		},
	})
}

func (w *decodeWorker) start() {
	w.stop = make(chan struct{})
	w.done = make(chan struct{})
	go w.loop()
}

func (w *decodeWorker) loop() {
	defer close(w.done)
	ticker := time.NewTicker(decodeEvery)
	defer ticker.Stop()
	for {
		w.drain()
		select {
		case <-ticker.C:
		case <-w.stop:
			return
		}
	}
}

// drain runs batches until a batch is not full. A failure is logged; the
// next tick tries again.
func (w *decodeWorker) drain() {
	for {
		select {
		case <-w.stop:
			return
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		n, err := w.runOnce(ctx)
		cancel()
		if err != nil {
			w.logger.Warn("device: decode uplinks", slog.Any("err", err))
			return
		}
		if n < w.batch {
			return
		}
	}
}

func (w *decodeWorker) close(ctx context.Context) error {
	close(w.stop)
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runOnce decodes one batch of pending uplinks in one transaction and
// updates the alarms of their devices. It returns how many uplinks it
// decoded.
func (w *decodeWorker) runOnce(ctx context.Context) (int, error) {
	var n int
	err := pgx.BeginFunc(ctx, w.pool, func(tx pgx.Tx) error {
		repo := newDecodeRepo(tx)
		batch, err := repo.pending(ctx, w.batch)
		if err != nil {
			return err
		}
		n = len(batch)
		if n == 0 {
			return nil
		}

		ids := make([]int64, n)
		values := make([]decodedValues, n)
		fresh := make(map[deviceKey][]decodedRow)
		var order []deviceKey
		for i, p := range batch {
			ids[i] = p.ID
			values[i] = decodeOne(p)
			k := deviceKey{p.OrganizationID, p.DeviceID}
			if _, seen := fresh[k]; !seen {
				order = append(order, k)
				fresh[k] = nil
			}
			if len(values[i].Reported) > 0 {
				fresh[k] = append(fresh[k], decodedRow{
					UplinkID: p.ID, ReceivedAt: p.ReceivedAt,
					Reported: values[i].Reported, Raised: values[i].Raised,
				})
			}
		}
		if err := repo.save(ctx, ids, values); err != nil {
			return err
		}
		for _, k := range order {
			if err := updateAlarms(ctx, repo, k, fresh[k], batch); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

// decodeOne decodes one uplink with the decoder of its device model. A model
// with no decoder, or with a decoder this build does not hold, clears the
// decoded values.
func decodeOne(p pendingUplink) decodedValues {
	if p.Decoder == nil {
		return decodedValues{}
	}
	d, ok := decoder.Get(*p.Decoder)
	if !ok {
		return decodedValues{}
	}
	id := d.ID()
	res, err := d.Decode(p.Payload)
	if err != nil {
		msg := err.Error()
		return decodedValues{Decoder: &id, Error: &msg}
	}
	v := decodedValues{Decoder: &id, PulseCount: res.PulseCount}
	for _, k := range res.Reported {
		v.Reported = append(v.Reported, string(k))
	}
	for _, k := range res.Raised {
		v.Raised = append(v.Raised, string(k))
	}
	return v
}

// updateAlarms brings the alarms of a device up to date after a batch. While
// the device still has pending uplinks, it marks the alarms for a replay,
// because a later batch changes the history. When the new rows all come after the computed point,
// it applies them to the active alarms; else it replays all the rows.
func updateAlarms(ctx context.Context, repo *decodeRepo, k deviceKey, fresh []decodedRow, batch []pendingUplink) error {
	pending, err := repo.hasPending(ctx, k)
	if err != nil {
		return err
	}
	if pending {
		// The rows of this batch are saved, but their alarms are not
		// applied. A later batch sees only its own rows, so it must replay.
		return repo.setAlarmsThrough(ctx, k, nil)
	}
	through, err := repo.alarmsThrough(ctx, k)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if through != nil && allAfter(batch, k, *through) {
		// The batch is in uplink id order; the alarm rules need time order.
		slices.SortFunc(fresh, func(a, b decodedRow) int {
			if c := a.ReceivedAt.Compare(b.ReceivedAt); c != 0 {
				return c
			}
			return cmp.Compare(a.UplinkID, b.UplinkID)
		})
		open, err := repo.activeAlarms(ctx, k)
		if err != nil {
			return err
		}
		if err := repo.writeAlarms(ctx, k, applyAlarms(open, fresh)); err != nil {
			return err
		}
	} else {
		rows, err := repo.decodedRows(ctx, k)
		if err != nil {
			return err
		}
		if err := repo.deleteAlarms(ctx, k); err != nil {
			return err
		}
		if err := repo.writeAlarms(ctx, k, applyAlarms(nil, rows)); err != nil {
			return err
		}
	}

	last, err := repo.lastReceivedAt(ctx, k)
	if err != nil {
		return err
	}
	return repo.setAlarmsThrough(ctx, k, last)
}

// allAfter reports whether every uplink of device k in the batch came after
// through.
func allAfter(batch []pendingUplink, k deviceKey, through time.Time) bool {
	for _, p := range batch {
		if p.OrganizationID == k.OrganizationID && p.DeviceID == k.DeviceID && !p.ReceivedAt.After(through) {
			return false
		}
	}
	return true
}

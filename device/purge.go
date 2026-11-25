package device

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/waneock/app-lora/pkg/clock"
)

// purgeEvery is the period of the unmatched uplink purge.
const purgeEvery = time.Hour

// purger deletes the unmatched uplinks that are older than unmatchedKeep
// (device:D4), once at start and then every purgeEvery.
type purger struct {
	uplinks *uplinkRepo
	clock   clock.Clock
	logger  *slog.Logger
	stop    chan struct{}
	done    chan struct{}
}

func newPurger(pool *pgxpool.Pool, clk clock.Clock, logger *slog.Logger) *purger {
	return &purger{uplinks: newUplinkRepo(pool), clock: clk, logger: logger}
}

// runPurger ties the purge loop to the lifecycle of the process.
func runPurger(lc fx.Lifecycle, p *purger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			p.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return p.close(ctx)
		},
	})
}

func (p *purger) start() {
	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	go p.loop()
}

func (p *purger) loop() {
	defer close(p.done)
	ticker := time.NewTicker(purgeEvery)
	defer ticker.Stop()
	for {
		p.purge()
		select {
		case <-ticker.C:
		case <-p.stop:
			return
		}
	}
}

// purge runs one delete. A failure is logged; the next tick tries again.
func (p *purger) purge() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	n, err := p.uplinks.purgeUnmatched(ctx, p.clock.Now().Add(-unmatchedKeep))
	if err != nil {
		p.logger.WarnContext(ctx, "device: purge unmatched uplinks", slog.Any("err", err))
		return
	}
	if n > 0 {
		p.logger.InfoContext(ctx, "device: purged unmatched uplinks", slog.Int64("deleted", n))
	}
}

// close stops the loop and waits for it, or for ctx.
func (p *purger) close(ctx context.Context) error {
	close(p.stop)
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

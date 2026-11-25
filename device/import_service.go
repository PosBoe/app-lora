package device

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waneock/app-lora/pkg/pgdb"
)

// Import row statuses. The frontend matches on them.
const (
	importNew    = "new"
	importExists = "exists"
	importError  = "error"
)

// importedRow is the result of one row in a preview.
type importedRow struct {
	Line      int
	Status    string
	Code      string
	DevEUI    string
	Name      string
	Vendor    string
	Model     string
	Groups    []string
	NewModel  bool
	NewGroups []string
}

// importTotals counts the rows by status, and the device models and groups
// that the new rows create.
type importTotals struct {
	New       int
	Exists    int
	Error     int
	NewModels int
	NewGroups int
}

// importPlan is what an import of some rows does now: the result of each
// row, the totals, and the names to create.
type importPlan struct {
	rows      []importedRow
	totals    importTotals
	newModels []modelName
	newGroups []string
	// models and groups map a lower-case name to an id of the organization.
	models map[string]int64
	groups map[string]int64
	// creates are the rows to create, as parsed.
	creates []importRow
}

type modelName struct{ vendor, model string }

func modelKey(vendor, model string) string {
	return strings.ToLower(vendor) + "\x00" + strings.ToLower(model)
}

// importService previews and runs a CSV import of devices.
type importService struct {
	pool *pgxpool.Pool
}

func newImportService(pool *pgxpool.Pool) *importService {
	return &importService{pool: pool}
}

// preview tells what an import of rows would do now, and changes nothing.
func (s *importService) preview(ctx context.Context, orgID int64, rows []importRow) (*importPlan, error) {
	return plan(ctx, s.pool, orgID, rows)
}

// commit creates the devices of the new rows, with the missing device models
// and groups, in one transaction. It plans again inside the transaction, so
// a stale preview changes only the totals. A row whose DevEUI the
// organization has is skipped with its groups.
func (s *importService) commit(ctx context.Context, orgID int64, rows []importRow) (importTotals, error) {
	var totals importTotals
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		p, err := plan(ctx, tx, orgID, rows)
		if err != nil {
			return err
		}
		totals = p.totals
		if len(p.creates) == 0 {
			return nil
		}
		if err := createNames(ctx, tx, orgID, p); err != nil {
			return err
		}
		return createDevices(ctx, tx, orgID, p)
	})
	if errors.Is(err, errDuplicate) || errors.Is(err, errReferenced) {
		// Another request wrote the same names or DevEUIs at the same time.
		return importTotals{}, errImportRetry
	}
	return totals, err
}

// plan reads the device models, the groups, and the DevEUIs of the
// organization, and gives each row its status.
func plan(ctx context.Context, q pgdb.Querier, orgID int64, rows []importRow) (*importPlan, error) {
	p := &importPlan{models: make(map[string]int64), groups: make(map[string]int64)}

	models, err := newModelRepo(q).list(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		p.models[modelKey(m.Vendor, m.Model)] = m.ID
	}
	groups, err := newGroupRepo(q).list(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		p.groups[strings.ToLower(g.Name)] = g.ID
	}
	var euis []string
	for i := range rows {
		if r := &rows[i]; r.Err == nil {
			euis = append(euis, r.Fields.DevEUI)
		}
	}
	existing, err := newDeviceRepo(q).existingDevEUIs(ctx, orgID, euis)
	if err != nil {
		return nil, err
	}

	newModel := make(map[string]bool)
	newGroup := make(map[string]bool)
	for i := range rows {
		r := &rows[i]
		out := importedRow{Line: r.Line, DevEUI: r.RawDevEUI, Name: r.RawName}
		switch {
		case r.Err != nil:
			out.Status, out.Code = importError, r.Err.code
			p.totals.Error++
		case existing[r.Fields.DevEUI]:
			out.Status = importExists
			out.DevEUI, out.Name = r.Fields.DevEUI, r.Fields.Name
			p.totals.Exists++
		default:
			out.Status = importNew
			out.DevEUI, out.Name = r.Fields.DevEUI, r.Fields.Name
			out.Vendor, out.Model, out.Groups = r.Vendor, r.Model, r.Groups
			p.totals.New++
			p.creates = append(p.creates, *r)

			if key := modelKey(r.Vendor, r.Model); p.models[key] == 0 {
				out.NewModel = true
				if !newModel[key] {
					newModel[key] = true
					p.newModels = append(p.newModels, modelName{r.Vendor, r.Model})
				}
			}
			for _, g := range r.Groups {
				key := strings.ToLower(g)
				if p.groups[key] != 0 {
					continue
				}
				out.NewGroups = append(out.NewGroups, g)
				if !newGroup[key] {
					newGroup[key] = true
					p.newGroups = append(p.newGroups, g)
				}
			}
		}
		p.rows = append(p.rows, out)
	}
	p.totals.NewModels = len(p.newModels)
	p.totals.NewGroups = len(p.newGroups)
	return p, nil
}

// createNames creates the device models and groups of a plan, and adds their
// ids to the plan. It mutates p.
func createNames(ctx context.Context, tx pgx.Tx, orgID int64, p *importPlan) error {
	for _, m := range p.newModels {
		created, err := newModelRepo(tx).insert(ctx, orgID, m.vendor, m.model)
		if err != nil {
			return err
		}
		p.models[modelKey(m.vendor, m.model)] = created.ID
	}
	for _, name := range p.newGroups {
		created, err := newGroupRepo(tx).insert(ctx, orgID, name)
		if err != nil {
			return err
		}
		p.groups[strings.ToLower(name)] = created.ID
	}
	return nil
}

// createDevices creates the devices of a plan and their group memberships.
func createDevices(ctx context.Context, tx pgx.Tx, orgID int64, p *importPlan) error {
	fields := make([]deviceFields, len(p.creates))
	for i := range p.creates {
		r := &p.creates[i]
		f := r.Fields
		f.DeviceModelID = p.models[modelKey(r.Vendor, r.Model)]
		fields[i] = f
	}
	ids, err := newDeviceRepo(tx).insertMany(ctx, orgID, fields)
	if err != nil {
		return err
	}
	if err := attachImported(ctx, tx, orgID, ids); err != nil {
		return err
	}

	var groupIDs, deviceIDs []int64
	for i := range p.creates {
		r := &p.creates[i]
		for _, g := range r.Groups {
			groupIDs = append(groupIDs, p.groups[strings.ToLower(g)])
			deviceIDs = append(deviceIDs, ids[r.Fields.DevEUI])
		}
	}
	if len(groupIDs) == 0 {
		return nil
	}
	return newGroupRepo(tx).addPairs(ctx, orgID, groupIDs, deviceIDs)
}

// attachImported gives each new device the unmatched uplinks with its
// DevEUI (device:G8).
func attachImported(ctx context.Context, tx pgx.Tx, orgID int64, ids map[string]int64) error {
	deviceIDs := make([]int64, 0, len(ids))
	devEUIs := make([]string, 0, len(ids))
	for eui, id := range ids {
		deviceIDs = append(deviceIDs, id)
		devEUIs = append(devEUIs, eui)
	}
	return newUplinkRepo(tx).attachMany(ctx, orgID, deviceIDs, devEUIs)
}

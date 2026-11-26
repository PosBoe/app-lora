package device

import (
	"slices"
	"time"
)

// decodedRow is a decoded uplink as the alarm rules read it.
type decodedRow struct {
	UplinkID   int64
	ReceivedAt time.Time
	Reported   []string
	Raised     []string
}

// alarmPeriod is one alarm. ID is 0 for an alarm that is not stored yet.
// ClearedAt is nil while the alarm is active.
type alarmPeriod struct {
	ID              int64
	Kind            string
	RaisedAt        time.Time
	RaisedUplinkID  int64
	ClearedAt       *time.Time
	ClearedUplinkID int64
	// changed marks a stored alarm that applyAlarms cleared.
	changed bool
}

// applyAlarms runs the alarm rules (device:G11) over rows in time order,
// starting from the active alarms in open. A flag on raises an alarm of its
// kind when none is active. A kind that a row reports with the flag off
// clears the active alarm of that kind. A kind that a row does not report
// leaves the alarm as it is.
//
// It returns every alarm it touched: the ones in open that it cleared
// (changed), and the new ones, active or cleared. It mutates the alarms in
// open.
func applyAlarms(open []*alarmPeriod, rows []decodedRow) []*alarmPeriod {
	active := make(map[string]*alarmPeriod, len(open))
	for _, a := range open {
		active[a.Kind] = a
	}
	var touched []*alarmPeriod

	for _, r := range rows {
		for _, kind := range r.Reported {
			a, isActive := active[kind]
			on := slices.Contains(r.Raised, kind)
			switch {
			case on && !isActive:
				a = &alarmPeriod{Kind: kind, RaisedAt: r.ReceivedAt, RaisedUplinkID: r.UplinkID}
				active[kind] = a
				touched = append(touched, a)
			case !on && isActive:
				at := r.ReceivedAt
				a.ClearedAt = &at
				a.ClearedUplinkID = r.UplinkID
				delete(active, kind)
				if a.ID != 0 {
					a.changed = true
					touched = append(touched, a)
				}
			}
		}
	}
	return touched
}

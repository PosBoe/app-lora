// Package decoder holds the payload decoders of the device models. A decoder
// is Go code that all organizations share; the admin assigns one to a device
// model (device:D3). README.md states how to add one.
package decoder

import (
	"errors"
	"slices"
	"sort"
)

// AlarmKind is a condition that an alarm flag names. The values are
// persisted in the uplinks and alarms tables, and the frontend names each
// one in Romanian and Russian.
type AlarmKind string

// The shared list of alarm kinds (device:G12).
const (
	LowBattery  AlarmKind = "low_battery"
	Tamper      AlarmKind = "tamper"
	Leak        AlarmKind = "leak"
	ReverseFlow AlarmKind = "reverse_flow"
	CableCut    AlarmKind = "cable_cut"
)

// AlarmKinds lists every alarm kind.
var AlarmKinds = []AlarmKind{LowBattery, Tamper, Leak, ReverseFlow, CableCut}

// Result is what a decoder gets from one payload.
type Result struct {
	// PulseCount is the cumulative pulse count, or nil when the payload
	// holds none.
	PulseCount *int64
	// Reported lists the alarm kinds that the payload tells about. An alarm
	// of a kind that is not reported stays as it is.
	Reported []AlarmKind
	// Raised lists the reported kinds whose flag is on.
	Raised []AlarmKind
}

// report records a kind as reported, and as raised when on is true.
func (r *Result) report(kind AlarmKind, on bool) {
	if !slices.Contains(r.Reported, kind) {
		r.Reported = append(r.Reported, kind)
	}
	if on && !slices.Contains(r.Raised, kind) {
		r.Raised = append(r.Raised, kind)
	}
}

// ErrMalformed means a payload that the decoder cannot read.
var ErrMalformed = errors.New("decoder: malformed payload")

// Decoder reads the payload format of one vendor's device.
type Decoder interface {
	// ID names the decoder in the database. It is persisted, so it never
	// changes.
	ID() string
	// Name is the name that the admin sees.
	Name() string
	// Decode reads one payload.
	Decode(payload []byte) (Result, error)
}

var registry = map[string]Decoder{}

func register(d Decoder) {
	registry[d.ID()] = d
}

// Get returns the decoder with id, and false when there is none.
func Get(id string) (Decoder, bool) {
	d, ok := registry[id]
	return d, ok
}

// All returns every decoder, sorted by name.
func All() []Decoder {
	out := make([]Decoder, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

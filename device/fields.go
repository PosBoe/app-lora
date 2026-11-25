package device

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxNameChars    = 200
	maxAddressChars = 500
	maxNoteChars    = 2000
	maxModelChars   = 100
)

// Meter types. They are persisted in devices.meter_type.
const (
	meterWater = "water"
	meterGas   = "gas"
)

var devEUIPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// devEUISeparators are the characters that a written DevEUI can hold between
// its bytes.
var devEUISeparators = strings.NewReplacer(":", "", "-", "", " ", "")

// normalizeDevEUI returns a DevEUI as 16 lower-case hex characters. It takes
// the forms that network servers and labels use: upper or lower case, with
// or without ":" or "-" between the bytes.
func normalizeDevEUI(s string) (string, error) {
	s = strings.ToLower(devEUISeparators.Replace(strings.TrimSpace(s)))
	if !devEUIPattern.MatchString(s) {
		return "", errDevEUIInvalid
	}
	return s, nil
}

// meterTypeAliases maps the written forms of a meter type to its stored
// value. A CSV file can give the type in English, Romanian, or Russian.
var meterTypeAliases = map[string]string{
	"water": meterWater, "apă": meterWater, "apa": meterWater, "вода": meterWater,
	"gas": meterGas, "gaz": meterGas, "газ": meterGas,
}

// normalizeMeterType returns the stored meter type, or "" for no type.
func normalizeMeterType(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", nil
	}
	t, ok := meterTypeAliases[s]
	if !ok {
		return "", errMeterTypeInvalid
	}
	return t, nil
}

// deviceFields are the fields of a device that a user writes.
type deviceFields struct {
	DevEUI        string
	Name          string
	DeviceModelID int64
	MeterType     string
	Address       string
	Note          string
	// GroupIDs are the groups of the device. A create or an update sets
	// exactly these.
	GroupIDs []int64
}

// normalize trims the fields and checks the rules of a device that a user
// writes in the UI. It returns a copy.
func (f deviceFields) normalize() (deviceFields, error) {
	f, err := f.normalizeOwn()
	if err != nil {
		return f, err
	}
	if f.DeviceModelID <= 0 {
		return f, errModelNotFound
	}
	f.GroupIDs = uniqueIDs(f.GroupIDs)
	return f, nil
}

// normalizeOwn checks the fields that a device holds itself, without its
// device model and groups. A CSV row names those, so the import checks them
// in its own way.
func (f deviceFields) normalizeOwn() (deviceFields, error) {
	var err error
	if f.DevEUI, err = normalizeDevEUI(f.DevEUI); err != nil {
		return f, err
	}
	f.Name = strings.TrimSpace(f.Name)
	switch {
	case f.Name == "":
		return f, errNameRequired
	case utf8.RuneCountInString(f.Name) > maxNameChars:
		return f, errNameTooLong
	}
	if f.MeterType, err = normalizeMeterType(f.MeterType); err != nil {
		return f, err
	}
	f.Address = strings.TrimSpace(f.Address)
	if utf8.RuneCountInString(f.Address) > maxAddressChars {
		return f, errAddressTooLong
	}
	f.Note = strings.TrimSpace(f.Note)
	if utf8.RuneCountInString(f.Note) > maxNoteChars {
		return f, errNoteTooLong
	}
	return f, nil
}

// uniqueIDs drops the ids that are not positive and the repeats, and keeps
// the order. It returns an empty slice, never nil, for SQL arrays.
func uniqueIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// normalizeModel trims a vendor and a model name and checks the rules.
func normalizeModel(vendor, model string) (string, string, error) {
	vendor = strings.TrimSpace(vendor)
	model = strings.TrimSpace(model)
	switch {
	case vendor == "":
		return "", "", errVendorRequired
	case model == "":
		return "", "", errModelRequired
	case utf8.RuneCountInString(vendor) > maxModelChars || utf8.RuneCountInString(model) > maxModelChars:
		return "", "", errModelTooLong
	}
	return vendor, model, nil
}

// searchPattern turns a search text into a LIKE pattern on devices.search.
// A text that is a DevEUI with separators loses them, so "70:B3:D5" finds
// "70b3d5…". The LIKE wildcards in the text match only themselves.
func searchPattern(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	if dense := devEUISeparators.Replace(q); dense != q && isHex(dense) {
		q = dense
	}
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	return "%" + q + "%"
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

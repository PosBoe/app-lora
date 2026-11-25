package device

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	// maxImportBytes bounds the body of an import request.
	maxImportBytes = 2 << 20
	// maxImportRows bounds the data rows of one file.
	maxImportRows = 5000
)

// The CSV columns. They are a contract with the files that users keep: see
// the CSV format in README.md.
const (
	colDevEUI  = "dev_eui"
	colName    = "name"
	colVendor  = "vendor"
	colModel   = "model"
	colType    = "type"
	colAddress = "address"
	colNote    = "note"
	colGroups  = "groups"
)

var (
	importColumns         = []string{colDevEUI, colName, colVendor, colModel, colType, colAddress, colNote, colGroups}
	requiredImportColumns = []string{colDevEUI, colName, colVendor, colModel}
)

// groupSeparator splits the group names of one row.
const groupSeparator = ";"

// importRow is one data row of a CSV file, checked on its own. Err is the
// first rule the row breaks, or nil. The other fields are normalized only
// when Err is nil.
type importRow struct {
	Line int
	// RawDevEUI and RawName are the cells as written, for a row with an error.
	RawDevEUI string
	RawName   string
	Fields    deviceFields
	Vendor    string
	Model     string
	Groups    []string
	Err       *ruleError
}

// parseCSV reads a device file: a header row, then up to maxImportRows data
// rows. It takes a UTF-8 BOM, and `,` or `;` as the delimiter, which it reads
// from the header row. A file-level problem is an error; a row-level problem
// is the Err of that row.
func parseCSV(data []byte) ([]importRow, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		return nil, errImportNotUTF8
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = delimiterOf(data)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if errors.Is(err, io.EOF) {
		return nil, errImportEmpty
	}
	if err != nil {
		return nil, errImportMalformed
	}
	cols, err := columnIndex(header)
	if err != nil {
		return nil, err
	}

	var rows []importRow
	seen := make(map[string]bool)
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errImportMalformed
		}
		if blank(record) {
			continue
		}
		if len(rows) == maxImportRows {
			return nil, errImportTooManyRows
		}
		line, _ := r.FieldPos(0)
		row := parseRow(line, record, cols)
		if row.Err == nil {
			if seen[row.Fields.DevEUI] {
				row.Err = errDuplicateInFile
			}
			seen[row.Fields.DevEUI] = true
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// delimiterOf returns `;` when the header row holds more `;` than `,`, and
// `,` otherwise. Excel writes `;` in the Romanian and Russian locales.
func delimiterOf(data []byte) rune {
	header, _, _ := bytes.Cut(data, []byte("\n"))
	if bytes.Count(header, []byte(";")) > bytes.Count(header, []byte(",")) {
		return ';'
	}
	return ','
}

// columnIndex maps each known column to its position. Names are
// case-insensitive and can come in any order.
func columnIndex(header []string) (map[string]int, error) {
	known := make(map[string]bool, len(importColumns))
	for _, c := range importColumns {
		known[c] = true
	}
	cols := make(map[string]int, len(header))
	for i, name := range header {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if _, dup := cols[name]; dup || !known[name] {
			return nil, errImportUnknownColumn
		}
		cols[name] = i
	}
	for _, c := range requiredImportColumns {
		if _, ok := cols[c]; !ok {
			return nil, errImportMissingColumn
		}
	}
	return cols, nil
}

func blank(record []string) bool {
	for _, v := range record {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

// parseRow checks one data row with the rules of a single device, and the
// rules of device model and group names.
func parseRow(line int, record []string, cols map[string]int) importRow {
	get := func(col string) string {
		i, ok := cols[col]
		if !ok || i >= len(record) {
			return ""
		}
		return record[i]
	}
	row := importRow{
		Line:      line,
		RawDevEUI: strings.TrimSpace(get(colDevEUI)),
		RawName:   strings.TrimSpace(get(colName)),
	}

	f, err := deviceFields{
		DevEUI:    get(colDevEUI),
		Name:      get(colName),
		MeterType: get(colType),
		Address:   get(colAddress),
		Note:      get(colNote),
	}.normalizeOwn()
	if err != nil {
		row.Err = asRule(err)
		return row
	}
	row.Fields = f

	if row.Vendor, row.Model, err = normalizeModel(get(colVendor), get(colModel)); err != nil {
		row.Err = asRule(err)
		return row
	}

	seen := make(map[string]bool)
	for _, name := range strings.Split(get(colGroups), groupSeparator) {
		if strings.TrimSpace(name) == "" {
			continue
		}
		name, err := normalizeGroupName(name)
		if err != nil {
			row.Err = asRule(err)
			return row
		}
		if key := strings.ToLower(name); !seen[key] {
			seen[key] = true
			row.Groups = append(row.Groups, name)
		}
	}
	return row
}

// asRule returns err as a rule error. The field rules return only rule
// errors, so another error is a bug; it becomes a generic row error.
func asRule(err error) *ruleError {
	if rerr, ok := errors.AsType[*ruleError](err); ok {
		return rerr
	}
	return errImportMalformed
}

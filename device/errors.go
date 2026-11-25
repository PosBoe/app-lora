package device

import (
	"errors"
	"net/http"

	"github.com/waneock/app-lora/pkg/httpkit"
)

// ruleError is a rule that a user can trip. A service returns it; a handler
// maps it to an HTTP status with toHTTP.
type ruleError struct {
	code    string
	message string
}

func (e *ruleError) Error() string { return e.code + ": " + e.message }

var (
	errDevEUIInvalid     = &ruleError{"dev_eui_invalid", "a DevEUI is 16 hex characters"}
	errDevEUITaken       = &ruleError{"dev_eui_taken", "another device of the organization has this DevEUI"}
	errDevEUIArchived    = &ruleError{"dev_eui_archived", "an archived device of the organization has this DevEUI"}
	errDevEUILocked      = &ruleError{"dev_eui_locked", "the DevEUI of a device with uplinks cannot change"}
	errNameRequired      = &ruleError{"device_name_required", "the device name is required"}
	errNameTooLong       = &ruleError{"device_name_too_long", "the device name has at most 200 characters"}
	errAddressTooLong    = &ruleError{"address_too_long", "the address has at most 500 characters"}
	errNoteTooLong       = &ruleError{"note_too_long", "the note has at most 2000 characters"}
	errMeterTypeInvalid  = &ruleError{"meter_type_invalid", "the meter type is water, gas, or empty"}
	errModelNotFound     = &ruleError{"model_not_found", "the device model is not in the organization"}
	errVendorRequired    = &ruleError{"vendor_required", "the vendor is required"}
	errModelRequired     = &ruleError{"model_name_required", "the model name is required"}
	errModelTooLong      = &ruleError{"model_too_long", "the vendor and the model name have at most 100 characters"}
	errModelTaken        = &ruleError{"model_taken", "the organization has this device model"}
	errModelInUse        = &ruleError{"model_in_use", "a device uses this device model"}
	errTooManyIDs        = &ruleError{"too_many_ids", "send at most 1000 ids"}
	errGroupNameRequired = &ruleError{"group_name_required", "the group name is required"}
	errGroupNameTooLong  = &ruleError{"group_name_too_long", "the group name has at most 100 characters"}
	errGroupTaken        = &ruleError{"group_taken", "the organization has a group with this name"}
	errGroupNotFound     = &ruleError{"group_not_found", "the group is not in the organization"}
	errNotFound          = &ruleError{"not_found", "not found"}

	errImportContentType   = &ruleError{"import_content_type", "send the file as text/csv"}
	errImportTooLarge      = &ruleError{"import_too_large", "the file has more than 2 MiB"}
	errImportNotUTF8       = &ruleError{"import_not_utf8", "save the file as CSV UTF-8"}
	errImportEmpty         = &ruleError{"import_empty", "the file has no header row"}
	errImportMalformed     = &ruleError{"import_malformed", "the file is not a valid CSV file"}
	errImportMissingColumn = &ruleError{"import_missing_column", "the header needs dev_eui, name, vendor, and model"}
	errImportUnknownColumn = &ruleError{"import_unknown_column", "the header has an unknown or repeated column"}
	errImportTooManyRows   = &ruleError{"import_too_many_rows", "the file has more than 5000 rows"}
	errImportRetry         = &ruleError{"import_retry", "the data changed during the import; try again"}
	errDuplicateInFile     = &ruleError{"duplicate_in_file", "an earlier row of the file has this DevEUI"}
)

// ruleStatus is the HTTP status of each rule error that is not 400.
var ruleStatus = map[*ruleError]int{
	errDevEUITaken:    http.StatusConflict,
	errDevEUIArchived: http.StatusConflict,
	errDevEUILocked:   http.StatusConflict,
	errModelTaken:     http.StatusConflict,
	errModelInUse:     http.StatusConflict,
	errGroupTaken:     http.StatusConflict,
	errGroupNotFound:  http.StatusNotFound,

	errImportContentType: http.StatusUnsupportedMediaType,
	errImportTooLarge:    http.StatusRequestEntityTooLarge,
	errImportRetry:       http.StatusConflict,
	errNotFound:          http.StatusNotFound,
}

// toHTTP turns a rule error into an *httpkit.Error. A rule error not in
// ruleStatus is a 400. It returns any other error unchanged, so the guard
// answers 500 for it.
func toHTTP(err error) error {
	rerr, ok := errors.AsType[*ruleError](err)
	if !ok {
		return err
	}
	status, ok := ruleStatus[rerr]
	if !ok {
		status = http.StatusBadRequest
	}
	return httpkit.NewError(status, rerr.code, rerr.message)
}

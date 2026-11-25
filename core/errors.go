package core

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
	errInvalidCredentials      = &ruleError{"invalid_credentials", "the login or the password is wrong"}
	errLoginLocked             = &ruleError{"login_locked", "too many failed logins; try again later"}
	errOrganizationDeactivated = &ruleError{"organization_deactivated", "the organization is deactivated"}
	errUnauthenticated         = &ruleError{"unauthenticated", "log in first"}
	errPasswordChangeRequired  = &ruleError{"password_change_required", "replace the temporary password first"}
	errForbidden               = &ruleError{"forbidden", "only an admin can do this"}
	errNotMember               = &ruleError{"forbidden", "only a member of an organization can do this"}
	errWrongCurrentPassword    = &ruleError{"invalid_current_password", "the current password is wrong"}
	errPasswordTooShort        = &ruleError{"password_too_short", "the password must have at least 8 characters"}
	errPasswordTooLong         = &ruleError{"password_too_long", "the password must have at most 72 bytes"}
	errLoginInvalid            = &ruleError{"login_invalid", "use 3 to 64 characters: a-z, 0-9, dot, dash, underscore"}
	errLoginTaken              = &ruleError{"login_taken", "another user has this login"}
	errNameRequired            = &ruleError{"name_required", "the organization name is required"}
	errNotFound                = &ruleError{"not_found", "not found"}
)

// ruleStatus is the HTTP status of each rule error.
var ruleStatus = map[*ruleError]int{
	errInvalidCredentials:      http.StatusBadRequest,
	errLoginLocked:             http.StatusTooManyRequests,
	errOrganizationDeactivated: http.StatusForbidden,
	errUnauthenticated:         http.StatusUnauthorized,
	errPasswordChangeRequired:  http.StatusForbidden,
	errForbidden:               http.StatusForbidden,
	errNotMember:               http.StatusForbidden,
	errWrongCurrentPassword:    http.StatusBadRequest,
	errPasswordTooShort:        http.StatusBadRequest,
	errPasswordTooLong:         http.StatusBadRequest,
	errLoginInvalid:            http.StatusBadRequest,
	errLoginTaken:              http.StatusConflict,
	errNameRequired:            http.StatusBadRequest,
	errNotFound:                http.StatusNotFound,
}

// toHTTP turns a rule error into an *httpkit.Error. It returns any other
// error unchanged, so httpkit.WriteError answers 500 for it.
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

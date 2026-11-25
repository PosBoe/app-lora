package device

import (
	"context"
	"errors"
	"net/http"

	"github.com/waneock/app-lora/core"
)

// guard lets a request reach a device route. *core.Guard is the production
// guard; it puts a core.Caller in the request context.
type guard interface {
	Member(fn func(w http.ResponseWriter, req *http.Request) error) http.Handler
}

// errNoCaller means a route ran outside a guard. It is a wiring bug, so the
// client gets 500.
var errNoCaller = errors.New("device: no caller in the request context")

// organizationOf returns the organization of the caller.
func organizationOf(ctx context.Context) (int64, error) {
	c, ok := core.CallerFrom(ctx)
	if !ok || c.OrganizationID == 0 {
		return 0, errNoCaller
	}
	return c.OrganizationID, nil
}

// apiFunc is a device route. It gets the organization of the caller and
// returns its error.
type apiFunc func(w http.ResponseWriter, req *http.Request, orgID int64) error

// member wraps a device route with the guard. It reads the organization and
// maps the rule errors to their HTTP status.
func member(g guard, fn apiFunc) http.Handler {
	return g.Member(func(w http.ResponseWriter, req *http.Request) error {
		orgID, err := organizationOf(req.Context())
		if err != nil {
			return err
		}
		return toHTTP(fn(w, req, orgID))
	})
}

package core

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/waneock/app-lora/pkg/appconf"
	"github.com/waneock/app-lora/pkg/httpkit"
)

// sessionCookieName is the name of the session cookie. It is persisted in
// the browsers of the users, so a rename logs everybody out.
const sessionCookieName = "lora_session"

// apiFunc is a handler that returns its error. The middleware writes the
// error with httpkit.WriteError.
type apiFunc func(w http.ResponseWriter, req *http.Request) error

type principalKey struct{}

// principalFrom returns the principal that the middleware put in ctx.
func principalFrom(ctx context.Context) *principal {
	p, _ := ctx.Value(principalKey{}).(*principal)
	return p
}

// access is what a route requires of its caller.
type access int

const (
	// accessPublic needs no session.
	accessPublic access = iota
	// accessTemporary needs a session, also with a temporary password.
	accessTemporary
	// accessUser needs a session with a password the user chose.
	accessUser
	// accessMember is accessUser for a member of an organization.
	accessMember
	// accessAdmin needs an admin session with a password the admin chose.
	accessAdmin
)

// Guard resolves the session cookie of a request and enforces the access of
// a route. Other contexts wrap their routes with Member.
type Guard struct {
	auth   *authService
	cookie cookieSettings
	logger *slog.Logger
}

// cookieSettings are the attributes of the session cookie.
type cookieSettings struct {
	secure bool
	ttl    time.Duration
}

func newGuard(auth *authService, conf *appconf.Config, logger *slog.Logger) *Guard {
	return &Guard{
		auth: auth,
		cookie: cookieSettings{
			secure: strings.HasPrefix(conf.Site.URL, "https://"),
			ttl:    conf.Auth.SessionTTL,
		},
		logger: logger,
	}
}

// Member returns a handler that lets in only a member of an organization
// whose password is not temporary. fn reads the member with CallerFrom. A
// returned *httpkit.Error gives its status; any other error answers 500.
func (m *Guard) Member(fn func(w http.ResponseWriter, req *http.Request) error) http.Handler {
	return m.wrap(accessMember, fn)
}

// Admin returns a handler that lets in only an admin whose password is not
// temporary.
func (m *Guard) Admin(fn func(w http.ResponseWriter, req *http.Request) error) http.Handler {
	return m.wrap(accessAdmin, fn)
}

// Caller is the user of a request, for a context outside core.
type Caller struct {
	UserID int64
	// OrganizationID is 0 for an admin.
	OrganizationID int64
	IsAdmin        bool
}

type callerKey struct{}

// CallerFrom returns the user that a Guard handler put in ctx. It reports
// false outside such a handler.
func CallerFrom(ctx context.Context) (Caller, bool) {
	c, ok := ctx.Value(callerKey{}).(Caller)
	return c, ok
}

// ContextWithCaller puts c in ctx, as a Guard handler does. A test of another
// context uses it in place of a session.
func ContextWithCaller(ctx context.Context, c Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, c)
}

func callerOf(p *principal) Caller {
	c := Caller{UserID: p.ID, IsAdmin: p.IsAdmin}
	if p.Organization != nil {
		c.OrganizationID = p.Organization.ID
	}
	return c
}

// wrap returns an http.Handler that checks the access level, puts the
// principal in the request context, and writes the error of fn.
func (m *Guard) wrap(level access, fn apiFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if level != accessPublic {
			p, err := m.authenticate(w, req)
			if err == nil {
				err = checkAccess(level, p)
			}
			if err != nil {
				httpkit.WriteError(w, req, m.logger, toHTTP(err))
				return
			}
			ctx := context.WithValue(req.Context(), principalKey{}, p)
			req = req.WithContext(ContextWithCaller(ctx, callerOf(p)))
		}

		if err := fn(w, req); err != nil {
			httpkit.WriteError(w, req, m.logger, toHTTP(err))
		}
	})
}

// authenticate resolves the session cookie. It sends the cookie again when
// the session expiry moved, and clears a cookie that names no live session.
func (m *Guard) authenticate(w http.ResponseWriter, req *http.Request) (*principal, error) {
	cookie, err := req.Cookie(sessionCookieName)
	if err != nil {
		return nil, errUnauthenticated
	}
	p, extended, err := m.auth.resolve(req.Context(), cookie.Value)
	if errors.Is(err, errUnauthenticated) || errors.Is(err, errOrganizationDeactivated) {
		m.clearCookie(w)
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if extended {
		m.setCookie(w, cookie.Value)
	}
	return p, nil
}

func checkAccess(level access, p *principal) error {
	if level >= accessUser && p.PasswordTemporary {
		return errPasswordChangeRequired
	}
	if level == accessAdmin && !p.IsAdmin {
		return errForbidden
	}
	if level == accessMember && p.Organization == nil {
		return errNotMember
	}
	return nil
}

// setCookie sends the session cookie. Secure follows the scheme of site.url,
// because a browser drops a Secure cookie that a plain http dev server sets.
// HttpOnly and SameSite are always set.
func (m *Guard) setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure follows site.url; see the doc comment
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(m.cookie.ttl.Seconds()),
		HttpOnly: true,
		Secure:   m.cookie.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Guard) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure follows site.url; see setCookie
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.cookie.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

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
	// accessAdmin needs an admin session with a password the admin chose.
	accessAdmin
)

// authMiddleware resolves the session cookie of a request and enforces the
// access of a route.
type authMiddleware struct {
	auth   *authService
	cookie cookieSettings
	logger *slog.Logger
}

// cookieSettings are the attributes of the session cookie.
type cookieSettings struct {
	secure bool
	ttl    time.Duration
}

func newAuthMiddleware(auth *authService, conf *appconf.Config, logger *slog.Logger) *authMiddleware {
	return &authMiddleware{
		auth: auth,
		cookie: cookieSettings{
			secure: strings.HasPrefix(conf.Site.URL, "https://"),
			ttl:    conf.Auth.SessionTTL,
		},
		logger: logger,
	}
}

// wrap returns an http.Handler that checks the access level, puts the
// principal in the request context, and writes the error of fn.
func (m *authMiddleware) wrap(level access, fn apiFunc) http.Handler {
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
			req = req.WithContext(context.WithValue(req.Context(), principalKey{}, p))
		}

		if err := fn(w, req); err != nil {
			httpkit.WriteError(w, req, m.logger, toHTTP(err))
		}
	})
}

// authenticate resolves the session cookie. It sends the cookie again when
// the session expiry moved, and clears a cookie that names no live session.
func (m *authMiddleware) authenticate(w http.ResponseWriter, req *http.Request) (*principal, error) {
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
	return nil
}

// setCookie sends the session cookie. Secure follows the scheme of site.url,
// because a browser drops a Secure cookie that a plain http dev server sets.
// HttpOnly and SameSite are always set.
func (m *authMiddleware) setCookie(w http.ResponseWriter, token string) {
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

func (m *authMiddleware) clearCookie(w http.ResponseWriter) {
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

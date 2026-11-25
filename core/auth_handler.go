package core

import (
	"net/http"

	"github.com/waneock/app-lora/pkg/httpkit"
)

type authHandler struct {
	auth       *authService
	middleware *authMiddleware
}

func newAuthHandler(auth *authService, middleware *authMiddleware) *authHandler {
	return &authHandler{auth: auth, middleware: middleware}
}

func registerAuthHandler(mux *http.ServeMux, h *authHandler) {
	m := h.middleware
	mux.Handle("POST /api/v1/auth/login", m.wrap(accessPublic, h.login))
	mux.Handle("POST /api/v1/auth/logout", m.wrap(accessTemporary, h.logout))
	mux.Handle("GET /api/v1/auth/me", m.wrap(accessTemporary, h.me))
	mux.Handle("POST /api/v1/auth/password", m.wrap(accessTemporary, h.password))
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// meResponse is the logged-in user. The frontend mirrors it in
// vue/src/use/session.ts.
type meResponse struct {
	ID                int64                    `json:"id"`
	Login             string                   `json:"login"`
	IsAdmin           bool                     `json:"is_admin"`
	PasswordTemporary bool                     `json:"password_temporary"`
	Language          string                   `json:"language"`
	Organization      *organizationRefResponse `json:"organization"`
}

type organizationRefResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func newMeResponse(u currentUser) meResponse {
	resp := meResponse{
		ID:                u.ID,
		Login:             u.Login,
		IsAdmin:           u.IsAdmin,
		PasswordTemporary: u.PasswordTemporary,
		Language:          u.Language,
	}
	if u.Organization != nil {
		resp.Organization = &organizationRefResponse{ID: u.Organization.ID, Name: u.Organization.Name}
	}
	return resp
}

func (h *authHandler) login(w http.ResponseWriter, req *http.Request) error {
	var in loginRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	token, p, err := h.auth.login(req.Context(), in.Login, in.Password)
	if err != nil {
		return err
	}
	h.middleware.setCookie(w, token)
	httpkit.JSON(w, http.StatusOK, newMeResponse(p.currentUser))
	return nil
}

func (h *authHandler) logout(w http.ResponseWriter, req *http.Request) error {
	if err := h.auth.logout(req.Context(), principalFrom(req.Context())); err != nil {
		return err
	}
	h.middleware.clearCookie(w)
	httpkit.NoContent(w)
	return nil
}

func (h *authHandler) me(w http.ResponseWriter, req *http.Request) error {
	httpkit.JSON(w, http.StatusOK, newMeResponse(principalFrom(req.Context()).currentUser))
	return nil
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *authHandler) password(w http.ResponseWriter, req *http.Request) error {
	var in passwordRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	p := principalFrom(req.Context())
	if err := h.auth.changePassword(req.Context(), p, in.CurrentPassword, in.NewPassword); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

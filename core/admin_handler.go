package core

import (
	"net/http"
	"strconv"
	"time"

	"github.com/waneock/app-lora/pkg/httpkit"
)

type adminHandler struct {
	admin      *adminService
	middleware *Guard
}

func newAdminHandler(admin *adminService, middleware *Guard) *adminHandler {
	return &adminHandler{admin: admin, middleware: middleware}
}

func registerAdminHandler(mux *http.ServeMux, h *adminHandler) {
	m := h.middleware
	mux.Handle("GET /api/v1/admin/organizations", m.wrap(accessAdmin, h.list))
	mux.Handle("POST /api/v1/admin/organizations", m.wrap(accessAdmin, h.create))
	mux.Handle("POST /api/v1/admin/organizations/{id}/deactivate", m.wrap(accessAdmin, h.deactivate))
	mux.Handle("POST /api/v1/admin/organizations/{id}/reactivate", m.wrap(accessAdmin, h.reactivate))
	mux.Handle("POST /api/v1/admin/users/{id}/reset-password", m.wrap(accessAdmin, h.resetPassword))
}

// organizationResponse is an organization in the admin list. The frontend
// mirrors it in vue/src/core/use-organizations.ts.
type organizationResponse struct {
	ID            int64            `json:"id"`
	Name          string           `json:"name"`
	DeactivatedAt *time.Time       `json:"deactivated_at"`
	CreatedAt     time.Time        `json:"created_at"`
	DeviceCount   int              `json:"device_count"`
	LastUplinkAt  *time.Time       `json:"last_uplink_at"`
	Users         []memberResponse `json:"users"`
}

type memberResponse struct {
	ID                int64      `json:"id"`
	Login             string     `json:"login"`
	PasswordTemporary bool       `json:"password_temporary"`
	FirstName         *string    `json:"first_name"`
	LastName          *string    `json:"last_name"`
	Email             *string    `json:"email"`
	Phone             *string    `json:"phone"`
	LastLoginAt       *time.Time `json:"last_login_at"`
}

func newOrganizationResponse(om *organizationMembers) organizationResponse {
	resp := organizationResponse{
		ID:            om.organization.ID,
		Name:          om.organization.Name,
		DeactivatedAt: om.organization.DeactivatedAt,
		CreatedAt:     om.organization.CreatedAt,
		DeviceCount:   om.deviceCount,
		LastUplinkAt:  om.lastUplinkAt,
		Users:         make([]memberResponse, len(om.users)),
	}
	for i, u := range om.users {
		resp.Users[i] = memberResponse{
			ID:                u.ID,
			Login:             u.Login,
			PasswordTemporary: u.PasswordTemporary,
			FirstName:         u.FirstName,
			LastName:          u.LastName,
			Email:             u.Email,
			Phone:             u.Phone,
			LastLoginAt:       u.LastLoginAt,
		}
	}
	return resp
}

type listOrganizationsResponse struct {
	Organizations []organizationResponse `json:"organizations"`
}

func (h *adminHandler) list(w http.ResponseWriter, req *http.Request) error {
	orgs, err := h.admin.listOrganizations(req.Context())
	if err != nil {
		return err
	}
	resp := listOrganizationsResponse{Organizations: make([]organizationResponse, len(orgs))}
	for i, om := range orgs {
		resp.Organizations[i] = newOrganizationResponse(om)
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

type createOrganizationRequest struct {
	Name  string `json:"name"`
	Login string `json:"login"`
}

type createOrganizationResponse struct {
	Organization      organizationResponse `json:"organization"`
	TemporaryPassword string               `json:"temporary_password"`
}

func (h *adminHandler) create(w http.ResponseWriter, req *http.Request) error {
	var in createOrganizationRequest
	if err := httpkit.DecodeJSON(w, req, &in); err != nil {
		return err
	}
	om, password, err := h.admin.createOrganization(req.Context(), in.Name, in.Login)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusCreated, createOrganizationResponse{
		Organization:      newOrganizationResponse(om),
		TemporaryPassword: password,
	})
	return nil
}

func (h *adminHandler) deactivate(w http.ResponseWriter, req *http.Request) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if err := h.admin.deactivate(req.Context(), id); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

func (h *adminHandler) reactivate(w http.ResponseWriter, req *http.Request) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if err := h.admin.reactivate(req.Context(), id); err != nil {
		return err
	}
	httpkit.NoContent(w)
	return nil
}

type resetPasswordResponse struct {
	TemporaryPassword string `json:"temporary_password"`
}

func (h *adminHandler) resetPassword(w http.ResponseWriter, req *http.Request) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	password, err := h.admin.resetPassword(req.Context(), id)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, resetPasswordResponse{TemporaryPassword: password})
	return nil
}

// pathID reads the {id} path value. A value that is not a positive integer
// names nothing, so it answers errNotFound.
func pathID(req *http.Request) (int64, error) {
	id, err := strconv.ParseInt(req.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errNotFound
	}
	return id, nil
}

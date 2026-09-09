package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

var knownPermissions = map[string]bool{
	"record_attendance": true,
	"view_reports":      true,
	"manage_schedules":  true,
	"manage_users":      true,
}

func (a *API) roles(w http.ResponseWriter, r *http.Request) {
	if !a.requireSystemPermission(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		roles, err := a.repo.ListRoles(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, roles)
		return
	}
	var role Role
	if !decode(w, r, &role) {
		return
	}
	role.Key = newRoleKey()
	role.Scope = "organization"
	role.IsSystem = false
	if err := validEditableRole(role); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if err := a.repo.CreateRole(r.Context(), &role); err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	respond(w, http.StatusCreated, role)
}

func (a *API) role(w http.ResponseWriter, r *http.Request) {
	if !a.requireSystemPermission(w, r) {
		return
	}
	id := r.PathValue("id")
	existing, err := a.repo.GetRole(r.Context(), id)
	if notFound(w, err) {
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if r.Method == http.MethodGet {
		respond(w, http.StatusOK, existing)
		return
	}
	if existing.IsSystem {
		fail(w, http.StatusForbidden, errors.New("System Administrator is built in and cannot be changed"))
		return
	}
	if r.Method == http.MethodDelete {
		if existing.AssignmentCount > 0 {
			fail(w, http.StatusConflict, errors.New("reassign or remove users before deleting this role"))
			return
		}
		if err := a.repo.DeleteRole(r.Context(), id); err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var role Role
	if !decode(w, r, &role) {
		return
	}
	role.ID, role.Key, role.Scope, role.IsSystem = existing.ID, existing.Key, existing.Scope, false
	if err := validEditableRole(role); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if err := a.repo.UpdateRole(r.Context(), &role); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	role.AssignmentCount = existing.AssignmentCount
	respond(w, http.StatusOK, role)
}

func validEditableRole(role Role) error {
	if strings.TrimSpace(role.Name) == "" {
		return errors.New("role name is required")
	}
	if role.Scope != "organization" || role.IsSystem {
		return errors.New("only organization-scoped roles can be created or edited")
	}
	seen := map[string]bool{}
	for _, permission := range role.Permissions {
		if !knownPermissions[permission] {
			return errors.New("unknown permission: " + permission)
		}
		if seen[permission] {
			return errors.New("permissions must be unique")
		}
		seen[permission] = true
	}
	return nil
}

func newRoleKey() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "custom-role"
	}
	return "custom-" + hex.EncodeToString(b)
}

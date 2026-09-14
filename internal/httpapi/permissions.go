package httpapi

import (
	"errors"
	"net/http"
	"parishattendance/internal/auth"
	"strings"
)

// Authentication establishes the Cognito subject; these checks then enforce
// the application's system and parish roles.
func (a *API) requireSystemPermission(w http.ResponseWriter, r *http.Request) bool {
	return a.requireParishPermission(w, r, "", "system")
}

func (a *API) requireParishPermission(w http.ResponseWriter, r *http.Request, parishID, permission string) bool {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		fail(w, http.StatusUnauthorized, errors.New("authentication required"))
		return false
	}
	userID := principal.Subject
	roles, err := a.repo.UserRoles(r.Context(), userID, parishID)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return false
	}
	for _, role := range roles {
		if role.Role == "system_administrator" {
			return true
		}
		if permission == "system" || role.ParishID == nil || *role.ParishID != parishID {
			continue
		}
		permissions, err := a.repo.RolePermissions(r.Context(), role.RoleID)
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return false
		}
		for _, granted := range permissions {
			if granted == permission {
				return true
			}
		}
	}
	fail(w, http.StatusForbidden, errors.New("your assigned role does not allow this action"))
	return false
}

func (a *API) prepareUserAccess(r *http.Request, access *UserAccess) error {
	if strings.TrimSpace(access.UserID) == "" {
		return errors.New("userId is required")
	}
	if strings.TrimSpace(access.RoleID) == "" {
		return errors.New("roleId is required")
	}
	role, err := a.repo.GetRole(r.Context(), access.RoleID)
	if err != nil {
		return errors.New("roleId must reference an existing role")
	}
	if role.IsSystem {
		if access.ParishID != nil {
			return errors.New("System Administrator cannot be scoped to a parish")
		}
	} else if access.ParishID == nil || strings.TrimSpace(*access.ParishID) == "" {
		return errors.New("parishId is required for parish roles")
	}
	access.Role, access.RoleName = role.Key, role.Name
	return nil
}

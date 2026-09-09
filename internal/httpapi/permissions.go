package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

// Authentication is intentionally separate from authorization. Until an
// identity provider is configured, an absent X-User-ID keeps local demo use
// open. When the header is present, these role checks are enforced.
func (a *API) requireSystemPermission(w http.ResponseWriter, r *http.Request) bool {
	return a.requireOrganizationPermission(w, r, "", "system")
}

func (a *API) requireOrganizationPermission(w http.ResponseWriter, r *http.Request, organizationID, permission string) bool {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return true
	}
	roles, err := a.repo.UserRoles(r.Context(), userID, organizationID)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return false
	}
	for _, role := range roles {
		if role.Role == "system_administrator" {
			return true
		}
		if permission == "system" || role.OrganizationID == nil || *role.OrganizationID != organizationID {
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
		if access.OrganizationID != nil {
			return errors.New("System Administrator cannot be scoped to an organization")
		}
	} else if access.OrganizationID == nil || strings.TrimSpace(*access.OrganizationID) == "" {
		return errors.New("organizationId is required for organization roles")
	}
	access.Role, access.RoleName = role.Key, role.Name
	return nil
}

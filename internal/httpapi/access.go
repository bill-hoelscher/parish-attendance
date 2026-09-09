package httpapi

import (
	"errors"
	"net/http"
)

// requireWritableOrganization is the server-side authorization point for
// schedule and attendance changes. UI controls may be disabled for clarity,
// but callers cannot bypass this check by calling the API directly.
func (a *API) requireWritableOrganization(w http.ResponseWriter, r *http.Request, organizationID, permission string) bool {
	if !a.requireOrganizationPermission(w, r, organizationID, permission) {
		return false
	}
	access, err := a.repo.OrganizationAccess(r.Context(), organizationID)
	if notFound(w, err) {
		return false
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return false
	}
	if !access.CanWrite {
		fail(w, http.StatusForbidden, errors.New("organization is inactive or its subscription does not allow updates"))
		return false
	}
	return true
}

func (a *API) requireWritableResource(w http.ResponseWriter, r *http.Request, resource, id, permission string) (string, bool) {
	organizationID, err := a.repo.ResourceOrganizationID(r.Context(), resource, id)
	if notFound(w, err) {
		return "", false
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return "", false
	}
	return organizationID, a.requireWritableOrganization(w, r, organizationID, permission)
}

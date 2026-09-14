package httpapi

import (
	"errors"
	"net/http"
)

// requireWritableParish is the server-side authorization point for
// schedule and attendance changes. UI controls may be disabled for clarity,
// but callers cannot bypass this check by calling the API directly.
func (a *API) requireWritableParish(w http.ResponseWriter, r *http.Request, parishID, permission string) bool {
	if !a.requireParishPermission(w, r, parishID, permission) {
		return false
	}
	access, err := a.repo.ParishAccess(r.Context(), parishID)
	if notFound(w, err) {
		return false
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return false
	}
	if !access.CanWrite {
		fail(w, http.StatusForbidden, errors.New("parish is inactive or its subscription does not allow updates"))
		return false
	}
	return true
}

func (a *API) requireWritableResource(w http.ResponseWriter, r *http.Request, resource, id, permission string) (string, bool) {
	parishID, err := a.repo.ResourceParishID(r.Context(), resource, id)
	if notFound(w, err) {
		return "", false
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return "", false
	}
	return parishID, a.requireWritableParish(w, r, parishID, permission)
}

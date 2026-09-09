package httpapi

import "net/http"

func (a *API) userAccess(w http.ResponseWriter, r *http.Request) {
	organizationID := r.URL.Query().Get("organizationId")
	if r.Method == http.MethodGet {
		if organizationID == "" {
			if !a.requireSystemPermission(w, r) {
				return
			}
		} else if !a.requireOrganizationPermission(w, r, organizationID, "manage_users") {
			return
		}
		access, err := a.repo.ListUserAccess(r.Context(), organizationID)
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, access)
		return
	}
	var access UserAccess
	if !decode(w, r, &access) {
		return
	}
	if err := a.prepareUserAccess(r, &access); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if !a.requireUserAccessManagement(w, r, access) {
		return
	}
	if err := a.repo.CreateUserAccess(r.Context(), &access); err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	respond(w, http.StatusCreated, access)
}

func (a *API) userAccessItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := a.repo.GetUserAccess(r.Context(), id)
	if notFound(w, err) {
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if r.Method == http.MethodDelete {
		if !a.requireUserAccessManagement(w, r, *existing) {
			return
		}
		if err := a.repo.DeleteUserAccess(r.Context(), id); err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var access UserAccess
	if !decode(w, r, &access) {
		return
	}
	if err := a.prepareUserAccess(r, &access); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if !a.requireUserAccessManagement(w, r, *existing) || !a.requireUserAccessManagement(w, r, access) {
		return
	}
	access.ID = id
	if err := a.repo.UpdateUserAccess(r.Context(), &access); err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	respond(w, http.StatusOK, access)
}

func (a *API) requireUserAccessManagement(w http.ResponseWriter, r *http.Request, access UserAccess) bool {
	if access.Role == "system_administrator" {
		return a.requireSystemPermission(w, r)
	}
	return a.requireOrganizationPermission(w, r, *access.OrganizationID, "manage_users")
}

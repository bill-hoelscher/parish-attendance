package httpapi

import (
	"errors"
	"net/http"
)

func (a *API) userAccess(w http.ResponseWriter, r *http.Request) {
	parishID := r.URL.Query().Get("parishId")
	if r.Method == http.MethodGet {
		if parishID == "" {
			if !a.requireSystemPermission(w, r) {
				return
			}
		} else if !a.requireParishPermission(w, r, parishID, "manage_users") {
			return
		}
		access, err := a.repo.ListUserAccess(r.Context(), parishID)
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		for index := range access {
			if access[index].Email != "" || a.inviter == nil {
			} else if email, lookupErr := a.inviter.Email(r.Context(), access[index].UserID); lookupErr == nil {
				access[index].Email = email
			}
			if access[index].Email == "" || a.inviter == nil {
				access[index].Status = "Unknown"
				continue
			}
			if status, statusErr := a.inviter.Status(r.Context(), access[index].Email); statusErr == nil {
				access[index].Status = status
			} else {
				access[index].Status = "Unknown"
			}
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

func (a *API) disableUserAccess(w http.ResponseWriter, r *http.Request) {
	existing, err := a.repo.GetUserAccess(r.Context(), r.PathValue("id"))
	if notFound(w, err) {
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if !a.requireUserAccessManagement(w, r, *existing) {
		return
	}
	if a.inviter == nil {
		fail(w, http.StatusServiceUnavailable, errors.New("user management is unavailable"))
		return
	}
	email := existing.Email
	if email == "" {
		email, err = a.inviter.Email(r.Context(), existing.UserID)
		if err != nil {
			fail(w, http.StatusBadGateway, err)
			return
		}
	}
	if err := a.inviter.Disable(r.Context(), email); err != nil {
		fail(w, http.StatusBadGateway, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	return a.requireParishPermission(w, r, *access.ParishID, "manage_users")
}

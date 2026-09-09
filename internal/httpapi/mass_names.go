package httpapi

import (
	"errors"
	"net/http"
)

func (a *API) massNames(w http.ResponseWriter, r *http.Request) {
	organizationID := r.PathValue("id")
	if r.Method == http.MethodGet {
		x, err := a.repo.ListMassNames(r.Context(), organizationID)
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, x)
		return
	}
	if !a.requireWritableOrganization(w, r, organizationID, "manage_schedules") {
		return
	}
	var x MassName
	if !decode(w, r, &x) {
		return
	}
	if x.Name == "" {
		fail(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	x.OrganizationID = organizationID
	if e := a.repo.CreateMassName(r.Context(), &x); e != nil {
		fail(w, http.StatusInternalServerError, e)
		return
	}
	respond(w, http.StatusCreated, x)
}

func (a *API) massName(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.requireWritableResource(w, r, "mass_name", id, "manage_schedules"); !ok {
		return
	}
	if r.Method == http.MethodDelete {
		if err := a.repo.DeleteMassName(r.Context(), id); notFound(w, err) {
			return
		} else if err != nil {
			fail(w, http.StatusConflict, errors.New("a Mass name in use cannot be deleted; retire it instead"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var x MassName
	if !decode(w, r, &x) {
		return
	}
	if x.Name == "" {
		fail(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	x.ID = id
	if err := a.repo.UpdateMassName(r.Context(), &x); notFound(w, err) {
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, x)
}

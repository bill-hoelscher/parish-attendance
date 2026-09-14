package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

func (a *API) dioceses(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		x, err := a.repo.ListDioceses(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, x)
		return
	}
	if !a.requireSystemPermission(w, r) {
		return
	}
	var x Diocese
	if !decode(w, r, &x) {
		return
	}
	if strings.TrimSpace(x.Name) == "" {
		fail(w, http.StatusBadRequest, errors.New("diocese name is required"))
		return
	}
	x.Name = strings.TrimSpace(x.Name)
	if err := a.repo.CreateDiocese(r.Context(), &x); err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	respond(w, http.StatusCreated, x)
}

func (a *API) diocese(w http.ResponseWriter, r *http.Request) {
	if !a.requireSystemPermission(w, r) {
		return
	}
	id := r.PathValue("id")
	if r.Method == http.MethodGet {
		x, err := a.repo.GetDiocese(r.Context(), id)
		if notFound(w, err) {
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusOK, x)
		return
	}
	if r.Method == http.MethodDelete {
		parishes, err := a.repo.ListParishes(r.Context())
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		for _, parish := range parishes {
			if parish.DioceseID != nil && *parish.DioceseID == id {
				fail(w, http.StatusConflict, errors.New("move or remove this diocese's parishes before deleting it"))
				return
			}
		}
		if err := a.repo.DeleteDiocese(r.Context(), id); notFound(w, err) {
			return
		} else if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var x Diocese
	if !decode(w, r, &x) {
		return
	}
	if strings.TrimSpace(x.Name) == "" {
		fail(w, http.StatusBadRequest, errors.New("diocese name is required"))
		return
	}
	x.ID, x.Name = id, strings.TrimSpace(x.Name)
	if err := a.repo.UpdateDiocese(r.Context(), &x); notFound(w, err) {
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, x)
}

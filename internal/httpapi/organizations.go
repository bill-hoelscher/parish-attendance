package httpapi

import (
	"errors"
	"net/http"
	"parishattendance/internal/auth"
	"strings"
)

func (a *API) parishes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var (
			x []Parish
			e error
		)
		principal, ok := auth.PrincipalFrom(r.Context())
		if !ok {
			fail(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		x, e = a.repo.ListAuthorizedParishes(r.Context(), principal.Subject)
		if e != nil {
			fail(w, 500, e)
			return
		}
		respond(w, 200, x)
		return
	}
	if !a.requireSystemPermission(w, r) {
		return
	}
	var x Parish
	if !decode(w, r, &x) {
		return
	}
	if x.Timezone == "" {
		x.Timezone = "America/Chicago"
	}
	// New parishes begin active. They may be deactivated later while
	// preserving all historical schedules, attendance, and reports.
	x.IsActive = true
	if err := validParish(x); err != nil {
		fail(w, 400, err)
		return
	}
	if x.DioceseID != nil && strings.TrimSpace(*x.DioceseID) != "" {
		if _, err := a.repo.GetDiocese(r.Context(), *x.DioceseID); err != nil {
			fail(w, http.StatusBadRequest, errors.New("dioceseId must reference an existing Diocese"))
			return
		}
	}
	if e := a.repo.CreateParish(r.Context(), &x); e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 201, x)
}
func (a *API) parish(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	switch r.Method {
	case http.MethodGet:
		if !a.requireParishPermission(w, r, id, "view_reports") {
			return
		}
		x, e := a.repo.GetParish(r.Context(), id)
		if notFound(w, e) {
			return
		}
		if e != nil {
			fail(w, 500, e)
			return
		}
		respond(w, 200, x)
	case http.MethodPatch:
		if !a.requireSystemPermission(w, r) {
			return
		}
		var x Parish
		if !decode(w, r, &x) {
			return
		}
		if x.Timezone == "" {
			x.Timezone = "America/Chicago"
		}
		if err := validParish(x); err != nil {
			fail(w, 400, err)
			return
		}
		if x.DioceseID != nil && strings.TrimSpace(*x.DioceseID) != "" {
			if _, err := a.repo.GetDiocese(r.Context(), *x.DioceseID); err != nil {
				fail(w, http.StatusBadRequest, errors.New("dioceseId must reference an existing Diocese"))
				return
			}
		}
		x.ID = id
		if e := a.repo.UpdateParish(r.Context(), &x); notFound(w, e) {
			return
		} else if e != nil {
			fail(w, 500, e)
			return
		}
		respond(w, 200, x)
	case http.MethodDelete:
		if !a.requireSystemPermission(w, r) {
			return
		}
		if e := a.repo.DeleteParish(r.Context(), id); notFound(w, e) {
			return
		} else if e != nil {
			fail(w, 500, e)
			return
		}
		w.WriteHeader(204)
	}
}

func validParish(x Parish) error {
	required := []struct {
		name  string
		value string
	}{
		{"parish name", x.Name},
		{"street address", x.StreetAddress},
		{"city", x.City},
		{"state or province", x.StateProvince},
		{"postal code", x.PostalCode},
		{"time zone", x.Timezone},
		{"contact name", x.ContactName},
		{"contact email", x.ContactEmail},
		{"contact phone", x.ContactPhone},
	}
	missing := make([]string, 0)
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) > 0 {
		return errors.New(strings.Join(missing, ", ") + " are required")
	}
	if x.BillingAccountID == nil || strings.TrimSpace(*x.BillingAccountID) == "" {
		return errors.New("billing account is required")
	}
	return nil
}

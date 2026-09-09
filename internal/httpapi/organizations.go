package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

func (a *API) organizations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var (
			x []Organization
			e error
		)
		if userID := r.Header.Get("X-User-ID"); userID != "" {
			x, e = a.repo.ListAuthorizedOrganizations(r.Context(), userID)
		} else {
			x, e = a.repo.ListOrganizations(r.Context())
		}
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
	var x Organization
	if !decode(w, r, &x) {
		return
	}
	if x.Timezone == "" {
		x.Timezone = "America/Chicago"
	}
	// New organizations begin active. They may be deactivated later while
	// preserving all historical schedules, attendance, and reports.
	x.IsActive = true
	if err := validOrganization(x); err != nil {
		fail(w, 400, err)
		return
	}
	if e := a.repo.CreateOrganization(r.Context(), &x); e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 201, x)
}
func (a *API) organization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	switch r.Method {
	case http.MethodGet:
		if !a.requireOrganizationPermission(w, r, id, "view_reports") {
			return
		}
		x, e := a.repo.GetOrganization(r.Context(), id)
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
		var x Organization
		if !decode(w, r, &x) {
			return
		}
		if x.Timezone == "" {
			x.Timezone = "America/Chicago"
		}
		if err := validOrganization(x); err != nil {
			fail(w, 400, err)
			return
		}
		x.ID = id
		if e := a.repo.UpdateOrganization(r.Context(), &x); notFound(w, e) {
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
		if e := a.repo.DeleteOrganization(r.Context(), id); notFound(w, e) {
			return
		} else if e != nil {
			fail(w, 500, e)
			return
		}
		w.WriteHeader(204)
	}
}

func validOrganization(x Organization) error {
	required := []struct {
		name  string
		value string
	}{
		{"organization name", x.Name},
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

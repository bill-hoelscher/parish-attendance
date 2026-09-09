package httpapi

import (
	"errors"
	"net/http"
)

func (a *API) massReport(w http.ResponseWriter, r *http.Request) {
	if !a.requireOrganizationPermission(w, r, r.PathValue("id"), "view_reports") {
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	source := r.URL.Query().Get("type")
	if source != "recurring" && source != "special" && source != "" {
		fail(w, 400, errors.New("type must be recurring or special"))
		return
	}
	x, e := a.repo.MassAttendanceReport(r.Context(), r.PathValue("id"), from, to, source)
	if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}

func (a *API) massReportEntries(w http.ResponseWriter, r *http.Request) {
	if !a.requireOrganizationPermission(w, r, r.PathValue("id"), "view_reports") {
		return
	}
	source := r.URL.Query().Get("type")
	if source != "recurring" && source != "special" {
		fail(w, 400, errors.New("type must be recurring or special"))
		return
	}
	x, e := a.repo.MassAttendanceEntries(r.Context(), r.PathValue("id"), r.PathValue("massNameID"), r.URL.Query().Get("from"), r.URL.Query().Get("to"), source)
	if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}

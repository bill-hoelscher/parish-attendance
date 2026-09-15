package httpapi

import (
	"errors"
	"net/http"
)

func (a *API) massReport(w http.ResponseWriter, r *http.Request) {
	if !a.requireParishPermission(w, r, r.PathValue("id"), "view_reports") {
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	source := r.URL.Query().Get("type")
	if source != "recurring" && source != "weekend" && source != "weekday" && source != "special" && source != "" {
		fail(w, 400, errors.New("type must be recurring, weekend, weekday, or special"))
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
	if !a.requireParishPermission(w, r, r.PathValue("id"), "view_reports") {
		return
	}
	source := r.URL.Query().Get("type")
	if source != "recurring" && source != "weekend" && source != "weekday" && source != "special" {
		fail(w, 400, errors.New("type must be recurring, weekend, weekday, or special"))
		return
	}
	x, e := a.repo.MassAttendanceEntries(r.Context(), r.PathValue("id"), r.PathValue("massNameID"), r.URL.Query().Get("from"), r.URL.Query().Get("to"), source)
	if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}

func (a *API) weekendTotalsReport(w http.ResponseWriter, r *http.Request) {
	parishID := r.PathValue("id")
	if !a.requireParishPermission(w, r, parishID, "view_reports") {
		return
	}
	totals, err := a.repo.WeekendAttendanceTotals(r.Context(), parishID, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, totals)
}

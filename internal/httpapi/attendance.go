package httpapi

import (
	"errors"
	"net/http"
)

func (a *API) attendance(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("id")
	if r.Method == http.MethodGet {
		if !a.requireOrganizationPermission(w, r, org, "record_attendance") {
			return
		}
		x, e := a.repo.ListAttendance(r.Context(), org, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
		if e != nil {
			fail(w, 500, e)
			return
		}
		respond(w, 200, x)
		return
	}
	if !a.requireWritableOrganization(w, r, org, "record_attendance") {
		return
	}
	var x Attendance
	if !decode(w, r, &x) {
		return
	}
	if e := validAttendance(x); e != nil {
		fail(w, 400, e)
		return
	}
	x.OrganizationID = org
	if x.RecordedByUserID == "" {
		x.RecordedByUserID = r.Header.Get("X-User-ID")
	}
	if e := a.repo.UpsertAttendance(r.Context(), &x); e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 201, x)
}
func (a *API) attendanceLedger(w http.ResponseWriter, r *http.Request) {
	if !a.requireOrganizationPermission(w, r, r.PathValue("id"), "record_attendance") {
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		fail(w, 400, errors.New("from and to query parameters are required (YYYY-MM-DD)"))
		return
	}
	x, e := a.repo.AttendanceLedger(r.Context(), r.PathValue("id"), from, to)
	if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}
func (a *API) attendanceItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.requireWritableResource(w, r, "attendance", id, "record_attendance"); !ok {
		return
	}
	if r.Method == http.MethodDelete {
		if e := a.repo.DeleteAttendance(r.Context(), id); notFound(w, e) {
			return
		} else if e != nil {
			fail(w, 500, e)
			return
		}
		w.WriteHeader(204)
		return
	}
	var x Attendance
	if !decode(w, r, &x) {
		return
	}
	if e := validAttendance(x); e != nil {
		fail(w, 400, e)
		return
	}
	x.ID = id
	if e := a.repo.UpdateAttendance(r.Context(), &x); notFound(w, e) {
		return
	} else if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}
func validAttendance(x Attendance) error {
	if x.ServiceDate == "" || x.ServiceTime == "" || x.MassName == "" || x.MassNameID == nil {
		return errors.New("massNameId, serviceDate, serviceTime, and massName are required")
	}
	if x.AttendanceCount < 0 {
		return errors.New("attendanceCount cannot be negative")
	}
	if x.MassTemplateID != nil && x.SpecialMassID != nil {
		return errors.New("only one of massTemplateId or specialMassId may be set")
	}
	return nil
}

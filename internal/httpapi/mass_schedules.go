package httpapi

import (
	"errors"
	"net/http"
	"parishattendance/internal"
)

func (a *API) templates(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("id")
	if r.Method == http.MethodGet {
		x, e := a.repo.ListMassTemplates(r.Context(), org)
		if e != nil {
			fail(w, 500, e)
			return
		}
		respond(w, 200, x)
		return
	}
	if !a.requireWritableParish(w, r, org, "manage_schedules") {
		return
	}
	var x MassTemplate
	if !decode(w, r, &x) {
		return
	}
	if x.MassNameID == nil || x.Name == "" || x.ServiceTime == "" || x.Weekday < 0 || x.Weekday > 6 {
		fail(w, 400, errors.New("massNameId, name, weekday (0=Sunday), and serviceTime are required"))
		return
	}
	x.ParishID = org
	if e := a.repo.CreateMassTemplate(r.Context(), &x); e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 201, x)
}
func (a *API) template(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.requireWritableResource(w, r, "mass_template", id, "manage_schedules"); !ok {
		return
	}
	if r.Method == http.MethodDelete {
		if e := a.repo.DeleteMassTemplate(r.Context(), id); notFound(w, e) {
			return
		} else if e != nil {
			fail(w, 500, e)
			return
		}
		w.WriteHeader(204)
		return
	}
	var x MassTemplate
	if !decode(w, r, &x) {
		return
	}
	if x.MassNameID == nil || x.Name == "" || x.ServiceTime == "" || x.Weekday < 0 || x.Weekday > 6 {
		fail(w, 400, errors.New("massNameId, name, weekday, and serviceTime are required"))
		return
	}
	x.ID = id
	if e := a.repo.UpdateMassTemplate(r.Context(), &x); notFound(w, e) {
		return
	} else if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}
func (a *API) specialMasses(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("id")
	if r.Method == http.MethodGet {
		x, e := a.repo.ListSpecialMasses(r.Context(), org)
		if e != nil {
			fail(w, 500, e)
			return
		}
		respond(w, 200, x)
		return
	}
	if !a.requireWritableParish(w, r, org, "manage_schedules") {
		return
	}
	var x SpecialMass
	if !decode(w, r, &x) {
		return
	}
	if e := validSpecial(x); e != nil {
		fail(w, 400, e)
		return
	}
	x.ParishID = org
	if e := a.repo.CreateSpecialMass(r.Context(), &x); e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 201, x)
}
func (a *API) specialMass(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.requireWritableResource(w, r, "special_mass", id, "manage_schedules"); !ok {
		return
	}
	if r.Method == http.MethodDelete {
		if e := a.repo.DeleteSpecialMass(r.Context(), id); notFound(w, e) {
			return
		} else if e != nil {
			fail(w, 500, e)
			return
		}
		w.WriteHeader(204)
		return
	}
	var x SpecialMass
	if !decode(w, r, &x) {
		return
	}
	if e := validSpecial(x); e != nil {
		fail(w, 400, e)
		return
	}
	x.ID = id
	if e := a.repo.UpdateSpecialMass(r.Context(), &x); notFound(w, e) {
		return
	} else if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, x)
}
func validSpecial(x SpecialMass) error {
	if x.Action != "ADD" && x.Action != "CANCEL" {
		return errors.New("action must be ADD or CANCEL")
	}
	if x.ServiceDate == "" || x.ServiceTime == "" || x.Name == "" || x.MassNameID == nil {
		return errors.New("massNameId, serviceDate, serviceTime, and name are required")
	}
	if x.Action == "ADD" && x.MassTemplateID != nil {
		return errors.New("ADD cannot reference massTemplateId")
	}
	if x.Action == "CANCEL" && x.MassTemplateID == nil {
		return errors.New("CANCEL requires massTemplateId")
	}
	if x.Action == "CANCEL" && x.RecursAnnually {
		return errors.New("only an added special mass can recur annually")
	}
	return nil
}
func (a *API) scheduledMasses(w http.ResponseWriter, r *http.Request) {
	if !a.requireParishPermission(w, r, r.PathValue("id"), "record_attendance") {
		return
	}
	d := r.URL.Query().Get("date")
	if d == "" {
		fail(w, 400, errors.New("date query parameter is required (YYYY-MM-DD)"))
		return
	}
	x, e := a.repo.ScheduledMasses(r.Context(), r.PathValue("id"), d)
	if e != nil {
		fail(w, 500, e)
		return
	}
	respond(w, 200, []internal.ScheduledMass(x))
}

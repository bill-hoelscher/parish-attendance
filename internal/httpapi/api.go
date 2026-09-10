package httpapi

import (
	"errors"
	"net/http"

	"parishattendance/internal"
	"parishattendance/internal/auth"
)

type API struct {
	repo     internal.Repository
	verifier *auth.Verifier
	inviter  *auth.Inviter
}

func New(repo internal.Repository, verifier *auth.Verifier, inviter *auth.Inviter) http.Handler {
	a := &API{repo: repo, verifier: verifier, inviter: inviter}
	m := http.NewServeMux()
	m.HandleFunc("GET /health", a.health)
	m.HandleFunc("GET /me", a.me)
	m.HandleFunc("GET /organizations", a.organizations)
	m.HandleFunc("POST /organizations", a.organizations)
	m.HandleFunc("GET /organizations/{id}", a.organization)
	m.HandleFunc("PATCH /organizations/{id}", a.organization)
	m.HandleFunc("DELETE /organizations/{id}", a.organization)
	m.HandleFunc("GET /billing-accounts", a.billingAccounts)
	m.HandleFunc("POST /billing-accounts", a.billingAccounts)
	m.HandleFunc("GET /billing-accounts/{id}", a.billingAccount)
	m.HandleFunc("PATCH /billing-accounts/{id}", a.billingAccount)
	m.HandleFunc("GET /billing-accounts/{id}/subscription", a.billingSubscription)
	m.HandleFunc("PUT /billing-accounts/{id}/subscription", a.billingSubscription)
	m.HandleFunc("GET /user-access", a.userAccess)
	m.HandleFunc("POST /user-access", a.userAccess)
	m.HandleFunc("PATCH /user-access/{id}", a.userAccessItem)
	m.HandleFunc("DELETE /user-access/{id}", a.userAccessItem)
	m.HandleFunc("GET /roles", a.roles)
	m.HandleFunc("POST /roles", a.roles)
	m.HandleFunc("GET /roles/{id}", a.role)
	m.HandleFunc("PATCH /roles/{id}", a.role)
	m.HandleFunc("DELETE /roles/{id}", a.role)
	m.HandleFunc("GET /organizations/{id}/mass-names", a.massNames)
	m.HandleFunc("POST /organizations/{id}/accounts", a.organizationAccount)
	m.HandleFunc("POST /organizations/{id}/mass-names", a.massNames)
	m.HandleFunc("PATCH /mass-names/{id}", a.massName)
	m.HandleFunc("DELETE /mass-names/{id}", a.massName)
	m.HandleFunc("GET /organizations/{id}/mass-templates", a.templates)
	m.HandleFunc("POST /organizations/{id}/mass-templates", a.templates)
	m.HandleFunc("PATCH /mass-templates/{id}", a.template)
	m.HandleFunc("DELETE /mass-templates/{id}", a.template)
	m.HandleFunc("GET /organizations/{id}/special-masses", a.specialMasses)
	m.HandleFunc("POST /organizations/{id}/special-masses", a.specialMasses)
	m.HandleFunc("PATCH /special-masses/{id}", a.specialMass)
	m.HandleFunc("DELETE /special-masses/{id}", a.specialMass)
	m.HandleFunc("GET /organizations/{id}/scheduled-masses", a.scheduledMasses)
	m.HandleFunc("GET /organizations/{id}/attendance", a.attendance)
	m.HandleFunc("GET /organizations/{id}/attendance-ledger", a.attendanceLedger)
	m.HandleFunc("POST /organizations/{id}/attendance", a.attendance)
	m.HandleFunc("PATCH /attendance/{id}", a.attendanceItem)
	m.HandleFunc("DELETE /attendance/{id}", a.attendanceItem)
	m.HandleFunc("GET /organizations/{id}/reports/mass-attendance", a.massReport)
	m.HandleFunc("GET /organizations/{id}/reports/mass-attendance/{massNameID}", a.massReportEntries)
	return withCORS(a.requireAuthentication(m))
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		fail(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	roles, err := a.repo.UserRoles(r.Context(), principal.Subject, "")
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"email": principal.Email, "roles": roles})
}

func (a *API) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		principal, err := a.verifier.Verify(r.Context(), auth.Bearer(r.Header.Get("Authorization")))
		if err != nil {
			fail(w, http.StatusUnauthorized, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
	})
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.repo.Ping(r.Context()); err != nil {
		fail(w, http.StatusServiceUnavailable, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

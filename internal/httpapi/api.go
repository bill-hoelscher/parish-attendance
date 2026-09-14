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
	m.HandleFunc("GET /dioceses", a.dioceses)
	m.HandleFunc("POST /dioceses", a.dioceses)
	m.HandleFunc("GET /dioceses/{id}", a.diocese)
	m.HandleFunc("PATCH /dioceses/{id}", a.diocese)
	m.HandleFunc("DELETE /dioceses/{id}", a.diocese)
	m.HandleFunc("GET /parishes", a.parishes)
	m.HandleFunc("POST /parishes", a.parishes)
	m.HandleFunc("GET /parishes/{id}", a.parish)
	m.HandleFunc("PATCH /parishes/{id}", a.parish)
	m.HandleFunc("DELETE /parishes/{id}", a.parish)
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
	m.HandleFunc("POST /user-access/{id}/disable", a.disableUserAccess)
	m.HandleFunc("GET /roles", a.roles)
	m.HandleFunc("POST /roles", a.roles)
	m.HandleFunc("GET /roles/{id}", a.role)
	m.HandleFunc("PATCH /roles/{id}", a.role)
	m.HandleFunc("DELETE /roles/{id}", a.role)
	m.HandleFunc("GET /parishes/{id}/mass-names", a.massNames)
	m.HandleFunc("POST /parishes/{id}/accounts", a.parishAccount)
	m.HandleFunc("POST /parishes/{id}/mass-names", a.massNames)
	m.HandleFunc("PATCH /mass-names/{id}", a.massName)
	m.HandleFunc("DELETE /mass-names/{id}", a.massName)
	m.HandleFunc("GET /parishes/{id}/mass-templates", a.templates)
	m.HandleFunc("POST /parishes/{id}/mass-templates", a.templates)
	m.HandleFunc("PATCH /mass-templates/{id}", a.template)
	m.HandleFunc("DELETE /mass-templates/{id}", a.template)
	m.HandleFunc("GET /parishes/{id}/special-masses", a.specialMasses)
	m.HandleFunc("POST /parishes/{id}/special-masses", a.specialMasses)
	m.HandleFunc("PATCH /special-masses/{id}", a.specialMass)
	m.HandleFunc("DELETE /special-masses/{id}", a.specialMass)
	m.HandleFunc("GET /parishes/{id}/scheduled-masses", a.scheduledMasses)
	m.HandleFunc("GET /parishes/{id}/attendance", a.attendance)
	m.HandleFunc("GET /parishes/{id}/attendance-ledger", a.attendanceLedger)
	m.HandleFunc("POST /parishes/{id}/attendance", a.attendance)
	m.HandleFunc("PATCH /attendance/{id}", a.attendanceItem)
	m.HandleFunc("DELETE /attendance/{id}", a.attendanceItem)
	m.HandleFunc("GET /parishes/{id}/reports/mass-attendance", a.massReport)
	m.HandleFunc("GET /parishes/{id}/reports/mass-attendance/{massNameID}", a.massReportEntries)
	return withCORS(a.requireAuthentication(m))
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		fail(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	// A person can have different parish-scoped roles. When the app supplies an
	// active parish, return only that parish's assignment (plus any system role)
	// so the client can present the matching workspace.
	roles, err := a.repo.UserRoles(r.Context(), principal.Subject, r.URL.Query().Get("parishId"))
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	granted := make(map[string]bool)
	for _, role := range roles {
		// System administrators are deliberately not stored with a duplicate
		// list of every permission; their role grants full application access.
		if role.Role == "system_administrator" {
			for permission := range knownPermissions {
				granted[permission] = true
			}
			continue
		}
		permissions, permissionErr := a.repo.RolePermissions(r.Context(), role.RoleID)
		if permissionErr != nil {
			fail(w, http.StatusInternalServerError, permissionErr)
			return
		}
		for _, permission := range permissions {
			granted[permission] = true
		}
	}
	permissionOrder := []string{"record_attendance", "view_reports", "manage_schedules", "manage_users"}
	permissions := make([]string, 0, len(permissionOrder))
	for _, permission := range permissionOrder {
		if granted[permission] {
			permissions = append(permissions, permission)
		}
	}
	respond(w, http.StatusOK, map[string]any{"email": principal.Email, "roles": roles, "permissions": permissions})
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

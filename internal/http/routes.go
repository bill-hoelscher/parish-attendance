package http

// routes is the single place that declares the public HTTP surface.
func (s *Server) routes() {
	// Operations
	s.router.HandleFunc("GET /healthcheck", s.handleHealthCheck())
	s.router.HandleFunc("GET /smoketest", s.handleSmokeTest())

	// Parishes
	s.router.Handle("GET /parishes", s.api)
	s.router.Handle("POST /parishes", s.api)
	s.router.Handle("GET /parishes/{id}", s.api)
	s.router.Handle("PATCH /parishes/{id}", s.api)
	s.router.Handle("DELETE /parishes/{id}", s.api)
	s.router.Handle("GET /billing-accounts", s.api)
	s.router.Handle("POST /billing-accounts", s.api)
	s.router.Handle("GET /billing-accounts/{id}", s.api)
	s.router.Handle("PATCH /billing-accounts/{id}", s.api)
	s.router.Handle("GET /billing-accounts/{id}/subscription", s.api)
	s.router.Handle("PUT /billing-accounts/{id}/subscription", s.api)
	s.router.Handle("GET /user-access", s.api)
	s.router.Handle("POST /user-access", s.api)
	s.router.Handle("PATCH /user-access/{id}", s.api)
	s.router.Handle("DELETE /user-access/{id}", s.api)
	s.router.Handle("POST /user-access/{id}/disable", s.api)
	s.router.Handle("GET /roles", s.api)
	s.router.Handle("POST /roles", s.api)
	s.router.Handle("GET /roles/{id}", s.api)
	s.router.Handle("PATCH /roles/{id}", s.api)
	s.router.Handle("DELETE /roles/{id}", s.api)
	s.router.Handle("GET /parishes/{id}/mass-names", s.api)
	s.router.Handle("POST /parishes/{id}/mass-names", s.api)
	s.router.Handle("PATCH /mass-names/{id}", s.api)
	s.router.Handle("DELETE /mass-names/{id}", s.api)

	// Mass schedules
	s.router.Handle("GET /parishes/{id}/mass-templates", s.api)
	s.router.Handle("POST /parishes/{id}/mass-templates", s.api)
	s.router.Handle("PATCH /mass-templates/{id}", s.api)
	s.router.Handle("DELETE /mass-templates/{id}", s.api)
	s.router.Handle("GET /parishes/{id}/special-masses", s.api)
	s.router.Handle("POST /parishes/{id}/special-masses", s.api)
	s.router.Handle("PATCH /special-masses/{id}", s.api)
	s.router.Handle("DELETE /special-masses/{id}", s.api)
	s.router.Handle("GET /parishes/{id}/scheduled-masses", s.api)

	// Attendance and reports
	s.router.Handle("GET /parishes/{id}/attendance", s.api)
	s.router.Handle("GET /parishes/{id}/attendance-ledger", s.api)
	s.router.Handle("POST /parishes/{id}/attendance", s.api)
	s.router.Handle("PATCH /attendance/{id}", s.api)
	s.router.Handle("DELETE /attendance/{id}", s.api)
	s.router.Handle("GET /parishes/{id}/reports/mass-attendance", s.api)
	s.router.Handle("GET /parishes/{id}/reports/mass-attendance/{massNameID}", s.api)

	// API documentation
	s.router.Handle("GET /docs/openapi.yaml", s.openAPI())
	s.router.Handle("GET /docs/", s.swaggerUI())
	s.router.Handle("GET /app/", s.webPage())
	s.router.HandleFunc("GET /auth/config", s.authConfig)
	s.router.HandleFunc("POST /auth/sign-in", s.signIn)
	s.router.HandleFunc("POST /auth/new-password", s.newPassword)
	s.router.HandleFunc("POST /auth/password-reset", s.passwordReset)
	s.router.HandleFunc("POST /auth/password-reset/confirm", s.confirmPasswordReset)

	// Retain the existing CORS preflight behavior and /health compatibility route.
	s.router.Handle("/", s.api)
}

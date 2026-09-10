// Package http exposes the application's HTTP transport.
package http

import (
	stdhttp "net/http"

	"parishattendance/internal"
	"parishattendance/internal/auth"
	"parishattendance/internal/httpapi"
)

type Server struct {
	repo   internal.Repository
	router *stdhttp.ServeMux
	api    stdhttp.Handler
}

func NewServer(services internal.Services, verifier *auth.Verifier, inviter *auth.Inviter) *Server {
	s := &Server{repo: services.Repository, router: stdhttp.NewServeMux(), api: httpapi.New(services.Repository, verifier, inviter)}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) { s.router.ServeHTTP(w, r) }

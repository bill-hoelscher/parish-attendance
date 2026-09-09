// Package http exposes the application's HTTP transport.
package http

import (
	"database/sql"
	stdhttp "net/http"

	"parishattendance/internal"
	"parishattendance/internal/httpapi"
)

type Server struct {
	db     *sql.DB
	router *stdhttp.ServeMux
	api    stdhttp.Handler
}

func NewServer(services internal.Services) *Server {
	s := &Server{db: services.Repository.DB(), router: stdhttp.NewServeMux(), api: httpapi.New(services.Repository)}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) { s.router.ServeHTTP(w, r) }

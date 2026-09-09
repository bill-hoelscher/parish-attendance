package http

import (
	_ "embed"
	"net/http"
)

//go:embed assets/index.html
var webApp []byte

func (s *Server) webPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(webApp)
	})
}

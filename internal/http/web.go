package http

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
)

//go:embed assets/index.html
var webApp []byte

func (s *Server) webPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(webApp)
	})
}

func (s *Server) authConfig(w http.ResponseWriter, _ *http.Request) {
	redirectURI := os.Getenv("COGNITO_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/app/"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"domain": os.Getenv("COGNITO_DOMAIN"), "clientId": os.Getenv("COGNITO_CLIENT_ID"),
		"redirectUri": redirectURI,
	})
}

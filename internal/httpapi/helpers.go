package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, 400, err)
		return false
	}
	return true
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, err error) {
	respond(w, status, map[string]string{"error": err.Error()})
}
func notFound(w http.ResponseWriter, err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, errors.New("not found"))
		return true
	}
	return false
}
func changed(w http.ResponseWriter, res sql.Result) int64 {
	n, err := res.RowsAffected()
	if err != nil {
		fail(w, 500, err)
		return 0
	}
	if n == 0 {
		fail(w, 404, errors.New("not found"))
	}
	return n
}
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("APP_ORIGIN")
		if origin == "" {
			origin = "http://localhost:8080"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

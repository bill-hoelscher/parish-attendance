package http

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
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

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type resetRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func decodeAuthRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil || json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(target) != nil {
		http.Error(w, `{"error":"Enter the requested sign-in details."}`, http.StatusBadRequest)
		return false
	}
	return true
}

func authError(err error) string {
	var invalid *types.NotAuthorizedException
	if errors.As(err, &invalid) {
		return "The email address or password is incorrect."
	}
	var userNotFound *types.UserNotFoundException
	if errors.As(err, &userNotFound) {
		return "The email address or password is incorrect."
	}
	var invalidPassword *types.InvalidPasswordException
	if errors.As(err, &invalidPassword) {
		return "Choose a password that meets the required security rules."
	}
	return "We could not complete that request. Please try again."
}

func writeAuthJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) signIn(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decodeAuthRequest(w, r, &request) {
		return
	}
	request.Email, request.Password = strings.TrimSpace(request.Email), strings.TrimSpace(request.Password)
	if request.Email == "" || request.Password == "" {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter your email address and password."})
		return
	}
	result, err := s.authenticator.SignIn(r.Context(), request.Email, request.Password)
	if err != nil {
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{"error": authError(err)})
		return
	}
	if result.AccessToken != "" {
		writeAuthJSON(w, http.StatusOK, map[string]string{"accessToken": result.AccessToken})
		return
	}
	if result.Challenge == string(types.ChallengeNameTypeNewPasswordRequired) && result.Session != "" {
		writeAuthJSON(w, http.StatusOK, map[string]string{"challenge": "new_password", "session": result.Session})
		return
	}
	writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "This sign-in method needs an additional security step that is not configured in the app."})
}

func (s *Server) newPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Session  string `json:"session"`
	}
	if !decodeAuthRequest(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Email) == "" || request.Password == "" || request.Session == "" {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter a new password to continue."})
		return
	}
	result, err := s.authenticator.CompleteNewPassword(r.Context(), strings.TrimSpace(request.Email), request.Password, request.Session)
	if err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": authError(err)})
		return
	}
	if result.AccessToken == "" {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "We could not complete your password update. Please try again."})
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]string{"accessToken": result.AccessToken})
}

func (s *Server) passwordReset(w http.ResponseWriter, r *http.Request) {
	var request resetRequest
	if !decodeAuthRequest(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Email) == "" {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter your email address."})
		return
	}
	if err := s.authenticator.RequestPasswordReset(r.Context(), strings.TrimSpace(request.Email)); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": authError(err)})
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var request resetRequest
	if !decodeAuthRequest(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.Code) == "" || request.Password == "" {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter your email, confirmation code, and new password."})
		return
	}
	if err := s.authenticator.ConfirmPasswordReset(r.Context(), strings.TrimSpace(request.Email), strings.TrimSpace(request.Code), request.Password); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": authError(err)})
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

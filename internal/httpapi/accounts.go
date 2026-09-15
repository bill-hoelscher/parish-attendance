package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"parishattendance/internal"
)

type accountInvitation struct {
	Mode     string `json:"mode"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   string `json:"roleId"`
}

// parishAccount lets a parish administrator create an account or grant a user
// access only to the parish they administer. Email invitations retain the
// existing Cognito behavior; direct accounts never send email.
func (a *API) parishAccount(w http.ResponseWriter, r *http.Request) {
	parishID := r.PathValue("id")
	if !a.requireParishPermission(w, r, parishID, "manage_users") {
		return
	}
	var request accountInvitation
	if !decode(w, r, &request) {
		return
	}
	request.Mode = strings.TrimSpace(request.Mode)
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	request.Username = strings.TrimSpace(request.Username)
	if request.RoleID == "" {
		fail(w, http.StatusBadRequest, errors.New("roleId is required"))
		return
	}
	role, err := a.repo.GetRole(r.Context(), request.RoleID)
	if err != nil || role.IsSystem {
		fail(w, http.StatusBadRequest, errors.New("a parish-scoped role is required"))
		return
	}
	if a.inviter == nil {
		fail(w, http.StatusServiceUnavailable, errors.New("user management is unavailable"))
		return
	}
	var userID string
	access := internal.UserAccess{ParishID: &parishID, RoleID: role.ID, Role: role.Key, RoleName: role.Name}
	switch request.Mode {
	case "", "invite":
		if request.Email == "" {
			fail(w, http.StatusBadRequest, errors.New("email and roleId are required"))
			return
		}
		userID, err = a.inviter.InviteOrFind(r.Context(), request.Email)
		access.Email = request.Email
	case "password":
		if request.Email == "" || request.Password == "" {
			fail(w, http.StatusBadRequest, errors.New("email, password, and roleId are required"))
			return
		}
		userID, err = a.inviter.CreateWithPassword(r.Context(), request.Email, request.Password)
		access.Email = request.Email
	default:
		fail(w, http.StatusBadRequest, errors.New("account mode must be invite or password"))
		return
	}
	if err != nil {
		fail(w, http.StatusBadGateway, err)
		return
	}
	access.UserID = userID
	if err := a.repo.CreateUserAccess(r.Context(), &access); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusCreated, map[string]string{"userId": userID, "username": access.Username, "email": access.Email})
}

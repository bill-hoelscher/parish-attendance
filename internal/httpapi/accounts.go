package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"parishattendance/internal"
)

type accountInvitation struct {
	Email  string `json:"email"`
	RoleID string `json:"roleId"`
}

// parishAccount lets a parish administrator grant a user access only to the
// parish they administer. New accounts receive a Cognito invitation; an
// existing Cognito account is linked without sending a duplicate invitation.
func (a *API) parishAccount(w http.ResponseWriter, r *http.Request) {
	parishID := r.PathValue("id")
	if !a.requireParishPermission(w, r, parishID, "manage_users") {
		return
	}
	var request accountInvitation
	if !decode(w, r, &request) {
		return
	}
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	if request.Email == "" || request.RoleID == "" {
		fail(w, http.StatusBadRequest, errors.New("email and roleId are required"))
		return
	}
	role, err := a.repo.GetRole(r.Context(), request.RoleID)
	if err != nil || role.IsSystem {
		fail(w, http.StatusBadRequest, errors.New("a parish-scoped role is required"))
		return
	}
	userID, err := a.inviter.InviteOrFind(r.Context(), request.Email)
	if err != nil {
		fail(w, http.StatusBadGateway, err)
		return
	}
	access := internal.UserAccess{UserID: userID, Email: request.Email, ParishID: &parishID, RoleID: role.ID, Role: role.Key, RoleName: role.Name}
	if err := a.repo.CreateUserAccess(r.Context(), &access); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusCreated, map[string]string{"userId": userID, "email": request.Email})
}

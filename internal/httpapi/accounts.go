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

// organizationAccount lets a parish administrator invite a user only into the
// organization they administer. System roles are deliberately excluded.
func (a *API) organizationAccount(w http.ResponseWriter, r *http.Request) {
	organizationID := r.PathValue("id")
	if !a.requireOrganizationPermission(w, r, organizationID, "manage_users") {
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
		fail(w, http.StatusBadRequest, errors.New("an organization-scoped role is required"))
		return
	}
	userID, err := a.inviter.Invite(r.Context(), request.Email)
	if err != nil {
		fail(w, http.StatusBadGateway, err)
		return
	}
	access := internal.UserAccess{UserID: userID, OrganizationID: &organizationID, RoleID: role.ID, Role: role.Key, RoleName: role.Name}
	if err := a.repo.CreateUserAccess(r.Context(), &access); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusCreated, map[string]string{"userId": userID, "email": request.Email})
}

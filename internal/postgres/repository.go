// Package postgres provides the PostgreSQL adapter for the application.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"parishattendance/internal"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository           { return &Repository{db: db} }
func (r *Repository) DB() *sql.DB                    { return r.db }
func (r *Repository) Ping(ctx context.Context) error { return r.db.PingContext(ctx) }

func (r *Repository) ListOrganizations(ctx context.Context) ([]internal.Organization, error) {
	return r.listOrganizations(ctx, `SELECT id::text,billing_account_id::text,is_active,name,address,street_address,city,state_province,postal_code,contact_name,contact_email,contact_phone,timezone FROM organizations ORDER BY name`)
}

func (r *Repository) ListAuthorizedOrganizations(ctx context.Context, userID string) ([]internal.Organization, error) {
	return r.listOrganizations(ctx, `SELECT o.id::text,o.billing_account_id::text,o.is_active,o.name,o.address,o.street_address,o.city,o.state_province,o.postal_code,o.contact_name,o.contact_email,o.contact_phone,o.timezone FROM organizations o WHERE EXISTS (SELECT 1 FROM user_access ua WHERE ua.user_id=$1 AND (ua.role='system_administrator' OR ua.organization_id=o.id)) ORDER BY o.name`, userID)
}

func (r *Repository) listOrganizations(ctx context.Context, query string, args ...any) ([]internal.Organization, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.Organization{}
	for rows.Next() {
		var x internal.Organization
		if err := rows.Scan(&x.ID, &x.BillingAccountID, &x.IsActive, &x.Name, &x.Address, &x.StreetAddress, &x.City, &x.StateProvince, &x.PostalCode, &x.ContactName, &x.ContactEmail, &x.ContactPhone, &x.Timezone); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateOrganization(ctx context.Context, x *internal.Organization) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO organizations(billing_account_id,is_active,name,address,street_address,city,state_province,postal_code,contact_name,contact_email,contact_phone,timezone) VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id::text`, x.BillingAccountID, x.IsActive, x.Name, x.Address, x.StreetAddress, x.City, x.StateProvince, x.PostalCode, x.ContactName, x.ContactEmail, x.ContactPhone, x.Timezone).Scan(&x.ID)
}
func (r *Repository) GetOrganization(ctx context.Context, id string) (*internal.Organization, error) {
	var x internal.Organization
	err := r.db.QueryRowContext(ctx, `SELECT id::text,billing_account_id::text,is_active,name,address,street_address,city,state_province,postal_code,contact_name,contact_email,contact_phone,timezone FROM organizations WHERE id=$1`, id).Scan(&x.ID, &x.BillingAccountID, &x.IsActive, &x.Name, &x.Address, &x.StreetAddress, &x.City, &x.StateProvince, &x.PostalCode, &x.ContactName, &x.ContactEmail, &x.ContactPhone, &x.Timezone)
	return &x, err
}
func (r *Repository) UpdateOrganization(ctx context.Context, x *internal.Organization) error {
	res, err := r.db.ExecContext(ctx, `UPDATE organizations SET billing_account_id=$2::uuid,is_active=$3,name=$4,address=$5,street_address=$6,city=$7,state_province=$8,postal_code=$9,contact_name=$10,contact_email=$11,contact_phone=$12,timezone=$13 WHERE id=$1`, x.ID, x.BillingAccountID, x.IsActive, x.Name, x.Address, x.StreetAddress, x.City, x.StateProvince, x.PostalCode, x.ContactName, x.ContactEmail, x.ContactPhone, x.Timezone)
	return rowsAffected(res, err)
}
func (r *Repository) DeleteOrganization(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM organizations WHERE id=$1`, id)
	return rowsAffected(res, err)
}

func (r *Repository) ListBillingAccounts(ctx context.Context) ([]internal.BillingAccount, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT b.id::text,b.name,b.billing_email,COUNT(o.id)::int FROM billing_accounts b LEFT JOIN organizations o ON o.billing_account_id=b.id GROUP BY b.id,b.name,b.billing_email ORDER BY b.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []internal.BillingAccount{}
	for rows.Next() {
		var account internal.BillingAccount
		if err := rows.Scan(&account.ID, &account.Name, &account.BillingEmail, &account.OrganizationCount); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (r *Repository) CreateBillingAccount(ctx context.Context, account *internal.BillingAccount) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO billing_accounts(name,billing_email) VALUES($1,$2) RETURNING id::text`, account.Name, account.BillingEmail).Scan(&account.ID)
}

func (r *Repository) GetBillingAccount(ctx context.Context, id string) (*internal.BillingAccount, error) {
	var account internal.BillingAccount
	err := r.db.QueryRowContext(ctx, `SELECT b.id::text,b.name,b.billing_email,COUNT(o.id)::int FROM billing_accounts b LEFT JOIN organizations o ON o.billing_account_id=b.id WHERE b.id=$1 GROUP BY b.id,b.name,b.billing_email`, id).Scan(&account.ID, &account.Name, &account.BillingEmail, &account.OrganizationCount)
	return &account, err
}

func (r *Repository) UpdateBillingAccount(ctx context.Context, account *internal.BillingAccount) error {
	res, err := r.db.ExecContext(ctx, `UPDATE billing_accounts SET name=$2,billing_email=$3 WHERE id=$1`, account.ID, account.Name, account.BillingEmail)
	return rowsAffected(res, err)
}

func (r *Repository) GetSubscription(ctx context.Context, billingAccountID string) (*internal.Subscription, error) {
	var subscription internal.Subscription
	err := r.db.QueryRowContext(ctx, `SELECT id::text,billing_account_id::text,plan_code,status,stripe_subscription_id,to_char(access_through,'YYYY-MM-DD') FROM billing_subscriptions WHERE billing_account_id=$1`, billingAccountID).Scan(&subscription.ID, &subscription.BillingAccountID, &subscription.PlanCode, &subscription.Status, &subscription.StripeSubscriptionID, &subscription.AccessThrough)
	return &subscription, err
}

func (r *Repository) UpsertSubscription(ctx context.Context, subscription *internal.Subscription) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO billing_subscriptions(billing_account_id,plan_code,status,stripe_subscription_id,access_through) VALUES($1::uuid,$2,$3,$4,$5::date) ON CONFLICT(billing_account_id) DO UPDATE SET plan_code=EXCLUDED.plan_code,status=EXCLUDED.status,stripe_subscription_id=EXCLUDED.stripe_subscription_id,access_through=EXCLUDED.access_through RETURNING id::text`, subscription.BillingAccountID, subscription.PlanCode, subscription.Status, subscription.StripeSubscriptionID, subscription.AccessThrough).Scan(&subscription.ID)
}

func (r *Repository) OrganizationAccess(ctx context.Context, organizationID string) (*internal.OrganizationAccess, error) {
	var access internal.OrganizationAccess
	err := r.db.QueryRowContext(ctx, `SELECT o.id::text,o.is_active,COALESCE(s.status,'canceled'),to_char(s.access_through,'YYYY-MM-DD') FROM organizations o LEFT JOIN billing_subscriptions s ON s.billing_account_id=o.billing_account_id WHERE o.id=$1`, organizationID).Scan(&access.OrganizationID, &access.OrganizationActive, &access.SubscriptionStatus, &access.AccessThrough)
	if err != nil {
		return nil, err
	}
	access.CanWrite = access.OrganizationActive && (access.SubscriptionStatus == "active" || access.SubscriptionStatus == "trialing")
	if access.AccessThrough != nil && *access.AccessThrough < time.Now().UTC().Format("2006-01-02") {
		access.CanWrite = false
	}
	return &access, nil
}

func (r *Repository) ResourceOrganizationID(ctx context.Context, resource, id string) (string, error) {
	var query string
	switch resource {
	case "mass_name":
		query = `SELECT organization_id::text FROM mass_names WHERE id=$1`
	case "mass_template":
		query = `SELECT organization_id::text FROM mass_templates WHERE id=$1`
	case "special_mass":
		query = `SELECT organization_id::text FROM special_masses WHERE id=$1`
	case "attendance":
		query = `SELECT organization_id::text FROM attendance WHERE id=$1`
	default:
		return "", fmt.Errorf("unsupported resource %q", resource)
	}
	var organizationID string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&organizationID)
	return organizationID, err
}

func (r *Repository) ListRoles(ctx context.Context) ([]internal.Role, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT r.id::text,r.role_key,r.name,r.scope,r.is_system,COUNT(ua.id)::int FROM app_roles r LEFT JOIN user_access ua ON ua.role_id=r.id GROUP BY r.id,r.role_key,r.name,r.scope,r.is_system ORDER BY r.is_system DESC,r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []internal.Role{}
	for rows.Next() {
		var role internal.Role
		if err := rows.Scan(&role.ID, &role.Key, &role.Name, &role.Scope, &role.IsSystem, &role.AssignmentCount); err != nil {
			return nil, err
		}
		permissions, err := r.RolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) CreateRole(ctx context.Context, role *internal.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `INSERT INTO app_roles(role_key,name,scope,is_system) VALUES($1,$2,$3,false) RETURNING id::text`, role.Key, role.Name, role.Scope).Scan(&role.ID); err != nil {
		return err
	}
	for _, permission := range role.Permissions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO app_role_permissions(role_id,permission) VALUES($1::uuid,$2)`, role.ID, permission); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) GetRole(ctx context.Context, id string) (*internal.Role, error) {
	var role internal.Role
	err := r.db.QueryRowContext(ctx, `SELECT r.id::text,r.role_key,r.name,r.scope,r.is_system,COUNT(ua.id)::int FROM app_roles r LEFT JOIN user_access ua ON ua.role_id=r.id WHERE r.id=$1 GROUP BY r.id,r.role_key,r.name,r.scope,r.is_system`, id).Scan(&role.ID, &role.Key, &role.Name, &role.Scope, &role.IsSystem, &role.AssignmentCount)
	if err != nil {
		return nil, err
	}
	role.Permissions, err = r.RolePermissions(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) UpdateRole(ctx context.Context, role *internal.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := rowsAffected(tx.ExecContext(ctx, `UPDATE app_roles SET name=$2 WHERE id=$1 AND NOT is_system`, role.ID, role.Name)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM app_role_permissions WHERE role_id=$1`, role.ID); err != nil {
		return err
	}
	for _, permission := range role.Permissions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO app_role_permissions(role_id,permission) VALUES($1::uuid,$2)`, role.ID, permission); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) DeleteRole(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM app_roles WHERE id=$1 AND NOT is_system`, id)
	return rowsAffected(res, err)
}

func (r *Repository) RolePermissions(ctx context.Context, roleID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT permission FROM app_role_permissions WHERE role_id=$1 ORDER BY permission`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := []string{}
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	return permissions, rows.Err()
}

func (r *Repository) ListUserAccess(ctx context.Context, organizationID string) ([]internal.UserAccess, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ua.id::text,ua.user_id,ua.organization_id::text,r.id::text,r.role_key,r.name FROM user_access ua JOIN app_roles r ON r.id=ua.role_id WHERE (NULLIF($1,'') IS NULL OR ua.organization_id=NULLIF($1,'')::uuid) ORDER BY r.name,ua.user_id`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	access := []internal.UserAccess{}
	for rows.Next() {
		var item internal.UserAccess
		if err := rows.Scan(&item.ID, &item.UserID, &item.OrganizationID, &item.RoleID, &item.Role, &item.RoleName); err != nil {
			return nil, err
		}
		access = append(access, item)
	}
	return access, rows.Err()
}

func (r *Repository) CreateUserAccess(ctx context.Context, access *internal.UserAccess) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO user_access(user_id,organization_id,role,role_id) VALUES($1,$2::uuid,$3,$4::uuid) RETURNING id::text`, access.UserID, access.OrganizationID, access.Role, access.RoleID).Scan(&access.ID)
}

func (r *Repository) GetUserAccess(ctx context.Context, id string) (*internal.UserAccess, error) {
	var access internal.UserAccess
	err := r.db.QueryRowContext(ctx, `SELECT ua.id::text,ua.user_id,ua.organization_id::text,r.id::text,r.role_key,r.name FROM user_access ua JOIN app_roles r ON r.id=ua.role_id WHERE ua.id=$1`, id).Scan(&access.ID, &access.UserID, &access.OrganizationID, &access.RoleID, &access.Role, &access.RoleName)
	return &access, err
}

func (r *Repository) UpdateUserAccess(ctx context.Context, access *internal.UserAccess) error {
	res, err := r.db.ExecContext(ctx, `UPDATE user_access SET user_id=$2,organization_id=$3::uuid,role=$4,role_id=$5::uuid WHERE id=$1`, access.ID, access.UserID, access.OrganizationID, access.Role, access.RoleID)
	return rowsAffected(res, err)
}

func (r *Repository) DeleteUserAccess(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM user_access WHERE id=$1`, id)
	return rowsAffected(res, err)
}

func (r *Repository) UserRoles(ctx context.Context, userID, organizationID string) ([]internal.UserAccess, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ua.id::text,ua.user_id,ua.organization_id::text,r.id::text,r.role_key,r.name FROM user_access ua JOIN app_roles r ON r.id=ua.role_id WHERE ua.user_id=$1 AND (ua.organization_id=NULLIF($2,'')::uuid OR ua.organization_id IS NULL)`, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []internal.UserAccess{}
	for rows.Next() {
		var role internal.UserAccess
		if err := rows.Scan(&role.ID, &role.UserID, &role.OrganizationID, &role.RoleID, &role.Role, &role.RoleName); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) ListMassNames(ctx context.Context, orgID string) ([]internal.MassName, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id::text,organization_id::text,name,is_active FROM mass_names WHERE organization_id=$1 ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.MassName{}
	for rows.Next() {
		var x internal.MassName
		if err := rows.Scan(&x.ID, &x.OrganizationID, &x.Name, &x.IsActive); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateMassName(ctx context.Context, x *internal.MassName) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO mass_names(organization_id,name,is_active) VALUES($1,$2,$3) RETURNING id::text`, x.OrganizationID, x.Name, x.IsActive).Scan(&x.ID)
}
func (r *Repository) UpdateMassName(ctx context.Context, x *internal.MassName) error {
	return r.db.QueryRowContext(ctx, `UPDATE mass_names SET name=$2,is_active=$3 WHERE id=$1 RETURNING organization_id::text`, x.ID, x.Name, x.IsActive).Scan(&x.OrganizationID)
}
func (r *Repository) DeleteMassName(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM mass_names WHERE id=$1`, id)
	return rowsAffected(res, err)
}

func (r *Repository) ListMassTemplates(ctx context.Context, orgID string) ([]internal.MassTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t.id::text,t.organization_id::text,t.mass_name_id::text,n.name,t.weekday,t.service_time::text,to_char(t.active_from,'YYYY-MM-DD'),to_char(t.active_to,'YYYY-MM-DD'),t.is_active FROM mass_templates t LEFT JOIN mass_names n ON n.id=t.mass_name_id WHERE t.organization_id=$1 ORDER BY t.weekday,t.service_time`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.MassTemplate{}
	for rows.Next() {
		var x internal.MassTemplate
		if err := rows.Scan(&x.ID, &x.OrganizationID, &x.MassNameID, &x.Name, &x.Weekday, &x.ServiceTime, &x.ActiveFrom, &x.ActiveTo, &x.IsActive); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateMassTemplate(ctx context.Context, x *internal.MassTemplate) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO mass_templates(organization_id,mass_name_id,name,weekday,service_time,active_from,active_to,is_active) VALUES($1,$2::uuid,$3,$4,$5::time,$6::date,$7::date,$8) RETURNING id::text`, x.OrganizationID, x.MassNameID, x.Name, x.Weekday, x.ServiceTime, x.ActiveFrom, x.ActiveTo, x.IsActive).Scan(&x.ID)
}
func (r *Repository) UpdateMassTemplate(ctx context.Context, x *internal.MassTemplate) error {
	return r.db.QueryRowContext(ctx, `UPDATE mass_templates SET mass_name_id=$2::uuid,name=$3,weekday=$4,service_time=$5::time,active_from=$6::date,active_to=$7::date,is_active=$8 WHERE id=$1 RETURNING organization_id::text`, x.ID, x.MassNameID, x.Name, x.Weekday, x.ServiceTime, x.ActiveFrom, x.ActiveTo, x.IsActive).Scan(&x.OrganizationID)
}
func (r *Repository) DeleteMassTemplate(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM mass_templates WHERE id=$1`, id)
	return rowsAffected(res, err)
}

func (r *Repository) ListSpecialMasses(ctx context.Context, orgID string) ([]internal.SpecialMass, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id::text,s.organization_id::text,s.action::text,to_char(s.service_date,'YYYY-MM-DD'),s.service_time::text,n.name,s.mass_name_id::text,s.mass_template_id::text,s.recurs_annually FROM special_masses s LEFT JOIN mass_names n ON n.id=s.mass_name_id WHERE s.organization_id=$1 ORDER BY s.service_date,s.service_time`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.SpecialMass{}
	for rows.Next() {
		var x internal.SpecialMass
		if err := rows.Scan(&x.ID, &x.OrganizationID, &x.Action, &x.ServiceDate, &x.ServiceTime, &x.Name, &x.MassNameID, &x.MassTemplateID, &x.RecursAnnually); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateSpecialMass(ctx context.Context, x *internal.SpecialMass) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO special_masses(organization_id,action,service_date,service_time,name,mass_name_id,mass_template_id,recurs_annually) VALUES($1,$2::special_mass_action,$3::date,$4::time,$5,$6::uuid,$7::uuid,$8) RETURNING id::text`, x.OrganizationID, x.Action, x.ServiceDate, x.ServiceTime, x.Name, x.MassNameID, x.MassTemplateID, x.RecursAnnually).Scan(&x.ID)
}
func (r *Repository) UpdateSpecialMass(ctx context.Context, x *internal.SpecialMass) error {
	return r.db.QueryRowContext(ctx, `UPDATE special_masses SET action=$2::special_mass_action,service_date=$3::date,service_time=$4::time,name=$5,mass_name_id=$6::uuid,mass_template_id=$7::uuid,recurs_annually=$8 WHERE id=$1 RETURNING organization_id::text`, x.ID, x.Action, x.ServiceDate, x.ServiceTime, x.Name, x.MassNameID, x.MassTemplateID, x.RecursAnnually).Scan(&x.OrganizationID)
}
func (r *Repository) DeleteSpecialMass(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM special_masses WHERE id=$1`, id)
	return rowsAffected(res, err)
}
func (r *Repository) ScheduledMasses(ctx context.Context, orgID, date string) ([]internal.ScheduledMass, error) {
	q := `WITH regular AS (SELECT t.mass_name_id::text,t.id::text AS template_id,NULL::text AS special_id,n.name,t.service_time::text FROM mass_templates t JOIN mass_names n ON n.id=t.mass_name_id WHERE t.organization_id=$1 AND t.is_active AND t.weekday=EXTRACT(DOW FROM $2::date) AND (t.active_from IS NULL OR t.active_from <= $2::date) AND (t.active_to IS NULL OR t.active_to >= $2::date) AND NOT EXISTS (SELECT 1 FROM special_masses s WHERE s.organization_id=t.organization_id AND s.action='CANCEL' AND s.service_date=$2::date AND s.mass_template_id=t.id)), added AS (SELECT s.mass_name_id::text,NULL::text,s.id::text,n.name,s.service_time::text FROM special_masses s JOIN mass_names n ON n.id=s.mass_name_id WHERE s.organization_id=$1 AND s.action='ADD' AND (s.service_date=$2::date OR (s.recurs_annually AND EXTRACT(MONTH FROM s.service_date)=EXTRACT(MONTH FROM $2::date) AND EXTRACT(DAY FROM s.service_date)=EXTRACT(DAY FROM $2::date)))) SELECT mass_name_id,template_id,special_id,name,service_time FROM regular UNION ALL SELECT * FROM added ORDER BY service_time`
	rows, err := r.db.QueryContext(ctx, q, orgID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.ScheduledMass{}
	for rows.Next() {
		var x internal.ScheduledMass
		x.ServiceDate = date
		if err := rows.Scan(&x.MassNameID, &x.MassTemplateID, &x.SpecialMassID, &x.Name, &x.ServiceTime); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) ListAttendance(ctx context.Context, orgID, from, to string) ([]internal.Attendance, error) {
	q := `SELECT id::text,organization_id::text,to_char(service_date,'YYYY-MM-DD'),service_time::text,mass_name,mass_name_id::text,mass_template_id::text,special_mass_id::text,attendance_count,recorded_by_user_id FROM attendance WHERE organization_id=$1 AND ($2::date IS NULL OR service_date >= $2::date) AND ($3::date IS NULL OR service_date <= $3::date) ORDER BY service_date DESC,service_time`
	rows, err := r.db.QueryContext(ctx, q, orgID, nilIfEmpty(from), nilIfEmpty(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.Attendance{}
	for rows.Next() {
		var x internal.Attendance
		if err := rows.Scan(&x.ID, &x.OrganizationID, &x.ServiceDate, &x.ServiceTime, &x.MassName, &x.MassNameID, &x.MassTemplateID, &x.SpecialMassID, &x.AttendanceCount, &x.RecordedByUserID); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) AttendanceLedger(ctx context.Context, orgID, from, to string) ([]internal.AttendanceLedgerItem, error) {
	q := `WITH days AS (SELECT generate_series($2::date,$3::date,'1 day')::date AS service_date), scheduled AS (SELECT d.service_date,t.mass_name_id,t.id AS template_id,NULL::uuid AS special_id,n.name,t.service_time FROM days d JOIN mass_templates t ON t.organization_id=$1 AND t.is_active AND t.weekday=EXTRACT(DOW FROM d.service_date) AND (t.active_from IS NULL OR t.active_from<=d.service_date) AND (t.active_to IS NULL OR t.active_to>=d.service_date) JOIN mass_names n ON n.id=t.mass_name_id WHERE NOT EXISTS (SELECT 1 FROM special_masses c WHERE c.organization_id=$1 AND c.action='CANCEL' AND c.service_date=d.service_date AND c.mass_template_id=t.id) UNION ALL SELECT d.service_date,s.mass_name_id,NULL::uuid,s.id,n.name,s.service_time FROM days d JOIN special_masses s ON s.organization_id=$1 AND s.action='ADD' AND (s.service_date=d.service_date OR (s.recurs_annually AND EXTRACT(MONTH FROM s.service_date)=EXTRACT(MONTH FROM d.service_date) AND EXTRACT(DAY FROM s.service_date)=EXTRACT(DAY FROM d.service_date))) JOIN mass_names n ON n.id=s.mass_name_id) SELECT to_char(s.service_date,'YYYY-MM-DD'),s.service_time::text,s.name,s.mass_name_id::text,s.template_id::text,s.special_id::text,a.id::text,a.attendance_count FROM scheduled s LEFT JOIN attendance a ON a.organization_id=$1 AND a.service_date=s.service_date AND a.service_time=s.service_time ORDER BY s.service_date,s.service_time`
	rows, err := r.db.QueryContext(ctx, q, orgID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.AttendanceLedgerItem{}
	for rows.Next() {
		var x internal.AttendanceLedgerItem
		if err := rows.Scan(&x.ServiceDate, &x.ServiceTime, &x.MassName, &x.MassNameID, &x.MassTemplateID, &x.SpecialMassID, &x.AttendanceID, &x.AttendanceCount); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) UpsertAttendance(ctx context.Context, x *internal.Attendance) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO attendance(organization_id,service_date,service_time,mass_name,mass_name_id,mass_template_id,special_mass_id,attendance_count,recorded_by_user_id) VALUES($1,$2::date,$3::time,$4,$5::uuid,$6::uuid,$7::uuid,$8,$9) ON CONFLICT(organization_id,service_date,service_time) DO UPDATE SET mass_name=EXCLUDED.mass_name,mass_name_id=EXCLUDED.mass_name_id,mass_template_id=EXCLUDED.mass_template_id,special_mass_id=EXCLUDED.special_mass_id,attendance_count=EXCLUDED.attendance_count,recorded_by_user_id=EXCLUDED.recorded_by_user_id RETURNING id::text`, x.OrganizationID, x.ServiceDate, x.ServiceTime, x.MassName, x.MassNameID, x.MassTemplateID, x.SpecialMassID, x.AttendanceCount, x.RecordedByUserID).Scan(&x.ID)
}
func (r *Repository) UpdateAttendance(ctx context.Context, x *internal.Attendance) error {
	return r.db.QueryRowContext(ctx, `UPDATE attendance SET service_date=$2::date,service_time=$3::time,mass_name=$4,mass_name_id=$5::uuid,mass_template_id=$6::uuid,special_mass_id=$7::uuid,attendance_count=$8,recorded_by_user_id=$9 WHERE id=$1 RETURNING organization_id::text`, x.ID, x.ServiceDate, x.ServiceTime, x.MassName, x.MassNameID, x.MassTemplateID, x.SpecialMassID, x.AttendanceCount, x.RecordedByUserID).Scan(&x.OrganizationID)
}
func (r *Repository) DeleteAttendance(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM attendance WHERE id=$1`, id)
	return rowsAffected(res, err)
}
func (r *Repository) MassAttendanceReport(ctx context.Context, orgID, from, to, source string) ([]internal.MassAttendanceReport, error) {
	q := `SELECT a.mass_name_id::text,n.name,COUNT(*)::int,ROUND(AVG(a.attendance_count))::int,MIN(a.attendance_count),MAX(a.attendance_count) FROM attendance a JOIN mass_names n ON n.id=a.mass_name_id WHERE a.organization_id=$1 AND ($2::date IS NULL OR a.service_date >= $2::date) AND ($3::date IS NULL OR a.service_date <= $3::date) AND ($4='' OR ($4='recurring' AND a.mass_template_id IS NOT NULL) OR ($4='special' AND a.special_mass_id IS NOT NULL)) GROUP BY a.mass_name_id,n.name ORDER BY n.name`
	rows, err := r.db.QueryContext(ctx, q, orgID, nilIfEmpty(from), nilIfEmpty(to), source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.MassAttendanceReport{}
	for rows.Next() {
		var x internal.MassAttendanceReport
		if err := rows.Scan(&x.MassNameID, &x.MassName, &x.ServicesCounted, &x.AverageAttendance, &x.LowestAttendance, &x.HighestAttendance); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) MassAttendanceEntries(ctx context.Context, orgID, massNameID, from, to, source string) ([]internal.MassAttendanceEntry, error) {
	q := `SELECT to_char(service_date,'YYYY-MM-DD'),service_time::text,attendance_count FROM attendance WHERE organization_id=$1 AND mass_name_id=$2::uuid AND ($3::date IS NULL OR service_date >= $3::date) AND ($4::date IS NULL OR service_date <= $4::date) AND (($5='recurring' AND mass_template_id IS NOT NULL) OR ($5='special' AND special_mass_id IS NOT NULL)) ORDER BY service_date,service_time`
	rows, err := r.db.QueryContext(ctx, q, orgID, massNameID, nilIfEmpty(from), nilIfEmpty(to), source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []internal.MassAttendanceEntry{}
	for rows.Next() {
		var x internal.MassAttendanceEntry
		if err := rows.Scan(&x.ServiceDate, &x.ServiceTime, &x.AttendanceCount); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func rowsAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

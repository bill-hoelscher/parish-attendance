// Package internal contains the church-attendance application domain.
package internal

import (
	"context"
	"database/sql"
)

type Organization struct {
	ID               string  `json:"id"`
	BillingAccountID *string `json:"billingAccountId,omitempty"`
	IsActive         bool    `json:"isActive"`
	Name             string  `json:"name"`
	Address          string  `json:"address"`
	StreetAddress    string  `json:"streetAddress"`
	City             string  `json:"city"`
	StateProvince    string  `json:"stateProvince"`
	PostalCode       string  `json:"postalCode"`
	ContactName      string  `json:"contactName"`
	ContactEmail     string  `json:"contactEmail"`
	ContactPhone     string  `json:"contactPhone"`
	Timezone         string  `json:"timezone"`
}

// BillingAccount owns a Stripe-ready subscription and may cover one or more
// organizations. No Stripe API calls are made by this application yet.
type BillingAccount struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	BillingEmail      string `json:"billingEmail"`
	OrganizationCount int    `json:"organizationCount"`
}

// Subscription is the application's internal subscription record. Stripe IDs
// are optional until payment and webhook integration is introduced.
type Subscription struct {
	ID                   string  `json:"id"`
	BillingAccountID     string  `json:"billingAccountId"`
	PlanCode             string  `json:"planCode"`
	Status               string  `json:"status"`
	StripeSubscriptionID string  `json:"stripeSubscriptionId"`
	AccessThrough        *string `json:"accessThrough,omitempty"`
}

type OrganizationAccess struct {
	OrganizationID     string  `json:"organizationId"`
	OrganizationActive bool    `json:"organizationActive"`
	SubscriptionStatus string  `json:"subscriptionStatus"`
	AccessThrough      *string `json:"accessThrough,omitempty"`
	CanWrite           bool    `json:"canWrite"`
}

// Role is organization-scoped unless IsSystem is true. Its key is immutable
// and supports stable user assignments when its display name changes.
type Role struct {
	ID              string   `json:"id"`
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	Scope           string   `json:"scope"`
	IsSystem        bool     `json:"isSystem"`
	Permissions     []string `json:"permissions"`
	AssignmentCount int      `json:"assignmentCount"`
}

// UserAccess assigns an application role. System Administrators have no
// organization ID; all other roles are scoped to exactly one organization.
type UserAccess struct {
	ID             string  `json:"id"`
	UserID         string  `json:"userId"`
	OrganizationID *string `json:"organizationId,omitempty"`
	RoleID         string  `json:"roleId"`
	Role           string  `json:"role"`
	RoleName       string  `json:"roleName"`
}

// MassName is an organization-managed reporting identity shared by recurring
// templates and one-time special Masses.
type MassName struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	IsActive       bool   `json:"isActive"`
}

type MassTemplate struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organizationId"`
	MassNameID     *string `json:"massNameId,omitempty"`
	Name           string  `json:"name"`
	ServiceTime    string  `json:"serviceTime"`
	Weekday        int     `json:"weekday"`
	ActiveFrom     *string `json:"activeFrom,omitempty"`
	ActiveTo       *string `json:"activeTo,omitempty"`
	IsActive       bool    `json:"isActive"`
}

type SpecialMass struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organizationId"`
	Action         string  `json:"action"`
	ServiceDate    string  `json:"serviceDate"`
	ServiceTime    string  `json:"serviceTime"`
	Name           string  `json:"name"`
	MassNameID     *string `json:"massNameId,omitempty"`
	RecursAnnually bool    `json:"recursAnnually"`
	MassTemplateID *string `json:"massTemplateId,omitempty"`
}

type Attendance struct {
	ID               string  `json:"id"`
	OrganizationID   string  `json:"organizationId"`
	ServiceDate      string  `json:"serviceDate"`
	ServiceTime      string  `json:"serviceTime"`
	MassName         string  `json:"massName"`
	MassNameID       *string `json:"massNameId,omitempty"`
	MassTemplateID   *string `json:"massTemplateId,omitempty"`
	SpecialMassID    *string `json:"specialMassId,omitempty"`
	AttendanceCount  int     `json:"attendanceCount"`
	RecordedByUserID string  `json:"recordedByUserId"`
}

type MassAttendanceReport struct {
	MassNameID        string `json:"massNameId"`
	MassName          string `json:"massName"`
	ServicesCounted   int    `json:"servicesCounted"`
	AverageAttendance int    `json:"averageAttendance"`
	LowestAttendance  int    `json:"lowestAttendance"`
	HighestAttendance int    `json:"highestAttendance"`
}

type MassAttendanceEntry struct {
	ServiceDate     string `json:"serviceDate"`
	ServiceTime     string `json:"serviceTime"`
	AttendanceCount int    `json:"attendanceCount"`
}

type ScheduledMass struct {
	MassNameID     *string `json:"massNameId,omitempty"`
	MassTemplateID *string `json:"massTemplateId,omitempty"`
	SpecialMassID  *string `json:"specialMassId,omitempty"`
	Name           string  `json:"name"`
	ServiceDate    string  `json:"serviceDate"`
	ServiceTime    string  `json:"serviceTime"`
}

type AttendanceLedgerItem struct {
	ServiceDate     string  `json:"serviceDate"`
	ServiceTime     string  `json:"serviceTime"`
	MassName        string  `json:"massName"`
	MassNameID      *string `json:"massNameId,omitempty"`
	MassTemplateID  *string `json:"massTemplateId,omitempty"`
	SpecialMassID   *string `json:"specialMassId,omitempty"`
	AttendanceID    *string `json:"attendanceId,omitempty"`
	AttendanceCount *int    `json:"attendanceCount,omitempty"`
}

// Repository is the database port used by the application. Domain services can
// depend on this contract instead of a particular PostgreSQL driver.
type Repository interface {
	DB() *sql.DB
	Ping(ctx context.Context) error
	ListOrganizations(ctx context.Context) ([]Organization, error)
	ListAuthorizedOrganizations(ctx context.Context, userID string) ([]Organization, error)
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id string) error
	ListBillingAccounts(ctx context.Context) ([]BillingAccount, error)
	CreateBillingAccount(ctx context.Context, account *BillingAccount) error
	GetBillingAccount(ctx context.Context, id string) (*BillingAccount, error)
	UpdateBillingAccount(ctx context.Context, account *BillingAccount) error
	GetSubscription(ctx context.Context, billingAccountID string) (*Subscription, error)
	UpsertSubscription(ctx context.Context, subscription *Subscription) error
	OrganizationAccess(ctx context.Context, organizationID string) (*OrganizationAccess, error)
	ResourceOrganizationID(ctx context.Context, resource, id string) (string, error)
	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, id string) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id string) error
	RolePermissions(ctx context.Context, roleID string) ([]string, error)
	ListUserAccess(ctx context.Context, organizationID string) ([]UserAccess, error)
	CreateUserAccess(ctx context.Context, access *UserAccess) error
	GetUserAccess(ctx context.Context, id string) (*UserAccess, error)
	UpdateUserAccess(ctx context.Context, access *UserAccess) error
	DeleteUserAccess(ctx context.Context, id string) error
	UserRoles(ctx context.Context, userID, organizationID string) ([]UserAccess, error)
	ListMassNames(ctx context.Context, organizationID string) ([]MassName, error)
	CreateMassName(ctx context.Context, massName *MassName) error
	UpdateMassName(ctx context.Context, massName *MassName) error
	DeleteMassName(ctx context.Context, id string) error
	ListMassTemplates(ctx context.Context, organizationID string) ([]MassTemplate, error)
	CreateMassTemplate(ctx context.Context, template *MassTemplate) error
	UpdateMassTemplate(ctx context.Context, template *MassTemplate) error
	DeleteMassTemplate(ctx context.Context, id string) error
	ListSpecialMasses(ctx context.Context, organizationID string) ([]SpecialMass, error)
	CreateSpecialMass(ctx context.Context, mass *SpecialMass) error
	UpdateSpecialMass(ctx context.Context, mass *SpecialMass) error
	DeleteSpecialMass(ctx context.Context, id string) error
	ScheduledMasses(ctx context.Context, organizationID, date string) ([]ScheduledMass, error)
	ListAttendance(ctx context.Context, organizationID, from, to string) ([]Attendance, error)
	AttendanceLedger(ctx context.Context, organizationID, from, to string) ([]AttendanceLedgerItem, error)
	UpsertAttendance(ctx context.Context, attendance *Attendance) error
	UpdateAttendance(ctx context.Context, attendance *Attendance) error
	DeleteAttendance(ctx context.Context, id string) error
	MassAttendanceReport(ctx context.Context, organizationID, from, to, source string) ([]MassAttendanceReport, error)
	MassAttendanceEntries(ctx context.Context, organizationID, massNameID, from, to, source string) ([]MassAttendanceEntry, error)
}

type Services struct {
	Repository Repository
}

func NewServices(repository Repository) Services {
	return Services{Repository: repository}
}

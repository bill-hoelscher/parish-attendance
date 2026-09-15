// Package internal contains the parish-attendance application domain.
package internal

import (
	"context"
)

type Parish struct {
	ID               string  `json:"id"`
	DioceseID        *string `json:"dioceseId,omitempty"`
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

// Diocese is the parent grouping for parishes. Billing remains associated
// directly with parishes, not dioceses.
type Diocese struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BillingAccount owns a Stripe-ready subscription and may cover one or more
// parishes. No Stripe API calls are made by this application yet.
type BillingAccount struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BillingEmail string `json:"billingEmail"`
	ParishCount  int    `json:"parishCount"`
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

type ParishAccess struct {
	ParishID           string  `json:"parishId"`
	ParishActive       bool    `json:"parishActive"`
	SubscriptionStatus string  `json:"subscriptionStatus"`
	AccessThrough      *string `json:"accessThrough,omitempty"`
	CanWrite           bool    `json:"canWrite"`
}

// Role is parish-scoped unless IsSystem is true. Its key is immutable
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
// parish ID; all other roles are scoped to exactly one parish.
type UserAccess struct {
	ID       string  `json:"id"`
	UserID   string  `json:"userId"`
	Username string  `json:"username,omitempty"`
	Email    string  `json:"email,omitempty"`
	Status   string  `json:"status,omitempty"`
	ParishID *string `json:"parishId,omitempty"`
	RoleID   string  `json:"roleId"`
	Role     string  `json:"role"`
	RoleName string  `json:"roleName"`
}

// MassName is a parish-managed reporting identity shared by recurring
// templates and one-time special Masses.
type MassName struct {
	ID       string `json:"id"`
	ParishID string `json:"parishId"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

type MassTemplate struct {
	ID          string  `json:"id"`
	ParishID    string  `json:"parishId"`
	MassNameID  *string `json:"massNameId,omitempty"`
	Name        string  `json:"name"`
	ServiceTime string  `json:"serviceTime"`
	Weekday     int     `json:"weekday"`
	ActiveFrom  *string `json:"activeFrom,omitempty"`
	ActiveTo    *string `json:"activeTo,omitempty"`
	IsActive    bool    `json:"isActive"`
}

type SpecialMass struct {
	ID             string  `json:"id"`
	ParishID       string  `json:"parishId"`
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
	ParishID         string  `json:"parishId"`
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
// depend on this contract instead of a particular storage driver.
type Repository interface {
	Ping(ctx context.Context) error
	ListDioceses(ctx context.Context) ([]Diocese, error)
	CreateDiocese(ctx context.Context, diocese *Diocese) error
	GetDiocese(ctx context.Context, id string) (*Diocese, error)
	UpdateDiocese(ctx context.Context, diocese *Diocese) error
	DeleteDiocese(ctx context.Context, id string) error
	ListParishes(ctx context.Context) ([]Parish, error)
	ListAuthorizedParishes(ctx context.Context, userID string) ([]Parish, error)
	CreateParish(ctx context.Context, org *Parish) error
	GetParish(ctx context.Context, id string) (*Parish, error)
	UpdateParish(ctx context.Context, org *Parish) error
	DeleteParish(ctx context.Context, id string) error
	ListBillingAccounts(ctx context.Context) ([]BillingAccount, error)
	CreateBillingAccount(ctx context.Context, account *BillingAccount) error
	GetBillingAccount(ctx context.Context, id string) (*BillingAccount, error)
	UpdateBillingAccount(ctx context.Context, account *BillingAccount) error
	GetSubscription(ctx context.Context, billingAccountID string) (*Subscription, error)
	UpsertSubscription(ctx context.Context, subscription *Subscription) error
	ParishAccess(ctx context.Context, parishID string) (*ParishAccess, error)
	ResourceParishID(ctx context.Context, resource, id string) (string, error)
	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, id string) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id string) error
	RolePermissions(ctx context.Context, roleID string) ([]string, error)
	ListUserAccess(ctx context.Context, parishID string) ([]UserAccess, error)
	CreateUserAccess(ctx context.Context, access *UserAccess) error
	GetUserAccess(ctx context.Context, id string) (*UserAccess, error)
	UpdateUserAccess(ctx context.Context, access *UserAccess) error
	DeleteUserAccess(ctx context.Context, id string) error
	UserRoles(ctx context.Context, userID, parishID string) ([]UserAccess, error)
	ListMassNames(ctx context.Context, parishID string) ([]MassName, error)
	CreateMassName(ctx context.Context, massName *MassName) error
	UpdateMassName(ctx context.Context, massName *MassName) error
	DeleteMassName(ctx context.Context, id string) error
	ListMassTemplates(ctx context.Context, parishID string) ([]MassTemplate, error)
	CreateMassTemplate(ctx context.Context, template *MassTemplate) error
	UpdateMassTemplate(ctx context.Context, template *MassTemplate) error
	DeleteMassTemplate(ctx context.Context, id string) error
	ListSpecialMasses(ctx context.Context, parishID string) ([]SpecialMass, error)
	CreateSpecialMass(ctx context.Context, mass *SpecialMass) error
	UpdateSpecialMass(ctx context.Context, mass *SpecialMass) error
	DeleteSpecialMass(ctx context.Context, id string) error
	ScheduledMasses(ctx context.Context, parishID, date string) ([]ScheduledMass, error)
	ListAttendance(ctx context.Context, parishID, from, to string) ([]Attendance, error)
	AttendanceLedger(ctx context.Context, parishID, from, to string) ([]AttendanceLedgerItem, error)
	UpsertAttendance(ctx context.Context, attendance *Attendance) error
	UpdateAttendance(ctx context.Context, attendance *Attendance) error
	DeleteAttendance(ctx context.Context, id string) error
	MassAttendanceReport(ctx context.Context, parishID, from, to, source string) ([]MassAttendanceReport, error)
	MassAttendanceEntries(ctx context.Context, parishID, massNameID, from, to, source string) ([]MassAttendanceEntry, error)
}

type Services struct {
	Repository Repository
}

func NewServices(repository Repository) Services {
	return Services{Repository: repository}
}

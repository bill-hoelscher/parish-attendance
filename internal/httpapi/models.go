package httpapi

import "parishattendance/internal"

// HTTP handlers use aliases so request and response JSON remains unchanged
// while business concepts live in domain-specific packages.
type Parish = internal.Parish
type Diocese = internal.Diocese
type BillingAccount = internal.BillingAccount
type Subscription = internal.Subscription
type UserAccess = internal.UserAccess
type Role = internal.Role
type MassName = internal.MassName
type MassTemplate = internal.MassTemplate
type SpecialMass = internal.SpecialMass
type Attendance = internal.Attendance

// Package dynamodb implements the application's repository port with one
// DynamoDB table. Items retain a JSON domain payload while their keys model
// the application's query paths.
package dynamodb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"parishattendance/internal"
)

const (
	byIDIndex   = "LookupByEntityId"
	byUserIndex = "AccessByUser"
	byMassIndex = "AttendanceByMassReport"
)

type item struct {
	PartitionKey           string `dynamodbav:"PartitionKey"`
	SortKey                string `dynamodbav:"SortKey"`
	EntityType             string `dynamodbav:"EntityType"`
	ID                     string `dynamodbav:"EntityId"`
	ParishID               string `dynamodbav:"ParishId"`
	IdLookupPartitionKey   string `dynamodbav:"IdLookupPartitionKey,omitempty"`
	IdLookupSortKey        string `dynamodbav:"IdLookupSortKey,omitempty"`
	UserAccessPartitionKey string `dynamodbav:"UserAccessPartitionKey,omitempty"`
	UserAccessSortKey      string `dynamodbav:"UserAccessSortKey,omitempty"`
	MassReportPartitionKey string `dynamodbav:"MassReportPartitionKey,omitempty"`
	MassReportSortKey      string `dynamodbav:"MassReportSortKey,omitempty"`
	Data                   string `dynamodbav:"Payload"`
}

type Repository struct {
	client *dynamodb.Client
	table  string
}

func NewRepository(ctx context.Context, table string) (*Repository, error) {
	options := []func(*awsconfig.LoadOptions) error{}
	if endpoint := os.Getenv("DYNAMODB_ENDPOINT"); endpoint != "" {
		options = append(options, awsconfig.WithBaseEndpoint(endpoint))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, err
	}
	return &Repository{client: dynamodb.NewFromConfig(cfg), table: table}, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	_, err := r.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(r.table)})
	return err
}

func parishPK(id string) string  { return "PARISH#" + id }
func diocesePK(id string) string { return "DIOCESE#" + id }
func idPK(id string) string      { return "ID#" + id }

func (r *Repository) put(ctx context.Context, x item, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	x.Data = string(b)
	x.IdLookupPartitionKey, x.IdLookupSortKey = idPK(x.ID), x.EntityType
	m, err := attributevalue.MarshalMap(x)
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.table), Item: m})
	return err
}
func (r *Repository) delete(ctx context.Context, pk, sk string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.table), Key: map[string]types.AttributeValue{"PartitionKey": &types.AttributeValueMemberS{Value: pk}, "SortKey": &types.AttributeValueMemberS{Value: sk}}})
	return err
}
func (r *Repository) byID(ctx context.Context, id string) (item, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{TableName: aws.String(r.table), IndexName: aws.String(byIDIndex), KeyConditionExpression: aws.String("IdLookupPartitionKey = :id"), ExpressionAttributeValues: map[string]types.AttributeValue{":id": &types.AttributeValueMemberS{Value: idPK(id)}}})
	if err != nil {
		return item{}, err
	}
	if len(out.Items) == 0 {
		return item{}, sql.ErrNoRows
	}
	var x item
	err = attributevalue.UnmarshalMap(out.Items[0], &x)
	return x, err
}
func decode[T any](x item) (T, error) {
	var v T
	err := json.Unmarshal([]byte(x.Data), &v)
	return v, err
}
func (r *Repository) list(ctx context.Context, entity, org string) ([]item, error) {
	values := map[string]types.AttributeValue{}
	filter := ""
	if entity != "" {
		filter = "EntityType = :e"
		values[":e"] = &types.AttributeValueMemberS{Value: entity}
	}
	if org != "" {
		if filter != "" {
			filter += " AND "
		}
		filter += "ParishId = :o"
		values[":o"] = &types.AttributeValueMemberS{Value: org}
	}
	var all []item
	var start map[string]types.AttributeValue
	for {
		input := &dynamodb.ScanInput{TableName: aws.String(r.table), ExpressionAttributeValues: values, ExclusiveStartKey: start}
		if filter != "" {
			input.FilterExpression = aws.String(filter)
		}
		out, err := r.client.Scan(ctx, input)
		if err != nil {
			return nil, err
		}
		for _, m := range out.Items {
			var x item
			if err := attributevalue.UnmarshalMap(m, &x); err != nil {
				return nil, err
			}
			all = append(all, x)
		}
		if out.LastEvaluatedKey == nil {
			break
		}
		start = out.LastEvaluatedKey
	}
	return all, nil
}
func listValue[T any](ctx context.Context, r *Repository, entity, org string) ([]T, error) {
	xs, e := r.list(ctx, entity, org)
	if e != nil {
		return nil, e
	}
	out := make([]T, 0, len(xs))
	for _, x := range xs {
		v, e := decode[T](x)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}
func getValue[T any](ctx context.Context, r *Repository, id string) (T, error) {
	x, e := r.byID(ctx, id)
	if e != nil {
		var z T
		return z, e
	}
	return decode[T](x)
}

func (r *Repository) ListDioceses(ctx context.Context) ([]internal.Diocese, error) {
	x, err := listValue[internal.Diocese](ctx, r, "diocese", "")
	sort.Slice(x, func(i, j int) bool { return x[i].Name < x[j].Name })
	return x, err
}
func (r *Repository) CreateDiocese(ctx context.Context, x *internal.Diocese) error {
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, item{PartitionKey: diocesePK(x.ID), SortKey: "DIOCESE", EntityType: "diocese", ID: x.ID}, x)
}
func (r *Repository) GetDiocese(ctx context.Context, id string) (*internal.Diocese, error) {
	x, err := getValue[internal.Diocese](ctx, r, id)
	return &x, err
}
func (r *Repository) UpdateDiocese(ctx context.Context, x *internal.Diocese) error {
	old, err := r.byID(ctx, x.ID)
	if err != nil {
		return err
	}
	return r.put(ctx, item{PartitionKey: old.PartitionKey, SortKey: old.SortKey, EntityType: "diocese", ID: x.ID}, x)
}
func (r *Repository) DeleteDiocese(ctx context.Context, id string) error {
	x, err := r.byID(ctx, id)
	if err != nil {
		return err
	}
	return r.delete(ctx, x.PartitionKey, x.SortKey)
}

func (r *Repository) ListParishes(ctx context.Context) ([]internal.Parish, error) {
	x, e := listValue[internal.Parish](ctx, r, "parish", "")
	sort.Slice(x, func(i, j int) bool { return x[i].Name < x[j].Name })
	return x, e
}
func (r *Repository) ListAuthorizedParishes(ctx context.Context, user string) ([]internal.Parish, error) {
	a, e := r.UserRoles(ctx, user, "")
	if e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	out := []internal.Parish{}
	for _, v := range a {
		if v.Role == "system_administrator" {
			return r.ListParishes(ctx)
		}
		if v.ParishID != nil && !seen[*v.ParishID] {
			o, e := r.GetParish(ctx, *v.ParishID)
			if e != nil {
				return nil, e
			}
			seen[o.ID] = true
			out = append(out, *o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (r *Repository) CreateParish(ctx context.Context, x *internal.Parish) error {
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, item{PartitionKey: parishPK(x.ID), SortKey: "PARISH", EntityType: "parish", ID: x.ID, ParishID: x.ID}, x)
}
func (r *Repository) GetParish(ctx context.Context, id string) (*internal.Parish, error) {
	x, e := getValue[internal.Parish](ctx, r, id)
	return &x, e
}
func (r *Repository) UpdateParish(ctx context.Context, x *internal.Parish) error {
	old, e := r.byID(ctx, x.ID)
	if e != nil {
		return e
	}
	return r.put(ctx, item{PartitionKey: old.PartitionKey, SortKey: old.SortKey, EntityType: "parish", ID: x.ID, ParishID: x.ID}, x)
}
func (r *Repository) DeleteParish(ctx context.Context, id string) error {
	xs, e := r.list(ctx, "", id)
	if e != nil {
		return e
	}
	if len(xs) == 0 {
		return sql.ErrNoRows
	}
	for _, x := range xs {
		if e := r.delete(ctx, x.PartitionKey, x.SortKey); e != nil {
			return e
		}
	}
	return nil
}

func (r *Repository) ListBillingAccounts(ctx context.Context) ([]internal.BillingAccount, error) {
	x, e := listValue[internal.BillingAccount](ctx, r, "billing", "")
	if e != nil {
		return nil, e
	}
	orgs, e := r.ListParishes(ctx)
	if e != nil {
		return nil, e
	}
	setParishCounts(x, orgs)
	sort.Slice(x, func(i, j int) bool { return x[i].Name < x[j].Name })
	return x, nil
}
func (r *Repository) CreateBillingAccount(ctx context.Context, x *internal.BillingAccount) error {
	if x.ID == "" {
		x.ID = newID()
	}
	x.ParishCount = 0
	return r.put(ctx, item{PartitionKey: "BILLING#" + x.ID, SortKey: "ACCOUNT", EntityType: "billing", ID: x.ID}, x)
}
func (r *Repository) GetBillingAccount(ctx context.Context, id string) (*internal.BillingAccount, error) {
	x, e := getValue[internal.BillingAccount](ctx, r, id)
	if e != nil {
		return &x, e
	}
	all, e := r.ListBillingAccounts(ctx)
	if e != nil {
		return &x, e
	}
	for _, a := range all {
		if a.ID == id {
			return &a, nil
		}
	}
	return &x, sql.ErrNoRows
}
func (r *Repository) UpdateBillingAccount(ctx context.Context, x *internal.BillingAccount) error {
	old, e := r.byID(ctx, x.ID)
	if e != nil {
		return e
	}
	x.ParishCount = 0
	return r.put(ctx, item{PartitionKey: old.PartitionKey, SortKey: old.SortKey, EntityType: "billing", ID: x.ID}, x)
}

// ParishCount is derived from parish records and must never be
// trusted from a stored billing-account payload or import source.
func setParishCounts(accounts []internal.BillingAccount, parishes []internal.Parish) {
	for i := range accounts {
		accounts[i].ParishCount = 0
		for _, parish := range parishes {
			if parish.BillingAccountID != nil && *parish.BillingAccountID == accounts[i].ID {
				accounts[i].ParishCount++
			}
		}
	}
}
func (r *Repository) GetSubscription(ctx context.Context, billing string) (*internal.Subscription, error) {
	xs, e := listValue[internal.Subscription](ctx, r, "subscription", "")
	if e != nil {
		return nil, e
	}
	for _, x := range xs {
		if x.BillingAccountID == billing {
			return &x, nil
		}
	}
	return nil, sql.ErrNoRows
}
func (r *Repository) UpsertSubscription(ctx context.Context, x *internal.Subscription) error {
	if x.ID == "" {
		if old, e := r.GetSubscription(ctx, x.BillingAccountID); e == nil {
			x.ID = old.ID
		} else {
			x.ID = newID()
		}
	}
	return r.put(ctx, item{PartitionKey: "BILLING#" + x.BillingAccountID, SortKey: "SUBSCRIPTION", EntityType: "subscription", ID: x.ID}, x)
}
func (r *Repository) ParishAccess(ctx context.Context, id string) (*internal.ParishAccess, error) {
	o, e := r.GetParish(ctx, id)
	if e != nil {
		return nil, e
	}
	a := &internal.ParishAccess{ParishID: id, ParishActive: o.IsActive, SubscriptionStatus: "canceled"}
	if o.BillingAccountID != nil {
		if s, e := r.GetSubscription(ctx, *o.BillingAccountID); e == nil {
			a.SubscriptionStatus = s.Status
			a.AccessThrough = s.AccessThrough
		}
	}
	a.CanWrite = a.ParishActive && (a.SubscriptionStatus == "active" || a.SubscriptionStatus == "trialing")
	if a.AccessThrough != nil && *a.AccessThrough < time.Now().UTC().Format("2006-01-02") {
		a.CanWrite = false
	}
	return a, nil
}
func (r *Repository) ResourceParishID(ctx context.Context, resource, id string) (string, error) {
	x, e := r.byID(ctx, id)
	if e != nil {
		return "", e
	}
	if x.EntityType != resource {
		return "", sql.ErrNoRows
	}
	return x.ParishID, nil
}

func (r *Repository) ListRoles(ctx context.Context) ([]internal.Role, error) {
	x, e := listValue[internal.Role](ctx, r, "role", "")
	if e != nil {
		return nil, e
	}
	a, e := listValue[internal.UserAccess](ctx, r, "access", "")
	if e != nil {
		return nil, e
	}
	for i := range x {
		for _, u := range a {
			if u.RoleID == x[i].ID {
				x[i].AssignmentCount++
			}
		}
	}
	sort.Slice(x, func(i, j int) bool { return x[i].Name < x[j].Name })
	return x, nil
}
func (r *Repository) CreateRole(ctx context.Context, x *internal.Role) error {
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, item{PartitionKey: "ROLE#" + x.ID, SortKey: "ROLE", EntityType: "role", ID: x.ID}, x)
}
func (r *Repository) GetRole(ctx context.Context, id string) (*internal.Role, error) {
	x, e := getValue[internal.Role](ctx, r, id)
	return &x, e
}
func (r *Repository) UpdateRole(ctx context.Context, x *internal.Role) error {
	old, e := r.byID(ctx, x.ID)
	if e != nil {
		return e
	}
	return r.put(ctx, item{PartitionKey: old.PartitionKey, SortKey: old.SortKey, EntityType: "role", ID: x.ID}, x)
}
func (r *Repository) DeleteRole(ctx context.Context, id string) error {
	old, e := r.byID(ctx, id)
	if e != nil {
		return e
	}
	return r.delete(ctx, old.PartitionKey, old.SortKey)
}
func (r *Repository) RolePermissions(ctx context.Context, id string) ([]string, error) {
	x, e := r.GetRole(ctx, id)
	if e != nil {
		return nil, e
	}
	return x.Permissions, nil
}

func (r *Repository) ListUserAccess(ctx context.Context, org string) ([]internal.UserAccess, error) {
	x, e := listValue[internal.UserAccess](ctx, r, "access", "")
	if e != nil {
		return nil, e
	}
	if org == "" {
		return x, nil
	}
	out := []internal.UserAccess{}
	for _, v := range x {
		if v.ParishID != nil && *v.ParishID == org {
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *Repository) CreateUserAccess(ctx context.Context, x *internal.UserAccess) error {
	if x.ID == "" {
		x.ID = newID()
	}
	sk := "SYSTEM"
	if x.ParishID != nil {
		sk = "PARISH#" + *x.ParishID
	}
	return r.put(ctx, item{PartitionKey: "ACCESS#" + x.ID, SortKey: sk, EntityType: "access", ID: x.ID, ParishID: deref(x.ParishID), UserAccessPartitionKey: "USER#" + x.UserID, UserAccessSortKey: sk}, x)
}
func (r *Repository) GetUserAccess(ctx context.Context, id string) (*internal.UserAccess, error) {
	x, e := getValue[internal.UserAccess](ctx, r, id)
	return &x, e
}
func (r *Repository) UpdateUserAccess(ctx context.Context, x *internal.UserAccess) error {
	old, e := r.byID(ctx, x.ID)
	if e != nil {
		return e
	}
	if e = r.delete(ctx, old.PartitionKey, old.SortKey); e != nil {
		return e
	}
	return r.CreateUserAccess(ctx, x)
}
func (r *Repository) DeleteUserAccess(ctx context.Context, id string) error {
	old, e := r.byID(ctx, id)
	if e != nil {
		return e
	}
	return r.delete(ctx, old.PartitionKey, old.SortKey)
}
func (r *Repository) UserRoles(ctx context.Context, user, org string) ([]internal.UserAccess, error) {
	out, e := r.client.Query(ctx, &dynamodb.QueryInput{TableName: aws.String(r.table), IndexName: aws.String(byUserIndex), KeyConditionExpression: aws.String("UserAccessPartitionKey = :u"), ExpressionAttributeValues: map[string]types.AttributeValue{":u": &types.AttributeValueMemberS{Value: "USER#" + user}}})
	if e != nil {
		return nil, e
	}
	xs := []internal.UserAccess{}
	for _, m := range out.Items {
		var it item
		if e := attributevalue.UnmarshalMap(m, &it); e != nil {
			return nil, e
		}
		v, e := decode[internal.UserAccess](it)
		if e != nil {
			return nil, e
		}
		if org == "" || v.ParishID == nil || *v.ParishID == org {
			xs = append(xs, v)
		}
	}
	return xs, nil
}

func (r *Repository) ListMassNames(ctx context.Context, org string) ([]internal.MassName, error) {
	return listValue[internal.MassName](ctx, r, "mass_name", org)
}
func (r *Repository) CreateMassName(ctx context.Context, x *internal.MassName) error {
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, item{PartitionKey: parishPK(x.ParishID), SortKey: "MASS#" + x.ID, EntityType: "mass_name", ID: x.ID, ParishID: x.ParishID}, x)
}
func (r *Repository) UpdateMassName(ctx context.Context, x *internal.MassName) error {
	return r.updateOrg(ctx, "mass_name", x.ID, x.ParishID, x)
}
func (r *Repository) DeleteMassName(ctx context.Context, id string) error {
	return r.deleteID(ctx, "mass_name", id)
}
func (r *Repository) ListMassTemplates(ctx context.Context, org string) ([]internal.MassTemplate, error) {
	return listValue[internal.MassTemplate](ctx, r, "mass_template", org)
}
func (r *Repository) CreateMassTemplate(ctx context.Context, x *internal.MassTemplate) error {
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, item{PartitionKey: parishPK(x.ParishID), SortKey: "TEMPLATE#" + x.ID, EntityType: "mass_template", ID: x.ID, ParishID: x.ParishID}, x)
}
func (r *Repository) UpdateMassTemplate(ctx context.Context, x *internal.MassTemplate) error {
	return r.updateOrg(ctx, "mass_template", x.ID, x.ParishID, x)
}
func (r *Repository) DeleteMassTemplate(ctx context.Context, id string) error {
	return r.deleteID(ctx, "mass_template", id)
}
func (r *Repository) ListSpecialMasses(ctx context.Context, org string) ([]internal.SpecialMass, error) {
	return listValue[internal.SpecialMass](ctx, r, "special_mass", org)
}
func (r *Repository) CreateSpecialMass(ctx context.Context, x *internal.SpecialMass) error {
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, item{PartitionKey: parishPK(x.ParishID), SortKey: "SPECIAL#" + x.ServiceDate + "#" + x.ServiceTime + "#" + x.ID, EntityType: "special_mass", ID: x.ID, ParishID: x.ParishID}, x)
}
func (r *Repository) UpdateSpecialMass(ctx context.Context, x *internal.SpecialMass) error {
	return r.updateOrg(ctx, "special_mass", x.ID, x.ParishID, x)
}
func (r *Repository) DeleteSpecialMass(ctx context.Context, id string) error {
	return r.deleteID(ctx, "special_mass", id)
}

func (r *Repository) ScheduledMasses(ctx context.Context, org, date string) ([]internal.ScheduledMass, error) {
	ts, e := r.ListMassTemplates(ctx, org)
	if e != nil {
		return nil, e
	}
	ss, e := r.ListSpecialMasses(ctx, org)
	if e != nil {
		return nil, e
	}
	names, e := r.ListMassNames(ctx, org)
	if e != nil {
		return nil, e
	}
	nm := map[string]string{}
	for _, n := range names {
		nm[n.ID] = n.Name
	}
	day, e := time.Parse("2006-01-02", date)
	if e != nil {
		return nil, e
	}
	out := []internal.ScheduledMass{}
	for _, t := range ts {
		if !t.IsActive || t.Weekday != int(day.Weekday()) || (t.ActiveFrom != nil && *t.ActiveFrom > date) || (t.ActiveTo != nil && *t.ActiveTo < date) {
			continue
		}
		cancel := false
		for _, s := range ss {
			if s.Action == "CANCEL" && s.ServiceDate == date && s.MassTemplateID != nil && *s.MassTemplateID == t.ID {
				cancel = true
			}
		}
		if !cancel {
			out = append(out, internal.ScheduledMass{MassNameID: t.MassNameID, MassTemplateID: &t.ID, Name: nm[deref(t.MassNameID)], ServiceDate: date, ServiceTime: t.ServiceTime})
		}
	}
	for _, s := range ss {
		if s.Action != "ADD" || !(s.ServiceDate == date || (s.RecursAnnually && s.ServiceDate[5:] == date[5:])) {
			continue
		}
		out = append(out, internal.ScheduledMass{MassNameID: s.MassNameID, SpecialMassID: &s.ID, Name: nm[deref(s.MassNameID)], ServiceDate: date, ServiceTime: s.ServiceTime})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServiceTime < out[j].ServiceTime })
	return out, nil
}

func (r *Repository) ListAttendance(ctx context.Context, org, from, to string) ([]internal.Attendance, error) {
	values := map[string]types.AttributeValue{
		":pk":     &types.AttributeValueMemberS{Value: parishPK(org)},
		":prefix": &types.AttributeValueMemberS{Value: "ATTENDANCE#"},
	}
	var records []item
	var start map[string]types.AttributeValue
	for {
		page, err := r.client.Query(ctx, &dynamodb.QueryInput{
			TableName:                 aws.String(r.table),
			KeyConditionExpression:    aws.String("PartitionKey = :pk AND begins_with(SortKey, :prefix)"),
			ExpressionAttributeValues: values,
			ExclusiveStartKey:         start,
		})
		if err != nil {
			return nil, err
		}
		for _, raw := range page.Items {
			var record item
			if err := attributevalue.UnmarshalMap(raw, &record); err != nil {
				return nil, err
			}
			records = append(records, record)
		}
		if page.LastEvaluatedKey == nil {
			break
		}
		start = page.LastEvaluatedKey
	}
	xs := make([]internal.Attendance, 0, len(records))
	for _, record := range records {
		value, err := decode[internal.Attendance](record)
		if err != nil {
			return nil, err
		}
		xs = append(xs, value)
	}
	out := filterAttendance(xs, from, to)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ServiceDate > out[j].ServiceDate || (out[i].ServiceDate == out[j].ServiceDate && out[i].ServiceTime < out[j].ServiceTime)
	})
	return out, nil
}
func (r *Repository) AttendanceLedger(ctx context.Context, org, from, to string) ([]internal.AttendanceLedgerItem, error) {
	start, e := time.Parse("2006-01-02", from)
	if e != nil {
		return nil, e
	}
	end, e := time.Parse("2006-01-02", to)
	if e != nil {
		return nil, e
	}
	all, e := r.ListAttendance(ctx, org, from, to)
	if e != nil {
		return nil, e
	}
	recorded := map[string]internal.Attendance{}
	for _, a := range all {
		recorded[a.ServiceDate+"#"+a.ServiceTime] = a
	}
	out := []internal.AttendanceLedgerItem{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		scheduled, e := r.ScheduledMasses(ctx, org, date)
		if e != nil {
			return nil, e
		}
		for _, s := range scheduled {
			k := date + "#" + s.ServiceTime
			v, ok := recorded[k]
			entry := internal.AttendanceLedgerItem{ServiceDate: date, ServiceTime: s.ServiceTime, MassName: s.Name, MassNameID: s.MassNameID, MassTemplateID: s.MassTemplateID, SpecialMassID: s.SpecialMassID}
			if ok {
				entry.AttendanceID = &v.ID
				entry.AttendanceCount = &v.AttendanceCount
			}
			out = append(out, entry)
		}
	}
	return out, nil
}
func (r *Repository) UpsertAttendance(ctx context.Context, x *internal.Attendance) error {
	existing, e := r.ListAttendance(ctx, x.ParishID, x.ServiceDate, x.ServiceDate)
	if e != nil {
		return e
	}
	for _, v := range existing {
		if v.ServiceTime == x.ServiceTime {
			x.ID = v.ID
			return r.UpdateAttendance(ctx, x)
		}
	}
	if x.ID == "" {
		x.ID = newID()
	}
	return r.put(ctx, attendanceItem(*x), x)
}
func (r *Repository) UpdateAttendance(ctx context.Context, x *internal.Attendance) error {
	old, e := r.byID(ctx, x.ID)
	if e != nil {
		return e
	}
	if old.EntityType != "attendance" {
		return sql.ErrNoRows
	}
	x.ParishID = old.ParishID
	if e = r.delete(ctx, old.PartitionKey, old.SortKey); e != nil {
		return e
	}
	return r.put(ctx, attendanceItem(*x), x)
}
func (r *Repository) DeleteAttendance(ctx context.Context, id string) error {
	return r.deleteID(ctx, "attendance", id)
}
func attendanceItem(x internal.Attendance) item {
	source := "recurring"
	if x.SpecialMassID != nil {
		source = "special"
	}
	return item{PartitionKey: parishPK(x.ParishID), SortKey: "ATTENDANCE#" + x.ServiceDate + "#" + x.ServiceTime, EntityType: "attendance", ID: x.ID, ParishID: x.ParishID, MassReportPartitionKey: "PARISH#" + x.ParishID + "#MASS#" + deref(x.MassNameID) + "#" + source, MassReportSortKey: x.ServiceDate + "#" + x.ServiceTime}
}
func (r *Repository) MassAttendanceReport(ctx context.Context, org, from, to, source string) ([]internal.MassAttendanceReport, error) {
	xs, e := r.ListAttendance(ctx, org, from, to)
	if e != nil {
		return nil, e
	}
	names, e := r.ListMassNames(ctx, org)
	if e != nil {
		return nil, e
	}
	nm := map[string]string{}
	for _, n := range names {
		nm[n.ID] = n.Name
	}
	type agg struct{ n, sum, min, max int }
	m := map[string]*agg{}
	for _, x := range xs {
		if !reportAttendanceMatchesSource(x, source) {
			continue
		}
		id := deref(x.MassNameID)
		if m[id] == nil {
			m[id] = &agg{min: x.AttendanceCount, max: x.AttendanceCount}
		}
		a := m[id]
		a.n++
		a.sum += x.AttendanceCount
		if x.AttendanceCount < a.min {
			a.min = x.AttendanceCount
		}
		if x.AttendanceCount > a.max {
			a.max = x.AttendanceCount
		}
	}
	out := []internal.MassAttendanceReport{}
	for id, a := range m {
		out = append(out, internal.MassAttendanceReport{MassNameID: id, MassName: nm[id], ServicesCounted: a.n, AverageAttendance: (a.sum + a.n/2) / a.n, LowestAttendance: a.min, HighestAttendance: a.max})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MassName < out[j].MassName })
	return out, nil
}
func (r *Repository) MassAttendanceEntries(ctx context.Context, org, mass, from, to, source string) ([]internal.MassAttendanceEntry, error) {
	xs, e := r.ListAttendance(ctx, org, from, to)
	if e != nil {
		return nil, e
	}
	out := []internal.MassAttendanceEntry{}
	for _, x := range xs {
		if deref(x.MassNameID) == mass && reportAttendanceMatchesSource(x, source) {
			out = append(out, internal.MassAttendanceEntry{ServiceDate: x.ServiceDate, ServiceTime: x.ServiceTime, AttendanceCount: x.AttendanceCount})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ServiceDate < out[j].ServiceDate || (out[i].ServiceDate == out[j].ServiceDate && out[i].ServiceTime < out[j].ServiceTime)
	})
	return out, nil
}

func (r *Repository) WeekendAttendanceTotals(ctx context.Context, parishID, from, to string) ([]internal.WeekendAttendanceTotal, error) {
	attendance, err := r.ListAttendance(ctx, parishID, from, to)
	if err != nil {
		return nil, err
	}
	totals := map[string]*internal.WeekendAttendanceTotal{}
	for _, record := range attendance {
		if record.SpecialMassID != nil {
			continue
		}
		serviceDate, err := time.Parse("2006-01-02", record.ServiceDate)
		if err != nil || (serviceDate.Weekday() != time.Saturday && serviceDate.Weekday() != time.Sunday) {
			continue
		}
		weekendStart := serviceDate
		if serviceDate.Weekday() == time.Sunday {
			weekendStart = weekendStart.AddDate(0, 0, -1)
		}
		key := weekendStart.Format("2006-01-02")
		if totals[key] == nil {
			totals[key] = &internal.WeekendAttendanceTotal{
				WeekendStart: key,
				WeekendEnd:   weekendStart.AddDate(0, 0, 1).Format("2006-01-02"),
			}
		}
		totals[key].AttendanceTotal += record.AttendanceCount
	}
	result := make([]internal.WeekendAttendanceTotal, 0, len(totals))
	for _, total := range totals {
		result = append(result, *total)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].WeekendStart < result[j].WeekendStart })
	return result, nil
}

func reportAttendanceMatchesSource(x internal.Attendance, source string) bool {
	if source == "" {
		return true
	}
	if x.SpecialMassID != nil {
		return source == "special"
	}
	if source == "recurring" {
		return true
	}
	date, err := time.Parse("2006-01-02", x.ServiceDate)
	if err != nil {
		return false
	}
	weekend := date.Weekday() == time.Saturday || date.Weekday() == time.Sunday
	return (source == "weekend" && weekend) || (source == "weekday" && !weekend)
}

func (r *Repository) updateOrg(ctx context.Context, entity, id, org string, value any) error {
	old, e := r.byID(ctx, id)
	if e != nil {
		return e
	}
	if old.EntityType != entity {
		return sql.ErrNoRows
	}
	org = old.ParishID
	switch v := value.(type) {
	case *internal.MassName:
		v.ParishID = org
	case *internal.MassTemplate:
		v.ParishID = org
	case *internal.SpecialMass:
		v.ParishID = org
	}
	if e = r.delete(ctx, old.PartitionKey, old.SortKey); e != nil {
		return e
	}
	prefix := map[string]string{"mass_name": "MASS#", "mass_template": "TEMPLATE#", "special_mass": "SPECIAL#"}[entity]
	sk := prefix + id
	if entity == "special_mass" {
		v := value.(*internal.SpecialMass)
		sk = "SPECIAL#" + v.ServiceDate + "#" + v.ServiceTime + "#" + id
	}
	return r.put(ctx, item{PartitionKey: parishPK(org), SortKey: sk, EntityType: entity, ID: id, ParishID: org}, value)
}
func (r *Repository) deleteID(ctx context.Context, entity, id string) error {
	old, e := r.byID(ctx, id)
	if e != nil {
		return e
	}
	if old.EntityType != entity {
		return sql.ErrNoRows
	}
	return r.delete(ctx, old.PartitionKey, old.SortKey)
}
func deref(x *string) string {
	if x == nil {
		return ""
	}
	return *x
}
func filterAttendance(xs []internal.Attendance, from, to string) []internal.Attendance {
	out := []internal.Attendance{}
	for _, x := range xs {
		if (from == "" || x.ServiceDate >= from) && (to == "" || x.ServiceDate <= to) {
			out = append(out, x)
		}
	}
	return out
}
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

var _ internal.Repository = (*Repository)(nil)

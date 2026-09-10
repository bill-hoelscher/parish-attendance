# Parish Attendance API

Go REST API and DynamoDB schema for recording parish attendance by mass.

## Run locally

Create the DynamoDB table with the CloudFormation template in
[`infra/dynamodb.yaml`](infra/dynamodb.yaml), then configure AWS credentials,
an AWS Region, and the table name:

```sh
aws cloudformation deploy \
  --template-file infra/dynamodb.yaml \
  --stack-name parish-attendance-data \
  --region us-east-1

export AWS_REGION=us-east-1
export DYNAMODB_TABLE=parish-attendance
go run ./cmd/attendanceapi
```

The API listens on `http://localhost:8080`. The runtime no longer accepts a
PostgreSQL connection string.

CloudFormation creates AWS resources; DynamoDB Local does not implement
CloudFormation. For local development, create the equivalent table with the
AWS CLI and then add:

```sh
export DYNAMODB_ENDPOINT=http://localhost:8000
```

## Import existing PostgreSQL data

After creating the table, make the existing PostgreSQL database read-only and
run the one-time importer:

```sh
export DATABASE_URL='postgres://...'
export AWS_REGION=us-east-1
export DYNAMODB_TABLE=parish-attendance
go run ./cmd/migrate-postgres-to-dynamodb
```

The importer preserves record IDs and copies billing accounts, subscriptions,
roles, access assignments, organizations, schedules, special Masses, and
attendance. Run it again only while PostgreSQL remains the source of truth;
the API itself uses DynamoDB exclusively.

## Legacy PostgreSQL preparation

These commands apply only when preparing an existing PostgreSQL database for
the one-time DynamoDB importer. PostgreSQL is not used by the running API.
PostgreSQL only runs container initialization scripts when its data volume is
created. If you already have a database, apply the billing foundation migration:

```sh
psql "$DATABASE_URL" -f db/migrations/005_billing_foundation.sql
psql "$DATABASE_URL" -f db/migrations/006_role_permissions.sql
psql "$DATABASE_URL" -f db/migrations/007_editable_roles.sql
```

The migration attaches existing organizations to an active `Default billing
account`, so existing attendance entry remains available. Rename that account or
create additional accounts in **Administration → Billing accounts**. This release
does not send requests to Stripe or collect payment details.

The second migration adds System Administrator, Organization Administrator, and
Attendance Counter role assignments. Authentication is still intentionally not
configured: requests without `X-User-ID` retain open local-development access;
requests carrying that header are checked against the assigned role.

The third migration turns the organization-scoped roles into editable roles
with individually assigned permissions. System Administrator remains built in
and cannot be modified or deleted.

## Attendance workflow

1. Create an organization.
2. Create its recurring `mass-templates` once (`weekday` uses `0` for Sunday through `6` for Saturday).
3. Ask `GET /organizations/{organizationId}/scheduled-masses?date=2026-08-30` to pre-fill the day’s expected services.
4. `POST /organizations/{organizationId}/attendance` saves a count; a second POST for the same mass date/time updates it.
5. Use `GET /organizations/{organizationId}/reports/mass-attendance?from=2026-01-01&to=2026-06-30` for per-mass averages.

## Example: recurring mass and attendance

```sh
curl -X POST http://localhost:8080/organizations \
  -H 'Content-Type: application/json' \
  -d '{"name":"St. Mary Parish","timezone":"America/Chicago"}'

curl -X POST http://localhost:8080/organizations/ORG_ID/mass-templates \
  -H 'Content-Type: application/json' \
  -d '{"name":"Sunday 9:00 AM","weekday":0,"serviceTime":"09:00","isActive":true}'

curl -X POST http://localhost:8080/organizations/ORG_ID/attendance \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: local-user' \
  -d '{"serviceDate":"2026-08-30","serviceTime":"09:00","massName":"Sunday 9:00 AM","massTemplateId":"TEMPLATE_ID","attendanceCount":218}'
```

Authentication is intentionally not wired in yet. The `X-User-ID` header is a local-development placeholder; replace it with verified Cognito JWT claims before deployment.

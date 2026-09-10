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
  --parameter-overrides CognitoDomainPrefix=your-unique-prefix \
  --region us-east-1

export AWS_REGION=us-east-1
export DYNAMODB_TABLE=parish-attendance
```

The API listens on `http://localhost:8080`.

CloudFormation creates AWS resources; DynamoDB Local does not implement
CloudFormation. For local development, create the equivalent table with the
AWS CLI and then add:

```sh
export DYNAMODB_ENDPOINT=http://localhost:8000
```

## Authentication and first administrator

The application uses a Cognito User Pool with hosted login. Cognito accepts
only administrator-created accounts and emails each new user a temporary
password. Deploy the stack, then export its Cognito outputs and choose the
local callback URL configured in the stack:

```sh
export COGNITO_REGION=us-east-1
export COGNITO_USER_POOL_ID=us-east-1_example
export COGNITO_CLIENT_ID=exampleclientid
export COGNITO_DOMAIN=your-unique-prefix.auth.us-east-1.amazoncognito.com
export COGNITO_REDIRECT_URI=http://localhost:8080/app/
export APP_ORIGIN=http://localhost:8080
```

Initialize the built-in roles and send the first system administrator an email
with a temporary password. This command uses the same DynamoDB settings as the
application, so it works with DynamoDB Local when `DYNAMODB_ENDPOINT` is set:

```sh
go run ./cmd/bootstrap-admin --email admin@example.org
go run ./cmd/attendanceapi
```

Open `http://localhost:8080/app/`. Sign in with the emailed temporary
password, create billing accounts and organizations, then organization
administrators can invite accounts from **User access**. Invited users receive
their own temporary-password email and can only access their assigned
organization.

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
  -H 'Authorization: Bearer COGNITO_ACCESS_TOKEN' \
  -d '{"serviceDate":"2026-08-30","serviceTime":"09:00","massName":"Sunday 9:00 AM","massTemplateId":"TEMPLATE_ID","attendanceCount":218}'
```

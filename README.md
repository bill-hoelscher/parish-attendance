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

The API listens on `http://localhost:8080`.

CloudFormation creates AWS resources; DynamoDB Local does not implement
CloudFormation. For local development, create the equivalent table with the
AWS CLI and then add:

```sh
export DYNAMODB_ENDPOINT=http://localhost:8000
```

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

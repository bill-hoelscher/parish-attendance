# Parish Attendance API

Go REST API and DynamoDB schema for recording parish attendance by mass.

## Overview

The same binary runs locally as an HTTP server and in AWS as a single Lambda
behind a Lambda Function URL. DynamoDB Local is still supported for local
development.

## Deploy to AWS

[`infra/parish-attendance-infrastructure.yaml`](infra/parish-attendance-infrastructure.yaml)
creates Cognito and the on-demand DynamoDB table. The dependent
[`infra/parish-attendance-lambda.yaml`](infra/parish-attendance-lambda.yaml)
creates the Lambda, optional CloudFront distribution, and public application URL. Create a globally unique deployment
bucket once, build the custom-runtime executable as `bootstrap`, and upload it:

```sh
export AWS_REGION=us-east-1
export ENVIRONMENT=dev
export INFRASTRUCTURE_STACK=parish-attendance-${ENVIRONMENT}-infrastructure
export LAMBDA_STACK=parish-attendance-${ENVIRONMENT}-lambda
export DEPLOYMENT_BUCKET=parish-attendance-artifacts-${ENVIRONMENT}
export DEPLOYMENT_KEY=parish-attendance/bootstrap.zip

aws s3api create-bucket --bucket "$DEPLOYMENT_BUCKET" --region "$AWS_REGION"
mkdir -p build
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o build/bootstrap ./cmd/attendanceapi
(cd build && zip -j bootstrap.zip bootstrap)
aws s3 cp build/bootstrap.zip "s3://$DEPLOYMENT_BUCKET/$DEPLOYMENT_KEY"

aws cloudformation deploy \
  --template-file infra/parish-attendance-infrastructure.yaml \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    CognitoDomainPrefix=parish-attendance-${ENVIRONMENT} \
  --region "$AWS_REGION"

aws cloudformation deploy \
  --template-file infra/parish-attendance-lambda.yaml \
  --stack-name "$LAMBDA_STACK" \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    DeploymentBucket="$DEPLOYMENT_BUCKET" \
    DeploymentKey="$DEPLOYMENT_KEY" \
  --region "$AWS_REGION"
```

Read the generated CloudFront URL, then update the Cognito callback and the
Lambda redirect setting. This second deployment avoids a circular dependency
between the generated CloudFront URL and the Cognito client:

```sh
export APPLICATION_URL=$(aws cloudformation describe-stacks \
  --stack-name "$LAMBDA_STACK" \
  --region "$AWS_REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='CloudFrontURL'].OutputValue" \
  --output text)

echo "$APPLICATION_URL"

aws cloudformation deploy \
  --template-file infra/parish-attendance-infrastructure.yaml \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    CognitoDomainPrefix=parish-attendance-${ENVIRONMENT} \
    ApplicationCallbackURL="$APPLICATION_URL" \
  --region "$AWS_REGION"

aws cloudformation deploy \
  --template-file infra/parish-attendance-lambda.yaml \
  --stack-name "$LAMBDA_STACK" \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    DeploymentBucket="$DEPLOYMENT_BUCKET" \
    DeploymentKey="$DEPLOYMENT_KEY" \
    ApplicationCallbackURL="$APPLICATION_URL" \
  --region "$AWS_REGION"
```

Open `$APPLICATION_URL`. CloudFront is the public HTTPS entry point. The
Function URL remains its origin, and application API routes continue to require
a verified Cognito access token.

### Deploy without CloudFront

If your AWS account cannot create CloudFront distributions, set
`EnableCloudFront=false`. The Lambda Function URL is still a public HTTPS app
endpoint. First deploy the Lambda stack without an application callback, then
read its Function URL and use its `/app/` path for both the Cognito callback and
the Lambda redirect setting:

```sh
aws cloudformation deploy \
  --template-file infra/parish-attendance-lambda.yaml \
  --stack-name "$LAMBDA_STACK" \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    DeploymentBucket="$DEPLOYMENT_BUCKET" \
    DeploymentKey="$DEPLOYMENT_KEY" \
    EnableCloudFront=false \
  --region "$AWS_REGION"

export FUNCTION_URL=$(aws cloudformation describe-stacks \
  --stack-name "$LAMBDA_STACK" \
  --region "$AWS_REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='FunctionURL'].OutputValue" \
  --output text)
export APPLICATION_URL="${FUNCTION_URL%/}/app/"

aws cloudformation deploy \
  --template-file infra/parish-attendance-infrastructure.yaml \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    CognitoDomainPrefix=parish-attendance-${ENVIRONMENT} \
    ApplicationCallbackURL="$APPLICATION_URL" \
  --region "$AWS_REGION"

aws cloudformation deploy \
  --template-file infra/parish-attendance-lambda.yaml \
  --stack-name "$LAMBDA_STACK" \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
    Environment="$ENVIRONMENT" \
    DeploymentBucket="$DEPLOYMENT_BUCKET" \
    DeploymentKey="$DEPLOYMENT_KEY" \
    ApplicationCallbackURL="$APPLICATION_URL" \
    EnableCloudFront=false \
  --region "$AWS_REGION"
```

When CloudFront access becomes available, redeploy the Lambda stack with
`EnableCloudFront=true`, use its `CloudFrontURL` output as `APPLICATION_URL`,
and repeat the two callback-setting deployments above.

For production, repeat the same process with `ENVIRONMENT=prod`, a distinct
stack name, and a distinct globally unique Cognito domain prefix, for example
`parish-attendance-prod`. The dev and prod stacks have
separate User Pools, DynamoDB tables, Lambdas, Function URLs, and users.
Do not update an existing unsuffixed stack to add `Environment`; deploy a new
`dev` stack instead, because changing a named DynamoDB table can replace it.

The table uses DynamoDB `PAY_PER_REQUEST` billing. There is no provisioned
read/write capacity charge when it is idle.

## Run locally

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
export DYNAMODB_TABLE=$(aws cloudformation describe-stacks \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --region "$AWS_REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='TableName'].OutputValue" \
  --output text)
export COGNITO_USER_POOL_ID=$(aws cloudformation describe-stacks \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --region "$AWS_REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='CognitoUserPoolId'].OutputValue" \
  --output text)
export COGNITO_CLIENT_ID=$(aws cloudformation describe-stacks \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --region "$AWS_REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='CognitoClientId'].OutputValue" \
  --output text)
export COGNITO_DOMAIN=$(aws cloudformation describe-stacks \
  --stack-name "$INFRASTRUCTURE_STACK" \
  --region "$AWS_REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='CognitoDomain'].OutputValue" \
  --output text)
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
password, create billing accounts and parishes, then parish
administrators can invite accounts from **User access**. Invited users receive
their own temporary-password email and can only access their assigned
parish.

## Attendance workflow

1. Create a parish.
2. Create its recurring `mass-templates` once (`weekday` uses `0` for Sunday through `6` for Saturday).
3. Ask `GET /parishes/{parishId}/scheduled-masses?date=2026-08-30` to pre-fill the day’s expected services.
4. `POST /parishes/{parishId}/attendance` saves a count; a second POST for the same mass date/time updates it.
5. Use `GET /parishes/{parishId}/reports/mass-attendance?from=2026-01-01&to=2026-06-30` for per-mass averages.

## Example: recurring mass and attendance

```sh
curl -X POST http://localhost:8080/parishes \
  -H 'Content-Type: application/json' \
  -d '{"name":"St. Mary Parish","timezone":"America/Chicago"}'

curl -X POST http://localhost:8080/parishes/PARISH_ID/mass-templates \
  -H 'Content-Type: application/json' \
  -d '{"name":"Sunday 9:00 AM","weekday":0,"serviceTime":"09:00","isActive":true}'

curl -X POST http://localhost:8080/parishes/PARISH_ID/attendance \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer COGNITO_ACCESS_TOKEN' \
  -d '{"serviceDate":"2026-08-30","serviceTime":"09:00","massName":"Sunday 9:00 AM","massTemplateId":"TEMPLATE_ID","attendanceCount":218}'
```

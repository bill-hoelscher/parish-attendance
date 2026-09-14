package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-lambda-go/lambdaurl"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"log"
	"net/http"
	"os"
	"parishattendance/internal"
	"parishattendance/internal/auth"
	"parishattendance/internal/dynamodb"
	apphttp "parishattendance/internal/http"
)

func main() {
	port := flag.Int("port", 8080, "HTTP listen port")
	flag.Parse()
	server, err := newServer(context.Background())
	if err != nil {
		log.Fatalf("server startup failed: %v", err)
	}
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambdaurl.Start(server)
		return
	}
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), server); err != nil { //nolint:gosec
		log.Fatalf("server stopped: %v", err)
	}
}

func newServer(ctx context.Context) (http.Handler, error) {
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		return nil, fmt.Errorf("DYNAMODB_TABLE is required")
	}
	repository, err := dynamodb.NewRepository(ctx, tableName)
	if err != nil {
		return nil, err
	}
	region := os.Getenv("COGNITO_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	verifier, err := auth.NewVerifier(auth.Config{Region: region, UserPoolID: os.Getenv("COGNITO_USER_POOL_ID"), ClientID: os.Getenv("COGNITO_CLIENT_ID")})
	if err != nil {
		return nil, err
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	inviter := auth.NewInviter(cognitoidentityprovider.NewFromConfig(awsCfg), os.Getenv("COGNITO_USER_POOL_ID"))
	authenticator := auth.NewAuthenticator(cognitoidentityprovider.NewFromConfig(awsCfg), os.Getenv("COGNITO_CLIENT_ID"))
	services := internal.NewServices(repository)
	server := apphttp.NewServer(services, verifier, inviter, authenticator)
	return server, nil
}

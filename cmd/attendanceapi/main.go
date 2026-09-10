package main

import (
	"context"
	"flag"
	"fmt"
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
	if err := run(*port); err != nil {
		log.Fatalf("server startup failed: %v", err)
	}
}

func run(port int) error {
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		return fmt.Errorf("DYNAMODB_TABLE is required")
	}
	repository, err := dynamodb.NewRepository(context.Background(), tableName)
	if err != nil {
		return err
	}
	region := os.Getenv("COGNITO_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	verifier, err := auth.NewVerifier(auth.Config{Region: region, UserPoolID: os.Getenv("COGNITO_USER_POOL_ID"), ClientID: os.Getenv("COGNITO_CLIENT_ID")})
	if err != nil {
		return err
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return err
	}
	inviter := auth.NewInviter(cognitoidentityprovider.NewFromConfig(awsCfg), os.Getenv("COGNITO_USER_POOL_ID"))
	services := internal.NewServices(repository)
	server := apphttp.NewServer(services, verifier, inviter)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), server) //nolint:gosec
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"parishattendance/internal"
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
	services := internal.NewServices(repository)
	server := apphttp.NewServer(services)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), server) //nolint:gosec
}

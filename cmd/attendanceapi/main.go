package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"parishattendance/internal"
	apphttp "parishattendance/internal/http"
	"parishattendance/internal/postgres"
)

func main() {
	port := flag.Int("port", 8080, "HTTP listen port")
	flag.Parse()
	if err := run(*port); err != nil {
		log.Fatalf("server startup failed: %v", err)
	}
}

func run(port int) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://church:church@localhost:5432/parish_attendance?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	if err := db.PingContext(context.Background()); err != nil {
		return err
	}

	repository := postgres.NewRepository(db)
	services := internal.NewServices(repository)
	server := apphttp.NewServer(services)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), server) //nolint:gosec
}

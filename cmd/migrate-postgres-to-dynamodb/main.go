// migrate-postgres-to-dynamodb copies the complete application dataset into
// DynamoDB. It is intentionally separate from the production API binary.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"parishattendance/internal"
	ddb "parishattendance/internal/dynamodb"
	"parishattendance/internal/postgres"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	databaseURL, table := os.Getenv("DATABASE_URL"), os.Getenv("DYNAMODB_TABLE")
	if databaseURL == "" || table == "" {
		return errors.New("DATABASE_URL and DYNAMODB_TABLE are required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	source, target := postgres.NewRepository(db), mustDynamo(ctx, table)
	if err := copyAll(ctx, source, target); err != nil {
		return err
	}
	log.Print("PostgreSQL data copied to DynamoDB successfully")
	return nil
}

func mustDynamo(ctx context.Context, table string) *ddb.Repository {
	r, err := ddb.NewRepository(ctx, table)
	if err != nil {
		log.Fatal(err)
	}
	return r
}

func copyAll(ctx context.Context, source, target internal.Repository) error {
	roles, err := source.ListRoles(ctx)
	if err != nil {
		return err
	}
	for i := range roles {
		if err := target.CreateRole(ctx, &roles[i]); err != nil {
			return err
		}
	}
	accounts, err := source.ListBillingAccounts(ctx)
	if err != nil {
		return err
	}
	for i := range accounts {
		if err := target.CreateBillingAccount(ctx, &accounts[i]); err != nil {
			return err
		}
		if s, err := source.GetSubscription(ctx, accounts[i].ID); err == nil {
			if err := target.UpsertSubscription(ctx, s); err != nil {
				return err
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	organizations, err := source.ListOrganizations(ctx)
	if err != nil {
		return err
	}
	for i := range organizations {
		o := &organizations[i]
		if err := target.CreateOrganization(ctx, o); err != nil {
			return err
		}
		names, err := source.ListMassNames(ctx, o.ID)
		if err != nil {
			return err
		}
		for i := range names {
			if err := target.CreateMassName(ctx, &names[i]); err != nil {
				return err
			}
		}
		templates, err := source.ListMassTemplates(ctx, o.ID)
		if err != nil {
			return err
		}
		for i := range templates {
			if err := target.CreateMassTemplate(ctx, &templates[i]); err != nil {
				return err
			}
		}
		specials, err := source.ListSpecialMasses(ctx, o.ID)
		if err != nil {
			return err
		}
		for i := range specials {
			if err := target.CreateSpecialMass(ctx, &specials[i]); err != nil {
				return err
			}
		}
		attendance, err := source.ListAttendance(ctx, o.ID, "", "")
		if err != nil {
			return err
		}
		for i := range attendance {
			if err := target.UpsertAttendance(ctx, &attendance[i]); err != nil {
				return err
			}
		}
	}
	access, err := source.ListUserAccess(ctx, "")
	if err != nil {
		return err
	}
	for i := range access {
		if err := target.CreateUserAccess(ctx, &access[i]); err != nil {
			return err
		}
	}
	return nil
}

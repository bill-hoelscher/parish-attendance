// Command bootstrap-admin creates the application's built-in roles and invites
// the first system administrator to the Cognito user pool.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"parishattendance/internal"
	"parishattendance/internal/auth"
	"parishattendance/internal/dynamodb"
)

var builtInRoles = []internal.Role{
	{
		ID: "role-system-administrator", Key: "system_administrator", Name: "System Administrator", Scope: "system", IsSystem: true,
	},
	{
		ID: "role-organization-administrator", Key: "organization_administrator", Name: "Organization Administrator", Scope: "organization",
		Permissions: []string{"manage_users", "manage_schedules", "record_attendance", "view_reports"},
	},
	{
		ID: "role-attendance-counter", Key: "attendance_counter", Name: "Attendance Counter", Scope: "organization",
		Permissions: []string{"record_attendance", "view_reports"},
	},
}

func main() {
	email := flag.String("email", "", "email address for the first system administrator")
	flag.Parse()
	if *email == "" {
		log.Fatal("--email is required")
	}
	if err := run(context.Background(), *email); err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
}

func run(ctx context.Context, email string) error {
	table := os.Getenv("DYNAMODB_TABLE")
	poolID := os.Getenv("COGNITO_USER_POOL_ID")
	region := os.Getenv("COGNITO_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if table == "" || poolID == "" || region == "" {
		return fmt.Errorf("DYNAMODB_TABLE, COGNITO_USER_POOL_ID, and COGNITO_REGION (or AWS_REGION) are required")
	}
	repo, err := dynamodb.NewRepository(ctx, table)
	if err != nil {
		return err
	}
	if err := seedRoles(ctx, repo); err != nil {
		return err
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return err
	}
	userID, err := auth.NewInviter(cognitoidentityprovider.NewFromConfig(awsCfg), poolID).Invite(ctx, email)
	if err != nil {
		return err
	}
	role := builtInRoles[0]
	if err := repo.CreateUserAccess(ctx, &internal.UserAccess{UserID: userID, RoleID: role.ID, Role: role.Key, RoleName: role.Name}); err != nil {
		return err
	}
	fmt.Printf("Invited %s as the first System Administrator. Cognito sent a temporary password by email.\n", email)
	return nil
}

func seedRoles(ctx context.Context, repo internal.Repository) error {
	existing, err := repo.ListRoles(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(existing))
	for _, role := range existing {
		known[role.ID] = true
	}
	for _, role := range builtInRoles {
		if known[role.ID] {
			continue
		}
		if err := repo.CreateRole(ctx, &role); err != nil {
			return err
		}
	}
	return nil
}

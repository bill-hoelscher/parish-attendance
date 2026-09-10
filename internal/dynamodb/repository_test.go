package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"parishattendance/internal"
)

func TestOptionalIndexKeysAreOmittedForSparseIndexes(t *testing.T) {
	attributes, err := attributevalue.MarshalMap(item{
		PartitionKey: "ORG#one",
		SortKey:      "ORG",
		EntityType:   "organization",
		ID:           "one",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"IdLookupPartitionKey", "IdLookupSortKey",
		"UserAccessPartitionKey", "UserAccessSortKey",
		"MassReportPartitionKey", "MassReportSortKey",
	} {
		if _, exists := attributes[key]; exists {
			t.Fatalf("%s must be omitted when empty", key)
		}
	}
}

func TestOrganizationCountIsAlwaysDerived(t *testing.T) {
	billingAccountID := "billing-one"
	accounts := []internal.BillingAccount{{ID: billingAccountID, OrganizationCount: 2}}
	organizations := []internal.Organization{
		{BillingAccountID: &billingAccountID},
		{BillingAccountID: &billingAccountID},
	}

	setOrganizationCounts(accounts, organizations)
	if accounts[0].OrganizationCount != 2 {
		t.Fatalf("organization count = %d, want 2", accounts[0].OrganizationCount)
	}
}

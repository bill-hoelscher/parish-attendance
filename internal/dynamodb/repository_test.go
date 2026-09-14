package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"parishattendance/internal"
)

func TestOptionalIndexKeysAreOmittedForSparseIndexes(t *testing.T) {
	attributes, err := attributevalue.MarshalMap(item{
		PartitionKey: "PARISH#one",
		SortKey:      "PARISH",
		EntityType:   "parish",
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

func TestParishCountIsAlwaysDerived(t *testing.T) {
	billingAccountID := "billing-one"
	accounts := []internal.BillingAccount{{ID: billingAccountID, ParishCount: 2}}
	parishes := []internal.Parish{
		{BillingAccountID: &billingAccountID},
		{BillingAccountID: &billingAccountID},
	}

	setParishCounts(accounts, parishes)
	if accounts[0].ParishCount != 2 {
		t.Fatalf("parish count = %d, want 2", accounts[0].ParishCount)
	}
}

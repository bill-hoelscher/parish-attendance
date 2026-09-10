package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
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

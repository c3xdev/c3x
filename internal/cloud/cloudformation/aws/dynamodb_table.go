package aws

import (
	"github.com/awslabs/goformation/v7/cloudformation/dynamodb"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func GetDynamoDBTableRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "AWS::DynamoDB::Table",
		Notes: []string{
			"DAX is not yet supported.",
		},
		RFunc: NewDynamoDBTable,
	}
}

func NewDynamoDBTable(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	cfr, ok := d.CFResource.(*dynamodb.Table)
	if !ok {
		logging.Logger.Debug().Msgf("Skipping resource %s as it did not have the expected type (got %T)", d.Address, d.CFResource)
		return nil
	}

	region := d.Region
	billingMode := cfr.BillingMode
	var readCapacity int64
	if cfr.ProvisionedThroughput != nil {
		readCapacity = int64(cfr.ProvisionedThroughput.ReadCapacityUnits)
	}
	var writeCapacity int64
	if cfr.ProvisionedThroughput != nil {
		writeCapacity = int64(cfr.ProvisionedThroughput.WriteCapacityUnits)
	}

	a := &aws.DynamoDBTable{
		Address:        d.Address,
		Region:         region,
		BillingMode:    *billingMode,
		WriteCapacity:  &writeCapacity,
		ReadCapacity:   &readCapacity,
		ReplicaRegions: []string{}, // Global Tables are defined using AWS::DynamoDB::GlobalTable
	}
	a.PopulateUsage(u)

	resource := a.BuildResource()

	return resource
}

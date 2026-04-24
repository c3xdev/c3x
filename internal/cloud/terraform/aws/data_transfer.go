package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDataTransferRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "aws_data_transfer",
		RFunc: newDataTransfer,
	}
}

func newDataTransfer(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := strings.ToLower(u.Get("region").String())

	r := &aws.DataTransfer{
		Address: d.Address,
		Region:  region,
	}
	r.PopulateUsage(u)

	return r.BuildResource()
}

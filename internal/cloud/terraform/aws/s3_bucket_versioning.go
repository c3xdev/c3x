package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getS3BucketVersioningRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_s3_bucket_versioning",
		RFunc:               NewS3BucketVersioning,
		ReferenceAttributes: []string{"bucket"},
	}
}

func NewS3BucketVersioning(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	return &engine.Estimate{
		Name:         d.Address,
		ResourceType: d.Type,
		Tags:         d.Tags,
		DefaultTags:  d.DefaultTags,
		IsSkipped:    true,
		NoPrice:      true,
		SkipMessage:  "Free resource.",
	}
}

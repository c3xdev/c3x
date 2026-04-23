package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getS3BucketIntelligentTieringConfigurationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_s3_bucket_intelligent_tiering_configuration",
		RFunc:               NewS3BucketIntelligentTieringConfiguration,
		ReferenceAttributes: []string{"bucket"},
	}
}

func NewS3BucketIntelligentTieringConfiguration(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
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

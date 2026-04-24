package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNewEKSFargateProfileItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_eks_fargate_profile",
		CoreRFunc: NewEKSFargateProfile,
	}
}
func NewEKSFargateProfile(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EKSFargateProfile{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}

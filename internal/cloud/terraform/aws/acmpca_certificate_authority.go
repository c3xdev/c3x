package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getACMPCACertificateAuthorityRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_acmpca_certificate_authority",
		CoreRFunc: NewACMPCACertificateAuthority,
	}
}
func NewACMPCACertificateAuthority(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.ACMPCACertificateAuthority{
		Address:   d.Address,
		Region:    d.Get("region").String(),
		UsageMode: d.Get("usage_mode").String(),
	}
	return r
}

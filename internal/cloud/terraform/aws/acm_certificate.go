package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getACMCertificate() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_acm_certificate",
		CoreRFunc: NewACMCertificate,
	}
}
func NewACMCertificate(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.ACMCertificate{
		Address:                 d.Address,
		Region:                  d.Get("region").String(),
		CertificateAuthorityARN: d.Get("certificate_authority_arn").String(),
	}
	return r
}

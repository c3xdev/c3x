package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ACMCertificate struct {
	Address                 string
	Region                  string
	CertificateAuthorityARN string
}

func (r *ACMCertificate) CoreType() string {
	return "ACMCertificate"
}

func (r *ACMCertificate) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *ACMCertificate) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ACMCertificate) BuildResource() *engine.Estimate {
	if r.CertificateAuthorityARN == "" {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	certAuthority := &ACMPCACertificateAuthority{
		Region: r.Region,
	}

	certCostComponent := certAuthority.certificateCostComponent("Certificate", "0", decimalPtr(decimal.NewFromInt(1)))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{certCostComponent},
		UsageSchema:    r.UsageSchema(),
	}
}

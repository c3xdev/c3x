package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/usage"
)

// RecoveryServicesVault struct represents a storage vault that can azure users can back up
// various vms into.
//
// See the ProtectedVM struct for more information about backup services are charged.
//
// Resource information: https://learn.microsoft.com/en-us/azure/backup/backup-overview
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/backup/
type RecoveryServicesVault struct {
	Address      string
	Region       string
	ProtectedVMs []*BackupProtectedVM
}

func (r *RecoveryServicesVault) CoreType() string {
	return "RecoveryServicesVault"
}

// UsageSchema dynamically constructs a list of UsageItems based on the ProtectedVM sub catalog.
func (r *RecoveryServicesVault) UsageSchema() []*engine.ConsumptionField {
	items := make([]*engine.ConsumptionField, len(r.ProtectedVMs))
	for i, pm := range r.ProtectedVMs {
		items[i] = &engine.ConsumptionField{
			Key:          pm.Address,
			DefaultValue: &usage.ResourceUsage{Name: pm.Address, Items: pm.UsageSchema()},
			ValueType:    engine.SubResourceUsage,
		}
	}

	return items
}

// PopulateUsage parses the u engine.ConsumptionProfile into the RecoveryServicesVault's sub catalog.
//
// RecoveryServicesVault does not have any actual usage associated with itself and instead relies on
// users specifying usage for child ProtectedVM catalog.
func (r *RecoveryServicesVault) PopulateUsage(u *engine.ConsumptionProfile) {
	if u == nil {
		return
	}

	// build a new UsageMap so that we get the wildcard support.
	data := map[string]*engine.ConsumptionProfile{}
	for s, result := range u.Attributes {
		data[s] = engine.NewUsageData(s, result.Map())
	}
	um := engine.NewUsageMap(data)

	for _, pm := range r.ProtectedVMs {
		pm.PopulateUsage(um.Get(pm.Address))
	}

	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid RecoveryServicesVault struct.
//
// RecoveryServicesVault does not have any top level costs associated with it and instead returns a
// list of sub resources where the costs are encapsulated.
func (r *RecoveryServicesVault) BuildResource() *engine.Estimate {
	if len(r.ProtectedVMs) == 0 {
		logging.Logger.Warn().Msgf("recovery services vault %s has been marked as free as no associated protected VMs were found", r.Address)
		return &engine.Estimate{Name: r.Address, NoPrice: true}
	}

	subResources := make([]*engine.Estimate, len(r.ProtectedVMs))
	for i, pvm := range r.ProtectedVMs {
		subResources[i] = pvm.BuildResource()
	}

	return &engine.Estimate{
		Name:         r.Address,
		UsageSchema:  r.UsageSchema(),
		SubResources: subResources,
	}
}

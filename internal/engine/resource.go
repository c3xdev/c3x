package engine

import (
	"sort"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/logging"
)

var (
	HourToMonthUnitMultiplier = decimal.NewFromInt(730)
	MonthToHourUnitMultiplier = decimal.NewFromInt(1).Div(HourToMonthUnitMultiplier)
	DaysInMonth               = HourToMonthUnitMultiplier.DivRound(decimal.NewFromInt(24), 24)
	DayToMonthUnitMultiplier  = DaysInMonth.DivRound(HourToMonthUnitMultiplier, 24)
)

type LegacyBuilder func(*ResourceSpec, *ConsumptionProfile) *Estimate

type Estimate struct {
	Name                                    string
	CostComponents                          []*LineItem
	ActualCosts                             []*ObservedCost
	SubResources                            []*Estimate
	HourlyCost                              *decimal.Decimal
	MonthlyCost                             *decimal.Decimal
	MonthlyUsageCost                        *decimal.Decimal
	IsSkipped                               bool
	NoPrice                                 bool
	SkipMessage                             string
	ResourceType                            string
	Tags                                    *map[string]string
	DefaultTags                             *map[string]string
	TagPropagation                          *TagPropagation
	ProviderSupportsDefaultTags             bool
	ProviderLink                            string
	UsageSchema                             []*ConsumptionField
	EstimateUsage                           UsageEstimator
	EstimationSummary                       map[string]bool
	Metadata                                map[string]gjson.Result
	MissingVarsCausingUnknownTagKeys        []string
	MissingVarsCausingUnknownDefaultTagKeys []string

	// parent is the parent resource of this resource, this is only
	// applicable for sub resources. See FlattenedSubResources for more info
	// on how this is built and used.
	parent *Estimate
}

func ComputeCosts(project *Workspace) {
	for _, r := range project.AllResources() {
		r.CalculateCosts()
	}
}

// BaseResourceType returns the base resource type of the resource. This is the
// resource type of the top level resource in the hierarchy. For example, if the
// resource is a subresource of a `aws_instance` resource (e.g.
// ebs_block_device), the base resource type will be `aws_instance`.
func (r *Estimate) BaseResourceType() string {
	if r.parent == nil {
		return r.ResourceType
	}

	return r.parent.BaseResourceType()
}

// BaseResourceName returns the base resource name of the resource. This is the
// resource name of the top level resource in the hierarchy.
func (r *Estimate) BaseResourceName() string {
	if r.parent == nil {
		return r.Name
	}

	return r.parent.BaseResourceName()
}

func (r *Estimate) CalculateCosts() {
	h := decimal.Zero
	m := decimal.Zero
	var monthlyUsageCost *decimal.Decimal
	hasCost := false

	for _, c := range r.CostComponents {
		c.CalculateCosts()
		if c.HourlyCost != nil || c.MonthlyCost != nil {
			hasCost = true
		}
		if c.HourlyCost != nil {
			h = h.Add(*c.HourlyCost)
		}
		if c.MonthlyCost != nil {
			m = m.Add(*c.MonthlyCost)
			if c.UsageBased {
				if monthlyUsageCost == nil {
					monthlyUsageCost = &decimal.Zero
				}
				monthlyUsageCost = decimalPtr(monthlyUsageCost.Add(*c.MonthlyCost))
			}
		}
	}

	for _, s := range r.SubResources {
		s.CalculateCosts()
		if s.HourlyCost != nil || s.MonthlyCost != nil || s.MonthlyUsageCost != nil {
			hasCost = true
		}
		if s.HourlyCost != nil {
			h = h.Add(*s.HourlyCost)
		}
		if s.MonthlyCost != nil {
			m = m.Add(*s.MonthlyCost)
		}
		if s.MonthlyUsageCost != nil {
			if monthlyUsageCost == nil {
				monthlyUsageCost = &decimal.Zero
			}
			monthlyUsageCost = decimalPtr(monthlyUsageCost.Add(*s.MonthlyUsageCost))
		}
	}

	if hasCost {
		r.HourlyCost = &h
		r.MonthlyCost = &m
		r.MonthlyUsageCost = monthlyUsageCost
	}
	if r.NoPrice {
		logging.Logger.Debug().Msgf("Skipping free resource %s", r.Name)
	}
}

// FlattenedSubResources returns a list of resources from the given resources,
// flattening all sub resources recursively. It also sets the parent resource for
// each sub resource so that the full resource can be reconstructed.
func (r *Estimate) FlattenedSubResources() []*Estimate {
	resources := make([]*Estimate, 0, len(r.SubResources))

	for _, s := range r.SubResources {
		s.parent = r
		resources = append(resources, s)

		if len(s.SubResources) > 0 {
			resources = append(resources, s.FlattenedSubResources()...)
		}
	}

	return resources
}

func (r *Estimate) RemoveCostComponent(costComponent *LineItem) {
	n := make([]*LineItem, 0, len(r.CostComponents)-1)
	for _, c := range r.CostComponents {
		if c != costComponent {
			n = append(n, c)
		}
	}
	r.CostComponents = n
}

func SortEstimates(project *Workspace) {
	resources := project.AllResources()
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})
}

func ScaleQuantities(resource *Estimate, multiplier decimal.Decimal) {
	for _, costComponent := range resource.CostComponents {
		if costComponent.HourlyQuantity != nil {
			costComponent.HourlyQuantity = decimalPtr(costComponent.HourlyQuantity.Mul(multiplier))
		}
		if costComponent.MonthlyQuantity != nil {
			costComponent.MonthlyQuantity = decimalPtr(costComponent.MonthlyQuantity.Mul(multiplier))
		}
	}

	for _, subResource := range resource.SubResources {
		ScaleQuantities(subResource, multiplier)
	}
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

package aws

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type CodeBuildProject struct {
	Address          string
	Region           string
	ComputeType      string
	EnvironmentType  string
	MonthlyBuildMins *int64 `c3x_usage:"monthly_build_mins"`
}

func (r *CodeBuildProject) CoreType() string {
	return "CodeBuildProject"
}

func (r *CodeBuildProject) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_build_mins", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *CodeBuildProject) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CodeBuildProject) BuildResource() *engine.Estimate {
	var monthlyBuildMinutes *decimal.Decimal
	if r.MonthlyBuildMins != nil {
		monthlyBuildMinutes = decimalPtr(decimal.NewFromInt(*r.MonthlyBuildMins))
	}

	computeType := r.mapComputeType()
	return &engine.Estimate{
		Name:      r.Address,
		IsSkipped: computeType == "",
		CostComponents: []*engine.LineItem{
			{
				Name:            r.nameLabel(),
				Unit:            "minutes",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: monthlyBuildMinutes,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("CodeBuild"),
					ProductFamily: strPtr("Compute"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/Build-Min:%s:%s/", r.mapEnvironmentType(), computeType))},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *CodeBuildProject) nameLabel() string {
	switch r.EnvironmentType {
	case "WINDOWS_SERVER_2019_CONTAINER":
		return r.osWithComputeTypeLabel("Windows")
	case "ARM_CONTAINER":
		return "Linux (arm1.large)"
	case "LINUX_GPU_CONTAINER":
		return "Linux (gpu1.large)"
	default:
		return r.osWithComputeTypeLabel("Linux")
	}
}

func (r *CodeBuildProject) osWithComputeTypeLabel(os string) string {
	pieces := strings.SplitAfter(r.ComputeType, "BUILD_")
	if len(pieces) < 2 {
		return os
	}

	computeType := strings.Replace(strings.ToLower(pieces[1]), "_", ".", 1)
	return fmt.Sprintf("%s (%s)", os, computeType)
}

func (r *CodeBuildProject) mapEnvironmentType() string {
	switch r.EnvironmentType {
	case "LINUX_CONTAINER":
		return "Linux"
	case "LINUX_GPU_CONTAINER":
		return "LinuxGPU"
	case "ARM_CONTAINER":
		return "ARM"
	case "WINDOWS_SERVER_2019_CONTAINER":
		return "Windows"
	default:
		return ""
	}
}

func (r *CodeBuildProject) mapComputeType() string {
	switch r.ComputeType {
	case "BUILD_GENERAL1_SMALL":
		return "g1.small"
	case "BUILD_GENERAL1_MEDIUM":
		return "g1.medium"
	case "BUILD_GENERAL1_LARGE":
		return "g1.large"
	case "BUILD_GENERAL1_2XLARGE":
		return "g1.2xlarge"
	default:
		return ""
	}
}

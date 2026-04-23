package aws

import (
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type LaunchConfiguration struct {
	// "required" args that can't really be missing.
	Address          string
	Region           string
	Tenancy          string
	PurchaseOption   string
	AMI              string
	InstanceType     string
	EBSOptimized     bool
	EnableMonitoring bool
	CPUCredits       string

	// "optional" args, that may be empty depending on the resource config
	ElasticInferenceAcceleratorType *string
	RootBlockDevice                 *EBSVolume
	EBSBlockDevices                 []*EBSVolume

	// "usage" args
	// These are populated from the Autoscaling Group resource
	InstanceCount                 *int64  `c3x_usage:"instances"`
	OperatingSystem               *string `c3x_usage:"operating_system"`
	ReservedInstanceType          *string `c3x_usage:"reserved_instance_type"`
	ReservedInstanceTerm          *string `c3x_usage:"reserved_instance_term"`
	ReservedInstancePaymentOption *string `c3x_usage:"reserved_instance_payment_option"`
	MonthlyCPUCreditHours         *int64  `c3x_usage:"monthly_cpu_credit_hrs"`
	VCPUCount                     *int64  `c3x_usage:"vcpu_count"`
}

var LaunchConfigurationUsageSchema = InstanceUsageSchema

func (r *LaunchConfiguration) CoreType() string {
	return "LaunchConfiguration"
}

func (r *LaunchConfiguration) UsageSchema() []*engine.ConsumptionField {
	return LaunchConfigurationUsageSchema
}

func (r *LaunchConfiguration) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *LaunchConfiguration) BuildResource() *engine.Estimate {
	if strings.ToLower(r.Tenancy) == "host" {
		logging.Logger.Warn().Msgf("Skipping resource %s. C3X currently does not support host tenancy for AWS Launch Configurations", r.Address)
		return nil
	} else if strings.ToLower(r.Tenancy) == "dedicated" {
		r.Tenancy = "Dedicated"
	} else {
		r.Tenancy = "Shared"
	}

	instance := &Instance{
		Region:                          r.Region,
		Tenancy:                         r.Tenancy,
		PurchaseOption:                  r.PurchaseOption,
		AMI:                             r.AMI,
		InstanceType:                    r.InstanceType,
		EBSOptimized:                    r.EBSOptimized,
		EnableMonitoring:                r.EnableMonitoring,
		CPUCredits:                      r.CPUCredits,
		ElasticInferenceAcceleratorType: r.ElasticInferenceAcceleratorType,
		OperatingSystem:                 r.OperatingSystem,
		RootBlockDevice:                 r.RootBlockDevice,
		EBSBlockDevices:                 r.EBSBlockDevices,
		ReservedInstanceType:            r.ReservedInstanceType,
		ReservedInstanceTerm:            r.ReservedInstanceTerm,
		ReservedInstancePaymentOption:   r.ReservedInstancePaymentOption,
		MonthlyCPUCreditHours:           r.MonthlyCPUCreditHours,
		VCPUCount:                       r.VCPUCount,
	}
	instanceResource := instance.BuildResource()

	res := &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: instanceResource.CostComponents,
		SubResources:   instanceResource.SubResources,
		EstimateUsage:  instanceResource.EstimateUsage,
	}

	qty := int64(1)
	if r.InstanceCount != nil {
		qty = *r.InstanceCount
	}
	engine.ScaleQuantities(res, decimal.NewFromInt(qty))

	return res
}

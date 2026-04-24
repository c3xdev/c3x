package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

// ElasticBeanstalkEnvironment struct represents AWS Elastic Beanstalk environments.
//
// Resource information: https://aws.amazon.com/elasticbeanstalk/
// Pricing information: https://aws.amazon.com/elasticbeanstalk/pricing/
type ElasticBeanstalkEnvironment struct {
	Address string
	Region  string
	Name    string

	LoadBalancerType string

	RootBlockDevice     *EBSVolume
	CloudwatchLogGroup  *CloudwatchLogGroup
	LoadBalancer        *LB
	ElasticLoadBalancer *ELB
	DBInstance          *DBInstance
	LaunchConfiguration *LaunchConfiguration
}

func (r *ElasticBeanstalkEnvironment) CoreType() string {
	return "ElasticBeanstalkEnvironment"
}

// UsageSchema defines a list which represents the usage schema of ElasticBeanstalkEnvironment.
// Usage costs for Elastic Beanstalk come from sub resources as it is a wrapper for other AWS services.
func (r *ElasticBeanstalkEnvironment) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{
			Key:          "cloudwatch",
			DefaultValue: &usage.ResourceUsage{Name: "cloudwatch", Items: CloudwatchLogGroupUsageSchema},
			ValueType:    engine.SubResourceUsage,
		},
		{
			Key:          "lb",
			DefaultValue: &usage.ResourceUsage{Name: "lb", Items: LBUsageSchema},
			ValueType:    engine.SubResourceUsage,
		},
		{
			Key:          "elb",
			DefaultValue: &usage.ResourceUsage{Name: "elb", Items: ELBUsageSchema},
			ValueType:    engine.SubResourceUsage,
		},
		{
			Key:          "db",
			DefaultValue: &usage.ResourceUsage{Name: "db", Items: DBInstanceUsageSchema},
			ValueType:    engine.SubResourceUsage,
		},
		{
			Key:          "ec2",
			DefaultValue: &usage.ResourceUsage{Name: "ec2", Items: LaunchConfigurationUsageSchema},
			ValueType:    engine.SubResourceUsage,
		},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ElasticBeanstalkEnvironment.
// It uses the `c3x_usage` struct tags to populate data into the ElasticBeanstalkEnvironment.
func (r *ElasticBeanstalkEnvironment) PopulateUsage(u *engine.ConsumptionProfile) {
	if u == nil {
		return
	}

	if r.ElasticLoadBalancer != nil {
		catalog.PopulateArgsWithUsage(r.ElasticLoadBalancer, engine.NewUsageData("elb", u.Get("elb").Map()))
	}

	if r.LoadBalancer != nil {
		catalog.PopulateArgsWithUsage(r.LoadBalancer, engine.NewUsageData("lb", u.Get("lb").Map()))
	}

	if r.DBInstance != nil {
		catalog.PopulateArgsWithUsage(r.DBInstance, engine.NewUsageData("db", u.Get("db").Map()))
	}

	if r.CloudwatchLogGroup != nil {
		catalog.PopulateArgsWithUsage(r.CloudwatchLogGroup, engine.NewUsageData("cloudwatch", u.Get("cloudwatch").Map()))
	}

	catalog.PopulateArgsWithUsage(r.LaunchConfiguration, engine.NewUsageData("ec2", u.Get("ec2").Map()))
}

// BuildResource builds a engine.Estimate from a valid ElasticBeanstalkEnvironment struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ElasticBeanstalkEnvironment) BuildResource() *engine.Estimate {
	a := &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
	}

	a.SubResources = append(a.SubResources, r.LaunchConfiguration.BuildResource())

	if r.DBInstance != nil {
		a.SubResources = append(a.SubResources, r.DBInstance.BuildResource())
	}

	if r.CloudwatchLogGroup != nil {
		a.SubResources = append(a.SubResources, r.CloudwatchLogGroup.BuildResource())
	}

	if r.LoadBalancerType == "classic" {
		a.SubResources = append(a.SubResources, r.ElasticLoadBalancer.BuildResource())
	} else {
		a.SubResources = append(a.SubResources, r.LoadBalancer.BuildResource())
	}

	return a

}

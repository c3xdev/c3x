package recommend

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/engine"
)

// AWS instance generation upgrades: older → newer generation with same or better specs.
var awsInstanceGenerationUpgrades = map[string]string{
	"m4": "m7i", "m5": "m7i", "m5a": "m7a", "m5n": "m7i", "m6i": "m7i", "m6a": "m7a",
	"c4": "c7i", "c5": "c7i", "c5a": "c7a", "c5n": "c7i", "c6i": "c7i", "c6a": "c7a",
	"r4": "r7i", "r5": "r7i", "r5a": "r7a", "r5n": "r7i", "r6i": "r7i", "r6a": "r7a",
	"t2": "t3", "t3": "t3a",
	"i3": "i4i",
	"d2": "d3",
}

func analyzeAWS(resource *engine.Estimate) []Recommendation {
	var recs []Recommendation

	switch resource.ResourceType {
	case "aws_instance", "aws_launch_template", "aws_launch_configuration", "aws_autoscaling_group":
		recs = append(recs, checkAWSInstanceGeneration(resource)...)
	case "aws_ebs_volume":
		recs = append(recs, checkAWSEBSType(resource)...)
	case "aws_db_instance":
		recs = append(recs, checkAWSRDSGeneration(resource)...)
	case "aws_nat_gateway":
		recs = append(recs, checkAWSNATGateway(resource)...)
	}

	return recs
}

func checkAWSInstanceGeneration(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if !strings.Contains(cc.Name, "Instance usage") {
			continue
		}

		// Extract instance type from name like "Instance usage (Linux/UNIX, on-demand, m5.xlarge)"
		parts := strings.Split(cc.Name, ", ")
		if len(parts) < 3 {
			continue
		}
		instanceType := strings.TrimSuffix(parts[len(parts)-1], ")")

		// Split into family and size (e.g., "m5.xlarge" → "m5", "xlarge")
		typeParts := strings.SplitN(instanceType, ".", 2)
		if len(typeParts) != 2 {
			continue
		}
		family := typeParts[0]
		size := typeParts[1]

		if newFamily, ok := awsInstanceGenerationUpgrades[family]; ok {
			newType := newFamily + "." + size
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "instance-generation",
				Title:        fmt.Sprintf("Upgrade to newer instance generation (%s → %s)", instanceType, newType),
				Description:  fmt.Sprintf("The %s family has better price-performance than %s. Consider switching to %s.", newFamily, family, newType),
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}

func checkAWSEBSType(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if strings.Contains(cc.Name, "gp2") {
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "storage-type",
				Title:        "Switch EBS volume from gp2 to gp3",
				Description:  "gp3 volumes are up to 20% cheaper than gp2 with better baseline performance (3,000 IOPS vs 100/GB for gp2).",
				CurrentCost:  resource.MonthlyCost,
			}}
		}
		if strings.Contains(cc.Name, "io1") {
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "storage-type",
				Title:        "Consider switching EBS volume from io1 to io2",
				Description:  "io2 volumes offer higher durability (99.999%) at the same price as io1, with up to 64,000 IOPS.",
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}

func checkAWSRDSGeneration(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if !strings.Contains(cc.Name, "Database instance") {
			continue
		}

		parts := strings.Split(cc.Name, ", ")
		if len(parts) < 2 {
			continue
		}
		instanceClass := strings.TrimSuffix(parts[len(parts)-1], ")")

		typeParts := strings.SplitN(instanceClass, ".", 3)
		if len(typeParts) != 3 {
			continue
		}
		family := typeParts[1] // e.g., "m5" from "db.m5.large"

		if newFamily, ok := awsInstanceGenerationUpgrades[family]; ok {
			newClass := "db." + newFamily + "." + typeParts[2]
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "instance-generation",
				Title:        fmt.Sprintf("Upgrade RDS instance class (%s → %s)", instanceClass, newClass),
				Description:  fmt.Sprintf("The db.%s family offers better price-performance than db.%s.", newFamily, family),
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}

func checkAWSNATGateway(resource *engine.Estimate) []Recommendation {
	return []Recommendation{{
		ResourceName: resource.Name,
		ResourceType: resource.ResourceType,
		Category:     "architecture",
		Title:        "Consider VPC endpoints to reduce NAT Gateway costs",
		Description:  "NAT Gateway charges $0.045/GB for data processing. For AWS services like S3, DynamoDB, and SQS, VPC Gateway/Interface endpoints can eliminate these charges.",
		CurrentCost:  resource.MonthlyCost,
	}}
}

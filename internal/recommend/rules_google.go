package recommend

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/engine"
)

var gcpMachineGenerationUpgrades = map[string]string{
	"n1-standard": "n2-standard",
	"n1-highmem":  "n2-highmem",
	"n1-highcpu":  "n2-highcpu",
	"e2-standard": "n2-standard",
}

func analyzeGoogle(resource *engine.Estimate) []Recommendation {
	var recs []Recommendation

	switch resource.ResourceType {
	case "google_compute_instance":
		recs = append(recs, checkGCPInstanceGeneration(resource)...)
	case "google_compute_disk":
		recs = append(recs, checkGCPDiskType(resource)...)
	}

	return recs
}

func checkGCPInstanceGeneration(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if !strings.Contains(cc.Name, "Instance usage") {
			continue
		}

		parts := strings.Split(cc.Name, ", ")
		if len(parts) < 2 {
			continue
		}
		machineType := strings.TrimSuffix(parts[len(parts)-1], ")")

		// Split "n1-standard-4" into family "n1-standard" and size "4"
		lastDash := strings.LastIndex(machineType, "-")
		if lastDash < 0 {
			continue
		}
		family := machineType[:lastDash]
		size := machineType[lastDash+1:]

		if newFamily, ok := gcpMachineGenerationUpgrades[family]; ok {
			newType := newFamily + "-" + size
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "instance-generation",
				Title:        fmt.Sprintf("Upgrade to newer machine type (%s → %s)", machineType, newType),
				Description:  fmt.Sprintf("The %s series offers better price-performance than %s.", newFamily, family),
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}

func checkGCPDiskType(resource *engine.Estimate) []Recommendation {
	for _, cc := range resource.CostComponents {
		if strings.Contains(cc.Name, "Standard provisioned") {
			return []Recommendation{{
				ResourceName: resource.Name,
				ResourceType: resource.ResourceType,
				Category:     "storage-type",
				Title:        "Consider switching from pd-standard to pd-balanced",
				Description:  "pd-balanced offers significantly better IOPS and throughput at a modest price increase over pd-standard.",
				CurrentCost:  resource.MonthlyCost,
			}}
		}
	}
	return nil
}

package usage

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/settings"
)

type ContextEnv struct{}

type SyncResult struct {
	ResourceCount    int
	EstimationCount  int
	EstimationErrors map[string]error
}

type ReplaceResourceUsagesOpts struct {
	OverrideValueType bool
}

func (s *SyncResult) Merge(other *SyncResult) {
	s.ResourceCount += other.ResourceCount
	s.EstimationCount += other.EstimationCount
	for k, v := range other.EstimationErrors {
		s.EstimationErrors[k] = v
	}
}

func (s *SyncResult) ProjectContext() map[string]interface{} {
	r := make(map[string]interface{})

	r["usageSyncs"] = s.ResourceCount
	r["usageEstimates"] = s.EstimationCount
	r["usageEstimateErrors"] = len(s.EstimationErrors)

	var remediable, remAttempts, remErrors int
	for _, err := range s.EstimationErrors {
		if _, ok := err.(engine.Remediater); ok {
			remediable++
		}
	}

	r["remediationOpportunities"] = remediable
	r["remediationAttempts"] = remAttempts
	r["remediationErrors"] = remErrors

	return r
}

func SyncUsageData(projectCtx *settings.ProjectSession, usageFile *UsageFile, projects []*engine.Workspace) (*SyncResult, error) {
	referenceFile, err := LoadReferenceFile()
	if err != nil {
		return nil, err
	}
	referenceFile.SetDefaultValues()

	// Collect resources from all projects. When multiple projects exist,
	// resource addresses are already unique per project in the usage file.
	resources := make([]*engine.Estimate, 0)
	for _, project := range projects {
		resources = append(resources, project.Resources...)
	}

	syncResult := syncResourceUsages(projectCtx, usageFile, resources, referenceFile)

	return syncResult, nil
}

type syncResourceResult struct {
	ru *ResourceUsage
	sr *SyncResult
}

func syncResourceUsages(projectCtx *settings.ProjectSession, usageFile *UsageFile, resources []*engine.Estimate, referenceFile *ReferenceFile) *SyncResult {
	syncResult := &SyncResult{
		EstimationErrors: make(map[string]error),
	}

	existingResourceUsagesMap := resourceUsagesMap(usageFile.ResourceUsages)
	existingResourceTypeUsagesMap := resourceUsagesMap(usageFile.ResourceTypeUsages)

	resourceUsages := make([]*ResourceUsage, 0, len(resources))

	// Track the existing order so we can keep these at the top
	existingTypeOrder := make([]string, 0, len(usageFile.ResourceTypeUsages))
	for _, resourceUsage := range usageFile.ResourceTypeUsages {
		existingTypeOrder = append(existingTypeOrder, resourceUsage.Name)
	}

	existingOrder := make([]string, 0, len(usageFile.ResourceUsages))
	for _, resourceUsage := range usageFile.ResourceUsages {
		existingOrder = append(existingOrder, resourceUsage.Name)
	}

	wildCardResources := make(map[string]bool)
	for _, resource := range resources {
		ru := syncWildCardResource(wildCardResources, resource, referenceFile, existingResourceUsagesMap)
		if ru != nil {
			resourceUsages = append(resourceUsages, ru)
		}
	}

	resourceTypeUsages := make([]*ResourceUsage, 0, len(resources))

	resourceTypeSet := make(map[string]*engine.Estimate)

	for _, resource := range resources {
		resourceTypeSet[resource.ResourceType] = resource
	}

	for _, resource := range resourceTypeSet {
		ru := syncResourceType(projectCtx, resource, referenceFile, existingResourceTypeUsagesMap)
		if ru != nil {
			resourceTypeUsages = append(resourceTypeUsages, ru)
		}
	}

	sortResourceUsages(resourceTypeUsages, existingTypeOrder)

	usageFile.ResourceTypeUsages = resourceTypeUsages

	numWorkers := 4
	numCPU := runtime.NumCPU()
	if numCPU*4 > numWorkers {
		numWorkers = numCPU * 4
	}
	if numWorkers > 16 {
		numWorkers = 16
	}
	numJobs := len(resources)
	jobs := make(chan *engine.Estimate, numJobs)
	results := make(chan syncResourceResult, numJobs)

	// Fire up the workers
	for i := 0; i < numWorkers; i++ {
		go func(jobs <-chan *engine.Estimate, results chan<- syncResourceResult) {
			for r := range jobs {
				ru, sr := syncResource(projectCtx, r, referenceFile, existingResourceUsagesMap)
				results <- syncResourceResult{ru, sr}
			}
		}(jobs, results)
	}

	// Feed the workers the jobs of getting prices
	for _, r := range resources {
		jobs <- r
	}
	close(jobs)

	// Get the result of the jobs
	for i := 0; i < numJobs; i++ {
		result := <-results
		resourceUsages = append(resourceUsages, result.ru)
		syncResult.Merge(result.sr)
	}

	sortResourceUsages(resourceUsages, existingOrder)

	usageFile.ResourceUsages = resourceUsages

	return syncResult
}

func syncWildCardResource(wildCardResources map[string]bool, resource *engine.Estimate, referenceFile *ReferenceFile, existingResourceUsagesMap map[string]*ResourceUsage) *ResourceUsage {
	var resourceUsage *ResourceUsage

	// Add existing wildcard resource usages if not already added
	if strings.HasSuffix(resource.Name, "]") {
		lastIndexOfOpenBracket := strings.LastIndex(resource.Name, "[")
		wildCardName := fmt.Sprintf("%s[*]", resource.Name[:lastIndexOfOpenBracket])

		if !wildCardResources[wildCardName] {
			resourceUsage = &ResourceUsage{
				Name: wildCardName,
			}

			wildCardResources[wildCardName] = true
			if existingResourceUsage, ok := existingResourceUsagesMap[wildCardName]; ok {
				// Merge the usage schema from the reference usage file
				refResourceUsage := referenceFile.FindMatchingResourceUsage(resource.Name)
				if refResourceUsage != nil {
					replaceResourceUsages(resourceUsage, refResourceUsage, ReplaceResourceUsagesOpts{})
				}

				// Merge the usage schema from the resource struct
				// We want to override the value type from the usage schema since we can't always tell from the YAML
				// what the value type should be, e.g. user might add an int value for a float attribute.
				replaceResourceUsages(resourceUsage, &ResourceUsage{
					Name:  wildCardName,
					Items: resource.UsageSchema,
				}, ReplaceResourceUsagesOpts{OverrideValueType: true})

				replaceResourceUsages(resourceUsage, existingResourceUsage, ReplaceResourceUsagesOpts{})
			}
		}
	}

	return resourceUsage
}

func syncResourceType(projectCtx *settings.ProjectSession, resource *engine.Estimate, referenceFile *ReferenceFile, existingResourceUsagesMap map[string]*ResourceUsage) *ResourceUsage {
	resourceUsage := &ResourceUsage{
		Name: resource.ResourceType,
	}

	// Merge the usage schema from the reference usage file
	refResourceUsage := referenceFile.FindMatchingResourceTypeUsage(resource.ResourceType)
	if refResourceUsage != nil {
		replaceResourceUsages(resourceUsage, refResourceUsage, ReplaceResourceUsagesOpts{})
	}

	// Merge the usage schema from the resource struct
	// We want to override the value type from the usage schema since we can't always tell from the YAML
	// what the value type should be, e.g. user might add an int value for a float attribute.
	replaceResourceUsages(resourceUsage, &ResourceUsage{
		Name:  resourceUsage.Name,
		Items: resource.UsageSchema,
	}, ReplaceResourceUsagesOpts{OverrideValueType: true})

	// Merge any existing resource usage
	existingResourceUsage := existingResourceUsagesMap[resource.ResourceType]
	if existingResourceUsage != nil {
		replaceResourceUsages(resourceUsage, existingResourceUsage, ReplaceResourceUsagesOpts{})
	}

	return resourceUsage
}

func syncResource(projectCtx *settings.ProjectSession, resource *engine.Estimate, referenceFile *ReferenceFile, existingResourceUsagesMap map[string]*ResourceUsage) (*ResourceUsage, *SyncResult) {
	syncResult := &SyncResult{
		EstimationErrors: make(map[string]error),
	}

	resourceUsage := &ResourceUsage{
		Name: resource.Name,
	}

	// Merge the usage schema from the reference usage file
	refResourceUsage := referenceFile.FindMatchingResourceUsage(resource.Name)
	if refResourceUsage != nil {
		replaceResourceUsages(resourceUsage, refResourceUsage, ReplaceResourceUsagesOpts{})
	}

	// Merge the usage schema from the resource struct
	// We want to override the value type from the usage schema since we can't always tell from the YAML
	// what the value type should be, e.g. user might add an int value for a float attribute.
	replaceResourceUsages(resourceUsage, &ResourceUsage{
		Name:  resource.Name,
		Items: resource.UsageSchema,
	}, ReplaceResourceUsagesOpts{OverrideValueType: true})

	// Merge any existing resource usage
	existingResourceUsage := existingResourceUsagesMap[resource.Name]
	if existingResourceUsage != nil {
		replaceResourceUsages(resourceUsage, existingResourceUsage, ReplaceResourceUsagesOpts{})
	}

	syncResult.ResourceCount++
	if resource.EstimateUsage != nil {
		syncResult.EstimationCount++

		resourceUsageMap := resourceUsage.Map()

		ctx := context.WithValue(context.Background(), ContextEnv{}, projectCtx.ProjectConfig.Env)
		err := resource.EstimateUsage(ctx, resourceUsageMap)
		if err != nil {
			syncResult.EstimationErrors[resource.Name] = err
			logging.Logger.Warn().Msgf("Error estimating usage for resource %s: %v", resource.Name, err)
		}

		// Merge in the estimated usage
		estimatedUsageData := engine.NewUsageData(resource.Name, engine.ParseAttributes(resourceUsageMap))
		mergeResourceUsageWithUsageData(resourceUsage, estimatedUsageData)
	}

	return resourceUsage, syncResult
}

// replaceResourceUsages override usageItems from dest with usageItems from src
func replaceResourceUsages(dest *ResourceUsage, src *ResourceUsage, opts ReplaceResourceUsagesOpts) {
	if dest == nil || src == nil {
		return
	}

	destItemMap := make(map[string]*engine.ConsumptionField, len(dest.Items))
	for _, item := range dest.Items {
		destItemMap[item.Key] = item
	}

	for _, srcItem := range src.Items {
		destItem, ok := destItemMap[srcItem.Key]
		if !ok {
			destItem = &engine.ConsumptionField{Key: srcItem.Key, ValueType: srcItem.ValueType}
			dest.Items = append(dest.Items, destItem)
		}

		if opts.OverrideValueType {
			destItem.ValueType = srcItem.ValueType
		}

		if srcItem.Description != "" {
			destItem.Description = srcItem.Description
		}

		if srcItem.ValueType == engine.SubResourceUsage {
			if srcItem.DefaultValue != nil {
				srcDefaultValue := srcItem.DefaultValue.(*ResourceUsage)
				if destItem.DefaultValue == nil {
					destItem.DefaultValue = &ResourceUsage{
						Name: srcDefaultValue.Name,
					}
				}
				replaceResourceUsages(destItem.DefaultValue.(*ResourceUsage), srcDefaultValue, opts)
			}

			if srcItem.Value != nil {
				srcValue := srcItem.Value.(*ResourceUsage)
				if destItem.Value == nil {
					destItem.Value = destItem.DefaultValue
				}
				if destItem.Value == nil {
					destItem.Value = &ResourceUsage{
						Name: srcValue.Name,
					}
				}
				replaceResourceUsages(destItem.Value.(*ResourceUsage), srcValue, opts)
			}
		} else {
			if srcItem.DefaultValue != nil {
				destItem.DefaultValue = srcItem.DefaultValue
			}

			if srcItem.Value != nil {
				destItem.Value = srcItem.Value
			}
		}
	}
}

func mergeResourceUsageWithUsageData(resourceUsage *ResourceUsage, usageData *engine.ConsumptionProfile) {
	if usageData == nil {
		return
	}

	for _, item := range resourceUsage.Items {
		var val interface{}

		switch item.ValueType {
		case engine.Int64:
			if v := usageData.GetInt(item.Key); v != nil {
				val = *v
			}
		case engine.Float64:
			if v := usageData.GetFloat(item.Key); v != nil {
				val = *v
			}
		case engine.String:
			if v := usageData.GetString(item.Key); v != nil {
				val = *v
			}
		case engine.StringArray:
			if v := usageData.GetStringArray(item.Key); v != nil {
				val = *v
			}
		case engine.SubResourceUsage:
			subUsageMap := usageData.Get(item.Key).Map()
			subExisting := engine.NewUsageData(item.Key, subUsageMap)

			var subResourceUsage *ResourceUsage
			// If the item has a value, use it as the base
			if item.Value != nil {
				subResourceUsage = item.Value.(*ResourceUsage)
			}

			// If the resource usage is nil, but the usage data we want to merge has data
			// for any of its sub-items, we want to add the sub-items in first before we merge
			if item.Value == nil && item.DefaultValue != nil {
				subResourceUsage = &ResourceUsage{
					Name: item.Key,
				}

				hasSubItems := false
				for _, subItem := range item.DefaultValue.(*ResourceUsage).Items {
					if subExisting.Get(subItem.Key).Type != gjson.Null {
						hasSubItems = true
						subResourceUsage.Items = append(subResourceUsage.Items, subItem)
					}
				}

				if !hasSubItems {
					subResourceUsage = nil
				}
			}

			if subResourceUsage != nil {
				mergeResourceUsageWithUsageData(subResourceUsage, subExisting)
			}

			if subResourceUsage != nil {
				val = subResourceUsage
			}
		case engine.KeyValueMap:
			if v := usageData.Get(item.Key).Map(); v != nil {
				val = v
			}
		}

		if val != nil {
			item.Value = val
		}
	}
}

// sortResourceUsages sorts the resources by the existing order first, and then the rest by name
// Keep multiple-resource (count or each) together even if they are not in the existing
// order
func sortResourceUsages(resourceUsages []*ResourceUsage, existingOrder []string) {
	sort.Slice(resourceUsages, func(i, j int) bool {
		iExistingIndex := indexOf(resourceUsages[i].Name, existingOrder)
		jExistingIndex := indexOf(resourceUsages[j].Name, existingOrder)

		// If both resources are in the existing resource order, sort by the existing resource order
		if iExistingIndex != -1 && jExistingIndex != -1 {
			// this happens when the resource usage does not exists but the wildcard resource
			// usage exists
			if jExistingIndex == iExistingIndex {
				return resourceUsages[i].Name < resourceUsages[j].Name
			}
			return iExistingIndex < jExistingIndex
		}

		// If one resource is in the existing resource order, sort it first
		if iExistingIndex == -1 && jExistingIndex != -1 {
			return false
		}
		if jExistingIndex == -1 && iExistingIndex != -1 {
			return true
		}

		// If neither resource is in the existing resource order, sort resources that have a value first
		iHasUsage := resourceUsageHasValue(resourceUsages[i])
		jHasUsafe := resourceUsageHasValue(resourceUsages[j])
		if iHasUsage && !jHasUsafe {
			return true
		}
		if jHasUsafe && !iHasUsage {
			return false
		}

		// Otherwise sort by name
		return resourceUsages[i].Name < resourceUsages[j].Name
	})
}

func resourceUsageHasValue(resourceUsage *ResourceUsage) bool {
	for _, item := range resourceUsage.Items {
		if item.Value != nil {
			return true
		}
	}
	return false
}

func indexOf(s string, arr []string) int {
	for k, v := range arr {
		if s == v {
			return k
		}
	}

	if isMultipleResource := strings.HasSuffix(s, "]"); isMultipleResource {
		lastIndexOfOpenBracket := strings.LastIndex(s, "[")
		prefixName := s[:lastIndexOfOpenBracket]
		return lastIndexOfPrefix(prefixName, arr) + 1
	}

	return -1
}

func lastIndexOfPrefix(s string, arr []string) int {
	for i := len(arr) - 1; i >= 0; i-- {
		if strings.HasPrefix(arr[i], s) {
			return i
		}
	}
	return -1
}

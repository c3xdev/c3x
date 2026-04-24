package pricing

import (
	"runtime"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/apiclient"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/usage"
)

// PopulateActualCosts fetches cloud provider reported costs from the C3X
// Cloud Usage API and adds corresponding cost components to the project's resources
func PopulateActualCosts(ctx *settings.Session, project *engine.Workspace) error {
	resources := project.AllResources()

	c := apiclient.NewUsageAPIClient(ctx)

	err := popResourcesConcurrent(ctx, c, project, resources)
	if err != nil {
		return err
	}
	return nil
}

// popResourcesConcurrent gets the actual usage of all resources concurrently.
// Concurrency level is calculated using the following formula:
// max(min(4, numCPU * 4), 16)
func popResourcesConcurrent(ctx *settings.Session, c *apiclient.UsageAPIClient, project *engine.Workspace, resources []*engine.Estimate) error {
	// Set the number of workers
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
	resultErrors := make(chan error, numJobs)

	// Fire up the workers
	for i := 0; i < numWorkers; i++ {
		go func(jobs <-chan *engine.Estimate, resultErrors chan<- error) {
			for r := range jobs {
				err := popResourceActualCosts(ctx, c, project, r)
				resultErrors <- err
			}
		}(jobs, resultErrors)
	}

	// Feed the workers the jobs of getting prices
	for _, r := range resources {
		jobs <- r
	}
	close(jobs)

	// Get the result of the jobs
	for i := 0; i < numJobs; i++ {
		err := <-resultErrors
		if err != nil {
			return err
		}
	}
	return nil
}

func popResourceActualCosts(ctx *settings.Session, c *apiclient.UsageAPIClient, project *engine.Workspace, r *engine.Estimate) error {
	if r.IsSkipped {
		return nil
	}

	vars := apiclient.ActualCostsQueryVariables{
		RepoURL:              ctx.VCSRepositoryURL(),
		ProjectWithWorkspace: project.NameWithWorkspace(),
		Address:              r.Name,
		Currency:             c.Currency,
	}
	actualCostResults, err := c.ListActualCosts(vars)
	if actualCostResults == nil || err != nil {
		return err
	}

	for _, actualCost := range actualCostResults {
		actualCosts := &engine.ObservedCost{
			ResourceID:     actualCost.ResourceID,
			StartTimestamp: actualCost.StartTimestamp.UTC(),
			EndTimestamp:   actualCost.EndTimestamp.UTC(),
			CostComponents: make([]*engine.LineItem, 0, len(actualCost.CostComponents)),
		}

		for _, actual := range actualCost.CostComponents {
			monthlyCost, err := decimal.NewFromString(actual.MonthlyCost)
			if err != nil {
				logging.Logger.Debug().Err(err).Msgf("failed to parse monthlyCost for %s", actual.Description)
				continue
			}

			monthlyQuantity, err := decimal.NewFromString(actual.MonthlyQuantity)
			if err != nil {
				logging.Logger.Debug().Err(err).Msgf("failed to parse monthlyQuantity for %s", actual.Description)
				continue
			}
			price, err := decimal.NewFromString(actual.Price)
			if err != nil {
				logging.Logger.Debug().Err(err).Msgf("failed to parse price for %s", actual.Description)
				continue
			}

			cc := &engine.LineItem{
				Name:            actual.Description,
				Unit:            actual.Unit,
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyCost:     &monthlyCost,
				MonthlyQuantity: &monthlyQuantity,
			}
			cc.SetPrice(price)

			actualCosts.CostComponents = append(actualCosts.CostComponents, cc)
		}

		if len(actualCosts.CostComponents) > 0 {
			r.ActualCosts = append(r.ActualCosts, actualCosts)
		}
	}

	return nil
}

// chunkPartialResourcesWithUsage collects all partiral resources with a core resource usage schema
// into groups of the specified chunkSize
func chunkPartialResourcesWithUsage(resources []*engine.UnpricedEntry, chunkSize int) [][]*engine.UnpricedEntry {
	var usageResourceChunks [][]*engine.UnpricedEntry
	var currentChunk []*engine.UnpricedEntry
	for _, rb := range resources {
		if rb.CoreResource != nil && len(rb.CoreResource.UsageSchema()) > 0 {
			if len(currentChunk) >= chunkSize {
				usageResourceChunks = append(usageResourceChunks, currentChunk)
				currentChunk = nil
			}
			currentChunk = append(currentChunk, rb)
		}
	}
	if len(currentChunk) > 0 {
		usageResourceChunks = append(usageResourceChunks, currentChunk)
	}

	return usageResourceChunks
}

// FetchUsageData fetches usage estimates derived from cloud provider reported usage
// from the C3X Cloud Usage API for each supported resource in the project
func FetchUsageData(ctx *settings.Session, project *engine.Workspace) (engine.ConsumptionMap, error) {
	c := apiclient.NewUsageAPIClient(ctx)

	// Set the number of workers
	numWorkers := 4
	numCPU := runtime.NumCPU()
	if numCPU*4 > numWorkers {
		numWorkers = numCPU * 4
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	// gather all the CoreResource into chunks
	usageResourceChunks := chunkPartialResourcesWithUsage(project.AllPartialResources(), 10)

	usageMap := make(map[string]*engine.ConsumptionProfile, len(usageResourceChunks)*10)

	numJobs := len(usageResourceChunks)
	jobs := make(chan []*engine.UnpricedEntry, numJobs)
	responses := make(chan batchResponse, numJobs)

	// Fire up the workers
	for i := 0; i < numWorkers; i++ {
		go func(jobs <-chan []*engine.UnpricedEntry, responses chan<- batchResponse) {
			for req := range jobs {
				res := fetchUsageDataBatch(ctx, c, project, req)
				responses <- res
			}
		}(jobs, responses)
	}

	// Feed the workers the jobs
	for _, r := range usageResourceChunks {
		jobs <- r
	}
	close(jobs)

	// Get the result of the jobs
	for i := 0; i < numJobs; i++ {
		res := <-responses
		if res.Error != nil {
			return engine.NewUsageMap(usageMap), res.Error
		}

		for _, ud := range res.UsageData {
			usageMap[ud.Address] = ud
		}
	}

	return engine.NewUsageMap(usageMap), nil
}

type batchResponse struct {
	UsageData []*engine.ConsumptionProfile
	Error     error
}

func fetchUsageDataBatch(ctx *settings.Session, c *apiclient.UsageAPIClient, project *engine.Workspace, resources []*engine.UnpricedEntry) batchResponse {
	chunkedQueryVars := make([]*apiclient.UsageQuantitiesQueryVariables, 0, len(resources))
	for _, partialResource := range resources {
		var usageParams []engine.ConsumptionParam
		if crWithUsageParams, ok := partialResource.CoreResource.(engine.CatalogItemWithConsumptionParams); ok {
			usageParams = crWithUsageParams.UsageEstimationParams()
		}

		queryVars := apiclient.UsageQuantitiesQueryVariables{
			RepoURL:              ctx.VCSRepositoryURL(),
			ProjectWithWorkspace: project.NameWithWorkspace(),
			ResourceType:         partialResource.CoreResource.CoreType(),
			Address:              partialResource.Address,
			UsageKeys:            flattenUsageKeys(partialResource.CoreResource.UsageSchema()),
			UsageParams:          usageParams,
		}
		chunkedQueryVars = append(chunkedQueryVars, &queryVars)
	}

	ud, err := c.ListUsageQuantities(chunkedQueryVars)

	return batchResponse{
		UsageData: ud,
		Error:     err,
	}
}

// UploadCloudResourceIDs sends the project scoped cloud resource ids to the Usage API, so they can be used
// to provide usage estimates.
func UploadCloudResourceIDs(ctx *settings.Session, project *engine.Workspace) error {
	c := apiclient.NewUsageAPIClient(ctx)

	var resourceIDs []apiclient.ResourceIDAddress
	for _, partial := range project.AllPartialResources() {
		for _, resourceID := range partial.CloudResourceIDs {
			resourceIDs = append(resourceIDs, apiclient.ResourceIDAddress{
				Address:    partial.Address,
				ResourceID: resourceID},
			)
		}
	}

	vars := apiclient.CloudResourceIDVariables{
		RepoURL:              ctx.VCSRepositoryURL(),
		ProjectWithWorkspace: project.NameWithWorkspace(),
		ResourceIDAddresses:  resourceIDs,
	}

	err := c.UploadCloudResourceIDs(vars)
	if err != nil {
		return err
	}

	return nil
}

func flattenUsageKeys(usageSchema []*engine.ConsumptionField) []string {
	usageKeys := make([]string, 0, len(usageSchema))
	for _, usageItem := range usageSchema {
		if usageItem.ValueType == engine.SubResourceUsage {
			ru, ok := usageItem.DefaultValue.(*usage.ResourceUsage)
			if !ok {
				continue
			}
			// recursively flatten any nested keys, then add them to the current list
			for _, nestedKey := range flattenUsageKeys(ru.Items) {
				usageKeys = append(usageKeys, usageItem.Key+"."+nestedKey)
			}
		} else {
			usageKeys = append(usageKeys, usageItem.Key)
		}
	}

	return usageKeys
}

package azure

import (
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// SecurityCenterSubscriptionPricing struct represents the pricing structure for Microsoft Defender for Cloud.
// Currently, pricing is supported through the usage file.
//
// Resource information: https://learn.microsoft.com/en-us/azure/defender-for-cloud/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/defender-for-cloud/
type SecurityCenterSubscriptionPricing struct {
	Address      string
	Region       string
	Tier         string
	ResourceType string

	MonthlyServersPlan1Nodes *float64 `c3x_usage:"monthly_servers_plan_1_nodes"`
	MonthlyServersPlan2Nodes *float64 `c3x_usage:"monthly_servers_plan_2_nodes"`

	MonthlyContainersVCores        *float64 `c3x_usage:"monthly_containers_vcores"`
	MonthlyContainerRegistryImages *float64 `c3x_usage:"monthly_container_registry_images"`

	MonthlySQLAzureConnectedInstances *float64 `c3x_usage:"monthly_sql_azure_connected_instances"`
	MonthlySQLOutsideAzureVCores      *float64 `c3x_usage:"monthly_sql_outside_azure_vcores"`
	MonthlyMySQLInstances             *float64 `c3x_usage:"monthly_mysql_instances"`
	MonthlyPostgreSQLInstances        *float64 `c3x_usage:"monthly_postgresql_instances"`
	MonthlyMariaDBInstances           *float64 `c3x_usage:"monthly_mariadb_instances"`
	CosmosDBRequestUnits              *float64 `c3x_usage:"cosmosdb_request_units"`

	MonthlyStorageAccounts *float64 `c3x_usage:"monthly_storage_accounts"`

	MonthlyAppServiceNodes  *float64 `c3x_usage:"monthly_app_service_nodes"`
	MonthlyKeyVaults        *int64   `c3x_usage:"monthly_key_vaults"`
	MonthlyARMSubscriptions *int64   `c3x_usage:"monthly_arm_subscriptions"`
	MonthlyDNSQueries       *int64   `c3x_usage:"monthly_dns_queries"`

	MonthlyKubernetesCores *float64 `c3x_usage:"monthly_kubernetes_cores"`
}

// CoreType returns the name of this resource type
func (r *SecurityCenterSubscriptionPricing) CoreType() string {
	return "SecurityCenterSubscriptionPricing"
}

// UsageSchema defines a list which represents the usage schema of SecurityCenterSubscriptionPricing.
func (r *SecurityCenterSubscriptionPricing) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_servers_plan_1_nodes", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_servers_plan_2_nodes", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_containers_vcores", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_container_registry_images", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_sql_azure_connected_instances", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_sql_outside_azure_vcores", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_mysql_instances", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_postgresql_instances", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_mariadb_instances", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "cosmosdb_request_units", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_storage_accounts", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_app_service_nodes", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_key_vaults", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_arm_subscriptions", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_dns_queries", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_kubernetes_cores", DefaultValue: 0.0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the SecurityCenterSubscriptionPricing.
// It uses the `c3x_usage` struct tags to populate data into the SecurityCenterSubscriptionPricing.
func (r *SecurityCenterSubscriptionPricing) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid SecurityCenterSubscriptionPricing struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *SecurityCenterSubscriptionPricing) BuildResource() *engine.Estimate {
	if strings.ToLower(r.Tier) == "free" {
		return &engine.Estimate{
			Name:      r.Address,
			IsSkipped: true,
			NoPrice:   true,
		}
	}

	var costComponents []*engine.LineItem
	switch strings.ToLower(r.ResourceType) {
	case "appservices":
		costComponents = []*engine.LineItem{r.addAppServiceCostComponent()}
	case "containerregistry":
		costComponents = []*engine.LineItem{r.addContainerRegistryCostComponent()}
	case "keyvaults":
		costComponents = []*engine.LineItem{r.addKeyVaultCostComponent()}
	case "kubernetesservice":
		costComponents = []*engine.LineItem{r.addKubernetesCostComponent()}
	case "sqlservers":
		costComponents = []*engine.LineItem{r.addSQLOutsideAzureCostComponent()}
	case "sqlservervirtualmachines":
		costComponents = []*engine.LineItem{r.addSQLAzureConnectedCostComponent()}
	case "storageaccounts":
		costComponents = []*engine.LineItem{r.addStorageCostComponent()}
	case "virtualmachines":
		costComponents = []*engine.LineItem{
			r.addServersP1CostComponent(),
			r.addServersP2CostComponent(),
		}
	case "arm":
		costComponents = []*engine.LineItem{r.addARMCostComponent()}
	case "dns":
		costComponents = []*engine.LineItem{r.addDNSCostComponent()}
	case "opensourcerelationaldatabases":
		costComponents = []*engine.LineItem{
			r.addMySQLCostComponent(),
			r.addPostgreSQLCostComponent(),
			r.addMariaDBCostComponent(),
		}
	case "containers":
		costComponents = []*engine.LineItem{r.addContainersCostComponent()}
	case "cosmosdbs":
		costComponents = []*engine.LineItem{r.addCosmosDBCostComponent()}
	default:
		logging.Logger.Warn().Msgf("Skipping resource %s. Unknown resource type  '%s'", r.Address, r.ResourceType)
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *SecurityCenterSubscriptionPricing) addServersP1CostComponent() *engine.LineItem {
	var vmHours *decimal.Decimal
	if r.MonthlyServersPlan1Nodes != nil {
		vmHours = decimalPtr(decimal.NewFromFloat(*r.MonthlyServersPlan1Nodes).Mul(engine.HourToMonthUnitMultiplier))
	}

	return &engine.LineItem{
		Name:            "Defender for servers, plan 1",
		Unit:            "server",
		UnitMultiplier:  engine.HourToMonthUnitMultiplier,
		MonthlyQuantity: vmHours,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Servers")},
				{Key: "meterName", Value: strPtr("Standard P1 Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addServersP2CostComponent() *engine.LineItem {
	var vmHours *decimal.Decimal
	if r.MonthlyServersPlan2Nodes != nil {
		vmHours = decimalPtr(decimal.NewFromFloat(*r.MonthlyServersPlan2Nodes).Mul(engine.HourToMonthUnitMultiplier))
	}

	return &engine.LineItem{
		Name:            "Defender for servers, plan 2",
		Unit:            "server",
		UnitMultiplier:  engine.HourToMonthUnitMultiplier,
		MonthlyQuantity: vmHours,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Servers")},
				{Key: "meterName", Value: strPtr("Standard P2 Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addContainersCostComponent() *engine.LineItem {
	var vmHours *decimal.Decimal
	if r.MonthlyContainersVCores != nil {
		vmHours = decimalPtr(decimal.NewFromFloat(*r.MonthlyContainersVCores).Mul(engine.HourToMonthUnitMultiplier))
	}

	return &engine.LineItem{
		Name:            "Defender for containers",
		Unit:            "vCore",
		UnitMultiplier:  engine.HourToMonthUnitMultiplier,
		MonthlyQuantity: vmHours,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Containers")},
				{Key: "meterName", Value: strPtr("Standard vCore vCore Pack")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addSQLAzureConnectedCostComponent() *engine.LineItem {
	var instances *decimal.Decimal
	if r.MonthlySQLAzureConnectedInstances != nil {
		instances = decimalPtr(decimal.NewFromFloat(*r.MonthlySQLAzureConnectedInstances))
	}

	return &engine.LineItem{
		Name:            "Defender for SQL, Azure-connected",
		Unit:            "instance",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: instances,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for SQL")},
				{Key: "meterName", Value: strPtr("Standard Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addSQLOutsideAzureCostComponent() *engine.LineItem {
	var vCoreHours *decimal.Decimal
	if r.MonthlySQLOutsideAzureVCores != nil {
		vCoreHours = decimalPtr(decimal.NewFromFloat(*r.MonthlySQLOutsideAzureVCores).Mul(engine.HourToMonthUnitMultiplier))
	}

	return &engine.LineItem{
		Name:            "Defender for SQL, outside Azure",
		Unit:            "vCore",
		UnitMultiplier:  engine.HourToMonthUnitMultiplier,
		MonthlyQuantity: vCoreHours,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for SQL")},
				{Key: "meterName", Value: strPtr("Standard vCore")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addMySQLCostComponent() *engine.LineItem {
	var instances *decimal.Decimal
	if r.MonthlyMySQLInstances != nil {
		instances = decimalPtr(decimal.NewFromFloat(*r.MonthlyMySQLInstances))
	}

	return &engine.LineItem{
		Name:            "Defender for MySQL",
		Unit:            "instance",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: instances,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for MySQL")},
				{Key: "meterName", Value: strPtr("Standard Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addPostgreSQLCostComponent() *engine.LineItem {
	var instances *decimal.Decimal
	if r.MonthlyPostgreSQLInstances != nil {
		instances = decimalPtr(decimal.NewFromFloat(*r.MonthlyPostgreSQLInstances))
	}

	return &engine.LineItem{
		Name:            "Defender for PostgreSQL",
		Unit:            "instance",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: instances,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for PostgreSQL")},
				{Key: "meterName", Value: strPtr("Standard Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addMariaDBCostComponent() *engine.LineItem {
	var instances *decimal.Decimal
	if r.MonthlyMariaDBInstances != nil {
		instances = decimalPtr(decimal.NewFromFloat(*r.MonthlyMariaDBInstances))
	}

	region := r.normalizedRegion()
	if *region == "Global" {
		// force to west-us2 since price is not available in Global
		region = strPtr("westus2")
	}

	return &engine.LineItem{
		Name:           "Defender for MariaDB",
		Unit:           "instance",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: instances,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        region,
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for MariaDB")},
				{Key: "meterName", Value: strPtr("Standard Instance")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addCosmosDBCostComponent() *engine.LineItem {
	var averageRUs *decimal.Decimal
	if r.CosmosDBRequestUnits != nil {
		averageRUs = decimalPtr(decimal.NewFromFloat(*r.CosmosDBRequestUnits).Div(decimal.NewFromInt(100)))
	}

	return &engine.LineItem{
		Name:           "Defender for Cosmos DB",
		Unit:           "RU/s x 100",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: averageRUs,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Azure Cosmos DB")},
				{Key: "meterName", Value: strPtr("Standard 100 RU/s")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addStorageCostComponent() *engine.LineItem {
	var storageAccounts *decimal.Decimal
	if r.MonthlyStorageAccounts != nil {
		storageAccounts = decimalPtr(decimal.NewFromFloat(*r.MonthlyStorageAccounts))
	}

	return &engine.LineItem{
		Name:           "Defender for storage",
		Unit:           "storage account",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: storageAccounts,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Storage")},
				{Key: "meterName", Value: strPtr("Standard Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addAppServiceCostComponent() *engine.LineItem {
	var nodes *decimal.Decimal
	if r.MonthlyAppServiceNodes != nil {
		nodes = decimalPtr(decimal.NewFromFloat(*r.MonthlyAppServiceNodes))
	}

	return &engine.LineItem{
		Name:           "Defender for app service",
		Unit:           "node",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: nodes,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for App Service")},
				{Key: "meterName", Value: strPtr("Standard Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addKeyVaultCostComponent() *engine.LineItem {
	var keyVaults *decimal.Decimal
	if r.MonthlyKeyVaults != nil {
		keyVaults = decimalPtr(decimal.NewFromInt(*r.MonthlyKeyVaults))
	}

	region := r.normalizedRegion()
	if *region == "Global" {
		// force to west-us2 since price is not available in Global
		region = strPtr("westus2")
	}

	return &engine.LineItem{
		Name:           "Defender for Key Vault",
		Unit:           "key vault",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: keyVaults,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        region,
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Key Vault")},
				{Key: "meterName", Value: strPtr("Per node Std Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addARMCostComponent() *engine.LineItem {
	var subscriptions *decimal.Decimal
	if r.MonthlyARMSubscriptions != nil {
		subscriptions = decimalPtr(decimal.NewFromInt(*r.MonthlyARMSubscriptions))
	}

	region := r.normalizedRegion()
	if *region == "Global" {
		// force to west-us2 since price is not available in Global
		region = strPtr("westus2")
	}

	return &engine.LineItem{
		Name:           "Defender for ARM",
		Unit:           "subscription",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: subscriptions,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        region,
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Resource Manager")},
				{Key: "meterName", Value: strPtr("Per node Std Node")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addDNSCostComponent() *engine.LineItem {
	var apiCalls *decimal.Decimal
	if r.MonthlyDNSQueries != nil {
		apiCalls = decimalPtr(decimal.NewFromInt(*r.MonthlyDNSQueries).Div(decimal.NewFromInt(1000000)))
	}

	region := r.normalizedRegion()
	if *region == "Global" {
		// force to west-us2 since price is not available in Global
		region = strPtr("westus2")
	}

	return &engine.LineItem{
		Name:            "Defender for DNS",
		Unit:            "1M queries",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: apiCalls,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        region,
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Microsoft Defender for Cloud"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for DNS")},
				{Key: "meterName", Value: strPtr("Standard Queries")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addKubernetesCostComponent() *engine.LineItem {
	var nodes *decimal.Decimal
	if r.MonthlyKubernetesCores != nil {
		nodes = decimalPtr(decimal.NewFromFloat(*r.MonthlyKubernetesCores))
	}

	return &engine.LineItem{
		Name:           "Defender for kubernetes",
		Unit:           "core",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: nodes,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Advanced Threat Protection"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Kubernetes")},
				{Key: "meterName", Value: strPtr("Standard Cores")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) addContainerRegistryCostComponent() *engine.LineItem {
	var instances *decimal.Decimal
	if r.MonthlyContainerRegistryImages != nil {
		instances = decimalPtr(decimal.NewFromFloat(*r.MonthlyContainerRegistryImages))
	}
	return &engine.LineItem{
		Name:            "Defender for container registries",
		Unit:            "image",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: instances,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			ProductFamily: strPtr("Security"),
			Service:       strPtr("Advanced Threat Protection"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Microsoft Defender for Container Registries")},
				{Key: "meterName", Value: strPtr("Standard Images")},
			},
		},
		UsageBased: true,
	}
}

func (r *SecurityCenterSubscriptionPricing) normalizedRegion() *string {
	if r.Region == "global" {
		return strPtr("Global")
	}
	return strPtr(r.Region)
}

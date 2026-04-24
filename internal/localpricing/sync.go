package localpricing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// syncClient is a shared HTTP client with a generous timeout for downloading
// large pricing files from cloud provider APIs.
var syncClient = &http.Client{Timeout: 10 * time.Minute}

// SyncOptions configures which providers to sync.
type SyncOptions struct {
	Providers  []string // aws, azure, google
	Store      *Store
	OnProgress func(provider string, count int)
}

// Sync downloads pricing data from cloud provider bulk APIs into the local store.
func Sync(opts SyncOptions) error {
	for _, provider := range opts.Providers {
		switch provider {
		case "aws":
			if err := syncAWS(opts); err != nil {
				return fmt.Errorf("syncing AWS pricing: %w", err)
			}
		case "azure":
			if err := syncAzure(opts); err != nil {
				return fmt.Errorf("syncing Azure pricing: %w", err)
			}
		case "google":
			if err := syncGoogle(opts); err != nil {
				return fmt.Errorf("syncing Google pricing: %w", err)
			}
		default:
			return fmt.Errorf("unknown provider: %s", provider)
		}
	}

	_ = opts.Store.SetMetadata("last_sync", time.Now().UTC().Format(time.RFC3339))
	return nil
}

// syncAWS downloads AWS pricing from the bulk pricing API.
func syncAWS(opts SyncOptions) error {
	// AWS offers index: https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/index.json
	indexURL := "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/index.json"

	resp, err := syncClient.Get(indexURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AWS pricing index returned HTTP %d", resp.StatusCode)
	}

	var index struct {
		Offers map[string]struct {
			CurrentVersionURL string `json:"currentVersionUrl"`
		} `json:"offers"`
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024*1024))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &index); err != nil {
		return err
	}

	// Key services to download for cost estimation
	services := []string{
		"AmazonEC2", "AmazonRDS", "AmazonS3", "AWSLambda",
		"AmazonDynamoDB", "AmazonElastiCache", "AmazonEKS", "AmazonEFS",
		"AmazonES", "AmazonCloudWatch", "AmazonSNS", "AmazonSQS",
		"AmazonRoute53", "AmazonCloudFront", "AWSELB", "AmazonVPC",
		"AWSGlobalAccelerator", "AmazonKinesisFirehose", "AmazonRedshift",
		"AmazonMSK", "AWSSecretsManager", "AWSStepFunctions",
	}

	count := 0
	for _, svc := range services {
		offer, ok := index.Offers[svc]
		if !ok {
			continue
		}

		url := "https://pricing.us-east-1.amazonaws.com" + offer.CurrentVersionURL
		if err := downloadAndStoreAWSService(opts.Store, svc, url); err != nil {
			// Log but continue with other services
			fmt.Printf("  Warning: failed to sync %s: %v\n", svc, err)
			continue
		}
		count++
		if opts.OnProgress != nil {
			opts.OnProgress("aws", count)
		}
	}

	return nil
}

func downloadAndStoreAWSService(store *Store, service, url string) error {
	resp, err := syncClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AWS pricing for %s returned HTTP %d", service, resp.StatusCode)
	}

	var data struct {
		Products map[string]struct {
			SKU           string            `json:"sku"`
			ProductFamily string            `json:"productFamily"`
			Attributes    map[string]string `json:"attributes"`
		} `json:"products"`
		Terms struct {
			OnDemand map[string]map[string]struct {
				PriceDimensions map[string]struct {
					PricePerUnit map[string]string `json:"pricePerUnit"`
					Unit         string            `json:"unit"`
					Description  string            `json:"description"`
				} `json:"priceDimensions"`
			} `json:"OnDemand"`
		} `json:"terms"`
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024*1024))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	for sku, product := range data.Products {
		region := product.Attributes["regionCode"]
		if region == "" {
			region = product.Attributes["location"]
		}

		attrsJSON, _ := json.Marshal(product.Attributes)

		// Collect prices
		var prices []map[string]string
		if terms, ok := data.Terms.OnDemand[sku]; ok {
			for _, term := range terms {
				for _, dim := range term.PriceDimensions {
					prices = append(prices, map[string]string{
						"USD":         dim.PricePerUnit["USD"],
						"unit":        dim.Unit,
						"description": dim.Description,
					})
				}
			}
		}
		pricesJSON, _ := json.Marshal(prices)

		_ = store.UpsertProduct("aws", region, service, product.ProductFamily, sku, string(attrsJSON), string(pricesJSON))
	}

	return nil
}

// syncAzure downloads Azure pricing from the Retail Prices API.
func syncAzure(opts SyncOptions) error {
	baseURL := "https://prices.azure.com/api/retail/prices?$top=100"
	nextPage := baseURL
	count := 0

	for nextPage != "" {
		resp, err := syncClient.Get(nextPage)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("Azure pricing API returned HTTP %d", resp.StatusCode)
		}

		var result struct {
			Items []struct {
				CurrencyCode  string  `json:"currencyCode"`
				RetailPrice   float64 `json:"retailPrice"`
				UnitOfMeasure string  `json:"unitOfMeasure"`
				ArmRegionName string  `json:"armRegionName"`
				ServiceName   string  `json:"serviceName"`
				ProductName   string  `json:"productName"`
				MeterName     string  `json:"meterName"`
				SkuName       string  `json:"skuName"`
				SkuID         string  `json:"skuId"`
				ProductID     string  `json:"productId"`
			} `json:"Items"`
			NextPageLink string `json:"NextPageLink"`
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024*1024))
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("error reading Azure pricing response: %w", err)
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}

		for _, item := range result.Items {
			attrs := map[string]string{
				"productName": item.ProductName,
				"meterName":   item.MeterName,
				"skuName":     item.SkuName,
			}
			attrsJSON, _ := json.Marshal(attrs)
			prices := []map[string]string{{
				"USD":  fmt.Sprintf("%f", item.RetailPrice),
				"unit": item.UnitOfMeasure,
			}}
			pricesJSON, _ := json.Marshal(prices)

			sku := item.SkuID + "-" + item.MeterName
			_ = opts.Store.UpsertProduct("azure", item.ArmRegionName, item.ServiceName, item.ProductName, sku, string(attrsJSON), string(pricesJSON))
		}

		count += len(result.Items)
		if opts.OnProgress != nil {
			opts.OnProgress("azure", count)
		}

		nextPage = result.NextPageLink
		if count > 500000 {
			break // safety limit
		}
	}

	return nil
}

// gcpServices maps GCP service display names to their Cloud Billing service IDs.
var gcpServices = map[string]string{
	"Compute Engine":      "6F81-5844-456A",
	"Cloud SQL":           "9662-B51E-5089",
	"Cloud Storage":       "95FF-2EF5-5EA1",
	"Cloud Run Functions": "29E7-DA93-CA13",
	"Cloud Run":           "152E-C115-5142",
	"Cloud DNS":           "FA26-5236-B8B5",
	"Networking":          "E505-1604-58F8",
	"Cloud Pub/Sub":       "A1E8-BE35-7EBC",
	"Kubernetes Engine":   "CCD8-9BF1-090E",
}

// syncGoogle downloads GCP pricing from the Cloud Billing API.
func syncGoogle(opts SyncOptions) error {
	apiKey := os.Getenv("GCP_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GCP_API_KEY environment variable is not set — obtain a key from the Google Cloud Console and export it")
	}

	count := 0
	for serviceName, serviceID := range gcpServices {
		if err := syncGCPService(opts, serviceName, serviceID, apiKey, &count); err != nil {
			fmt.Printf("  Warning: failed to sync GCP %s: %v\n", serviceName, err)
			continue
		}
	}

	return nil
}

// gcpSKUResponse represents the JSON response from the Cloud Billing API.
type gcpSKUResponse struct {
	SKUs []struct {
		SkuID       string `json:"skuId"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    struct {
			ServiceDisplayName string `json:"serviceDisplayName"`
			ResourceFamily     string `json:"resourceFamily"`
			ResourceGroup      string `json:"resourceGroup"`
		} `json:"category"`
		ServiceRegions []string `json:"serviceRegions"`
		PricingInfo    []struct {
			PricingExpression struct {
				UsageUnit   string `json:"usageUnit"`
				TieredRates []struct {
					StartUsageAmount float64 `json:"startUsageAmount"`
					UnitPrice        struct {
						CurrencyCode string `json:"currencyCode"`
						Units        string `json:"units"`
						Nanos        int64  `json:"nanos"`
					} `json:"unitPrice"`
				} `json:"tieredRates"`
			} `json:"pricingExpression"`
		} `json:"pricingInfo"`
	} `json:"skus"`
	NextPageToken string `json:"nextPageToken"`
}

func syncGCPService(opts SyncOptions, serviceName, serviceID, apiKey string, count *int) error {
	baseURL := fmt.Sprintf("https://cloudbilling.googleapis.com/v1/services/%s/skus?key=%s&pageSize=500", serviceID, apiKey)
	nextPage := baseURL

	for nextPage != "" {
		resp, err := syncClient.Get(nextPage)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			return fmt.Errorf("GCP Cloud Billing API for %s returned HTTP %d: %s", serviceName, resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024*1024))
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("error reading GCP response for %s: %w", serviceName, err)
		}

		var result gcpSKUResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("error parsing GCP response for %s: %w", serviceName, err)
		}

		for _, sku := range result.SKUs {
			// Normalize service name: "Networking" SKUs are effectively Compute Engine networking
			svcName := sku.Category.ServiceDisplayName
			if svcName == "Networking" {
				svcName = "Compute Engine"
			}

			// Build prices from tiered rates
			var prices []map[string]string
			if len(sku.PricingInfo) > 0 {
				expr := sku.PricingInfo[0].PricingExpression
				for _, tier := range expr.TieredRates {
					// Normalize: parse and re-format to remove leading zeros in nanos
					var unitsVal int64
					_, _ = fmt.Sscanf(tier.UnitPrice.Units, "%d", &unitsVal)
					usdFloat := float64(unitsVal) + float64(tier.UnitPrice.Nanos)/1e9
					usd := fmt.Sprintf("%f", usdFloat)

					prices = append(prices, map[string]string{
						"USD":              usd,
						"unit":             expr.UsageUnit,
						"startUsageAmount": fmt.Sprintf("%g", tier.StartUsageAmount),
					})
				}
			}
			pricesJSON, _ := json.Marshal(prices)

			attrs := map[string]string{
				"description":    sku.Description,
				"resourceGroup":  sku.Category.ResourceGroup,
				"resourceFamily": sku.Category.ResourceFamily,
			}
			attrsJSON, _ := json.Marshal(attrs)

			// Create one product per region. Use skuId+region as the SKU
			// to ensure uniqueness (PK is vendor+sku).
			for _, region := range sku.ServiceRegions {
				regionSKU := sku.SkuID + "-" + region
				_ = opts.Store.UpsertProduct("gcp", region, svcName, sku.Category.ResourceFamily, regionSKU, string(attrsJSON), string(pricesJSON))
			}

			*count++
		}

		if opts.OnProgress != nil {
			opts.OnProgress("google", *count)
		}

		// Paginate
		if result.NextPageToken == "" {
			break
		}
		nextPage = fmt.Sprintf("https://cloudbilling.googleapis.com/v1/services/%s/skus?key=%s&pageSize=500&pageToken=%s", serviceID, apiKey, result.NextPageToken)
	}

	return nil
}

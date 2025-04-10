package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// GetEBSVolumePrice returns the price per GB-month for a given EBS volume type and region
func GetEBSVolumePrice(volumeType string, region string) float64 {
	// Initialize pricing client if not already done
	PricingInitOnce.Do(InitPricingClient)

	// Generate cache key
	cacheKey := fmt.Sprintf("ebs:%s:%s", volumeType, region)

	// Check cache first
	EBSPricingCacheLock.RLock()
	if price, found := EBSPricingCache[cacheKey]; found {
		EBSPricingCacheLock.RUnlock()

		// Update cache hit stats
		UpdateCacheHitStats("EBS", region)

		return price
	}
	EBSPricingCacheLock.RUnlock()

	var price float64
	var err error

	// If pricing client is available, try to get price from AWS API
	if PricingClient != nil {
		price, err = getEBSPriceFromAPI(volumeType, region)
	} else {
		err = fmt.Errorf("pricing client not initialized")
	}

	// If API call failed, use fallback pricing
	if err != nil {
		log.Printf("Error getting EBS price from API: %v. Using fallback pricing for %s in %s.", err, volumeType, region)

		// Update failure stats
		UpdateAPIFailureStats("EBS", region)

		// Fall back to default prices
		if regionPrices, found := DefaultEBSPrices[region]; found {
			if typePrice, found := regionPrices[volumeType]; found {
				price = typePrice
			} else {
				// Default to gp2 price if type not found
				price = regionPrices["gp2"]
			}
		} else {
			// If region not found, use us-east-1 prices
			price = DefaultEBSPrices["us-east-1"][volumeType]
			if price == 0 {
				price = DefaultEBSPrices["us-east-1"]["gp2"] // Default fallback
			}
		}
	} else {
		// Update success stats
		UpdateAPISuccessStats("EBS", region)
	}

	// Cache the result
	EBSPricingCacheLock.Lock()
	EBSPricingCache[cacheKey] = price
	EBSPricingCacheLock.Unlock()

	return price
}

// getEBSPriceFromAPI retrieves EBS volume pricing from the AWS Pricing API
func getEBSPriceFromAPI(volumeType, region string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Map volume type to API value
	volumeTypeValue := mapVolumeTypeToAPIValue(volumeType)

	// Construct filters for EBS volume types
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("volumeType"),
			Value: aws.String(volumeTypeValue),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("location"),
			Value: aws.String(GetRegionDescriptiveName(region)),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("productFamily"),
			Value: aws.String("Storage"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("regionCode"),
			Value: aws.String(region),
		},
	}

	// Get multiple products to find exact match
	pricingProducts, err := GetPricingProducts(ctx, "AmazonEC2", filters, "EBS", volumeType, region)
	if err != nil {
		return 0, err
	}

	// Find exact match for the volume type
	var matchedProduct string
	var matchFound bool

	for _, product := range pricingProducts {
		var priceData map[string]interface{}
		if err := json.Unmarshal([]byte(product), &priceData); err != nil {
			continue
		}

		productAttrs, ok := priceData["product"].(map[string]interface{})
		if !ok {
			continue
		}

		attributes, ok := productAttrs["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check exact volume type (gp2, gp3, etc.)
		if volApiName, ok := attributes["volumeApiName"].(string); ok {
			if volApiName == volumeType {
				matchedProduct = product
				matchFound = true
				break
			}
		}
	}

	if !matchFound {
		return 0, fmt.Errorf("no exact match found for EBS volume type %s in region %s", volumeType, region)
	}

	// Extract price from JSON data
	return extractEBSPrice(matchedProduct)
}

// mapVolumeTypeToAPIValue maps EBS volume types to their API filter values
func mapVolumeTypeToAPIValue(volumeType string) string {
	switch volumeType {
	case "gp2", "gp3":
		return "General Purpose"
	case "io1", "io2":
		return "Provisioned IOPS"
	case "st1":
		return "Throughput Optimized HDD"
	case "sc1":
		return "Cold HDD"
	case "standard":
		return "Magnetic"
	default:
		return "General Purpose" // Default value
	}
}

// extractEBSPrice extracts the price per GB-month from the EBS pricing data
func extractEBSPrice(matchedProduct string) (float64, error) {
	var priceData map[string]interface{}
	if err := json.Unmarshal([]byte(matchedProduct), &priceData); err != nil {
		return 0, fmt.Errorf("error parsing pricing data: %w", err)
	}

	// The structure of the pricing data can be complex and may change
	terms, ok := priceData["terms"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("terms field not found or invalid")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("OnDemand field not found or invalid")
	}

	// Extract the first skuOffer
	var skuOffer map[string]interface{}
	for _, offer := range onDemand {
		skuOffer, ok = offer.(map[string]interface{})
		if ok {
			break
		}
	}

	if skuOffer == nil {
		return 0, fmt.Errorf("no SKU offer found")
	}

	priceDimensions, ok := skuOffer["priceDimensions"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("priceDimensions field not found or invalid")
	}

	// Extract the first price dimension
	var dimension map[string]interface{}
	for _, dim := range priceDimensions {
		dimension, ok = dim.(map[string]interface{})
		if ok {
			break
		}
	}

	if dimension == nil {
		return 0, fmt.Errorf("no price dimension found")
	}

	// Check that this is a per GB-month price
	unit, ok := dimension["unit"].(string)
	if !ok || (unit != "GB-Mo" && unit != "GB-month") {
		return 0, fmt.Errorf("unexpected pricing unit: %s", unit)
	}

	pricePerUnit, ok := dimension["pricePerUnit"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("pricePerUnit field not found or invalid")
	}

	usd, ok := pricePerUnit["USD"].(string)
	if !ok {
		return 0, fmt.Errorf("USD price not found or invalid")
	}

	price, err := strconv.ParseFloat(usd, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing price: %w", err)
	}

	return price, nil
}

// CalculateEBSMonthlyCostWithSource calculates the monthly cost of an EBS volume and returns the pricing source
func CalculateEBSMonthlyCostWithSource(volumeType string, sizeGB int, region string) (float64, string) {
	// Initialize pricing client if not already done
	PricingInitOnce.Do(InitPricingClient)

	// Generate cache key
	cacheKey := fmt.Sprintf("ebs:%s:%s", volumeType, region)

	// Check cache first
	EBSPricingCacheLock.RLock()
	if price, found := EBSPricingCache[cacheKey]; found {
		EBSPricingCacheLock.RUnlock()

		// Update cache hit stats
		UpdateCacheHitStats("EBS", region)

		return float64(sizeGB) * price, string(PricingSourceCache)
	}
	EBSPricingCacheLock.RUnlock()

	// Try to get price from AWS API
	if PricingClient != nil {
		price, err := getEBSPriceFromAPI(volumeType, region)
		if err == nil {
			// Update success stats
			UpdateAPISuccessStats("EBS", region)

			// Cache the result
			EBSPricingCacheLock.Lock()
			EBSPricingCache[cacheKey] = price
			EBSPricingCacheLock.Unlock()

			return float64(sizeGB) * price, string(PricingSourceAPI)
		}

		// Log the error but continue to use fallback pricing
		log.Printf("Error getting EBS price from API: %v for %s in %s.", err, volumeType, region)
	}

	// Update failure stats
	UpdateAPIFailureStats("EBS", region)

	// Use fallback pricing instead of returning N/A
	if regionPrices, found := DefaultEBSPrices[region]; found {
		if typePrice, found := regionPrices[volumeType]; found {
			return float64(sizeGB) * typePrice, string(PricingSourceDefault)
		} else if typePrice, found := regionPrices["gp2"]; found {
			// Default to gp2 price if type not found
			return float64(sizeGB) * typePrice, string(PricingSourceDefault)
		}
	} else if regionPrices, found := DefaultEBSPrices["us-east-1"]; found {
		// If region not found, use us-east-1 prices
		if typePrice, found := regionPrices[volumeType]; found {
			return float64(sizeGB) * typePrice, string(PricingSourceDefault)
		} else if typePrice, found := regionPrices["gp2"]; found {
			return float64(sizeGB) * typePrice, string(PricingSourceDefault)
		}
	}

	// Only return N/A if all fallbacks fail
	return 0, string(PricingSourceNA)
}

// CalculateEBSMonthlyCost is a wrapper around CalculateEBSMonthlyCostWithSource
// that returns only the cost for backward compatibility
func CalculateEBSMonthlyCost(volumeType string, sizeGB int, region string) float64 {
	cost, _ := CalculateEBSMonthlyCostWithSource(volumeType, sizeGB, region)
	return cost
}

// CalculateEBSSavings calculates the estimated savings from an unused EBS volume
func CalculateEBSSavings(volumeType string, sizeGB int, region string, days int) float64 {
	monthlyCost, source := CalculateEBSMonthlyCostWithSource(volumeType, sizeGB, region)

	// If we couldn't get a price, return 0
	if source == string(PricingSourceNA) {
		return 0
	}

	// Simply return monthly cost (ignore elapsed days)
	return monthlyCost
}

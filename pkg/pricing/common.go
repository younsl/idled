package pricing

import (
	"fmt"
	"strconv"

	"github.com/younsl/idled/pkg/utils"
)

// UpdateCacheHitStats updates stats when a cache hit occurs
func UpdateCacheHitStats(service, region string) {
	updatePricingAPIStats(service, region, "cache")
}

// UpdateAPISuccessStats updates stats when an API call succeeds
func UpdateAPISuccessStats(service, region string) {
	updatePricingAPIStats(service, region, "success")
}

// UpdateAPIFailureStats updates stats when an API call fails
func UpdateAPIFailureStats(service, region string) {
	updatePricingAPIStats(service, region, "failure")
}

// updatePricingAPIStats updates the tracking statistics for Pricing API calls
func updatePricingAPIStats(service, region, statType string) {
	PricingAPIStatsLock.Lock()
	defer PricingAPIStatsLock.Unlock()

	// Initialize service map if needed
	if _, exists := PricingAPIStats[service]; !exists {
		PricingAPIStats[service] = make(map[string]map[string]int)
	}

	// Initialize region map if needed
	if _, exists := PricingAPIStats[service][region]; !exists {
		PricingAPIStats[service][region] = map[string]int{
			"success": 0,
			"failure": 0,
			"cache":   0,
		}
	}

	// Increment the appropriate counter
	PricingAPIStats[service][region][statType]++
}

// GetRegionDescriptiveName returns the human-readable region name used in AWS Pricing API
func GetRegionDescriptiveName(region string) string {
	return utils.GetRegionDescriptiveName(region)
}

// ExtractOnDemandPrice extracts the on-demand price from the pricing data JSON
func ExtractOnDemandPrice(priceJSON string) (float64, error) {
	priceData, err := utils.ParseJSON(priceJSON)
	if err != nil {
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
	skuOffer, err := utils.GetFirstMapValue(onDemand)
	if err != nil {
		return 0, fmt.Errorf("no SKU offer found")
	}

	skuOfferMap, ok := skuOffer.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("SKU offer is not a map")
	}

	priceDimensions, ok := skuOfferMap["priceDimensions"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("priceDimensions field not found or invalid")
	}

	// Extract the first price dimension
	dimension, err := utils.GetFirstMapValue(priceDimensions)
	if err != nil {
		return 0, fmt.Errorf("no price dimension found")
	}

	dimensionMap, ok := dimension.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("price dimension is not a map")
	}

	pricePerUnit, ok := dimensionMap["pricePerUnit"].(map[string]interface{})
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

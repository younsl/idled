package pricing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// GetInstanceHourlyPriceWithSource returns the hourly price for an EC2 instance and the source of the pricing
func GetInstanceHourlyPriceWithSource(instanceType, region string) (float64, string) {
	// Initialize pricing client if not already done
	PricingInitOnce.Do(InitPricingClient)

	// Generate cache key
	cacheKey := fmt.Sprintf("%s:%s", region, instanceType)

	// Check cache first
	EC2PricingCacheLock.RLock()
	if price, exists := EC2PricingCache[cacheKey]; exists {
		EC2PricingCacheLock.RUnlock()

		// Update cache hit stats
		UpdateCacheHitStats("EC2", region)

		return price, string(PricingSourceCache)
	}
	EC2PricingCacheLock.RUnlock()

	// Try to get pricing from AWS API only if the client is available
	if PricingClient != nil {
		price, err := getEC2PriceFromAPI(instanceType, region)
		if err == nil {
			// Update success stats
			UpdateAPISuccessStats("EC2", region)

			// Cache the result
			EC2PricingCacheLock.Lock()
			EC2PricingCache[cacheKey] = price
			EC2PricingCacheLock.Unlock()

			return price, string(PricingSourceAPI)
		}

		// Log the error but return N/A
		log.Printf("Error getting price from API: %v for %s in %s.", err, instanceType, region)
	}

	// Update failure stats
	UpdateAPIFailureStats("EC2", region)

	// Return 0 with N/A source, don't use fallback prices
	return 0, string(PricingSourceNA)
}

// GetInstanceHourlyPrice returns the hourly price for an EC2 instance based on its type and region
func GetInstanceHourlyPrice(instanceType, region string) float64 {
	price, _ := GetInstanceHourlyPriceWithSource(instanceType, region)
	return price
}

// getEC2PriceFromAPI retrieves EC2 instance pricing from the AWS Pricing API
func getEC2PriceFromAPI(instanceType, region string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Construct filters for EC2 Linux on-demand instances
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("instanceType"),
			Value: aws.String(instanceType),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("location"),
			Value: aws.String(GetRegionDescriptiveName(region)),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("operatingSystem"),
			Value: aws.String("Linux"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("tenancy"),
			Value: aws.String("Shared"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("preInstalledSw"),
			Value: aws.String("NA"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("capacitystatus"),
			Value: aws.String("Used"),
		},
	}

	// Get pricing data from API
	priceJSON, err := GetPriceFromAPI(ctx, "AmazonEC2", filters, "EC2", instanceType, region)
	if err != nil {
		return 0, err
	}

	// Extract price from JSON data
	return ExtractOnDemandPrice(priceJSON)
}

// CalculateMonthlyCostWithSource returns the estimated monthly cost for an instance and the source of the pricing
func CalculateMonthlyCostWithSource(instanceType, region string) (float64, string) {
	hourlyPrice, source := GetInstanceHourlyPriceWithSource(instanceType, region)

	// If we couldn't get a price, return 0 and N/A
	if source == string(PricingSourceNA) {
		return 0, string(PricingSourceNA)
	}

	// Assuming 730 hours per month (365 days / 12 months * 24 hours)
	return hourlyPrice * 730, source
}

// CalculateMonthlyCost returns the estimated monthly cost for an instance
func CalculateMonthlyCost(instanceType, region string) float64 {
	monthlyCost, _ := CalculateMonthlyCostWithSource(instanceType, region)
	return monthlyCost
}

// CalculateSavingsWithSource returns the estimated savings since the instance was stopped and the source of the pricing
func CalculateSavingsWithSource(instanceType, region string, elapsedDays int) (float64, string) {
	hourlyPrice, source := GetInstanceHourlyPriceWithSource(instanceType, region)

	// If we couldn't get a price, return 0 and N/A
	if source == string(PricingSourceNA) {
		return 0, string(PricingSourceNA)
	}

	// Calculate monthly cost (730 hours = one month)
	monthlyCost := hourlyPrice * 730

	// Calculate savings based on elapsed days (assuming 30 days per month)
	return monthlyCost * float64(elapsedDays) / 30.0, source
}

// CalculateSavings returns the estimated savings since the instance was stopped
func CalculateSavings(instanceType, region string, elapsedDays int) float64 {
	savings, _ := CalculateSavingsWithSource(instanceType, region, elapsedDays)
	return savings
}

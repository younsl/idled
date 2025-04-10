package pricing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"github.com/briandowns/spinner"
)

// AWS pricing client implementation
var (
	// PricingClient is the AWS Pricing API client
	PricingClient *pricing.Client

	// PricingInitOnce ensures the client is initialized only once
	PricingInitOnce sync.Once

	// Spinners for different services
	pricingSpinners = make(map[string]*spinner.Spinner)

	// InitMessage stores the API initialization message to be displayed after spinners
	InitMessage string
)

// InitPricingClient initializes the AWS pricing client
// The AWS Pricing API is only available in us-east-1 and ap-south-1 regions
func InitPricingClient() {
	pricingRegion := "us-east-1" // Pricing API is only available in us-east-1 and ap-south-1
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(pricingRegion))
	if err != nil {
		InitMessage = fmt.Sprintf("Error loading AWS config for pricing API: %v. Using fallback pricing.", err)
		return
	}

	PricingClient = pricing.NewFromConfig(cfg)
	InitMessage = fmt.Sprintf("AWS Pricing API initialized in %s region (https://api.pricing.%s.amazonaws.com)", pricingRegion, pricingRegion)
}

// GetInitMessage returns the initialization message and clears it
func GetInitMessage() string {
	msg := InitMessage
	InitMessage = "" // Clear the message after it's retrieved
	return msg
}

// initPricingSpinner initializes a spinner for a specific service
func initPricingSpinner(service string) *spinner.Spinner {
	if s, exists := pricingSpinners[service]; exists {
		return s
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Retrieving %s pricing information", service)
	s.Color("green")
	pricingSpinners[service] = s

	return s
}

// GetPricingSpinner returns the spinner for a specific service
func GetPricingSpinner(service string) *spinner.Spinner {
	// Initialize spinner if needed
	if _, exists := pricingSpinners[service]; !exists && PricingClient != nil {
		initPricingSpinner(service)
	}

	return pricingSpinners[service]
}

// StartServiceSpinner starts the spinner for a service with appropriate text
func StartServiceSpinner(service, resourceType, region string) {
	spinner := GetPricingSpinner(service)
	if spinner != nil {
		spinner.Suffix = fmt.Sprintf(" %s in %s", resourceType, region)
		spinner.Start()
	}
}

// StopServiceSpinner stops the spinner for a service
func StopServiceSpinner(service string) {
	if spinner, exists := pricingSpinners[service]; exists {
		spinner.Stop()
	}
}

// GetPriceFromAPI is a generic function to get pricing data from AWS API
func GetPriceFromAPI(ctx context.Context, serviceCode string, filters []types.Filter, service, resourceType, region string) (string, error) {
	// Ensure client is initialized
	PricingInitOnce.Do(InitPricingClient)

	if PricingClient == nil {
		return "", fmt.Errorf("AWS pricing client not initialized")
	}

	// Start spinner
	StartServiceSpinner(service, resourceType, GetRegionDescriptiveName(region))
	defer StopServiceSpinner(service)

	// Prepare the API input
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCode),
		Filters:     filters,
		MaxResults:  aws.Int32(1),
	}

	// Call the API
	resp, err := PricingClient.GetProducts(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error calling AWS Pricing API: %w", err)
	}

	if len(resp.PriceList) == 0 {
		return "", fmt.Errorf("no pricing found for %s in region %s", resourceType, region)
	}

	return resp.PriceList[0], nil
}

// GetPricingProducts gets multiple pricing products from AWS API
func GetPricingProducts(ctx context.Context, serviceCode string, filters []types.Filter, service, resourceType, region string) ([]string, error) {
	// Ensure client is initialized
	PricingInitOnce.Do(InitPricingClient)

	if PricingClient == nil {
		return nil, fmt.Errorf("AWS pricing client not initialized")
	}

	// Start spinner
	StartServiceSpinner(service, resourceType, GetRegionDescriptiveName(region))
	defer StopServiceSpinner(service)

	// Prepare the API input
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCode),
		Filters:     filters,
		MaxResults:  aws.Int32(100),
	}

	// Call the API
	resp, err := PricingClient.GetProducts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error calling AWS Pricing API: %w", err)
	}

	if len(resp.PriceList) == 0 {
		return nil, fmt.Errorf("no pricing found for %s in region %s", resourceType, region)
	}

	return resp.PriceList, nil
}

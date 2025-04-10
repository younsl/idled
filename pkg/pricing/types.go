package pricing

import (
	"sync"
)

// PricingSource represents the source of pricing information
type PricingSource string

const (
	// PricingSourceAPI indicates pricing data came from AWS API
	PricingSourceAPI PricingSource = "API"

	// PricingSourceCache indicates pricing data came from cache
	PricingSourceCache PricingSource = "Cache"

	// PricingSourceDefault indicates pricing data came from hardcoded defaults
	PricingSourceDefault PricingSource = "Default"

	// PricingSourceNA indicates pricing data is not available
	PricingSourceNA PricingSource = "N/A"
)

// Stats tracking for pricing API calls
var (
	// PricingAPIStats tracks API call statistics by service and region
	PricingAPIStats = make(map[string]map[string]map[string]int) // service -> region -> {success, failure, cache}

	// PricingAPIStatsLock protects the stats map from concurrent access
	PricingAPIStatsLock sync.RWMutex
)

// EC2 cache
var (
	// EC2PricingCache caches EC2 instance pricing data
	EC2PricingCache = make(map[string]float64)

	// EC2PricingCacheLock protects the EC2 cache from concurrent access
	EC2PricingCacheLock sync.RWMutex
)

// EBS cache
var (
	// EBSPricingCache caches EBS volume pricing data
	EBSPricingCache = make(map[string]float64)

	// EBSPricingCacheLock protects the EBS cache from concurrent access
	EBSPricingCacheLock sync.RWMutex
)

// Default EBS volume prices in USD per GB-month
// These are fallback prices if Pricing API fails
var DefaultEBSPrices = map[string]map[string]float64{
	"us-east-1": { // US East (N. Virginia)
		"gp2":      0.10,
		"gp3":      0.08,
		"io1":      0.125,
		"io2":      0.125,
		"st1":      0.045,
		"sc1":      0.025,
		"standard": 0.05,
	},
	"ap-northeast-2": { // Asia Pacific (Seoul)
		"gp2":      0.114, // Seoul region is about 14% more expensive
		"gp3":      0.092,
		"io1":      0.142,
		"io2":      0.142,
		"st1":      0.051,
		"sc1":      0.029,
		"standard": 0.057,
	},
	// Add more regions as needed
}

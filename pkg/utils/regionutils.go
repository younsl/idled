package utils

// RegionDescriptiveNames maps AWS region codes to descriptive names
var RegionDescriptiveNames = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"af-south-1":     "Africa (Cape Town)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ca-central-1":   "Canada (Central)",
	"eu-central-1":   "EU (Frankfurt)",
	"eu-west-1":      "EU (Ireland)",
	"eu-west-2":      "EU (London)",
	"eu-west-3":      "EU (Paris)",
	"eu-north-1":     "EU (Stockholm)",
	"eu-south-1":     "EU (Milan)",
	"me-south-1":     "Middle East (Bahrain)",
	"sa-east-1":      "South America (Sao Paulo)",
}

// GetRegionDescriptiveName returns the human-readable region name for AWS services
func GetRegionDescriptiveName(region string) string {
	if name, ok := RegionDescriptiveNames[region]; ok {
		return name
	}
	// Default to US East if region not found
	return "US East (N. Virginia)"
}

// IsValidRegion checks if a region is valid
func IsValidRegion(region string) bool {
	_, ok := RegionDescriptiveNames[region]
	return ok
}

// GetDefaultRegion returns the default AWS region
func GetDefaultRegion() string {
	return "us-east-1"
}

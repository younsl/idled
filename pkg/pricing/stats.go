package pricing

// GetAPIStats returns a copy of the current pricing API statistics
func GetAPIStats() map[string]map[string]map[string]int {
	PricingAPIStatsLock.RLock()
	defer PricingAPIStatsLock.RUnlock()

	// Create a deep copy of the stats
	statsCopy := make(map[string]map[string]map[string]int)
	for service, regions := range PricingAPIStats {
		statsCopy[service] = make(map[string]map[string]int)
		for region, stats := range regions {
			statsCopy[service][region] = make(map[string]int)
			for key, value := range stats {
				statsCopy[service][region][key] = value
			}
		}
	}

	return statsCopy
}

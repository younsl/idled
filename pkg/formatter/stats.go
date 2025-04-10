package formatter

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/younsl/idled/pkg/pricing"
)

// PrintPricingAPIStats prints the statistics of pricing API calls
func PrintPricingAPIStats() {
	stats := pricing.GetAPIStats()

	if len(stats) == 0 {
		return
	}

	fmt.Println("\n## AWS Pricing API Call Statistics")

	// Use tabwriter for clean tabular output
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "SERVICE\tREGION\tAPI CALLS\tSUCCESS\tFAILURE\tCACHE HITS\tSUCCESS RATE")

	// Print statistics for each service and region
	for service, regions := range stats {
		for region, statValues := range regions {
			success := statValues["success"]
			failure := statValues["failure"]
			cache := statValues["cache"]
			total := success + failure

			// Calculate success rate percentage
			successRate := 0.0
			if total > 0 {
				successRate = float64(success) / float64(total) * 100.0
			}

			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%.1f%%\n",
				service,
				region,
				total,
				success,
				failure,
				cache,
				successRate,
			)
		}
	}

	w.Flush()
}

package formatter

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// PrintEIPsTable prints a formatted table of unattached Elastic IPs
func PrintEIPsTable(eips []models.EIPInfo, scanTime time.Time, scanDuration time.Duration) {
	if len(eips) == 0 {
		fmt.Println("No unattached Elastic IPs found.")
		return
	}

	// Sort EIPs alphabetically by region
	sort.Slice(eips, func(i, j int) bool {
		if eips[i].Region == eips[j].Region {
			return eips[i].PublicIP < eips[j].PublicIP
		}
		return eips[i].Region < eips[j].Region
	})

	// Set up tabwriter with kubectl style spacing
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "ALLOCATION ID\tPUBLIC IP\tREGION\tSTATUS\tCOST/MO")

	// Print each EIP
	for _, eip := range eips {
		// Format the monthly cost with 2 decimal places
		monthlyCost := fmt.Sprintf("$%.2f", eip.EstimatedMonthlyCost)

		// Print row
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			eip.AllocationID,
			eip.PublicIP,
			eip.Region,
			eip.AssociationState,
			monthlyCost,
		)
	}

	// Print totals
	printEIPTotals(w, eips)

	w.Flush()
}

// printEIPTotals prints the summary information at the bottom of the table
func printEIPTotals(w *tabwriter.Writer, eips []models.EIPInfo) {
	totalEIPs := len(eips)

	// Calculate total potential monthly cost
	var totalMonthlyCost float64

	for _, eip := range eips {
		totalMonthlyCost += eip.EstimatedMonthlyCost
	}

	// Format totals with 2 decimal places
	formattedMonthlyCost := fmt.Sprintf("$%.2f", totalMonthlyCost)

	// Print summary with kubernetes style alignment
	fmt.Fprintf(w, "Total:\t\t\t\t%s (%d EIPs)\n",
		formattedMonthlyCost,
		totalEIPs,
	)
}

// PrintEIPsSummary displays summary information about unattached Elastic IPs
func PrintEIPsSummary(eips []models.EIPInfo) {
	if len(eips) == 0 {
		return
	}

	// Group by region
	regionCounts := make(map[string]int)

	for _, eip := range eips {
		regionCounts[eip.Region]++
	}

	// Prepare sorted list of regions
	var regions []string
	for region := range regionCounts {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	fmt.Println("\n## Unattached Elastic IPs by Region")

	// Set up tabwriter with kubectl style spacing
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "REGION\tCOUNT")

	// Print rows for each region
	for _, region := range regions {
		fmt.Fprintf(w, "%s\t%d\n", region, regionCounts[region])
	}

	w.Flush()
}

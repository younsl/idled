package formatter

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// PrintInstancesTable prints a formatted table of EC2 instances
func PrintInstancesTable(instances []models.InstanceInfo, scanTime time.Time, scanDuration time.Duration) {
	if len(instances) == 0 {
		fmt.Println("No idle instances found.")
		return
	}

	// Sort instances by elapsed days (longest first)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].ElapsedDays > instances[j].ElapsedDays
	})

	// kubectl 스타일 tabwriter 설정
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print scan timestamp first
	fmt.Fprintf(w, "Scan time: %s (completed in %.2f seconds)\n",
		scanTime.Format("2006-01-02 15:04:05"),
		scanDuration.Seconds())

	// Print header
	fmt.Fprintln(w, "INSTANCE ID\tNAME\tTYPE\tREGION\tSTOPPED SINCE\tDAYS\tCOST/MO\tTOTAL SAVED\tPRICING")

	// Print each instance
	for _, instance := range instances {
		// Format the stopped time
		stoppedTimeStr := ""
		if instance.StoppedTime != nil {
			stoppedTimeStr = instance.StoppedTime.Format("2006-01-02")
		} else {
			stoppedTimeStr = "Unknown"
		}

		// Format the monthly cost and savings with 2 decimal places
		var monthlyCost, savings string
		if instance.PricingSource == "N/A" {
			monthlyCost = "N/A"
			savings = "N/A"
		} else {
			monthlyCost = fmt.Sprintf("$%.2f", instance.EstimatedMonthlyCost)
			savings = fmt.Sprintf("$%.2f", instance.EstimatedSavings)
		}

		// Get pricing source marker
		pricingMarker := GetPricingMarker(instance.PricingSource)

		// Print row
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			instance.InstanceID,
			getInstanceName(instance.Name),
			instance.InstanceType,
			instance.Region,
			stoppedTimeStr,
			instance.ElapsedDays,
			monthlyCost,
			savings,
			pricingMarker,
		)
	}

	// Print totals without separator
	printTotals(w, instances)

	w.Flush()
}

// getInstanceName returns a formatted instance name or <unnamed> if empty
func getInstanceName(name string) string {
	if name == "" {
		return "<unnamed>"
	}
	return name
}

// printTotals prints the summary information at the bottom of the table
func printTotals(w *tabwriter.Writer, instances []models.InstanceInfo) {
	totalInstances := len(instances)

	// Calculate total potential monthly cost and actual savings
	var totalMonthlyCost float64
	var totalSavings float64

	for _, instance := range instances {
		totalMonthlyCost += instance.EstimatedMonthlyCost
		totalSavings += instance.EstimatedSavings
	}

	// Format totals with 2 decimal places
	formattedMonthlyCost := fmt.Sprintf("$%.2f", totalMonthlyCost)
	formattedSavings := fmt.Sprintf("$%.2f", totalSavings)

	// Print summary with kubernetes style alignment
	fmt.Fprintf(w, "Total:\t\t\t\t\t%d\t%s\t%s\n",
		totalInstances,
		formattedMonthlyCost,
		formattedSavings,
	)
}

// PrintInstancesSummary displays summary information about instances
func PrintInstancesSummary(instances []models.InstanceInfo) {
	if len(instances) == 0 {
		return
	}

	// Classify by days stopped
	dayRanges := map[string]int{
		"1 day or less": 0,
		"2-7 days":      0,
		"8-30 days":     0,
		"31-90 days":    0,
		"Over 90 days":  0,
		"Unknown":       0,
	}

	for _, instance := range instances {
		if instance.StoppedTime == nil {
			dayRanges["Unknown"]++
			continue
		}

		days := instance.ElapsedDays
		switch {
		case days <= 1:
			dayRanges["1 day or less"]++
		case days <= 7:
			dayRanges["2-7 days"]++
		case days <= 30:
			dayRanges["8-30 days"]++
		case days <= 90:
			dayRanges["31-90 days"]++
		default:
			dayRanges["Over 90 days"]++
		}
	}

	fmt.Println("\n## Stopped EC2 Instances Summary")

	// 요약 정보 출력을 kubectl 스타일로 설정
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "PERIOD STOPPED\tINSTANCE COUNT")

	// Keys array for ordered output
	keys := []string{"1 day or less", "2-7 days", "8-30 days", "31-90 days", "Over 90 days", "Unknown"}

	// Print rows
	for _, key := range keys {
		fmt.Fprintf(w, "%s\t%d\n", key, dayRanges[key])
	}

	w.Flush()
}

// GetPricingMarker returns a suitable marker for the pricing source
func GetPricingMarker(source string) string {
	switch source {
	case "API":
		return "API"
	case "Cache":
		return "CACHE"
	case "N/A":
		return "N/A"
	default:
		return "-"
	}
}

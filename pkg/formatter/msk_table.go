package formatter

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	// "github.com/jedib0t/go-pretty/v6/table"
	"github.com/younsl/idled/internal/models"
	// "github.com/younsl/idled/pkg/pricing" // Not needed for table, maybe for summary?
)

// PrintMskTable prints the MSK cluster information in a table format using tabwriter.
func PrintMskTable(clusters []models.MskClusterInfo, scanStartTime time.Time, scanDuration time.Duration) {
	if len(clusters) == 0 {
		// fmt.Println("\nNo idle/underutilized MSK clusters found.") // Spinner handles this
		return
	}

	// Sort clusters (Idle first, then by Creation Time ascending)
	sort.SliceStable(clusters, func(i, j int) bool {
		if clusters[i].IsIdle != clusters[j].IsIdle {
			return clusters[i].IsIdle // true comes before false
		}
		// If Idle status is the same, sort by CreationTime ascending (older first)
		return clusters[i].CreationTime.Before(clusters[j].CreationTime)
	})

	// Setup tabwriter for kubernetes style tables
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header - remove Idle Days, add Instance Type
	fmt.Fprintln(w, "CLUSTER NAME\tARN\tREGION\tSTATE\tINSTANCE TYPE\tCREATION TIME\tIS IDLE\tREASON\tMAX CONN (30d)\tAVG CPU (30d %)")

	// Print table rows
	for _, cluster := range clusters {
		connCountStr := "N/A"
		if cluster.ConnectionCount != nil {
			connCountStr = fmt.Sprintf("%.0f", *cluster.ConnectionCount)
		}
		cpuUtilStr := "N/A"
		if cluster.AvgCPUUtilization != nil {
			cpuUtilStr = fmt.Sprintf("%.2f", *cluster.AvgCPUUtilization)
		}

		// Truncate ARN if necessary (using the function from this package)
		truncatedARN := truncateString(cluster.ARN, 50)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
			cluster.ClusterName,
			truncatedARN,
			cluster.Region,
			cluster.State,
			cluster.InstanceType, // Add Instance Type
			cluster.CreationTime.Format("2006-01-02"),
			cluster.IsIdle,
			cluster.Reason,
			connCountStr,
			cpuUtilStr,
		)
	}

	// Update footer to show total scanned and idle/underutilized count
	idleCount := 0
	for _, cluster := range clusters {
		if cluster.IsIdle {
			idleCount++
		}
	}
	footerStr := fmt.Sprintf("Showing %d scanned MSK clusters (%d Idle/Underutilized)", len(clusters), idleCount)
	// Need to add caption/footer logic to tabwriter if not standard footer
	// Tabwriter doesn't have a direct SetCaption like go-pretty.
	// We'll just print the summary line after flushing the table.
	w.Flush()
	fmt.Printf("\n%s\n", footerStr) // Print summary line after table
}

// PrintMskSummary prints the summary for MSK clusters using tabwriter.
func PrintMskSummary(clusters []models.MskClusterInfo) {
	// Count clusters by Reason (only those marked as idle/underutilized)
	reasonCounts := make(map[string]int)
	totalIdleCount := 0
	for _, cluster := range clusters {
		if cluster.IsIdle { // Consider only clusters identified as idle/underutilized
			reasonCounts[cluster.Reason]++
			totalIdleCount++
		}
	}

	if totalIdleCount == 0 {
		return // No summary needed if no idle/underutilized clusters found
	}

	// Setup tabwriter for summary
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	fmt.Fprintln(w, "\n## MSK SUMMARY:") // Consistent summary title
	fmt.Fprintln(w, "REASON\tCOUNT")

	// Sort reasons for consistent output
	reasons := make([]string, 0, len(reasonCounts))
	for reason := range reasonCounts {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)

	// Print counts per reason
	for _, reason := range reasons {
		count := reasonCounts[reason]
		fmt.Fprintf(w, "%s\t%d\n", reason, count)
	}

	// Print total count
	fmt.Fprintf(w, "Total Idle/Underutilized:\t%d\n", totalIdleCount)

	w.Flush()
}

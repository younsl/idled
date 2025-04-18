package formatter

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// PrintECRTable formats and prints ECR repository information in a table, mimicking EC2 style.
func PrintECRTable(repos []models.RepositoryInfo, _ time.Time, _ time.Duration) { // scanStartTime, scanDuration removed as spinner handles it
	if len(repos) == 0 {
		// Message handled by spinner
		return
	}

	// Sort by last push time (oldest first, nil/never last)
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].LastPush == nil && repos[j].LastPush == nil {
			return repos[i].Name < repos[j].Name // Secondary sort by name if both never pushed
		}
		if repos[i].LastPush == nil {
			return false // Never pushed comes after pushed
		}
		if repos[j].LastPush == nil {
			return true // Pushed comes before never pushed
		}
		return repos[i].LastPush.Before(*repos[j].LastPush)
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0) // Use tabwriter like EC2

	// Print header, matching EC2 style, with TOTAL IMAGE
	fmt.Fprintln(w, "NAME\tREGION\tLAST PUSH\tTOTAL IMAGE\tIDLE")

	for _, repo := range repos {
		lastPushStr := "Never"
		if repo.LastPush != nil {
			lastPushStr = utils.FormatTimeAgo(*repo.LastPush) // Use the shortened format
		}
		idleStr := fmt.Sprintf("%t", repo.Idle)

		// Print row using tabwriter, including image count
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			repo.Name,
			repo.Region,
			lastPushStr,
			repo.ImageCount, // Add image count here
			idleStr,
		)
	}

	w.Flush()
}

// PrintECRSummary prints a simple summary of total and idle repositories.
func PrintECRSummary(repos []models.RepositoryInfo) {
	if len(repos) == 0 {
		return // No summary needed if no repos found
	}
	idleCount := 0
	for _, repo := range repos {
		if repo.Idle {
			idleCount++
		}
	}
	fmt.Printf("\nECR Summary: %d total repositories found, %d identified as idle.\n", len(repos), idleCount)
}

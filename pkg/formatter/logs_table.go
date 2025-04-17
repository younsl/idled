package formatter

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// PrintLogGroupsTable prints the found idle log groups using tabwriter for consistency.
func PrintLogGroupsTable(logGroups []models.LogGroupInfo) {
	if len(logGroups) == 0 {
		// No need to print anything if the list is empty,
		// the calling function already prints a summary message.
		return
	}

	// Sort by effective timestamp (actual last event or creation time)
	sort.SliceStable(logGroups, func(i, j int) bool {
		if logGroups[i].LastEventMillis == 0 {
			return false
		} // Put groups with unknown time at the end
		if logGroups[j].LastEventMillis == 0 {
			return true
		}
		return logGroups[i].LastEventMillis < logGroups[j].LastEventMillis
	})

	fmt.Println("\nIdle CloudWatch Log Groups:")

	// Use tabwriter, same settings as EC2/EBS formatter
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header with tabs
	fmt.Fprintln(w, "LOG GROUP NAME\tRETENTION\tSIZE\tCREATED\tLAST EVENT")

	// Print rows with tabs
	for _, lg := range logGroups {
		// Format CreationTime (short date)
		creationTimeStr := lg.CreationTime.Format("2006-01-02")
		if lg.CreationTime.IsZero() {
			creationTimeStr = "N/A"
		}

		// Format LastEventTime (short date or fallback string)
		lastEventTimeStr := lg.LastEventTime // This already contains fallback like "N/A (Created...)"
		if !strings.HasPrefix(lastEventTimeStr, "N/A") {
			// Try to parse the full timestamp and format as short date
			parsedTime, err := time.Parse("2006-01-02 15:04:05", lastEventTimeStr)
			if err == nil {
				lastEventTimeStr = parsedTime.Format("2006-01-02")
			} // Keep original if parsing fails
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			lg.Name,
			lg.RetentionDays,
			lg.StoredBytes,
			creationTimeStr,
			lastEventTimeStr,
		)
	}

	// Flush the writer to ensure output is displayed
	w.Flush()
}

package formatter

import (
	"fmt"
	"time"
)

// printTimestamp prints the scan timestamp and duration
// NOTE: This function is deprecated and kept for reference only.
// All formatters now use tabwriter for consistent output.
func printTimestamp(scanStartTime time.Time, scanDuration time.Duration) {
	// Format the scan time
	timeStr := scanStartTime.Format("2006-01-02 15:04:05")

	// Format the duration
	durationStr := fmt.Sprintf("%.2fs", scanDuration.Seconds())

	fmt.Printf("Scan completed at %s (took %s)\n", timeStr, durationStr)
}

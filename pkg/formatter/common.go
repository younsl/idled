package formatter

import (
	"fmt"
	"time"
)

// printTimestamp prints the scan timestamp and duration
func printTimestamp(scanStartTime time.Time, scanDuration time.Duration) {
	// Format the scan time
	timeStr := scanStartTime.Format("2006-01-02 15:04:05")

	// Format the duration
	durationStr := fmt.Sprintf("%.2fs", scanDuration.Seconds())

	fmt.Printf("Scan completed at %s (took %s)\n", timeStr, durationStr)
}

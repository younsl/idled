package models

import "time"

// LogGroupInfo holds information about a CloudWatch Log Group relevant for idle checking.
type LogGroupInfo struct {
	Name            string
	RetentionDays   string
	StoredBytes     string // Formatted string (e.g., using humanize)
	LastEventTime   string // Formatted string (actual last event or fallback)
	ARN             string
	CreationTime    time.Time // Original creation time
	LastEventMillis int64     // Timestamp for sorting (actual or creation)
}

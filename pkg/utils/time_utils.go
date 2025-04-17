package utils

import (
	"strings"
	"time"
)

// ParseStateTransitionTime extracts a time from EC2 state transition reason
// Example format: "User initiated (2023-04-01 12:34:56 GMT)"
func ParseStateTransitionTime(reason string) *time.Time {
	if len(reason) == 0 {
		return nil
	}

	// Simple approach: assume "User initiated (YYYY-MM-DD HH:MM:SS GMT)" format
	// A more sophisticated regex approach might be needed in practice
	parts := strings.Split(reason, "(")
	if len(parts) < 2 {
		return nil
	}

	dateStr := strings.TrimSuffix(parts[1], ")")
	dateStr = strings.TrimSpace(dateStr)

	t, err := time.Parse("2006-01-02 15:04:05 MST", dateStr)
	if err != nil {
		return nil
	}

	return &t
}

// CalculateElapsedDays calculates the number of days elapsed since a given time
func CalculateElapsedDays(since time.Time) int {
	return int(time.Since(since).Hours() / 24)
}

// GetMonthlyHours returns the number of hours in a month (approximation)
func GetMonthlyHours() float64 {
	return 730.0 // 365 days / 12 months * 24 hours
}

// DaysToMonthRatio converts days to a month ratio
func DaysToMonthRatio(days int) float64 {
	return float64(days) / 30.0
}

// FormatDuration formats a time.Duration in a human readable format
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return time.Unix(int64(days*24*60*60), 0).Format("2006-01-02")
	} else if hours > 0 {
		return time.Unix(int64(hours*60*60), 0).Format("15:04:05")
	} else {
		return time.Unix(int64(minutes*60), 0).Format("04:05")
	}
}

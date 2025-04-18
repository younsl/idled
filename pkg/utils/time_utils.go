package utils

import (
	"fmt"
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

// FormatTimeAgo formats a time.Time into a human-readable "time ago" string.
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	days := int(diff.Hours() / 24)
	hours := int(diff.Hours())
	minutes := int(diff.Minutes())
	seconds := int(diff.Seconds())

	const ( // Constants for calculation
		daysInYear  = 365
		daysInMonth = 30
	)

	if days >= daysInYear { // More than or equal to a year
		years := days / daysInYear
		remainingDaysAfterYears := days % daysInYear
		months := remainingDaysAfterYears / daysInMonth
		remainingDays := remainingDaysAfterYears % daysInMonth
		result := fmt.Sprintf("%dy", years)
		if months > 0 {
			result += fmt.Sprintf("%dm", months)
		}
		if remainingDays > 0 {
			result += fmt.Sprintf("%dd", remainingDays)
		}
		return result + " ago"
	} else if days >= daysInMonth { // More than or equal to a month but less than a year
		months := days / daysInMonth
		remainingDays := days % daysInMonth
		result := fmt.Sprintf("%dm", months)
		if remainingDays > 0 {
			result += fmt.Sprintf("%dd", remainingDays)
		}
		return result + " ago"
	} else if days > 1 {
		return fmt.Sprintf("%dd ago", days)
	} else if days == 1 {
		return "1d ago"
	} else if hours > 1 {
		return fmt.Sprintf("%dh ago", hours)
	} else if hours == 1 {
		return "1h ago"
	} else if minutes > 1 {
		return fmt.Sprintf("%dm ago", minutes)
	} else if minutes == 1 {
		return "1m ago"
	} else if seconds > 1 {
		return fmt.Sprintf("%ds ago", seconds)
	} else if seconds == 1 {
		return "1s ago"
	}
	return "now"
}

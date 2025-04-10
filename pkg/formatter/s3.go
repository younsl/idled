package formatter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// PrintBucketsTable prints S3 bucket information as a table
func PrintBucketsTable(buckets []models.BucketInfo, scanStartTime time.Time, scanDuration time.Duration) {
	if len(buckets) == 0 {
		fmt.Println("\nNo idle S3 buckets found")
		return
	}

	// Sort buckets by idle days (descending)
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].IdleDays > buckets[j].IdleDays
	})

	printTimestamp(scanStartTime, scanDuration)

	// Print table header
	fmt.Println("\nIDLE S3 BUCKETS:")
	fmt.Println(strings.Repeat("-", 130))
	fmt.Printf("%-32s %-15s %-11s %-15s %-12s %-15s %-9s %s\n",
		"BUCKET NAME", "REGION", "OBJECTS", "SIZE", "IDLE DAYS", "LAST MODIFIED", "EMPTY", "USAGE")
	fmt.Println(strings.Repeat("-", 130))

	// Print table rows
	for _, bucket := range buckets {
		var lastModified string
		if bucket.LastModified != nil {
			lastModified = bucket.LastModified.Format("2006-01-02")
		} else {
			lastModified = "N/A"
		}

		usage := formatBucketUsage(bucket)

		// Format size
		sizeFormatted := utils.FormatBytes(bucket.TotalSize)

		fmt.Printf("%-32s %-15s %-11d %-15s %-12d %-15s %-9t %s\n",
			bucket.BucketName,
			bucket.Region,
			bucket.ObjectCount,
			sizeFormatted,
			bucket.IdleDays,
			lastModified,
			bucket.IsEmpty,
			usage)
	}
	fmt.Println(strings.Repeat("-", 130))
}

// formatBucketUsage returns a human-readable description of bucket usage
func formatBucketUsage(bucket models.BucketInfo) string {
	var usage []string

	// Add "Recently Modified" tag if modified within last 30 days
	if bucket.LastModified != nil && utils.CalculateElapsedDays(*bucket.LastModified) <= 30 {
		usage = append(usage, "Recently Modified")
	}

	// Check if bucket is used for website hosting
	if bucket.HasWebsiteConfig {
		usage = append(usage, "Website")
	}

	// Check if bucket has policies (may indicate special usage)
	if bucket.HasBucketPolicy {
		usage = append(usage, "Policy")
	}

	// Check if bucket has event notifications
	if bucket.HasEventNotification {
		usage = append(usage, "Events")
	}

	// Check for API activity pattern
	if bucket.GetRequestsLast30Days > 1000 && bucket.PutRequestsLast30Days < 10 {
		usage = append(usage, "Static Content")
	} else if bucket.GetRequestsLast30Days > 0 || bucket.PutRequestsLast30Days > 0 {
		usage = append(usage, fmt.Sprintf("API: %d Get, %d Put",
			bucket.GetRequestsLast30Days, bucket.PutRequestsLast30Days))
	}

	if len(usage) == 0 {
		return "No detected usage"
	}

	return strings.Join(usage, ", ")
}

// PrintBucketsSummary prints a summary of idle S3 buckets
func PrintBucketsSummary(buckets []models.BucketInfo) {
	if len(buckets) == 0 {
		return
	}

	var emptyBuckets, idleBuckets, bucketsByAge []models.BucketInfo

	// Categorize buckets
	for _, bucket := range buckets {
		if bucket.IsEmpty {
			emptyBuckets = append(emptyBuckets, bucket)
		}
		if bucket.IsIdle {
			idleBuckets = append(idleBuckets, bucket)
			bucketsByAge = append(bucketsByAge, bucket)
		}
	}

	// Sort buckets by idle time
	sort.Slice(bucketsByAge, func(i, j int) bool {
		return bucketsByAge[i].IdleDays > bucketsByAge[j].IdleDays
	})

	// Calculate total size of IDLE buckets only
	var totalIdleSize int64
	for _, bucket := range idleBuckets {
		totalIdleSize += bucket.TotalSize
	}

	// Print summary
	fmt.Println("\nSUMMARY:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Total S3 buckets scanned: %d\n", len(buckets))
	fmt.Printf("  Empty buckets: %d\n", len(emptyBuckets))
	fmt.Printf("  Idle buckets: %d\n", len(idleBuckets))
	fmt.Printf("Total idle storage: %s\n", utils.FormatBytes(totalIdleSize))
	fmt.Println(strings.Repeat("-", 80))

	// Print additional recommendations for buckets by age category
	printBucketsAgeBreakdown(bucketsByAge)
}

// printBucketsAgeBreakdown prints breakdown of buckets by age categories
func printBucketsAgeBreakdown(buckets []models.BucketInfo) {
	if len(buckets) == 0 {
		return
	}

	var (
		b30Days, b90Days, b180Days, b365Days, bOlder int
	)

	for _, bucket := range buckets {
		switch {
		case bucket.IdleDays <= 30:
			b30Days++
		case bucket.IdleDays <= 90:
			b90Days++
		case bucket.IdleDays <= 180:
			b180Days++
		case bucket.IdleDays <= 365:
			b365Days++
		default:
			bOlder++
		}
	}

	fmt.Println("\nAGE BREAKDOWN:")
	fmt.Printf("  ≤ 30 days: %d buckets\n", b30Days)
	fmt.Printf("  31-90 days: %d buckets\n", b90Days)
	fmt.Printf("  91-180 days: %d buckets\n", b180Days)
	fmt.Printf("  181-365 days: %d buckets\n", b365Days)
	fmt.Printf("  > 365 days: %d buckets\n", bOlder)

	// Suggest lifecycle policy if older buckets exist
	if b180Days+b365Days+bOlder > 0 {
		fmt.Println("\nRECOMMENDATIONS:")
		fmt.Println("  - Consider implementing lifecycle policies for long-term idle buckets")
		fmt.Println("  - Review empty buckets for potential cleanup")
	}
}

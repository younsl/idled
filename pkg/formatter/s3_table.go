package formatter

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// PrintBucketsTable prints S3 bucket information as a table
func PrintBucketsTable(buckets []models.BucketInfo, scanStartTime time.Time, scanDuration time.Duration) {
	if len(buckets) == 0 {
		fmt.Println("No idle S3 buckets found.")
		return
	}

	// Sort buckets by idle days (descending)
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].IdleDays > buckets[j].IdleDays
	})

	// Setup tabwriter for kubernetes style tables
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "NAME\tREGION\tOBJECTS\tSIZE\tIDLE DAYS\tLAST MODIFIED\tEMPTY\tUSAGE")

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

		// Format empty value as a string instead of boolean
		emptyStr := "Yes"
		if !bucket.IsEmpty {
			emptyStr = "No"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t%s\t%s\t%s\n",
			bucket.BucketName,
			bucket.Region,
			bucket.ObjectCount,
			sizeFormatted,
			bucket.IdleDays,
			lastModified,
			emptyStr,
			usage)
	}

	// Print totals
	printBucketsTotals(w, buckets)

	w.Flush()
}

// printBucketsTotals prints the summary information at the bottom of the table
func printBucketsTotals(w *tabwriter.Writer, buckets []models.BucketInfo) {
	var totalObjects int64
	var totalSize int64

	for _, bucket := range buckets {
		totalObjects += int64(bucket.ObjectCount)
		totalSize += bucket.TotalSize
	}

	sizeFormatted := utils.FormatBytes(totalSize)

	// Print summary with kubernetes style alignment
	fmt.Fprintf(w, "Total:\t\t%d\t%s\t\t\t\n",
		totalObjects,
		sizeFormatted,
	)
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

	// Setup tabwriter for kubernetes style tables
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	fmt.Fprintln(w, "\n## S3 BUCKETS SUMMARY:")
	fmt.Fprintf(w, "Total buckets scanned:\t%d\n", len(buckets))
	fmt.Fprintf(w, "Empty buckets:\t%d\n", len(emptyBuckets))
	fmt.Fprintf(w, "Idle buckets:\t%d\n", len(idleBuckets))
	fmt.Fprintf(w, "Total idle storage:\t%s\n", utils.FormatBytes(totalIdleSize))

	w.Flush()

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

	// Setup tabwriter for kubernetes style tables
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	fmt.Fprintln(w, "\n## AGE BREAKDOWN:")
	fmt.Fprintf(w, "≤ 30 days:\t%d buckets\n", b30Days)
	fmt.Fprintf(w, "31-90 days:\t%d buckets\n", b90Days)
	fmt.Fprintf(w, "91-180 days:\t%d buckets\n", b180Days)
	fmt.Fprintf(w, "181-365 days:\t%d buckets\n", b365Days)
	fmt.Fprintf(w, "> 365 days:\t%d buckets\n", bOlder)

	// Suggest lifecycle policy if older buckets exist
	if b180Days+b365Days+bOlder > 0 {
		fmt.Fprintln(w, "\n## RECOMMENDATIONS:")
		fmt.Fprintln(w, "- Consider implementing lifecycle policies for long-term idle buckets")
		fmt.Fprintln(w, "- Review empty buckets for potential cleanup")
	}

	w.Flush()
}

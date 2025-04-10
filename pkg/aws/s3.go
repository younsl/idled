package aws

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/briandowns/spinner"
	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// S3Client struct for S3 client
type S3Client struct {
	client        *s3.Client
	cwClient      *cloudwatch.Client
	region        string
	idleThreshold int // in days
}

// NewS3Client creates a new S3Client
func NewS3Client(region string) (*S3Client, error) {
	// Use LoadDefaultConfig with explicit options
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithRetryMode(aws.RetryModeStandard),
		config.WithEC2IMDSClientEnableState(imds.ClientEnabled),
	)
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	// Initialize S3 client with explicit config
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Use path-style addressing which is more reliable
	})

	// Initialize CloudWatch client
	cwClient := cloudwatch.NewFromConfig(cfg)

	return &S3Client{
		client:        s3Client,
		cwClient:      cwClient,
		region:        region,
		idleThreshold: 30, // Default: consider buckets idle after 30 days of inactivity
	}, nil
}

// SetIdleThreshold sets the threshold in days for considering a bucket as idle
func (c *S3Client) SetIdleThreshold(days int) {
	c.idleThreshold = days
}

// GetIdleBuckets returns a list of S3 buckets with idle detection metrics
func (c *S3Client) GetIdleBuckets() ([]models.BucketInfo, error) {
	// Create and start a spinner for visual feedback
	sp := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Suffix = fmt.Sprintf(" Scanning S3 buckets in %s...", c.region)
	sp.Start()
	defer sp.Stop()

	// List all buckets
	result, err := c.client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("error listing S3 buckets: %w", err)
	}

	sp.Suffix = fmt.Sprintf(" Found %d total buckets, filtering for region %s...", len(result.Buckets), c.region)

	var bucketInfos []models.BucketInfo
	var regionBuckets []string // Store bucket names instead of bucket objects

	// First filter buckets by region (this is faster)
	for _, bucket := range result.Buckets {
		// Skip buckets from other regions
		location, err := c.getBucketRegion(*bucket.Name)
		if err != nil {
			// Skip buckets we can't access
			continue
		}

		// Skip buckets from other regions
		if location != c.region {
			continue
		}

		// Store just the bucket name
		regionBuckets = append(regionBuckets, *bucket.Name)
	}

	totalBuckets := len(regionBuckets)
	if totalBuckets == 0 {
		return bucketInfos, nil
	}

	// Process each bucket
	for i, bucketName := range regionBuckets {
		sp.Suffix = fmt.Sprintf(" Analyzing bucket %d/%d in %s: %s",
			i+1, totalBuckets, c.region, bucketName)

		// Find the matching bucket object to get creation date
		var creationDate time.Time
		for _, b := range result.Buckets {
			if *b.Name == bucketName {
				creationDate = *b.CreationDate
				break
			}
		}

		// Get basic bucket info
		bucketInfo, err := c.analyzeBucket(bucketName, creationDate)
		if err != nil {
			// Log error and continue with next bucket
			continue
		}

		bucketInfos = append(bucketInfos, bucketInfo)
	}

	return bucketInfos, nil
}

// getBucketRegion determines the region for a bucket
func (c *S3Client) getBucketRegion(bucketName string) (string, error) {
	ctx := context.TODO()
	location, err := c.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "", err
	}

	// Convert the location constraint to a region string
	// If LocationConstraint is empty, it's in us-east-1
	region := "us-east-1"
	if location.LocationConstraint != "" {
		region = string(location.LocationConstraint)
	}

	return region, nil
}

// analyzeBucket gathers information and analytics for a single bucket
func (c *S3Client) analyzeBucket(bucketName string, creationDate time.Time) (models.BucketInfo, error) {
	ctx := context.TODO()

	bucketInfo := models.BucketInfo{
		BucketName:   bucketName,
		Region:       c.region,
		CreationTime: creationDate,
	}

	// Check if bucket exists and is accessible
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return bucketInfo, fmt.Errorf("bucket not accessible: %w", err)
	}

	// Get object count and total size
	objCount, totalSize, lastModified, err := c.getBucketStats(bucketName)
	if err != nil {
		return bucketInfo, fmt.Errorf("error getting bucket stats: %w", err)
	}

	bucketInfo.ObjectCount = objCount
	bucketInfo.TotalSize = totalSize
	bucketInfo.LastModified = lastModified
	bucketInfo.IsEmpty = (objCount == 0)

	// Get CloudWatch metrics for API calls
	getRequests, putRequests, err := c.getBucketAPIActivity(bucketName)
	if err != nil {
		// Just log the error and continue - this is non-critical
		fmt.Printf("Warning: Could not retrieve CloudWatch metrics for bucket %s: %v\n", bucketName, err)
	} else {
		bucketInfo.GetRequestsLast30Days = getRequests
		bucketInfo.PutRequestsLast30Days = putRequests
	}

	// Check for website configuration
	hasWebsiteConfig, err := c.hasBucketWebsiteConfig(bucketName)
	if err == nil {
		bucketInfo.HasWebsiteConfig = hasWebsiteConfig
	}

	// Check for bucket policy
	hasBucketPolicy, err := c.hasBucketPolicy(bucketName)
	if err == nil {
		bucketInfo.HasBucketPolicy = hasBucketPolicy
	}

	// Check for event notifications
	hasNotification, err := c.hasBucketNotification(bucketName)
	if err == nil {
		bucketInfo.HasEventNotification = hasNotification
	}

	// Determine if bucket is idle
	bucketInfo.IsIdle = c.determineBucketIdleStatus(&bucketInfo)
	if bucketInfo.IsIdle && bucketInfo.LastModified != nil {
		bucketInfo.IdleDays = utils.CalculateElapsedDays(*bucketInfo.LastModified)
	}

	return bucketInfo, nil
}

// getBucketStats gets statistics about the bucket
func (c *S3Client) getBucketStats(bucketName string) (int64, int64, *time.Time, error) {
	// Use CloudWatch metrics instead of listing all objects
	ctx := context.TODO()
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30) // Last 30 days

	// Get bucket size from CloudWatch metrics
	sizeInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/S3"),
		MetricName: aws.String("BucketSizeBytes"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucketName),
			},
			{
				Name:  aws.String("StorageType"),
				Value: aws.String("StandardStorage"),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticAverage},
	}

	sizeResult, err := c.cwClient.GetMetricStatistics(ctx, sizeInput)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("error getting bucket size metrics: %w", err)
	}

	// Get object count from CloudWatch metrics
	countInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/S3"),
		MetricName: aws.String("NumberOfObjects"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucketName),
			},
			{
				Name:  aws.String("StorageType"),
				Value: aws.String("AllStorageTypes"),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticAverage},
	}

	countResult, err := c.cwClient.GetMetricStatistics(ctx, countInput)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("error getting object count metrics: %w", err)
	}

	// Initialize with default values
	var totalSize int64
	var objectCount int64
	var lastModified *time.Time

	// Process size metric results - get the most recent data point
	if len(sizeResult.Datapoints) > 0 {
		// Sort datapoints by timestamp (descending)
		sort.Slice(sizeResult.Datapoints, func(i, j int) bool {
			return sizeResult.Datapoints[i].Timestamp.After(*sizeResult.Datapoints[j].Timestamp)
		})

		// Use the most recent datapoint
		if sizeResult.Datapoints[0].Average != nil {
			totalSize = int64(*sizeResult.Datapoints[0].Average)
		}

		// Try to find when the bucket size last changed significantly
		lastChanged := findLastMetricChange(sizeResult.Datapoints)
		if lastChanged != nil && (lastModified == nil || lastChanged.Before(*lastModified)) {
			if !lastChanged.After(time.Now()) { // Ensure we don't use future dates
				lastModified = lastChanged
			}
		}
	}

	// Process object count metric results
	if len(countResult.Datapoints) > 0 {
		// Sort datapoints by timestamp (descending)
		sort.Slice(countResult.Datapoints, func(i, j int) bool {
			return countResult.Datapoints[i].Timestamp.After(*countResult.Datapoints[j].Timestamp)
		})

		// Use the most recent datapoint
		if countResult.Datapoints[0].Average != nil {
			objectCount = int64(*countResult.Datapoints[0].Average)
		}

		// If we don't have lastModified from size metrics, try from count metrics
		if lastModified == nil {
			lastChanged := findLastMetricChange(countResult.Datapoints)
			if lastChanged != nil && !lastChanged.After(time.Now()) {
				lastModified = lastChanged
			}
		}
	}

	// Fallback: if we couldn't determine lastModified from metrics or it's in the future,
	// use creation date or a reasonable fallback
	if lastModified == nil || lastModified.After(time.Now()) {
		// Try to use creation date if available
		for _, apiType := range []string{"GetRequests", "PutRequests"} {
			// Find the earliest API activity as a proxy for creation/first use
			activityTime := findEarliestActivity(c.cwClient, bucketName, apiType)
			if activityTime != nil && (lastModified == nil || activityTime.Before(*lastModified)) {
				lastModified = activityTime
			}
		}

		// If still no valid date, use a more conservative estimate
		if lastModified == nil || lastModified.After(time.Now()) {
			// Use 90 days ago as a safe fallback - better to potentially mark as idle
			// than to incorrectly mark as recently active
			t := time.Now().AddDate(0, 0, -90)
			lastModified = &t
		}
	}

	return objectCount, totalSize, lastModified, nil
}

// findLastMetricChange analyzes metric datapoints to find the last significant change
func findLastMetricChange(datapoints []cwTypes.Datapoint) *time.Time {
	if len(datapoints) < 2 {
		if len(datapoints) == 1 {
			return datapoints[0].Timestamp
		}
		return nil
	}

	// Sort by timestamp (ascending)
	sort.Slice(datapoints, func(i, j int) bool {
		return datapoints[i].Timestamp.Before(*datapoints[j].Timestamp)
	})

	var lastChangeTime *time.Time
	var prevValue float64
	if datapoints[0].Average != nil {
		prevValue = *datapoints[0].Average
	}

	for i := 1; i < len(datapoints); i++ {
		var currentValue float64
		if datapoints[i].Average != nil {
			currentValue = *datapoints[i].Average
		}

		// Look for any non-trivial change (0.1% is significant enough)
		if prevValue > 0 && math.Abs(currentValue-prevValue)/prevValue > 0.001 {
			lastChangeTime = datapoints[i].Timestamp
		} else if prevValue == 0 && currentValue > 0 {
			// Special case: from zero to non-zero is always significant
			lastChangeTime = datapoints[i].Timestamp
		}
		prevValue = currentValue
	}

	return lastChangeTime
}

// findEarliestActivity finds the earliest recorded API activity for a bucket
func findEarliestActivity(cwClient *cloudwatch.Client, bucketName string, metricName string) *time.Time {
	ctx := context.TODO()
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -90) // Look back 90 days max

	metricsInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/S3"),
		MetricName: aws.String(metricName),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucketName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticSum},
	}

	result, err := cwClient.GetMetricStatistics(ctx, metricsInput)
	if err != nil || len(result.Datapoints) == 0 {
		return nil
	}

	// Find the earliest datapoint with activity
	sort.Slice(result.Datapoints, func(i, j int) bool {
		return result.Datapoints[i].Timestamp.Before(*result.Datapoints[j].Timestamp)
	})

	// Find first datapoint with non-zero activity
	for _, dp := range result.Datapoints {
		if dp.Sum != nil && *dp.Sum > 0 {
			return dp.Timestamp
		}
	}

	return nil
}

// getBucketAPIActivity gets API call activity from CloudWatch metrics
func (c *S3Client) getBucketAPIActivity(bucketName string) (int64, int64, error) {
	// Time period for metrics: last 30 days
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)

	// GetObject requests
	getRequestsInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/S3"),
		MetricName: aws.String("GetRequests"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucketName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticSum},
	}

	getResult, err := c.cwClient.GetMetricStatistics(context.TODO(), getRequestsInput)
	if err != nil {
		return 0, 0, err
	}

	// PutObject requests
	putRequestsInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/S3"),
		MetricName: aws.String("PutRequests"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucketName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticSum},
	}

	putResult, err := c.cwClient.GetMetricStatistics(context.TODO(), putRequestsInput)
	if err != nil {
		return 0, 0, err
	}

	// Sum up the values
	var getRequests, putRequests int64
	for _, datapoint := range getResult.Datapoints {
		getRequests += int64(*datapoint.Sum)
	}
	for _, datapoint := range putResult.Datapoints {
		putRequests += int64(*datapoint.Sum)
	}

	return getRequests, putRequests, nil
}

// hasBucketWebsiteConfig checks if bucket has website configuration
func (c *S3Client) hasBucketWebsiteConfig(bucketName string) (bool, error) {
	_, err := c.client.GetBucketWebsite(context.TODO(), &s3.GetBucketWebsiteInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// NoSuchWebsiteConfiguration error means no website config
		return false, nil
	}
	return true, nil
}

// hasBucketPolicy checks if bucket has a policy
func (c *S3Client) hasBucketPolicy(bucketName string) (bool, error) {
	result, err := c.client.GetBucketPolicy(context.TODO(), &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// NoSuchBucketPolicy error means no policy
		return false, nil
	}
	// If there's a policy, it should have some content
	return result.Policy != nil && len(*result.Policy) > 0, nil
}

// hasBucketNotification checks if bucket has event notifications
func (c *S3Client) hasBucketNotification(bucketName string) (bool, error) {
	result, err := c.client.GetBucketNotificationConfiguration(context.TODO(),
		&s3.GetBucketNotificationConfigurationInput{
			Bucket: aws.String(bucketName),
		})
	if err != nil {
		return false, err
	}

	// Check if there are any notification configurations
	hasLambda := len(result.LambdaFunctionConfigurations) > 0
	hasQueue := len(result.QueueConfigurations) > 0
	hasTopic := len(result.TopicConfigurations) > 0

	return hasLambda || hasQueue || hasTopic, nil
}

// determineBucketIdleStatus determines if a bucket is idle based on multiple criteria
func (c *S3Client) determineBucketIdleStatus(bucketInfo *models.BucketInfo) bool {
	// Empty buckets are considered idle
	if bucketInfo.IsEmpty {
		return true
	}

	// No last modified date means we can't reliably determine status
	// Conservatively mark as not idle unless very clear evidence
	if bucketInfo.LastModified == nil {
		// Only mark as idle if zero API activity
		return bucketInfo.GetRequestsLast30Days == 0 && bucketInfo.PutRequestsLast30Days == 0
	}

	// Calculate days since last modification
	daysSinceModified := utils.CalculateElapsedDays(*bucketInfo.LastModified)

	// Debug logging removed for clarity

	// Primary idle check: No PUT requests and older than threshold
	if bucketInfo.PutRequestsLast30Days == 0 && daysSinceModified > c.idleThreshold {
		// For buckets with minimal GET activity
		if bucketInfo.GetRequestsLast30Days < 5 {
			return true
		}

		// For buckets with moderate GET activity but no changes
		if bucketInfo.GetRequestsLast30Days < 100 && daysSinceModified > c.idleThreshold*2 {
			return true
		}
	}

	// Not idle if it doesn't meet our criteria
	return false
}

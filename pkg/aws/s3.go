package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	// List all buckets
	result, err := c.client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("error listing S3 buckets: %w", err)
	}

	fmt.Printf("Found %d buckets in total\n", len(result.Buckets))

	var bucketInfos []models.BucketInfo

	// Filter buckets for the current region and process with progress indication
	var bucketsInRegion int
	totalBuckets := len(result.Buckets)

	fmt.Printf("Filtering buckets for region %s\n", c.region)

	for i, bucket := range result.Buckets {
		// Show progress for bucket filtering
		fmt.Printf("Checking bucket %d/%d: %s\n", i+1, totalBuckets, *bucket.Name)

		// Skip buckets from other regions
		location, err := c.getBucketRegion(*bucket.Name)
		if err != nil {
			// Skip buckets we can't access
			fmt.Printf("  Skipping: can't determine region\n")
			continue
		}

		// Skip buckets from other regions
		if location != c.region {
			fmt.Printf("  Skipping: in region %s\n", location)
			continue
		}

		bucketsInRegion++
		fmt.Printf("Analyzing bucket %d in region %s: %s\n", bucketsInRegion, c.region, *bucket.Name)

		// Get basic bucket info
		bucketInfo, err := c.analyzeBucket(*bucket.Name, *bucket.CreationDate)
		if err != nil {
			// Log error and continue with next bucket
			fmt.Printf("Error analyzing bucket %s: %v\n", *bucket.Name, err)
			continue
		}

		bucketInfos = append(bucketInfos, bucketInfo)
	}

	if bucketsInRegion == 0 {
		fmt.Printf("No buckets found in region %s\n", c.region)
	} else {
		fmt.Printf("Completed analyzing %d buckets in region %s\n", bucketsInRegion, c.region)
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
	var objectCount int64
	var totalSize int64
	var lastModified *time.Time

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}

	paginator := s3.NewListObjectsV2Paginator(c.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return 0, 0, nil, err
		}

		objectCount += int64(len(page.Contents))

		for _, object := range page.Contents {
			// Size is a pointer in the latest AWS SDK
			if object.Size != nil {
				totalSize += *object.Size
			}

			// Track the most recent LastModified time
			if lastModified == nil || object.LastModified.After(*lastModified) {
				lastModified = object.LastModified
			}
		}
	}

	return objectCount, totalSize, lastModified, nil
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

	// If bucket has been modified recently, it is NOT idle
	if bucketInfo.LastModified != nil {
		daysSinceModified := utils.CalculateElapsedDays(*bucketInfo.LastModified)

		// Buckets modified within threshold days are NEVER idle
		if daysSinceModified <= c.idleThreshold {
			return false
		}

		// For older buckets, check if there's ongoing read activity
		if bucketInfo.GetRequestsLast30Days < 10 {
			return true
		}
	}

	// Only consider buckets idle due to low API activity if:
	// 1. We don't have modification data OR
	// 2. The bucket is older than the threshold
	if (bucketInfo.LastModified == nil ||
		utils.CalculateElapsedDays(*bucketInfo.LastModified) > c.idleThreshold) &&
		bucketInfo.GetRequestsLast30Days == 0 &&
		bucketInfo.PutRequestsLast30Days == 0 {
		return true
	}

	// Not idle by our criteria
	return false
}

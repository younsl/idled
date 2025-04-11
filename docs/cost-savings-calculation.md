# Cost Savings Calculation in Idled

This document explains how Idled calculates cost savings for different AWS resources.

## EC2 Instance Savings

### How It Works

For EC2 instances, Idled calculates:

1. **Monthly Cost (`COST/MO`)**: The cost to run the instance for a full month.
   - Formula: `hourly price × 730 hours` (730 = average hours in a month)
   - This is the monthly cost if the instance were running.

2. **Total Savings (`TOTAL SAVED`)**: The accumulated savings since the instance was stopped.
   - Formula: `monthly cost × (days stopped ÷ 30)`
   - This counts all savings from the stop date until now.
   - For example, an instance stopped for 90 days saves 3 times its monthly cost.

### Example

If an instance:
- Costs $100 per month to run
- Has been stopped for 60 days

The calculation shows:
- Monthly Cost: $100
- Total Saved: $100 × (60 ÷ 30) = $200

## EBS Volume Savings

> [!WARNING]
> Current calculation does not include costs for IOPS or throughput. For io1, io2, and gp3 volumes with custom IOPS or throughput, actual savings may be higher than reported.

### How It Works

For EBS volumes, Idled calculates:

1. **Monthly Savings (`MONTHLY SAVINGS`)**: The potential savings per month if the unused volume were deleted.
   - Formula: `volume size × price per GB-month`
   - This shows how much you would save each month by removing this volume.
   - **Note**: Unlike EC2 instances, EBS volumes show only the monthly cost, not accumulated savings.

2. **Potential Monthly Savings (Summary)**: The total monthly savings across all volumes of a specific type.
   - This is the sum of monthly savings for all volumes, grouped by type.

### Example

If a volume:
- Is 100 GB in size
- Costs $0.10 per GB-month

The calculation shows:
- Monthly Savings: 100 × $0.10 = $10.00 per month
- This is regardless of how long the volume has been unused

## S3 Bucket Analysis

Idled analyzes S3 buckets to identify those that may be unused or underutilized. While S3 buckets have different cost considerations than EC2 or EBS resources, identifying idle buckets can help with:

### Identification Criteria

Idled considers a bucket "idle" if it meets one of these criteria:

1. **Empty Buckets**: Buckets with no objects (0 objects)
2. **No Recent Modifications**: Buckets with no modifications within the threshold period (default: 30 days)
3. **No API Activity**: Buckets with no GetObject or PutObject requests in the last 30 days (requires CloudWatch metrics)

### Analysis Information

For S3 buckets, Idled shows:

1. **Bucket Name**: The name of the S3 bucket
2. **Region**: AWS region where the bucket is located
3. **Objects**: Total number of objects in the bucket
4. **Size**: Total storage used by the bucket
5. **Idle Days**: Days since the last modification
6. **Last Modified**: Date of the last object modification
7. **Empty**: Whether the bucket has no objects
8. **Usage**: Detected usage patterns and configurations:
   - Recently Modified: Modified within last 30 days
   - Website: Configured for static website hosting
   - Policy: Has bucket policy
   - Events: Has event notifications
   - Static Content: High read, low write access pattern
   - API usage statistics

### Cost Considerations

While Idled does not currently calculate exact S3 cost savings, potential savings from idle S3 buckets include:

1. **Storage Costs**: Removing unused data
2. **API Request Costs**: Fewer GET/PUT operations
3. **Data Transfer Costs**: Less data transferred out
4. **Lifecycle Management**: Potential savings from moving data to cheaper storage classes

### Progress Indicators

For S3 operations that may take a long time (especially for buckets with millions of objects), Idled displays real-time progress information:

1. **Bucket discovery**: Shows total bucket count
2. **Region filtering**: Shows which buckets belong to the target region
3. **Analysis progress**: Shows which bucket is currently being analyzed
4. **Completion summary**: Shows total buckets processed

## Price Data Sources

Idled collects pricing information from:

1. **AWS Pricing API**: Real-time prices direct from AWS
2. **Cache**: Previously retrieved prices stored for better performance
3. **Default**: Built-in fallback prices when API data is unavailable

The source is marked in the `PRICING` column: API, CACHE, or N/A.

## Lambda Function Cost Estimation

Idled includes cost estimation for AWS Lambda functions to help identify potential savings from idle functions.

### Calculation Method

For Lambda functions, Idled calculates the estimated monthly cost based on:

1. **Request Costs**: The cost of function invocations
   - Formula: `monthly invocations × $0.20 per million requests`
   - Based on the actual number of invocations over the past 30 days

2. **Compute Costs**: The cost of execution time and memory usage
   - Formula: `GB-seconds × $0.0000166667 per GB-second`
   - GB-seconds = `invocations × average duration (seconds) × memory (GB)`
   - Memory is converted from MB to GB (divided by 1024)
   - Duration is converted from ms to seconds (divided by 1000)

3. **Total Monthly Cost**: The sum of request and compute costs
   - Formula: `request costs + compute costs`
   - This is the estimated monthly cost based on current usage patterns
   - Note: Free tier benefits are not included in this calculation

### Idle Function Identification

Idled considers a Lambda function "idle" if it meets one of these criteria:

1. **No Invocations**: Function has not been invoked in the last 30 days
2. **Low Usage**: Function's last invocation is older than the threshold (default: 30 days)

### Cost Savings Potential

While Idled does not currently show accumulated savings for Lambda functions as it does for EC2 instances, you can:

1. Use the "ESTIMATED COST" column to identify functions with minimal cost impact
2. Focus first on functions with zero invocations but significant memory allocations
3. Consider reducing memory allocations for rarely invoked functions
4. Remove functions that have been idle for extended periods

### Example 

For a Lambda function with:
- 10,000 invocations per month
- 200ms average duration
- 512MB memory allocation

The calculation would be:
- Request cost: 10,000 × $0.20 / 1,000,000 = $0.002
- GB-seconds: 10,000 × (200/1000) × (512/1024) = 1,000
- Compute cost: 1,000 × $0.0000166667 = $0.0167
- Total monthly cost: $0.002 + $0.0167 = $0.0187 (approximately 2 cents) 
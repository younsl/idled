package aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/briandowns/spinner"
	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// LambdaClient struct for Lambda client
type LambdaClient struct {
	client        *lambda.Client
	cwClient      *cloudwatch.Client
	region        string
	idleThreshold int // in days
}

// NewLambdaClient creates a new LambdaClient
func NewLambdaClient(region string) (*LambdaClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	client := lambda.NewFromConfig(cfg)
	cwClient := cloudwatch.NewFromConfig(cfg)

	return &LambdaClient{
		client:        client,
		cwClient:      cwClient,
		region:        region,
		idleThreshold: 30, // Default: consider functions idle after 30 days of inactivity
	}, nil
}

// SetIdleThreshold sets the threshold in days for considering a function as idle
func (c *LambdaClient) SetIdleThreshold(days int) {
	c.idleThreshold = days
}

// GetIdleFunctions returns a list of Lambda functions with their usage metrics
func (c *LambdaClient) GetIdleFunctions() ([]models.LambdaFunctionInfo, error) {
	fmt.Printf("Scanning Lambda functions in %s...\n", c.region)

	// 단계 1: Lambda 함수 목록 가져오기
	var functions []lambdaTypes.FunctionConfiguration
	var nextMarker *string
	var functionInfos []models.LambdaFunctionInfo

	for {
		input := &lambda.ListFunctionsInput{
			Marker: nextMarker,
		}

		result, err := c.client.ListFunctions(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("error listing Lambda functions: %w", err)
		}

		functions = append(functions, result.Functions...)

		if result.NextMarker == nil || *result.NextMarker == "" {
			break
		}
		nextMarker = result.NextMarker
	}

	totalFunctions := len(functions)
	fmt.Printf("Found %d Lambda functions in %s\n", totalFunctions, c.region)

	if totalFunctions == 0 {
		return functionInfos, nil
	}

	// 단계 2: 진행 상태 표시용 스피너 생성
	fmt.Printf("Analyzing Lambda functions usage metrics (this may take a while)...\n")

	// 단일 스피너 사용, 일관된 위치에 표시
	sp := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Suffix = fmt.Sprintf(" Progress: 0/%d functions", totalFunctions)
	sp.Start()
	defer sp.Stop()

	// 각 함수 분석, 진행률과 현재 함수 이름을 표시
	processedCount := 0
	lastPercentage := 0
	currentFunctionName := ""

	for _, function := range functions {
		// 현재 처리 중인 함수 이름 업데이트 (모든 함수마다 표시 업데이트)
		if function.FunctionName != nil {
			currentFunctionName = *function.FunctionName
			sp.Lock()
			sp.Suffix = fmt.Sprintf(" [%d/%d] Analyzing: %s",
				processedCount+1, totalFunctions, currentFunctionName)
			sp.Unlock()
		}

		// Get function metrics
		functionInfo, err := c.analyzeFunction(function)
		if err != nil {
			// Log error and continue with next function
			continue
		}

		functionInfos = append(functionInfos, functionInfo)

		// 진행률 업데이트
		processedCount++
		currentPercentage := (processedCount * 100) / totalFunctions

		// 진행률이 10% 포인트 증가할 때만 퍼센트 정보 추가 (깜빡임 줄이기)
		if currentPercentage >= lastPercentage+10 || processedCount == totalFunctions {
			sp.Lock()
			sp.Suffix = fmt.Sprintf(" %d/%d functions completed (%d%%) - Last: %s",
				processedCount, totalFunctions, currentPercentage, currentFunctionName)
			sp.Unlock()
			lastPercentage = currentPercentage
		}
	}

	sp.FinalMSG = fmt.Sprintf("✓ Completed analysis of %d Lambda functions\n", totalFunctions)

	return functionInfos, nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// analyzeFunction gathers information and metrics for a single Lambda function
func (c *LambdaClient) analyzeFunction(function lambdaTypes.FunctionConfiguration) (models.LambdaFunctionInfo, error) {
	functionName := *function.FunctionName

	// Initialize with basic information
	functionInfo := models.LambdaFunctionInfo{
		FunctionName: functionName,
		Region:       c.region,
		Runtime:      string(function.Runtime),
	}

	// Handle pointer values
	if function.MemorySize != nil {
		functionInfo.MemorySize = *function.MemorySize
	}

	if function.Timeout != nil {
		functionInfo.Timeout = *function.Timeout
	}

	// Add description if available
	if function.Description != nil {
		functionInfo.Description = *function.Description
	}

	// Set last modified time
	if function.LastModified != nil {
		parsedTime, err := time.Parse(time.RFC3339, *function.LastModified)
		if err == nil {
			functionInfo.LastModified = &parsedTime
		}
	}

	// Get CloudWatch metrics for invocations
	invocations, errors, lastInvocation, duration, err := c.getFunctionMetrics(functionName)
	if err != nil {
		// Just log the error and continue - this is non-critical
		fmt.Printf("Warning: Could not retrieve CloudWatch metrics for function %s: %v\n", functionName, err)
	} else {
		functionInfo.InvocationsLast30Days = invocations
		functionInfo.ErrorsLast30Days = errors
		functionInfo.LastInvocation = lastInvocation
		functionInfo.DurationP95Last30Days = duration

		// Calculate idle days if we have last invocation data
		if lastInvocation != nil {
			functionInfo.IdleDays = utils.CalculateElapsedDays(*lastInvocation)
		}
	}

	// Calculate estimated monthly cost
	functionInfo.EstimatedMonthlyCost = calculateLambdaCost(functionInfo)

	// Determine if the function is idle
	functionInfo.IsIdle = c.determineFunctionIdleStatus(&functionInfo)

	return functionInfo, nil
}

// getFunctionMetrics retrieves CloudWatch metrics for a Lambda function
func (c *LambdaClient) getFunctionMetrics(functionName string) (int64, int64, *time.Time, float64, error) {
	ctx := context.TODO()
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30) // Last 30 days

	// Get invocation metrics
	invocationsInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Lambda"),
		MetricName: aws.String("Invocations"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("FunctionName"),
				Value: aws.String(functionName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticSum},
	}

	invocationsResult, err := c.cwClient.GetMetricStatistics(ctx, invocationsInput)
	if err != nil {
		return 0, 0, nil, 0, err
	}

	// Get error metrics
	errorsInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Lambda"),
		MetricName: aws.String("Errors"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("FunctionName"),
				Value: aws.String(functionName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400), // 1 day
		Statistics: []cwTypes.Statistic{cwTypes.StatisticSum},
	}

	errorsResult, err := c.cwClient.GetMetricStatistics(ctx, errorsInput)
	if err != nil {
		return 0, 0, nil, 0, err
	}

	// Get duration metrics (average)
	durationInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Lambda"),
		MetricName: aws.String("Duration"),
		Dimensions: []cwTypes.Dimension{
			{
				Name:  aws.String("FunctionName"),
				Value: aws.String(functionName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(2592000), // 30 days
		Statistics: []cwTypes.Statistic{cwTypes.StatisticAverage},
	}

	durationResult, err := c.cwClient.GetMetricStatistics(ctx, durationInput)
	if err != nil {
		return 0, 0, nil, 0, err
	}

	// Sum up invocations
	var totalInvocations, totalErrors int64
	var lastInvocationTime *time.Time
	var avgDuration float64

	// Process invocations, tracking the most recent non-zero invocation
	if len(invocationsResult.Datapoints) > 0 {
		// Sort by timestamp (descending)
		sort.Slice(invocationsResult.Datapoints, func(i, j int) bool {
			return invocationsResult.Datapoints[i].Timestamp.After(*invocationsResult.Datapoints[j].Timestamp)
		})

		for _, datapoint := range invocationsResult.Datapoints {
			if datapoint.Sum != nil {
				sum := int64(*datapoint.Sum)
				totalInvocations += sum

				// If we have invocations and haven't set last invocation time yet
				if sum > 0 && lastInvocationTime == nil {
					lastInvocationTime = datapoint.Timestamp
				}
			}
		}
	}

	// Sum up errors
	for _, datapoint := range errorsResult.Datapoints {
		if datapoint.Sum != nil {
			totalErrors += int64(*datapoint.Sum)
		}
	}

	// Get average duration
	if len(durationResult.Datapoints) > 0 {
		// Sort by timestamp (descending) to get most recent
		sort.Slice(durationResult.Datapoints, func(i, j int) bool {
			return durationResult.Datapoints[i].Timestamp.After(*durationResult.Datapoints[j].Timestamp)
		})

		if durationResult.Datapoints[0].Average != nil {
			avgDuration = *durationResult.Datapoints[0].Average
		}
	}

	return totalInvocations, totalErrors, lastInvocationTime, avgDuration, nil
}

// calculateLambdaCost estimates the monthly cost of a Lambda function
func calculateLambdaCost(functionInfo models.LambdaFunctionInfo) float64 {
	// Lambda pricing (simplified model):
	// - Free tier: 1M requests free and 400,000 GB-seconds of compute time per month
	// - $0.20 per 1M requests
	// - $0.0000166667 per GB-second

	// Estimate monthly invocations based on 30-day history
	monthlyInvocations := functionInfo.InvocationsLast30Days

	// Estimate average duration in seconds
	avgDurationSec := functionInfo.DurationP95Last30Days / 1000 // convert ms to seconds
	if avgDurationSec <= 0 {
		avgDurationSec = 0.1 // assume 100ms if we don't have data
	}

	// Calculate GB-seconds
	gbSeconds := float64(monthlyInvocations) * avgDurationSec * float64(functionInfo.MemorySize) / 1024

	// Calculate cost (ignoring free tier for simplicity)
	requestsCost := float64(monthlyInvocations) * 0.20 / 1000000
	computeCost := gbSeconds * 0.0000166667

	// Total monthly cost
	return requestsCost + computeCost
}

// determineFunctionIdleStatus determines if a function is idle based on metrics
func (c *LambdaClient) determineFunctionIdleStatus(functionInfo *models.LambdaFunctionInfo) bool {
	// If no invocations in the last 30 days, it's definitely idle
	if functionInfo.InvocationsLast30Days == 0 {
		return true
	}

	// If we have last invocation data, check against threshold
	if functionInfo.LastInvocation != nil {
		daysSinceInvocation := utils.CalculateElapsedDays(*functionInfo.LastInvocation)

		// If last invocation is older than threshold, consider it idle
		if daysSinceInvocation > c.idleThreshold {
			return true
		}
	}

	// Not idle by our criteria
	return false
}

package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/younsl/idled/internal/models"
)

const (
	// Define the period for CloudWatch checks
	cloudWatchPeriodDays = 14

	// AWS CloudWatch Namespaces
	namespaceALB = "AWS/ApplicationELB"
	namespaceNLB = "AWS/NetworkELB"

	// AWS CloudWatch Metric Names
	metricRequestCount    = "RequestCount"
	metricActiveFlowCount = "ActiveFlowCount"
)

// ELBScanner contains the AWS clients needed for scanning ELB resources
type ELBScanner struct {
	ELBV2Client *elbv2.Client
	CWClient    *cloudwatch.Client
}

// NewELBScanner creates a new ELBScanner for a given region
func NewELBScanner(cfg aws.Config) *ELBScanner {
	return &ELBScanner{
		ELBV2Client: elbv2.NewFromConfig(cfg),
		CWClient:    cloudwatch.NewFromConfig(cfg),
	}
}

// GetIdleELBs scans for idle ALB and NLB resources in a specific region sequentially
func (s *ELBScanner) GetIdleELBs(ctx context.Context, region string) ([]models.ELBResource, error) {
	var idleELBs []models.ELBResource
	var errs []error // Collect errors encountered during the scan

	// Fetch Load Balancers using ELBv2 client
	paginator := elbv2.NewDescribeLoadBalancersPaginator(s.ELBV2Client, &elbv2.DescribeLoadBalancersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			// If pagination fails, we can't continue scanning this region
			fetchErr := fmt.Errorf("error describing v2 load balancers in %s: %w", region, err)
			// Return immediately with this error, potentially wrapping existing errors
			if len(errs) > 0 {
				return idleELBs, fmt.Errorf("pagination failed after encountering %d errors: %w. First error: %v", len(errs), fetchErr, errs[0])
			}
			return nil, fetchErr
		}

		for _, lb := range page.LoadBalancers {
			lbDesc := lb // Local copy for clarity

			// Skip unsupported types
			if lbDesc.Type != elbv2types.LoadBalancerTypeEnumApplication && lbDesc.Type != elbv2types.LoadBalancerTypeEnumNetwork {
				continue
			}

			// --- Process each LB sequentially ---
			lbArn := aws.ToString(lbDesc.LoadBalancerArn)
			lbName := aws.ToString(lbDesc.LoadBalancerName)
			lbType := lbDesc.Type

			isIdle, reason, healthyTargets, unhealthyTargets, lastActivitySum, checkErr := s.checkLoadBalancerIdleStatus(ctx, lbArn, lbType)

			if checkErr != nil {
				// Record error for this specific LB check and continue to the next LB
				newErr := fmt.Errorf("error checking idle status for %s %s in %s: %w", lbType, lbName, region, checkErr)
				errs = append(errs, newErr) // Assign back to errs
				continue                    // Don't add to idleELBs if check failed
			}

			if isIdle {
				// Determine short type string
				shortType := "Unknown"
				if lbType == elbv2types.LoadBalancerTypeEnumApplication {
					shortType = "ALB"
				} else if lbType == elbv2types.LoadBalancerTypeEnumNetwork {
					shortType = "NLB"
				}

				idleELBs = append(idleELBs, models.ELBResource{
					Name:                 lbName,
					Type:                 shortType,
					Region:               region,
					State:                string(lbDesc.State.Code),
					CreatedTime:          *lbDesc.CreatedTime,
					ARN:                  lbArn,
					HealthyTargetCount:   healthyTargets,
					UnhealthyTargetCount: unhealthyTargets,
					IdleReason:           reason,
					LastActivitySum:      lastActivitySum,
				})
			}
			// --- End sequential processing for this LB ---
		}
	}

	if len(errs) > 0 {
		// Return results found so far, along with the first error encountered
		return idleELBs, fmt.Errorf("encountered %d errors during ELB scan (results might be incomplete), first error: %w", len(errs), errs[0])
	}

	return idleELBs, nil // Success, no errors
}

// checkLoadBalancerIdleStatus determines if an ALB or NLB is idle
func (s *ELBScanner) checkLoadBalancerIdleStatus(ctx context.Context, lbArn string, lbType elbv2types.LoadBalancerTypeEnum) (isIdle bool, reason string, healthyTargets, unhealthyTargets int, metricSum *float64, err error) {
	// 1. Get Target Counts
	healthyTargets, unhealthyTargets, totalTargets, err := s.getTargetCounts(ctx, lbArn)
	if err != nil {
		return false, "", 0, 0, nil, fmt.Errorf("failed to get target counts: %w", err)
	}

	// 2. Determine CloudWatch parameters based on LB type
	var cwNamespace, cwMetricName, cwMetricReason string
	var cwStatistic cwtypes.Statistic
	switch lbType {
	case elbv2types.LoadBalancerTypeEnumApplication:
		cwNamespace = namespaceALB        // Use constant
		cwMetricName = metricRequestCount // Use constant
		cwStatistic = cwtypes.StatisticSum
		cwMetricReason = "Zero RequestCount (14d)"
	case elbv2types.LoadBalancerTypeEnumNetwork:
		cwNamespace = namespaceNLB           // Use constant
		cwMetricName = metricActiveFlowCount // Use constant
		cwStatistic = cwtypes.StatisticAverage
		cwMetricReason = "Zero ActiveFlowCount (Avg, 14d)"
	default:
		// Should not happen due to earlier check, but handle defensively
		return false, "", 0, 0, nil, fmt.Errorf("unsupported load balancer type: %s", lbType)
	}

	// 3. Check CloudWatch Metric
	sum, cwErr := s.getMetricSum(ctx, lbArn, cwNamespace, cwMetricName, cwStatistic)
	if cwErr != nil {
		// If CloudWatch fails, we cannot definitively say it's idle based on traffic.
		// We might still consider it idle if there are no healthy targets.
		if healthyTargets == 0 {
			reason = "No healthy targets registered"
			if totalTargets == 0 {
				reason = "No targets registered"
			}
			fmt.Printf("Warning: CloudWatch check failed for %s (%s), considering idle based on target health: %v\n", lbType, lbArn, cwErr)
			return true, reason + " (CW Check Failed)", healthyTargets, unhealthyTargets, nil, nil // Return idle, but note CW failed
		}
		// Healthy targets exist, but CW failed - cannot determine idle status reliably.
		return false, "", healthyTargets, unhealthyTargets, nil, fmt.Errorf("CloudWatch check failed: %w", cwErr)
	}
	metricSum = &sum

	// 4. Determine Idle Status based on targets and metrics
	if healthyTargets == 0 {
		reason = "No healthy targets registered"
		if totalTargets == 0 {
			reason = "No targets registered"
		}
		if sum == 0 {
			return true, reason + " & " + cwMetricReason, healthyTargets, unhealthyTargets, metricSum, nil
		} else {
			// No healthy targets, but recent traffic? Not idle.
			return false, "", healthyTargets, unhealthyTargets, metricSum, nil
		}
	}

	// Healthy targets > 0
	if sum == 0 {
		// Healthy targets exist, but no recent traffic.
		return true, cwMetricReason, healthyTargets, unhealthyTargets, metricSum, nil
	}

	// Healthy targets and recent traffic.
	return false, "", healthyTargets, unhealthyTargets, metricSum, nil
}

// getTargetCounts finds the number of healthy and unhealthy targets for a given ALB/NLB ARN
func (s *ELBScanner) getTargetCounts(ctx context.Context, lbArn string) (healthyCount, unhealthyCount, totalCount int, err error) {
	tgPaginator := elbv2.NewDescribeTargetGroupsPaginator(s.ELBV2Client, &elbv2.DescribeTargetGroupsInput{
		LoadBalancerArn: aws.String(lbArn),
	})

	healthyCount = 0
	unhealthyCount = 0
	totalCount = 0

	for tgPaginator.HasMorePages() {
		tgPage, pageErr := tgPaginator.NextPage(ctx)
		if pageErr != nil {
			return 0, 0, 0, fmt.Errorf("error describing target groups for %s: %w", lbArn, pageErr)
		}

		for _, tg := range tgPage.TargetGroups {
			if tg.TargetGroupArn == nil {
				continue
			}
			healthInput := &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: tg.TargetGroupArn,
			}
			healthOutput, healthErr := s.ELBV2Client.DescribeTargetHealth(ctx, healthInput)
			if healthErr != nil {
				fmt.Printf("Warning: error describing target health for TG %s: %v\n", *tg.TargetGroupArn, healthErr)
				continue // Skip this TG, but don't fail the whole LB check
			}

			for _, thd := range healthOutput.TargetHealthDescriptions {
				totalCount++
				if thd.TargetHealth != nil {
					switch thd.TargetHealth.State {
					case elbv2types.TargetHealthStateEnumHealthy:
						healthyCount++
					case elbv2types.TargetHealthStateEnumUnhealthy:
						unhealthyCount++
						// Other states (initial, draining, unused) are not counted as healthy or unhealthy explicitly here
					}
				}
			}
		}
	}
	return healthyCount, unhealthyCount, totalCount, nil
}

// getMetricSum retrieves the sum of a specific CloudWatch metric over the last N days
func (s *ELBScanner) getMetricSum(ctx context.Context, lbArn, namespace, metricName string, statistic cwtypes.Statistic) (float64, error) {
	// Extract LoadBalancer name/ID from ARN for dimensions
	arnParts := strings.Split(lbArn, ":")
	if len(arnParts) < 6 {
		return 0, fmt.Errorf("invalid ELB ARN format: %s", lbArn)
	}
	lbPart := arnParts[5]
	// Handle different ARN formats (e.g., app/my-alb/id, net/my-nlb/id)
	if !strings.HasPrefix(lbPart, "loadbalancer/") {
		return 0, fmt.Errorf("unexpected ELB ARN resource format: %s", lbPart)
	}
	lbDimensionValue := lbPart[len("loadbalancer/"):] // Get the part after loadbalancer/

	dimensionName := "LoadBalancer"

	now := time.Now()
	startTime := now.AddDate(0, 0, -cloudWatchPeriodDays)
	endTime := now

	periodSeconds := int32(cloudWatchPeriodDays * 24 * 60 * 60) // Total seconds in the period

	metricInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Dimensions: []cwtypes.Dimension{
			{
				Name:  aws.String(dimensionName),
				Value: aws.String(lbDimensionValue),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(periodSeconds),
		Statistics: []cwtypes.Statistic{statistic},
	}

	resp, err := s.CWClient.GetMetricStatistics(ctx, metricInput)
	if err != nil {
		// Check for specific errors? e.g., no metrics found might not be a hard error
		return 0, fmt.Errorf("failed to get CloudWatch metric %s for %s (dimension: %s=%s): %w",
			metricName, lbArn, dimensionName, lbDimensionValue, err)
	}

	sum := 0.0
	if len(resp.Datapoints) > 0 {
		dp := resp.Datapoints[0] // Assuming one datapoint for the whole period
		switch statistic {
		case cwtypes.StatisticSum:
			if dp.Sum != nil {
				sum = *dp.Sum
			}
		case cwtypes.StatisticAverage:
			if dp.Average != nil {
				sum = *dp.Average
			}
		default:
			if dp.Sum != nil { // Default to Sum if available
				sum = *dp.Sum
			}
		}
	}

	return sum, nil
}

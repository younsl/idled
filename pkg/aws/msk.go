package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"

	// kafkaconnecttypes "github.com/aws/aws-sdk-go-v2/service/kafkaconnect/types" // State type might be directly in kafka types
	"github.com/younsl/idled/internal/models"
	// Alias for pkg utils
)

const (
	mskCheckPeriodDays = 30
	mskNamespace       = "AWS/Kafka"
	// Connection Check
	mskMetricConnectionCount = "ConnectionCount"
	mskConnStatistic         = cwtypes.StatisticMaximum
	idleConnectionThreshold  = 0
	// CPU Check
	mskMetricCPUSystem     = "CpuSystem"
	mskMetricCPUUser       = "CpuUser"
	mskCPUStatistic        = cwtypes.StatisticAverage
	lowCPUThresholdPercent = 30.0 // Changed threshold to 30%
)

// MskScanner contains the AWS clients needed for scanning MSK resources
type MskScanner struct {
	KafkaClient *kafka.Client
	CWClient    *cloudwatch.Client
	Region      string
}

// NewMskScanner creates a new MskScanner for a given region
func NewMskScanner(cfg aws.Config) *MskScanner {
	return &MskScanner{
		KafkaClient: kafka.NewFromConfig(cfg),
		CWClient:    cloudwatch.NewFromConfig(cfg),
		Region:      cfg.Region,
	}
}

// GetIdleMskClusters scans all MSK clusters and identifies idle/underutilized ones
func (s *MskScanner) GetIdleMskClusters(ctx context.Context) ([]models.MskClusterInfo, []error) {
	var allClusters []models.MskClusterInfo
	var clusterArns []string
	var scanErrs []error
	clusterDetails := make(map[string]*types.ClusterInfo)

	// 1. List all clusters using ListClusters (pagination)
	listPaginator := kafka.NewListClustersPaginator(s.KafkaClient, &kafka.ListClustersInput{})
	pageCount := 0
	for listPaginator.HasMorePages() {
		pageCount++
		listOutput, err := listPaginator.NextPage(ctx)
		if err != nil {
			// Error message is handled by the main error processing logic
			// sp.FinalMSG = fmt.Sprintf("✗ Error listing MSK clusters page %d in %s\n", pageCount, s.Region)
			scanErrs = append(scanErrs, fmt.Errorf("error listing MSK clusters page %d: %w", pageCount, err))
			break // Stop processing this region on pagination error
		}
		if listOutput != nil {
			for _, clusterInfo := range listOutput.ClusterInfoList {
				if clusterInfo.ClusterArn != nil {
					arn := *clusterInfo.ClusterArn
					clusterArns = append(clusterArns, arn)
					// Store the pointer to ClusterInfo from ListClusters initially
					// Need to make a copy if we modify it later based on DescribeCluster?
					// Let's store the value first, then update with DescribeCluster info.
					tempInfo := clusterInfo // Create a copy
					clusterDetails[arn] = &tempInfo
				}
			}
		}
	}

	if len(clusterArns) == 0 {
		// No need for specific message here, main handler reports 0 items found.
		// sp.FinalMSG = fmt.Sprintf("✓ No MSK clusters found in %s\n", s.Region)
		return allClusters, scanErrs
	}

	// 2. Describe each cluster and List Nodes
	brokerIDsMap := make(map[string][]string) // Map ARN to list of Broker IDs
	for arn, detailsPtr := range clusterDetails {
		// Describe Cluster (mainly for CreationTime?)
		descInput := &kafka.DescribeClusterInput{ClusterArn: aws.String(arn)}
		descOutput, descErr := s.KafkaClient.DescribeCluster(ctx, descInput)
		if descErr != nil {
			warnMsg := fmt.Sprintf("Warning: could not describe MSK cluster %s in %s: %v", arn, s.Region, descErr)
			fmt.Println(warnMsg) // Print warning
			scanErrs = append(scanErrs, fmt.Errorf(warnMsg))
			delete(clusterDetails, arn)
			continue
		}
		if descOutput != nil && descOutput.ClusterInfo != nil {
			describedInfo := descOutput.ClusterInfo
			detailsPtr.CreationTime = describedInfo.CreationTime
			detailsPtr.State = describedInfo.State
			detailsPtr.ClusterName = describedInfo.ClusterName
		} else {
			// Handle unexpected empty response
			warnMsg := fmt.Sprintf("Warning: DescribeCluster returned empty info for %s in %s", arn, s.Region)
			fmt.Println(warnMsg)
			delete(clusterDetails, arn)
			continue
		}

		// List Nodes to get Broker IDs
		nodesInput := &kafka.ListNodesInput{ClusterArn: aws.String(arn)}
		var brokerIDs []string
		nodesPaginator := kafka.NewListNodesPaginator(s.KafkaClient, nodesInput)
		for nodesPaginator.HasMorePages() {
			nodesOutput, nodesErr := nodesPaginator.NextPage(ctx)
			if nodesErr != nil {
				warnMsg := fmt.Sprintf("Warning: could not list nodes for cluster %s: %v", arn, nodesErr)
				fmt.Println(warnMsg)
				scanErrs = append(scanErrs, fmt.Errorf(warnMsg))
				// Mark broker list as potentially incomplete or break?
				// Let's break for now, as we can't reliably get metrics without all brokers
				brokerIDs = nil // Indicate failure to get broker IDs
				break
			}
			if nodesOutput != nil {
				for _, nodeInfo := range nodesOutput.NodeInfoList {
					if nodeInfo.BrokerNodeInfo != nil && nodeInfo.BrokerNodeInfo.BrokerId != nil {
						// Format BrokerId (*float64) as integer string for dimension
						brokerIDs = append(brokerIDs, fmt.Sprintf("%d", int64(*nodeInfo.BrokerNodeInfo.BrokerId)))
					}
				}
			}
		}
		if brokerIDs != nil { // Only store if ListNodes succeeded
			brokerIDsMap[arn] = brokerIDs
		} else {
			// Failed to get broker IDs, remove cluster from further processing
			delete(clusterDetails, arn)
		}
	}

	processedCount := 0
	for arn, details := range clusterDetails {
		processedCount++
		// Update suffix for progress
		// sp.Suffix = fmt.Sprintf(" (%d/%d)", processedCount, totalClusters)

		creationTime := aws.ToTime(details.CreationTime)
		state := details.State
		clusterName := aws.ToString(details.ClusterName)
		brokerIDs := brokerIDsMap[arn]

		// Get Instance Type from BrokerNodeGroupInfo
		instanceType := "N/A"
		if details.BrokerNodeGroupInfo != nil && details.BrokerNodeGroupInfo.InstanceType != nil { // Check pointer
			instanceType = *details.BrokerNodeGroupInfo.InstanceType // Dereference pointer
		}

		// Check Connection Count using broker IDs
		maxConnections, connErrs := s.getMaxConnectionCount(ctx, clusterName, brokerIDs)
		if len(connErrs) > 0 {
			scanErrs = append(scanErrs, connErrs...)
		}

		// Check CPU Utilization using broker IDs
		avgCPU, cpuErrs := s.getAvgCPUUtilization(ctx, clusterName, brokerIDs)
		if len(cpuErrs) > 0 {
			scanErrs = append(scanErrs, cpuErrs...)
		}

		isIdle := false
		reason := "" // Default reason is empty (not idle)
		connIdle := maxConnections != nil && *maxConnections <= idleConnectionThreshold
		cpuIdle := avgCPU != nil && *avgCPU < lowCPUThresholdPercent

		if connIdle && cpuIdle {
			isIdle = true
			reason = "No Conn & Low CPU"
		} else if connIdle {
			isIdle = true
			reason = "No Connections"
		} else if cpuIdle {
			isIdle = true
			reason = "Low CPU Usage"
		}

		// Append ALL successfully processed clusters to the result slice
		allClusters = append(allClusters, models.MskClusterInfo{
			ClusterName:       clusterName,
			ARN:               arn,
			Region:            s.Region,
			State:             string(state),
			InstanceType:      instanceType,
			CreationTime:      creationTime,
			IsIdle:            isIdle, // Mark true/false
			Reason:            reason, // Populate reason if idle, otherwise empty
			ConnectionCount:   maxConnections,
			AvgCPUUtilization: avgCPU,
		})
	}

	return allClusters, scanErrs // Return results and any errors encountered during the scan
}

// getMaxConnectionCount retrieves the maximum connection count across all brokers
func (s *MskScanner) getMaxConnectionCount(ctx context.Context, clusterName string, brokerIDs []string) (*float64, []error) {
	var maxConn *float64
	var errs []error
	foundData := false

	if len(brokerIDs) == 0 {
		return nil, []error{fmt.Errorf("no broker IDs provided for cluster %s", clusterName)}
	}

	for _, brokerID := range brokerIDs {
		brokerIDStr := brokerID // Capture loop variable for pointer
		conn, err := s.getMetricValue(ctx, clusterName, mskMetricConnectionCount, mskConnStatistic, &brokerIDStr)
		if err != nil {
			err := fmt.Errorf("broker %s: %w", brokerID, err)
			err_msg := fmt.Sprintf("getMaxConnectionCount error for %s", err.Error())
			err = fmt.Errorf(err_msg)
			err = fmt.Errorf("broker %s: %w", brokerID, err)
			fmt.Printf("Warning: %s\n", err.Error())
			errs = append(errs, err) // Append the error with broker context
			continue                 // Try next broker
		}
		if conn != nil {
			foundData = true
			if maxConn == nil || *conn > *maxConn {
				maxConn = conn // Keep track of the highest max found
			}
		}
	}

	if !foundData && len(errs) == len(brokerIDs) {
		// If we had errors for every broker and found no data, return the errors
		return nil, errs
	}

	// Return the highest max found, or nil if no data points were found for any broker
	// Return collected errors (might be empty if all succeeded)
	return maxConn, errs
}

// getAvgCPUUtilization retrieves the average CPU utilization across all brokers
func (s *MskScanner) getAvgCPUUtilization(ctx context.Context, clusterName string, brokerIDs []string) (*float64, []error) {
	var totalCPU float64
	var cpuCount int
	var errs []error
	foundData := false

	if len(brokerIDs) == 0 {
		return nil, []error{fmt.Errorf("no broker IDs provided for cluster %s", clusterName)}
	}

	for _, brokerID := range brokerIDs {
		brokerIDStr := brokerID // Capture loop variable
		avgSystem, errSys := s.getMetricValue(ctx, clusterName, mskMetricCPUSystem, mskCPUStatistic, &brokerIDStr)
		avgUser, errUser := s.getMetricValue(ctx, clusterName, mskMetricCPUUser, mskCPUStatistic, &brokerIDStr)

		if errSys != nil {
			err := fmt.Errorf("broker %s (CpuSystem): %w", brokerID, errSys)
			fmt.Printf("Warning: %s\n", err.Error())
			err = fmt.Errorf("broker %s (CpuSystem): %w", brokerID, errSys)
			fmt.Printf("Warning: %s\n", err.Error())
			errs = append(errs, err) // Append the error with broker context
		}
		if errUser != nil {
			err := fmt.Errorf("broker %s (CpuUser): %w", brokerID, errUser)
			fmt.Printf("Warning: %s\n", err.Error())
			err = fmt.Errorf("broker %s (CpuUser): %w", brokerID, errUser)
			fmt.Printf("Warning: %s\n", err.Error())
			errs = append(errs, err) // Append the error with broker context
		}

		// Only aggregate if both metrics were successfully retrieved for this broker
		if avgSystem != nil && avgUser != nil {
			foundData = true
			totalCPU += (*avgSystem + *avgUser)
			cpuCount++
		}
		// If either metric is nil, or if errors occurred (errSys or errUser != nil),
		// we simply don't update totalCPU or cpuCount for this broker.
		// Errors were already appended to the errs slice earlier.
	}

	if !foundData {
		// If no data was found for any broker (either due to errors or no datapoints)
		return nil, errs // Return nil value and any errors encountered
	}

	if cpuCount == 0 {
		// This should ideally not happen if foundData is true, but handle defensively
		return nil, errs
	}

	overallAvg := totalCPU / float64(cpuCount)
	return &overallAvg, errs
}

// getMetricValue is a generic helper to fetch a specific metric value
// Added brokerID parameter for broker-level metrics
func (s *MskScanner) getMetricValue(ctx context.Context, clusterName, metricName string, statistic cwtypes.Statistic, brokerID *string) (*float64, error) {
	now := time.Now()
	startTime := now.AddDate(0, 0, -mskCheckPeriodDays)
	endTime := now
	periodSeconds := int32(mskCheckPeriodDays * 24 * 60 * 60)

	dimensions := []cwtypes.Dimension{
		{
			Name:  aws.String("Cluster Name"),
			Value: aws.String(clusterName),
		},
	}

	// Add Broker ID dimension if provided
	if brokerID != nil {
		dimensions = append(dimensions, cwtypes.Dimension{
			Name:  aws.String("Broker ID"),
			Value: brokerID,
		})
	}

	metricInput := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(mskNamespace),
		MetricName: aws.String(metricName),
		Dimensions: dimensions, // Use the constructed dimensions slice
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(periodSeconds),
		Statistics: []cwtypes.Statistic{statistic},
	}

	resp, err := s.CWClient.GetMetricStatistics(ctx, metricInput)
	if err != nil {
		// Construct more informative error message including broker ID if present
		logDetail := fmt.Sprintf("metric %s for cluster %s", metricName, clusterName)
		if brokerID != nil {
			logDetail += fmt.Sprintf(" (broker %s)", *brokerID)
		}
		return nil, fmt.Errorf("CloudWatch API error for %s: %w", logDetail, err)
	}

	if len(resp.Datapoints) == 0 {
		return nil, nil // No data found
	}

	dp := resp.Datapoints[0]
	switch statistic {
	case cwtypes.StatisticMaximum:
		if dp.Maximum != nil {
			return dp.Maximum, nil
		}
	case cwtypes.StatisticAverage:
		if dp.Average != nil {
			return dp.Average, nil
		}
	case cwtypes.StatisticSum:
		if dp.Sum != nil {
			return dp.Sum, nil
		}
	default:
		return nil, fmt.Errorf("unsupported statistic %s requested", statistic)
	}

	logDetail := fmt.Sprintf("cluster %s", clusterName)
	if brokerID != nil {
		logDetail += fmt.Sprintf(" (broker %s)", *brokerID)
	}
	return nil, fmt.Errorf("no %s value found in datapoint for %s", statistic, logDetail)
}

package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/younsl/idled/internal/models"
)

// ConfigClient represents an AWS Config client
type ConfigClient struct {
	client *configservice.Client
	region string
}

// ConfigRule represents an AWS Config rule
type ConfigRule struct {
	Name           string
	ARN            string
	LastModified   time.Time
	CreationTime   time.Time
	Region         string
	State          string
	LastEvaluation time.Time
}

// ConfigRecorder represents an AWS Config recorder
type ConfigRecorder struct {
	Name         string
	ARN          string
	LastStatus   string
	Region       string
	LastStarted  time.Time
	LastStopped  time.Time
	RecordingAll bool
}

// DeliveryChannel represents an AWS Config delivery channel
type DeliveryChannel struct {
	Name             string
	Region           string
	S3BucketName     string
	S3KeyPrefix      string
	SNSTopicARN      string
	LastChangeTime   time.Time
	ConfigSnapshotOn bool
}

// NewConfigClient creates a new AWS Config client
func NewConfigClient(region string) (*ConfigClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for region %s: %w", region, err)
	}

	return &ConfigClient{
		client: configservice.NewFromConfig(cfg, func(o *configservice.Options) {
			o.Region = region
		}),
		region: region,
	}, nil
}

// GetAllConfigRules returns a list of models.ConfigRuleInfo objects representing Config rules
func (c *ConfigClient) GetAllConfigRules() ([]models.ConfigRuleInfo, error) {
	ctx := context.Background()

	input := &configservice.DescribeConfigRulesInput{}
	resp, err := c.client.DescribeConfigRules(ctx, input)
	if err != nil {
		return nil, err
	}

	var configRules []models.ConfigRuleInfo
	cutoffTime := time.Now().AddDate(0, 0, -90) // Default to 90 days for idle

	for _, rule := range resp.ConfigRules {
		// Initialize with default values
		now := time.Now()
		var createdTime time.Time = now
		var lastActivity time.Time = now

		// Convert to our model
		configRule := models.ConfigRuleInfo{
			RuleName: *rule.ConfigRuleName,
			RuleID:   *rule.ConfigRuleId,
			ARN:      *rule.ConfigRuleArn,
			Region:   c.region,
			IsActive: rule.ConfigRuleState == types.ConfigRuleState("ACTIVE"),
			IsCustom: rule.Source != nil && rule.Source.Owner != types.Owner("AWS"),
		}

		// Set creation time pointer
		configRule.CreatedTime = &createdTime
		configRule.LastActivity = &lastActivity

		// Check rule evaluations for compliance status
		complianceInput := &configservice.DescribeComplianceByConfigRuleInput{
			ConfigRuleNames: []string{*rule.ConfigRuleName},
		}
		complianceResp, err := c.client.DescribeComplianceByConfigRule(ctx, complianceInput)
		if err == nil && len(complianceResp.ComplianceByConfigRules) > 0 {
			for _, compliance := range complianceResp.ComplianceByConfigRules {
				// Set compliance status
				if compliance.Compliance != nil {
					configRule.IsCompliant = compliance.Compliance.ComplianceType == types.ComplianceType("COMPLIANT")
				}
			}
		}

		// Try to get more detailed timing information from ConfigRuleEvaluationStatus
		statusInput := &configservice.DescribeConfigRuleEvaluationStatusInput{
			ConfigRuleNames: []string{*rule.ConfigRuleName},
		}
		statusResp, err := c.client.DescribeConfigRuleEvaluationStatus(ctx, statusInput)
		if err == nil && len(statusResp.ConfigRulesEvaluationStatus) > 0 {
			status := statusResp.ConfigRulesEvaluationStatus[0]

			// Update creation time if available
			if status.FirstActivatedTime != nil {
				createdTime = *status.FirstActivatedTime
				configRule.CreatedTime = &createdTime
				configRule.LastActivity = &createdTime // Default last activity to creation
			}

			// Update last activity time based on the most recent evaluation
			if status.LastSuccessfulEvaluationTime != nil {
				lastActivity = *status.LastSuccessfulEvaluationTime
				configRule.LastActivity = &lastActivity
			}

			// If error occurred more recently, use that time
			if status.LastFailedEvaluationTime != nil &&
				status.LastFailedEvaluationTime.After(lastActivity) {
				lastActivity = *status.LastFailedEvaluationTime
				configRule.LastActivity = &lastActivity
			}
		}

		// Calculate idle days
		configRule.IdleDays = int(time.Since(lastActivity).Hours() / 24)
		configRule.IsIdle = lastActivity.Before(cutoffTime)

		// 모든 규칙을 추가 (유휴 상태 필터링 제거)
		configRules = append(configRules, configRule)
	}

	return configRules, nil
}

// GetAllConfigRecorders returns a list of models.ConfigRecorderInfo objects representing Config recorders
func (c *ConfigClient) GetAllConfigRecorders() ([]models.ConfigRecorderInfo, error) {
	ctx := context.Background()
	var recorders []models.ConfigRecorderInfo

	input := &configservice.DescribeConfigurationRecordersInput{}
	resp, err := c.client.DescribeConfigurationRecorders(ctx, input)
	if err != nil {
		return nil, err
	}

	statusInput := &configservice.DescribeConfigurationRecorderStatusInput{}
	statusResp, err := c.client.DescribeConfigurationRecorderStatus(ctx, statusInput)
	if err != nil {
		return nil, err
	}

	// Map status by recorder name
	statusMap := make(map[string]types.ConfigurationRecorderStatus)
	for _, status := range statusResp.ConfigurationRecordersStatus {
		if status.Name != nil {
			statusMap[*status.Name] = status
		}
	}

	for _, recorder := range resp.ConfigurationRecorders {
		if recorder.Name == nil {
			continue
		}

		configRecorder := models.ConfigRecorderInfo{
			RecorderName:     *recorder.Name,
			RecorderID:       *recorder.Name,
			Region:           c.region,
			IsRecording:      false,
			AllResourceTypes: false,
		}

		// Get the recording status
		if recorder.RecordingGroup != nil {
			configRecorder.AllResourceTypes = recorder.RecordingGroup.AllSupported
			configRecorder.ResourceCount = len(recorder.RecordingGroup.ResourceTypes)
		}

		// Get status details if available
		var lastActivity time.Time
		lastActivitySet := false
		if status, ok := statusMap[*recorder.Name]; ok {
			if status.LastStartTime != nil {
				lastActivity = *status.LastStartTime
				lastActivitySet = true
			}
			if status.LastStopTime != nil &&
				(status.LastStopTime.After(lastActivity) || !lastActivitySet) {
				lastActivity = *status.LastStopTime
				lastActivitySet = true
			}

			// Recording status is a boolean in SDK v2
			configRecorder.IsRecording = status.Recording
		}

		// Set last activity and idle status if we have timing data
		if lastActivitySet {
			activityTime := lastActivity
			configRecorder.LastActivity = &activityTime
			configRecorder.IdleDays = int(time.Since(lastActivity).Hours() / 24)
			configRecorder.IsIdle = configRecorder.IdleDays > 90
		}

		// 모든 레코더 추가 (유휴 상태 필터링 제거)
		recorders = append(recorders, configRecorder)
	}

	return recorders, nil
}

// GetAllConfigDeliveryChannels returns a list of models.ConfigDeliveryChannelInfo objects
func (c *ConfigClient) GetAllConfigDeliveryChannels() ([]models.ConfigDeliveryChannelInfo, error) {
	ctx := context.Background()
	var channels []models.ConfigDeliveryChannelInfo

	input := &configservice.DescribeDeliveryChannelsInput{}
	resp, err := c.client.DescribeDeliveryChannels(ctx, input)
	if err != nil {
		return nil, err
	}

	for _, channel := range resp.DeliveryChannels {
		if channel.Name == nil {
			continue
		}

		deliveryChannel := models.ConfigDeliveryChannelInfo{
			ChannelName: *channel.Name,
			ChannelID:   *channel.Name,
			Region:      c.region,
		}

		if channel.S3BucketName != nil {
			deliveryChannel.S3BucketName = *channel.S3BucketName
		}
		if channel.SnsTopicARN != nil {
			deliveryChannel.SNSTopicARN = *channel.SnsTopicARN
		}

		// Set frequency if available
		if channel.ConfigSnapshotDeliveryProperties != nil &&
			channel.ConfigSnapshotDeliveryProperties.DeliveryFrequency != "" {
			deliveryChannel.Frequency = string(channel.ConfigSnapshotDeliveryProperties.DeliveryFrequency)
		}

		// Get the delivery channel status for last activity time
		statusInput := &configservice.DescribeDeliveryChannelStatusInput{
			DeliveryChannelNames: []string{*channel.Name},
		}
		statusResp, err := c.client.DescribeDeliveryChannelStatus(ctx, statusInput)
		if err == nil && len(statusResp.DeliveryChannelsStatus) > 0 {
			var lastActivity time.Time
			lastActivitySet := false

			if statusResp.DeliveryChannelsStatus[0].ConfigHistoryDeliveryInfo != nil &&
				statusResp.DeliveryChannelsStatus[0].ConfigHistoryDeliveryInfo.LastSuccessfulTime != nil {
				lastActivity = *statusResp.DeliveryChannelsStatus[0].ConfigHistoryDeliveryInfo.LastSuccessfulTime
				lastActivitySet = true
			}

			if statusResp.DeliveryChannelsStatus[0].ConfigStreamDeliveryInfo != nil &&
				statusResp.DeliveryChannelsStatus[0].ConfigStreamDeliveryInfo.LastStatusChangeTime != nil &&
				(statusResp.DeliveryChannelsStatus[0].ConfigStreamDeliveryInfo.LastStatusChangeTime.After(lastActivity) ||
					!lastActivitySet) {
				lastActivity = *statusResp.DeliveryChannelsStatus[0].ConfigStreamDeliveryInfo.LastStatusChangeTime
				lastActivitySet = true
			}

			// Set last activity and determine if idle
			if lastActivitySet {
				activityTime := lastActivity
				deliveryChannel.LastActivity = &activityTime
				deliveryChannel.IdleDays = int(time.Since(lastActivity).Hours() / 24)
				deliveryChannel.IsIdle = deliveryChannel.IdleDays > 90 // Default 90 days for idle
			}
		}

		// 모든 전송 채널 추가
		channels = append(channels, deliveryChannel)
	}

	return channels, nil
}

// GetAllConfigResources retrieves all AWS Config resources across all regions
func GetAllConfigResources(regions []string, idleDays int) ([]models.ConfigRuleInfo, []models.ConfigRecorderInfo, []models.ConfigDeliveryChannelInfo, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allRules []models.ConfigRuleInfo
	var allRecorders []models.ConfigRecorderInfo
	var allChannels []models.ConfigDeliveryChannelInfo
	var errs []error

	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()

			client, err := NewConfigClient(region)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to load AWS config for region %s: %w", region, err))
				mu.Unlock()
				return
			}

			// Get all Config rules
			rules, err := client.GetAllConfigRules()
			if err == nil {
				mu.Lock()
				allRules = append(allRules, rules...)
				mu.Unlock()
			} else {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}

			// Get all Config recorders
			recorders, err := client.GetAllConfigRecorders()
			if err == nil {
				mu.Lock()
				allRecorders = append(allRecorders, recorders...)
				mu.Unlock()
			} else {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}

			// Get all delivery channels
			channels, err := client.GetAllConfigDeliveryChannels()
			if err == nil {
				mu.Lock()
				allChannels = append(allChannels, channels...)
				mu.Unlock()
			} else {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(region)
	}

	wg.Wait()

	// If we got some results but had some errors, we'll still return what we found
	if len(allRules) > 0 || len(allRecorders) > 0 || len(allChannels) > 0 {
		return allRules, allRecorders, allChannels, nil
	}

	// If we have no results and have errors, return the first error
	if len(errs) > 0 {
		return nil, nil, nil, errs[0]
	}

	return allRules, allRecorders, allChannels, nil
}

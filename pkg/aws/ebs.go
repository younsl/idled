package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/pricing"
	"github.com/younsl/idled/pkg/utils"
)

// EBSClient struct for EBS client
type EBSClient struct {
	client *ec2.Client
	region string
}

// NewEBSClient creates a new EBSClient
func NewEBSClient(region string) (*EBSClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	return &EBSClient{
		client: client,
		region: region,
	}, nil
}

// GetAvailableVolumes returns a list of all EBS volumes in Available state
func (c *EBSClient) GetAvailableVolumes() ([]models.VolumeInfo, error) {
	// Filter only volumes in 'available' state (unattached volumes)
	filter := types.Filter{
		Name:   aws.String("status"),
		Values: []string{"available"},
	}

	input := &ec2.DescribeVolumesInput{
		Filters: []types.Filter{filter},
	}

	result, err := c.client.DescribeVolumes(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("error querying EBS volumes: %w", err)
	}

	volumes := []models.VolumeInfo{}

	for _, volume := range result.Volumes {
		// Extract volume name
		name := utils.GetName(volume.Tags)

		// Get last attachment time
		var lastAttachmentTime *time.Time
		var elapsedDays int

		if len(volume.Attachments) > 0 {
			for _, attachment := range volume.Attachments {
				if attachment.AttachTime != nil {
					if lastAttachmentTime == nil || attachment.AttachTime.After(*lastAttachmentTime) {
						lastAttachmentTime = attachment.AttachTime
					}
				}
			}
		}

		// Calculate elapsed days if last attachment time is available
		if lastAttachmentTime != nil {
			elapsedDays = utils.CalculateElapsedDays(*lastAttachmentTime)
		} else if volume.CreateTime != nil {
			// If no attachment history, use creation time
			lastAttachmentTime = volume.CreateTime
			elapsedDays = utils.CalculateElapsedDays(*volume.CreateTime)
		}

		// Calculate cost estimates
		volumeType := string(volume.VolumeType)
		volumeSizeGB := int(*volume.Size)

		// Determine savings based on time since last use
		monthlyCost, pricingSource := pricing.CalculateEBSMonthlyCostWithSource(volumeType, volumeSizeGB, c.region)
		savings := pricing.CalculateEBSSavings(volumeType, volumeSizeGB, c.region, elapsedDays)

		volumeInfo := models.VolumeInfo{
			VolumeID:             *volume.VolumeId,
			Name:                 name,
			Size:                 volumeSizeGB,
			VolumeType:           volumeType,
			State:                string(volume.State),
			Region:               c.region,
			AvailabilityZone:     *volume.AvailabilityZone,
			CreationTime:         *volume.CreateTime,
			LastAttachmentTime:   lastAttachmentTime,
			ElapsedDaysSinceUsed: elapsedDays,
			EstimatedMonthlyCost: monthlyCost,
			EstimatedSavings:     savings,
			PricingSource:        pricingSource,
		}

		volumes = append(volumes, volumeInfo)
	}

	return volumes, nil
}

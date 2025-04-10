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

// EC2Client struct for EC2 client
type EC2Client struct {
	client *ec2.Client
	region string
}

// NewEC2Client creates a new EC2Client
func NewEC2Client(region string) (*EC2Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	return &EC2Client{
		client: client,
		region: region,
	}, nil
}

// GetStoppedInstances returns a list of all EC2 instances in Stopped state
func (c *EC2Client) GetStoppedInstances() ([]models.InstanceInfo, error) {
	// Filter only stopped instances
	filter := types.Filter{
		Name:   aws.String("instance-state-name"),
		Values: []string{"stopped"},
	}

	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{filter},
	}

	result, err := c.client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("error querying EC2 instances: %w", err)
	}

	instances := []models.InstanceInfo{}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			// Extract instance name
			name := utils.GetName(instance.Tags)

			// Calculate stop time (extract from StateTransitionReason)
			var stoppedTime *time.Time
			var elapsedDays int

			if instance.StateTransitionReason != nil && len(*instance.StateTransitionReason) > 0 {
				stoppedTime = utils.ParseStateTransitionTime(*instance.StateTransitionReason)
				if stoppedTime != nil {
					elapsedDays = utils.CalculateElapsedDays(*stoppedTime)
				}
			}

			// Calculate cost estimates
			instanceType := string(instance.InstanceType)
			monthlyCost, pricingSource := pricing.CalculateMonthlyCostWithSource(instanceType, c.region)
			savings, _ := pricing.CalculateSavingsWithSource(instanceType, c.region, elapsedDays)

			instanceInfo := models.InstanceInfo{
				InstanceID:           *instance.InstanceId,
				Name:                 name,
				InstanceType:         instanceType,
				Region:               c.region,
				AvailabilityZone:     *instance.Placement.AvailabilityZone,
				StoppedTime:          stoppedTime,
				LaunchTime:           *instance.LaunchTime,
				ElapsedDays:          elapsedDays,
				EstimatedMonthlyCost: monthlyCost,
				EstimatedSavings:     savings,
				PricingSource:        pricingSource,
			}

			instances = append(instances, instanceInfo)
		}
	}

	return instances, nil
}

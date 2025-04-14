package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// EIPClient struct for Elastic IP client
type EIPClient struct {
	client *ec2.Client
	region string
}

// NewEIPClient creates a new EIPClient
func NewEIPClient(region string) (*EIPClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	return &EIPClient{
		client: client,
		region: region,
	}, nil
}

// GetUnattachedEIPs returns a list of all Elastic IPs that are not attached to running instances
func (c *EIPClient) GetUnattachedEIPs() ([]models.EIPInfo, error) {
	input := &ec2.DescribeAddressesInput{}

	result, err := c.client.DescribeAddresses(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("error querying Elastic IPs: %w", err)
	}

	eips := []models.EIPInfo{}

	for _, eip := range result.Addresses {
		// Check if the EIP is not associated with any resources
		isUnattached := eip.AssociationId == nil || *eip.AssociationId == ""

		// We're only interested in unattached EIPs
		if !isUnattached {
			continue
		}

		// Set fixed cost for EIPs - currently AWS charges about $0.005 per hour ($3.60 per month) for unused EIPs
		monthlyCost := 3.60 // Fixed monthly cost for an unused EIP

		eipInfo := models.EIPInfo{
			AllocationID:         *eip.AllocationId,
			PublicIP:             *eip.PublicIp,
			AssociationID:        utils.SafeDeref(eip.AssociationId),
			AssociationState:     "Unattached",
			InstanceID:           utils.SafeDeref(eip.InstanceId),
			NetworkInterfaceID:   utils.SafeDeref(eip.NetworkInterfaceId),
			Region:               c.region,
			EstimatedMonthlyCost: monthlyCost,
			PricingSource:        "Fixed", // EIP pricing is fixed
		}

		eips = append(eips, eipInfo)
	}

	return eips, nil
}

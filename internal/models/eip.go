package models

// EIPInfo represents Elastic IP address information
type EIPInfo struct {
	AllocationID         string
	PublicIP             string
	AssociationID        string
	AssociationState     string
	InstanceID           string
	NetworkInterfaceID   string
	Region               string
	EstimatedMonthlyCost float64
	PricingSource        string // "API", "Cache", or "Fixed"
}

package models

import "time"

// VolumeInfo represents EBS volume information
type VolumeInfo struct {
	VolumeID             string
	Name                 string
	Size                 int
	VolumeType           string
	State                string
	Region               string
	AvailabilityZone     string
	CreationTime         time.Time
	LastAttachmentTime   *time.Time
	ElapsedDaysSinceUsed int
	EstimatedMonthlyCost float64
	EstimatedSavings     float64
	PricingSource        string // "API", "Cache", or "Default"
}

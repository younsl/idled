package models

import "time"

// InstanceInfo represents EC2 instance information
type InstanceInfo struct {
	InstanceID           string
	Name                 string
	InstanceType         string
	Region               string
	AvailabilityZone     string
	StoppedTime          *time.Time
	LaunchTime           time.Time
	ElapsedDays          int
	EstimatedMonthlyCost float64
	EstimatedSavings     float64
	PricingSource        string // "API", "Cache", or "N/A"
}

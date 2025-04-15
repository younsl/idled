package models

import "time"

// ELBResource holds information about an idle Elastic Load Balancer
type ELBResource struct {
	Name                 string
	Type                 string // ALB, NLB
	Region               string
	State                string // active, idle
	CreatedTime          time.Time
	ARN                  string
	HealthyTargetCount   int      // Renamed from TargetCount
	UnhealthyTargetCount int      // Added for unhealthy count
	IdleReason           string   // Reason why it's considered idle (e.g., No targets, Low traffic)
	LastActivitySum      *float64 // Sum of relevant CloudWatch metric over the check period (e.g., 14 days)
}

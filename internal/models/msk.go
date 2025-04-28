package models

import (
	"time"
)

// MskClusterInfo holds information about an MSK cluster
type MskClusterInfo struct {
	ClusterName       string    `header:"Cluster Name"`
	ARN               string    `header:"ARN"`
	Region            string    `header:"Region"`
	State             string    `header:"State"`
	InstanceType      string    `header:"Instance Type"`
	CreationTime      time.Time `header:"Creation Time"`
	IsIdle            bool      `header:"Is Idle"`
	Reason            string    `header:"Reason"`                // "No Connections", "Low CPU Usage", "No Conn & Low CPU"
	ConnectionCount   *float64  `header:"Max Connections (30d)"` // Max connection count over the check period
	AvgCPUUtilization *float64  `header:"Avg CPU (30d %)"`       // Average CPU Utilization over check period
}

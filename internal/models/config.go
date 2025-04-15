package models

import (
	"time"
)

// ConfigRuleInfo holds information about an AWS Config rule
type ConfigRuleInfo struct {
	// Basic information
	RuleName       string
	RuleID         string
	ARN            string
	CreatedTime    *time.Time
	LastUpdateTime *time.Time
	Region         string

	// Status information
	IsActive       bool
	IsCustom       bool
	IsCompliant    bool
	EvaluationMode string

	// Idle detection
	IdleDays     int
	IsIdle       bool
	LastActivity *time.Time
}

// ConfigRecorderInfo holds information about an AWS Config recorder
type ConfigRecorderInfo struct {
	// Basic information
	RecorderName   string
	RecorderID     string
	Region         string
	CreatedTime    *time.Time
	LastUpdateTime *time.Time

	// Configuration
	AllResourceTypes bool
	ResourceCount    int
	IsRecording      bool

	// Idle detection
	IdleDays     int
	IsIdle       bool
	LastActivity *time.Time
}

// ConfigDeliveryChannelInfo holds information about a Config delivery channel
type ConfigDeliveryChannelInfo struct {
	// Basic information
	ChannelName string
	ChannelID   string
	Region      string
	CreatedTime *time.Time

	// Configuration
	S3BucketName string
	SNSTopicARN  string
	Frequency    string

	// Idle detection
	IdleDays     int
	IsIdle       bool
	LastActivity *time.Time
}

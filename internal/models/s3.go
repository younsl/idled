package models

import "time"

// BucketInfo represents S3 bucket information with idle detection metrics
type BucketInfo struct {
	BucketName   string
	Region       string
	CreationTime time.Time
	ObjectCount  int64
	TotalSize    int64 // in bytes

	// Activity metrics
	LastModified *time.Time // Last object modification time
	LastAccessed *time.Time // Last access time (if logging enabled)

	// Activity change metrics
	ObjectCountChange int64 // Object count change over specified period
	SizeChange        int64 // Size change over specified period

	// API call statistics
	GetRequestsLast30Days int64 // GetObject requests in last 30 days
	PutRequestsLast30Days int64 // PutObject requests in last 30 days

	// Idle detection
	IsEmpty  bool // True if bucket has no objects
	IsIdle   bool // True if classified as idle based on criteria
	IdleDays int  // Number of days the bucket has been idle

	// Additional information
	HasWebsiteConfig     bool // True if bucket has website configuration
	HasBucketPolicy      bool // True if bucket has a policy
	HasEventNotification bool // True if bucket has event notifications
}

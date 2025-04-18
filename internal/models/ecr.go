package models

import "time"

// RepositoryInfo holds information about an ECR repository
type RepositoryInfo struct {
	Name       string
	Region     string
	ARN        string
	URI        string
	LastPush   *time.Time // Pointer to handle cases where no images are pushed
	CreatedAt  *time.Time
	Idle       bool
	ImageCount int // Add field for image count
}

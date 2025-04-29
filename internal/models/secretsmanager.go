package models

import "time"

// SecretInfo holds information about an AWS Secrets Manager secret.
type SecretInfo struct {
	ARN              string    `json:"arn"`
	Name             string    `json:"name"`
	Region           string    `json:"region"`
	LastAccessedDate time.Time `json:"lastAccessedDate"`
	IdleDays         int       `json:"idleDays"`
}

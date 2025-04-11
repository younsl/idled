package models

import "time"

// LambdaFunctionInfo represents information about a Lambda function
type LambdaFunctionInfo struct {
	FunctionName          string     // Lambda function name
	Description           string     // Function description (if available)
	Runtime               string     // Runtime (e.g., nodejs16.x, python3.9)
	Region                string     // AWS region
	MemorySize            int32      // Memory allocation in MB
	Timeout               int32      // Function timeout in seconds
	LastModified          *time.Time // Last modification time
	LastInvocation        *time.Time // Last invocation time (from CloudWatch)
	InvocationsLast30Days int64      // Number of invocations in last 30 days
	ErrorsLast30Days      int64      // Number of errors in last 30 days
	DurationP95Last30Days float64    // 95th percentile duration in milliseconds
	IsIdle                bool       // Whether the function is considered idle
	IdleDays              int        // Days since last invocation
	EstimatedMonthlyCost  float64    // Estimated monthly cost
}

package models

import "time"

// IAMUserInfo represents information about an IAM user
type IAMUserInfo struct {
	UserName              string     // IAM user name
	UserID                string     // IAM user ID
	ARN                   string     // Full ARN of the user
	Region                string     // AWS region (global for IAM)
	Path                  string     // Path to the user
	CreateDate            *time.Time // When the user was created
	PasswordLastUsed      *time.Time // When the password was last used for console login
	AccessKeysLastUsed    *time.Time // The most recent access key usage timestamp
	AccessKeyCount        int        // Number of access keys associated with the user
	LastActivity          *time.Time // The most recent activity timestamp (login or API call)
	IsIdle                bool       // Whether the user is considered idle
	IdleDays              int        // Days since last activity
	HasActiveAccessKeys   bool       // Whether the user has active access keys
	HasMFAEnabled         bool       // Whether MFA is enabled for the user
	HasInlinePolicies     bool       // Whether the user has inline policies
	AttachedPolicyCount   int        // Number of managed policies attached to the user
	UnusedPermissionsInfo []string   // Information about unused permissions
}

// IAMRoleInfo represents information about an IAM role
type IAMRoleInfo struct {
	RoleName              string     // IAM role name
	RoleID                string     // IAM role ID
	ARN                   string     // Full ARN of the role
	Region                string     // AWS region (global for IAM)
	Path                  string     // Path to the role
	CreateDate            *time.Time // When the role was created
	LastUsed              *time.Time // When the role was last assumed
	LastActivity          *time.Time // The most recent activity timestamp
	IsIdle                bool       // Whether the role is considered idle
	IdleDays              int        // Days since last activity
	IsServiceLinkedRole   bool       // Whether this is a service-linked role
	IsCrossAccountRole    bool       // Whether this role can be assumed by other accounts
	TrustPolicy           string     // Summary of the trust policy
	AttachedPolicyCount   int        // Number of managed policies attached to the role
	HasInlinePolicies     bool       // Whether the role has inline policies
	UnusedPermissionsInfo []string   // Information about unused permissions
}

// IAMPolicyInfo represents information about an IAM policy
type IAMPolicyInfo struct {
	PolicyName         string     // IAM policy name
	PolicyID           string     // IAM policy ID
	ARN                string     // Full ARN of the policy
	Region             string     // AWS region (global for IAM)
	Path               string     // Path to the policy
	CreateDate         *time.Time // When the policy was created
	UpdateDate         *time.Time // When the policy was last updated
	LastAccessed       *time.Time // When the policy was last accessed
	IsIdle             bool       // Whether the policy is considered idle
	IdleDays           int        // Days since last activity
	IsAWSManaged       bool       // Whether this is an AWS managed policy
	IsAttached         bool       // Whether this policy is attached to any entities
	AttachmentCount    int        // Number of entities this policy is attached to
	VersionCount       int        // Number of versions this policy has
	DefaultVersion     string     // Default version of the policy
	UsedServiceCount   int        // Number of services used through this policy
	UnusedServiceCount int        // Number of services granted but not used
}

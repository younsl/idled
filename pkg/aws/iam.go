package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/briandowns/spinner"
	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/utils"
)

// IAMClient struct for IAM client
type IAMClient struct {
	client        *iam.Client
	region        string
	idleThreshold int // in days
}

// NewIAMClient creates a new IAMClient
func NewIAMClient(region string) (*IAMClient, error) {
	// IAM is a global service but we maintain region for consistency with other clients
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	client := iam.NewFromConfig(cfg)

	return &IAMClient{
		client:        client,
		region:        region,
		idleThreshold: 90, // Default: consider IAM resources idle after 90 days of inactivity
	}, nil
}

// SetIdleThreshold sets the threshold in days for considering IAM resources as idle
func (c *IAMClient) SetIdleThreshold(days int) {
	c.idleThreshold = days
}

// GetIdleUsers returns a list of IAM users with their usage metrics and idle status
func (c *IAMClient) GetIdleUsers() ([]models.IAMUserInfo, error) {
	// Create spinner for progress indication
	sp := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Prefix = "Scanning IAM users "
	sp.Suffix = " (this is a global service)"
	sp.Start()

	// List all IAM users
	var users []types.User
	var marker *string

	for {
		input := &iam.ListUsersInput{
			Marker: marker,
		}

		result, err := c.client.ListUsers(context.TODO(), input)
		if err != nil {
			sp.Stop()
			return nil, fmt.Errorf("error listing IAM users: %w", err)
		}

		users = append(users, result.Users...)

		if !result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	totalUsers := len(users)
	sp.FinalMSG = fmt.Sprintf("✓ Found %d IAM users\n", totalUsers)
	sp.Stop()

	if totalUsers == 0 {
		return []models.IAMUserInfo{}, nil
	}

	// Process each user
	var userInfos []models.IAMUserInfo

	// Create a new spinner for analyzing users
	sp = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Prefix = "Analyzing IAM users activity and permissions "
	sp.Start()

	processedCount := 0
	for _, user := range users {
		userName := *user.UserName

		// Get user info
		userInfo, err := c.analyzeUser(user)
		if err != nil {
			fmt.Printf("Warning: Error analyzing user %s: %v\n", userName, err)
			continue
		}

		userInfos = append(userInfos, userInfo)
		processedCount++

		// Update progress
		percentage := (processedCount * 100) / totalUsers
		sp.Suffix = fmt.Sprintf(" (%d/%d, %d%%)", processedCount, totalUsers, percentage)
	}

	sp.FinalMSG = fmt.Sprintf("✓ Completed analysis of %d IAM users\n", processedCount)
	sp.Stop()

	return userInfos, nil
}

// GetIdleRoles returns a list of IAM roles with their usage metrics and idle status
func (c *IAMClient) GetIdleRoles() ([]models.IAMRoleInfo, error) {
	// Create spinner for progress indication
	sp := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Prefix = "Scanning IAM roles "
	sp.Suffix = " (this is a global service)"
	sp.Start()

	// List all IAM roles
	var roles []types.Role
	var marker *string

	for {
		input := &iam.ListRolesInput{
			Marker: marker,
		}

		result, err := c.client.ListRoles(context.TODO(), input)
		if err != nil {
			sp.Stop()
			return nil, fmt.Errorf("error listing IAM roles: %w", err)
		}

		roles = append(roles, result.Roles...)

		if !result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	totalRoles := len(roles)
	sp.FinalMSG = fmt.Sprintf("✓ Found %d IAM roles\n", totalRoles)
	sp.Stop()

	if totalRoles == 0 {
		return []models.IAMRoleInfo{}, nil
	}

	// Process each role
	var roleInfos []models.IAMRoleInfo

	// Create a new spinner for analyzing roles
	sp = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Prefix = "Analyzing IAM roles activity and permissions "
	sp.Start()

	processedCount := 0
	for _, role := range roles {
		roleName := *role.RoleName

		// Get role info
		roleInfo, err := c.analyzeRole(role)
		if err != nil {
			fmt.Printf("Warning: Error analyzing role %s: %v\n", roleName, err)
			continue
		}

		roleInfos = append(roleInfos, roleInfo)
		processedCount++

		// Update progress
		percentage := (processedCount * 100) / totalRoles
		sp.Suffix = fmt.Sprintf(" (%d/%d, %d%%)", processedCount, totalRoles, percentage)
	}

	sp.FinalMSG = fmt.Sprintf("✓ Completed analysis of %d IAM roles\n", processedCount)
	sp.Stop()

	return roleInfos, nil
}

// GetIdlePolicies returns a list of IAM policies with their usage metrics and idle status
func (c *IAMClient) GetIdlePolicies() ([]models.IAMPolicyInfo, error) {
	// Create spinner for progress indication
	sp := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Prefix = "Scanning IAM policies "
	sp.Suffix = " (this is a global service)"
	sp.Start()

	// List all customer managed IAM policies
	var policies []types.Policy
	var marker *string

	for {
		input := &iam.ListPoliciesInput{
			Marker:       marker,
			Scope:        types.PolicyScopeTypeLocal, // Only customer managed policies
			OnlyAttached: false,                      // Include non-attached policies
		}

		result, err := c.client.ListPolicies(context.TODO(), input)
		if err != nil {
			sp.Stop()
			return nil, fmt.Errorf("error listing IAM policies: %w", err)
		}

		policies = append(policies, result.Policies...)

		if !result.IsTruncated {
			break
		}
		marker = result.Marker
	}

	totalPolicies := len(policies)
	sp.FinalMSG = fmt.Sprintf("✓ Found %d customer managed IAM policies\n", totalPolicies)
	sp.Stop()

	if totalPolicies == 0 {
		return []models.IAMPolicyInfo{}, nil
	}

	// Process each policy
	var policyInfos []models.IAMPolicyInfo

	// Create a new spinner for analyzing policies
	sp = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	sp.Prefix = "Analyzing IAM policies usage and attachment "
	sp.Start()

	processedCount := 0
	for _, policy := range policies {
		policyName := *policy.PolicyName

		// Get policy info
		policyInfo, err := c.analyzePolicy(policy)
		if err != nil {
			fmt.Printf("Warning: Error analyzing policy %s: %v\n", policyName, err)
			continue
		}

		policyInfos = append(policyInfos, policyInfo)
		processedCount++

		// Update progress
		percentage := (processedCount * 100) / totalPolicies
		sp.Suffix = fmt.Sprintf(" (%d/%d, %d%%)", processedCount, totalPolicies, percentage)
	}

	sp.FinalMSG = fmt.Sprintf("✓ Completed analysis of %d IAM policies\n", processedCount)
	sp.Stop()

	return policyInfos, nil
}

// analyzeUser gathers information about a single IAM user
func (c *IAMClient) analyzeUser(user types.User) (models.IAMUserInfo, error) {
	ctx := context.TODO()
	userName := *user.UserName

	// Initialize with basic information
	userInfo := models.IAMUserInfo{
		UserName:   userName,
		ARN:        *user.Arn,
		Region:     "global", // IAM is a global service
		CreateDate: user.CreateDate,
	}

	if user.UserId != nil {
		userInfo.UserID = *user.UserId
	}

	if user.Path != nil {
		userInfo.Path = *user.Path
	}

	if user.PasswordLastUsed != nil {
		userInfo.PasswordLastUsed = user.PasswordLastUsed
	}

	// Get last activity time (password or access key)
	userInfo.LastActivity = user.CreateDate // Default to creation date

	if user.PasswordLastUsed != nil {
		userInfo.LastActivity = user.PasswordLastUsed
	}

	// Get access keys information
	accessKeys, err := c.client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: &userName,
	})
	if err == nil && accessKeys != nil {
		userInfo.AccessKeyCount = len(accessKeys.AccessKeyMetadata)
		userInfo.HasActiveAccessKeys = false

		// Check for active access keys
		for _, key := range accessKeys.AccessKeyMetadata {
			if key.Status == types.StatusTypeActive {
				userInfo.HasActiveAccessKeys = true

				// Get last used information for each access key
				keyLastUsed, err := c.client.GetAccessKeyLastUsed(ctx, &iam.GetAccessKeyLastUsedInput{
					AccessKeyId: key.AccessKeyId,
				})
				if err == nil && keyLastUsed.AccessKeyLastUsed.LastUsedDate != nil {
					lastUsedDate := keyLastUsed.AccessKeyLastUsed.LastUsedDate

					// Update access keys last used time
					if userInfo.AccessKeysLastUsed == nil || lastUsedDate.After(*userInfo.AccessKeysLastUsed) {
						userInfo.AccessKeysLastUsed = lastUsedDate
					}

					// Update last activity if access key was used more recently than password
					if userInfo.LastActivity == nil ||
						(lastUsedDate != nil && lastUsedDate.After(*userInfo.LastActivity)) {
						userInfo.LastActivity = lastUsedDate
					}
				}
			}
		}
	}

	// Check if user has MFA enabled
	mfaDevices, err := c.client.ListMFADevices(ctx, &iam.ListMFADevicesInput{
		UserName: &userName,
	})
	if err == nil && mfaDevices != nil {
		userInfo.HasMFAEnabled = len(mfaDevices.MFADevices) > 0
	}

	// Check for inline policies
	inlinePolicies, err := c.client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
		UserName: &userName,
	})
	if err == nil && inlinePolicies != nil {
		userInfo.HasInlinePolicies = len(inlinePolicies.PolicyNames) > 0
	}

	// Check for attached managed policies
	attachedPolicies, err := c.client.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{
		UserName: &userName,
	})
	if err == nil && attachedPolicies != nil {
		userInfo.AttachedPolicyCount = len(attachedPolicies.AttachedPolicies)
	}

	// Generate service last accessed details
	jobId, err := c.client.GenerateServiceLastAccessedDetails(ctx, &iam.GenerateServiceLastAccessedDetailsInput{
		Arn: &userInfo.ARN,
	})
	if err == nil && jobId != nil {
		// TODO: Implement retrieval of service last accessed details
		// This requires polling until the job is complete
		// For now, we'll skip this part to keep the implementation simpler
	}

	// Determine if user is idle
	if userInfo.LastActivity != nil {
		userInfo.IdleDays = utils.CalculateElapsedDays(*userInfo.LastActivity)
		userInfo.IsIdle = userInfo.IdleDays > c.idleThreshold
	} else {
		// If we couldn't determine last activity, consider the user idle if it's old enough
		if userInfo.CreateDate != nil {
			daysSinceCreation := utils.CalculateElapsedDays(*userInfo.CreateDate)
			userInfo.IdleDays = daysSinceCreation
			userInfo.IsIdle = daysSinceCreation > c.idleThreshold
		}
	}

	return userInfo, nil
}

// analyzeRole gathers information about a single IAM role
func (c *IAMClient) analyzeRole(role types.Role) (models.IAMRoleInfo, error) {
	ctx := context.TODO()
	roleName := *role.RoleName

	// Initialize with basic information
	roleInfo := models.IAMRoleInfo{
		RoleName:     roleName,
		ARN:          *role.Arn,
		Region:       "global", // IAM is a global service
		CreateDate:   role.CreateDate,
		LastActivity: role.CreateDate, // Default to creation date
	}

	if role.RoleId != nil {
		roleInfo.RoleID = *role.RoleId
	}

	if role.Path != nil {
		roleInfo.Path = *role.Path
	}

	// Determine if this is a service-linked role
	roleInfo.IsServiceLinkedRole = false
	if role.Path != nil && *role.Path == "/service-role/" {
		roleInfo.IsServiceLinkedRole = true
	}

	// The AWS SDK v2 has different function naming
	roleLastUsed, err := c.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: &roleName,
	})
	if err == nil && roleLastUsed.Role.RoleLastUsed != nil && roleLastUsed.Role.RoleLastUsed.LastUsedDate != nil {
		roleInfo.LastUsed = roleLastUsed.Role.RoleLastUsed.LastUsedDate
		roleInfo.LastActivity = roleLastUsed.Role.RoleLastUsed.LastUsedDate
	}

	// Check for inline policies
	inlinePolicies, err := c.client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: &roleName,
	})
	if err == nil && inlinePolicies != nil {
		roleInfo.HasInlinePolicies = len(inlinePolicies.PolicyNames) > 0
	}

	// Check for attached managed policies
	attachedPolicies, err := c.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	})
	if err == nil && attachedPolicies != nil {
		roleInfo.AttachedPolicyCount = len(attachedPolicies.AttachedPolicies)
	}

	// Analyze trust policy to detect cross-account access
	if role.AssumeRolePolicyDocument != nil {
		// TODO: Parse and analyze assume role policy document
		// This requires JSON parsing and analysis
		// For now, we'll skip detailed analysis
		roleInfo.TrustPolicy = "Available" // Placeholder

		// Basic check for cross-account access based on document content
		// This is a simple heuristic and may not be accurate in all cases
		policyDoc := *role.AssumeRolePolicyDocument
		roleInfo.IsCrossAccountRole = contains(policyDoc, "arn:aws:iam") && !roleInfo.IsServiceLinkedRole
	}

	// Generate service last accessed details
	jobId, err := c.client.GenerateServiceLastAccessedDetails(ctx, &iam.GenerateServiceLastAccessedDetailsInput{
		Arn: &roleInfo.ARN,
	})
	if err == nil && jobId != nil {
		// TODO: Implement retrieval of service last accessed details
		// This requires polling until the job is complete
		// For now, we'll skip this part to keep the implementation simpler
	}

	// Determine if role is idle
	if roleInfo.LastUsed != nil {
		roleInfo.IdleDays = utils.CalculateElapsedDays(*roleInfo.LastUsed)
		roleInfo.IsIdle = roleInfo.IdleDays > c.idleThreshold
	} else {
		// If we couldn't determine last usage, consider the role idle if it's old enough
		// and not a service-linked role (service-linked roles are special)
		if roleInfo.CreateDate != nil && !roleInfo.IsServiceLinkedRole {
			daysSinceCreation := utils.CalculateElapsedDays(*roleInfo.CreateDate)
			roleInfo.IdleDays = daysSinceCreation
			roleInfo.IsIdle = daysSinceCreation > c.idleThreshold
		}
	}

	return roleInfo, nil
}

// analyzePolicy gathers information about a single IAM policy
func (c *IAMClient) analyzePolicy(policy types.Policy) (models.IAMPolicyInfo, error) {
	ctx := context.TODO()
	policyName := *policy.PolicyName

	// Initialize with basic information
	policyInfo := models.IAMPolicyInfo{
		PolicyName:      policyName,
		ARN:             *policy.Arn,
		Region:          "global", // IAM is a global service
		CreateDate:      policy.CreateDate,
		UpdateDate:      policy.UpdateDate,
		IsAttached:      *policy.AttachmentCount > 0,
		AttachmentCount: int(*policy.AttachmentCount),
		IsAWSManaged:    false, // We're only listing customer managed policies
	}

	if policy.PolicyId != nil {
		policyInfo.PolicyID = *policy.PolicyId
	}

	if policy.Path != nil {
		policyInfo.Path = *policy.Path
	}

	if policy.DefaultVersionId != nil {
		policyInfo.DefaultVersion = *policy.DefaultVersionId
	}

	// Get policy versions
	versions, err := c.client.ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{
		PolicyArn: &policyInfo.ARN,
	})
	if err == nil && versions != nil {
		policyInfo.VersionCount = len(versions.Versions)
	}

	// Generate service last accessed details
	jobId, err := c.client.GenerateServiceLastAccessedDetails(ctx, &iam.GenerateServiceLastAccessedDetailsInput{
		Arn: &policyInfo.ARN,
	})
	if err == nil && jobId != nil {
		// TODO: Implement retrieval of service last accessed details
		// This requires polling until the job is complete
		// For now, we'll skip this part to keep the implementation simpler
	}

	// Determine if policy is idle
	// Policies are considered idle if they're not attached to any entity AND haven't been updated recently
	policyInfo.IsIdle = !policyInfo.IsAttached

	if policyInfo.UpdateDate != nil {
		daysSinceUpdate := utils.CalculateElapsedDays(*policyInfo.UpdateDate)
		policyInfo.IdleDays = daysSinceUpdate
		// Even if it's not attached, don't mark as idle if updated recently
		if daysSinceUpdate <= c.idleThreshold {
			policyInfo.IsIdle = false
		}
	} else if policyInfo.CreateDate != nil {
		daysSinceCreation := utils.CalculateElapsedDays(*policyInfo.CreateDate)
		policyInfo.IdleDays = daysSinceCreation
	}

	return policyInfo, nil
}

// Helper function to check if a string contains a substring
func contains(s string, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

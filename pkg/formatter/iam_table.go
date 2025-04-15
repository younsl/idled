package formatter

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// FormatIAMUserTable writes IAM user information in a table format
func FormatIAMUserTable(writer io.Writer, users []models.IAMUserInfo) {
	if len(users) == 0 {
		fmt.Fprintln(writer, "No IAM users found.")
		return
	}

	// Sort users: idle users first, then by idle days (descending)
	sort.Slice(users, func(i, j int) bool {
		if users[i].IsIdle != users[j].IsIdle {
			return users[i].IsIdle // true comes first
		}
		return users[i].IdleDays > users[j].IdleDays
	})

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "USER NAME\tUSER ID\tAGE (DAYS)\tLAST ACTIVITY\tACCESS KEYS\tMFA\tATTACHED POLICIES\tIDLE\tREGION")

	// Print each user
	for _, user := range users {
		lastActivityStr := "Never"
		if user.LastActivity != nil {
			lastActivityStr = formatDate(*user.LastActivity)
		}

		accessKeysInfo := fmt.Sprintf("%d", user.AccessKeyCount)
		if user.HasActiveAccessKeys {
			accessKeysInfo += " (active)"
		}

		mfaStatus := "No"
		if user.HasMFAEnabled {
			mfaStatus = "Yes"
		}

		idleStatus := "No"
		if user.IsIdle {
			idleStatus = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
			user.UserName,
			user.UserID,
			user.IdleDays,
			lastActivityStr,
			accessKeysInfo,
			mfaStatus,
			user.AttachedPolicyCount,
			idleStatus,
			user.Region,
		)
	}

	w.Flush()

	// Print summary
	idleCount := 0
	for _, user := range users {
		if user.IsIdle {
			idleCount++
		}
	}

	fmt.Fprintf(writer, "\nSummary: %d idle IAM users out of %d total users\n",
		idleCount, len(users))
}

// FormatIAMRoleTable writes IAM role information in a table format
func FormatIAMRoleTable(writer io.Writer, roles []models.IAMRoleInfo) {
	if len(roles) == 0 {
		fmt.Fprintln(writer, "No IAM roles found.")
		return
	}

	// Sort roles: idle roles first, then by idle days (descending)
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].IsIdle != roles[j].IsIdle {
			return roles[i].IsIdle // true comes first
		}
		return roles[i].IdleDays > roles[j].IdleDays
	})

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "ROLE NAME\tROLE ID\tAGE (DAYS)\tLAST USED\tSERVICE LINKED\tCROSS ACCOUNT\tATTACHED POLICIES\tIDLE\tREGION")

	// Print each role
	for _, role := range roles {
		lastUsedStr := "Never"
		if role.LastUsed != nil {
			lastUsedStr = formatDate(*role.LastUsed)
		}

		serviceLinked := "No"
		if role.IsServiceLinkedRole {
			serviceLinked = "Yes"
		}

		crossAccount := "No"
		if role.IsCrossAccountRole {
			crossAccount = "Yes"
		}

		idleStatus := "No"
		if role.IsIdle {
			idleStatus = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
			role.RoleName,
			role.RoleID,
			role.IdleDays,
			lastUsedStr,
			serviceLinked,
			crossAccount,
			role.AttachedPolicyCount,
			idleStatus,
			role.Region,
		)
	}

	w.Flush()

	// Print summary
	idleCount := 0
	serviceLinkedCount := 0
	crossAccountCount := 0

	for _, role := range roles {
		if role.IsIdle {
			idleCount++
		}
		if role.IsServiceLinkedRole {
			serviceLinkedCount++
		}
		if role.IsCrossAccountRole {
			crossAccountCount++
		}
	}

	// 요약 정보 출력
	fmt.Fprintf(writer, "\nSummary: %d idle IAM roles out of %d total roles (%d service-linked, %d cross-account)\n",
		idleCount, len(roles), serviceLinkedCount, crossAccountCount)
}

// FormatIAMPolicyTable writes IAM policy information in a table format
func FormatIAMPolicyTable(writer io.Writer, policies []models.IAMPolicyInfo) {
	if len(policies) == 0 {
		fmt.Fprintln(writer, "No IAM policies found.")
		return
	}

	// Sort policies: idle policies first, then by idle days (descending)
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].IsIdle != policies[j].IsIdle {
			return policies[i].IsIdle // true comes first
		}
		return policies[i].IdleDays > policies[j].IdleDays
	})

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "POLICY NAME\tPOLICY ID\tAGE (DAYS)\tLAST UPDATED\tVERSIONS\tATTACHMENTS\tIDLE\tREGION")

	// Print each policy
	for _, policy := range policies {
		lastUpdatedStr := "Unknown"
		if policy.UpdateDate != nil {
			lastUpdatedStr = formatDate(*policy.UpdateDate)
		}

		idleStatus := "No"
		if policy.IsIdle {
			idleStatus = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t%d\t%s\t%s\n",
			policy.PolicyName,
			policy.PolicyID,
			policy.IdleDays,
			lastUpdatedStr,
			policy.VersionCount,
			policy.AttachmentCount,
			idleStatus,
			policy.Region,
		)
	}

	w.Flush()

	// Print summary
	idleCount := 0
	unattachedCount := 0

	for _, policy := range policies {
		if policy.IsIdle {
			idleCount++
		}
		if policy.AttachmentCount == 0 {
			unattachedCount++
		}
	}

	fmt.Fprintf(writer, "\nSummary: %d idle IAM policies out of %d total policies (%d unattached)\n",
		idleCount, len(policies), unattachedCount)
}

// Helper function to format date
func formatDate(t time.Time) string {
	daysAgo := int(time.Since(t).Hours() / 24)
	if daysAgo < 1 {
		return "Today"
	}
	if daysAgo == 1 {
		return "Yesterday"
	}
	return fmt.Sprintf("%s (%d days ago)", t.Format("2006-01-02"), daysAgo)
}

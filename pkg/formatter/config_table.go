package formatter

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/younsl/idled/internal/models"
)

// FormatConfigRulesTable writes AWS Config rules information in a table format
func FormatConfigRulesTable(writer io.Writer, rules []models.ConfigRuleInfo) {
	if len(rules) == 0 {
		fmt.Fprintln(writer, "No AWS Config rules found.")
		return
	}

	// Sort rules: idle rules first, then by idle days (descending)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].IsIdle != rules[j].IsIdle {
			return rules[i].IsIdle // true comes first
		}
		return rules[i].IdleDays > rules[j].IdleDays
	})

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "RULE NAME\tRULE ID\tCUSTOM\tSTATUS\tCOMPLIANT\tEVALUATION MODE\tLAST ACTIVITY\tIDLE\tREGION")

	// Print each rule
	for _, rule := range rules {
		lastActivityStr := "Never"
		if rule.LastActivity != nil {
			lastActivityStr = formatDate(*rule.LastActivity)
		}

		statusStr := "Inactive"
		if rule.IsActive {
			statusStr = "Active"
		}

		customStr := "No"
		if rule.IsCustom {
			customStr = "Yes"
		}

		compliantStr := "Unknown"
		if rule.IsActive {
			if rule.IsCompliant {
				compliantStr = "Yes"
			} else {
				compliantStr = "No"
			}
		}

		idleStatus := "No"
		if rule.IsIdle {
			idleStatus = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			rule.RuleName,
			rule.RuleID,
			customStr,
			statusStr,
			compliantStr,
			rule.EvaluationMode,
			lastActivityStr,
			idleStatus,
			rule.Region,
		)
	}

	w.Flush()

	// Print summary
	idleCount := 0
	customCount := 0
	inactiveCount := 0

	for _, rule := range rules {
		if rule.IsIdle {
			idleCount++
		}
		if rule.IsCustom {
			customCount++
		}
		if !rule.IsActive {
			inactiveCount++
		}
	}

	fmt.Fprintf(writer, "\nSummary: %d idle AWS Config rules out of %d total rules (%d custom, %d inactive)\n",
		idleCount, len(rules), customCount, inactiveCount)
}

// FormatConfigRecordersTable writes AWS Config recorders information in a table format
func FormatConfigRecordersTable(writer io.Writer, recorders []models.ConfigRecorderInfo) {
	if len(recorders) == 0 {
		fmt.Fprintln(writer, "No AWS Config recorders found.")
		return
	}

	// Sort recorders: idle recorders first, then by idle days (descending)
	sort.Slice(recorders, func(i, j int) bool {
		if recorders[i].IsIdle != recorders[j].IsIdle {
			return recorders[i].IsIdle // true comes first
		}
		return recorders[i].IdleDays > recorders[j].IdleDays
	})

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "RECORDER NAME\tSTATUS\tRESOURCE COVERAGE\tLAST ACTIVITY\tIDLE DAYS\tIDLE\tREGION")

	// Print each recorder
	for _, recorder := range recorders {
		lastActivityStr := "Never"
		if recorder.LastActivity != nil {
			lastActivityStr = formatDate(*recorder.LastActivity)
		}

		statusStr := "Not Recording"
		if recorder.IsRecording {
			statusStr = "Recording"
		}

		resourceCoverageStr := fmt.Sprintf("%d resources", recorder.ResourceCount)
		if recorder.AllResourceTypes {
			resourceCoverageStr = "All resources"
		}

		idleStatus := "No"
		if recorder.IsIdle {
			idleStatus = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			recorder.RecorderName,
			statusStr,
			resourceCoverageStr,
			lastActivityStr,
			recorder.IdleDays,
			idleStatus,
			recorder.Region,
		)
	}

	w.Flush()

	// Print summary
	idleCount := 0
	notRecordingCount := 0

	for _, recorder := range recorders {
		if recorder.IsIdle {
			idleCount++
		}
		if !recorder.IsRecording {
			notRecordingCount++
		}
	}

	fmt.Fprintf(writer, "\nSummary: %d idle AWS Config recorders out of %d total recorders (%d not recording)\n",
		idleCount, len(recorders), notRecordingCount)
}

// FormatConfigDeliveryChannelsTable writes AWS Config delivery channels information in a table format
func FormatConfigDeliveryChannelsTable(writer io.Writer, channels []models.ConfigDeliveryChannelInfo) {
	if len(channels) == 0 {
		fmt.Fprintln(writer, "No AWS Config delivery channels found.")
		return
	}

	// Sort channels: idle channels first, then by idle days (descending)
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].IsIdle != channels[j].IsIdle {
			return channels[i].IsIdle // true comes first
		}
		return channels[i].IdleDays > channels[j].IdleDays
	})

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "CHANNEL NAME\tS3 BUCKET\tSNS TOPIC\tFREQUENCY\tLAST ACTIVITY\tIDLE DAYS\tIDLE\tREGION")

	// Print each channel
	for _, channel := range channels {
		lastActivityStr := "Never"
		if channel.LastActivity != nil {
			lastActivityStr = formatDate(*channel.LastActivity)
		}

		snsTopicStr := "-"
		if channel.SNSTopicARN != "" {
			snsTopicStr = channel.SNSTopicARN
		}

		frequencyStr := "Not set"
		if channel.Frequency != "" {
			frequencyStr = channel.Frequency
		}

		idleStatus := "No"
		if channel.IsIdle {
			idleStatus = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			channel.ChannelName,
			channel.S3BucketName,
			snsTopicStr,
			frequencyStr,
			lastActivityStr,
			channel.IdleDays,
			idleStatus,
			channel.Region,
		)
	}

	w.Flush()

	// Print summary
	idleCount := 0
	for _, channel := range channels {
		if channel.IsIdle {
			idleCount++
		}
	}

	fmt.Fprintf(writer, "\nSummary: %d idle AWS Config delivery channels out of %d total delivery channels\n",
		idleCount, len(channels))
}

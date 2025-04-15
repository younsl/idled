package formatter

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

const (
	elbHeader = "NAME\tTYPE\tREGION\tSTATE\tCREATED\tARN\tTG(H/U)\tTRAFFIC (14d)\tIDLE REASON"
	elbFormat = "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n"
)

// PrintELBTable prints the idle ELB results in a table format using tabwriter
func PrintELBTable(w io.Writer, elbs []models.ELBResource) {
	if len(elbs) == 0 {
		fmt.Fprintln(w, "No idle Elastic Load Balancers found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0) // minwidth, tabwidth, padding, padchar, flags
	fmt.Fprintln(tw, elbHeader)

	for _, elb := range elbs {
		createdStr := elb.CreatedTime.Format(time.RFC3339)

		// Format LastActivitySum nicely
		lastActivityStr := "N/A"
		if elb.LastActivitySum != nil {
			lastActivityStr = fmt.Sprintf("%.2f", *elb.LastActivitySum)
		}

		// Format targets as H/U
		targetsStr := fmt.Sprintf("%d/%d", elb.HealthyTargetCount, elb.UnhealthyTargetCount)

		fmt.Fprintf(tw, elbFormat,
			elb.Name,
			elb.Type,
			elb.Region,
			elb.State,
			createdStr,
			elb.ARN,
			targetsStr, // Use H/U formatted string
			lastActivityStr,
			elb.IdleReason,
		)
	}

	tw.Flush()
}

// PrintELBSummary prints a summary of the ELB scan results
func PrintELBSummary(w io.Writer, elbs []models.ELBResource) {
	// Optionally add a summary, similar to other resources if needed
	// For now, keep it simple.
	if len(elbs) > 0 {
		fmt.Fprintf(w, "\nFound %d idle Elastic Load Balancers.\n", len(elbs))
		fmt.Fprintf(w, "Idle Reason indicates why an ELB is considered idle (e.g., no healthy targets or zero traffic over 14 days).\n")
	}
}

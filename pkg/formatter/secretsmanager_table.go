package formatter

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// PrintSecretsTable prints the idle Secrets Manager secret information in a table format.
func PrintSecretsTable(secrets []models.SecretInfo, scanStartTime time.Time, scanDuration time.Duration) {
	if len(secrets) == 0 {
		// Spinner will indicate if nothing was found
		return
	}

	// Sort secrets by IdleDays descending (longest idle first)
	sort.SliceStable(secrets, func(i, j int) bool {
		return secrets[i].IdleDays > secrets[j].IdleDays
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "NAME\tARN\tREGION\tLAST ACCESSED\tIDLE DAYS")

	// Print table rows
	for _, secret := range secrets {
		// Truncate ARN if necessary
		truncatedARN := truncateString(secret.ARN, 60) // Assuming truncateString exists in common.go or similar

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			secret.Name,
			truncatedARN,
			secret.Region,
			secret.LastAccessedDate.Format("2006-01-02"),
			secret.IdleDays,
		)
	}

	footerStr := fmt.Sprintf("Showing %d idle Secrets Manager secrets (unused for over %d days)", len(secrets), 90) // Assuming 90 days threshold
	w.Flush()
	fmt.Printf("\n%s\n", footerStr)
}

// PrintSecretsSummary prints a simple summary for idle secrets.
func PrintSecretsSummary(secrets []models.SecretInfo) {
	if len(secrets) == 0 {
		return
	}

	// For Secrets Manager, a simple count might be sufficient as the criteria is straightforward.
	// If more complex summaries are needed later, this can be expanded.
	fmt.Printf("\n## Secrets Manager Summary:")
	fmt.Printf("\nTotal Idle Secrets Found: %d\n", len(secrets))
}

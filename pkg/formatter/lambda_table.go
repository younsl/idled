package formatter

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// PrintLambdaTable formats and prints Lambda functions info in a table
func PrintLambdaTable(functions []models.LambdaFunctionInfo, scanTime time.Time, scanDuration time.Duration) {
	// Early return if no results
	if len(functions) == 0 {
		fmt.Println("No Lambda functions found.")
		return
	}

	// Sort functions by idle status and then by idle days (descending)
	sort.Slice(functions, func(i, j int) bool {
		if functions[i].IsIdle != functions[j].IsIdle {
			return functions[i].IsIdle // Idle functions first
		}
		return functions[i].IdleDays > functions[j].IdleDays // Then by idle days (descending)
	})

	// Use tabwriter for aligned columns with kubectl style spacing
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "FUNCTION\tRUNTIME\tMEMORY\tREGION\tLAST INVOCATION\tIDLE DAYS\tCOST/MO\tSTATUS")

	// Loop through each function
	for _, function := range functions {
		// Format last invocation
		lastInvocation := "Unknown"
		if function.LastInvocation != nil {
			lastInvocation = function.LastInvocation.Format("2006-01-02")
		}

		// Format memory size
		memorySize := fmt.Sprintf("%d MB", function.MemorySize)

		// Format idle days
		idleDays := strconv.Itoa(function.IdleDays)
		if function.IdleDays == 0 && !function.IsIdle {
			idleDays = "-"
		}

		// Format cost estimation
		cost := fmt.Sprintf("$%.2f", function.EstimatedMonthlyCost)

		// Determine status
		status := "Active"
		if function.IsIdle {
			status = "Idle"
		}

		// Format and print the row
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			truncateString(function.FunctionName, 50),
			function.Runtime,
			memorySize,
			function.Region,
			lastInvocation,
			idleDays,
			cost,
			status,
		)
	}

	// Print totals
	printLambdaTotals(w, functions)

	// Flush the tabwriter buffer
	w.Flush()
}

// printLambdaTotals prints the summary information at the bottom of the table
func printLambdaTotals(w *tabwriter.Writer, functions []models.LambdaFunctionInfo) {
	totalFunctions := len(functions)
	idleCount := 0
	var totalMonthlyCost float64

	for _, function := range functions {
		if function.IsIdle {
			idleCount++
		}
		totalMonthlyCost += function.EstimatedMonthlyCost
	}

	// Format totals with 2 decimal places
	formattedMonthlyCost := fmt.Sprintf("$%.2f", totalMonthlyCost)

	// Print summary with kubernetes style alignment
	fmt.Fprintf(w, "Total:\t\t\t\t\t%d\t%s\t%d idle\n",
		totalFunctions,
		formattedMonthlyCost,
		idleCount,
	)
}

// PrintLambdaSummary displays summary information about Lambda functions
func PrintLambdaSummary(functions []models.LambdaFunctionInfo) {
	if len(functions) == 0 {
		return
	}

	// Group by runtime
	runtimeCounts := make(map[string]int)
	for _, function := range functions {
		runtimeCounts[function.Runtime]++
	}

	// Prepare sorted list of runtimes
	var runtimes []string
	for runtime := range runtimeCounts {
		runtimes = append(runtimes, runtime)
	}
	sort.Strings(runtimes)

	fmt.Println("\n## Lambda Functions Summary")

	// Set up tabwriter with kubectl style spacing
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header for status summary
	fmt.Fprintln(w, "STATUS\tCOUNT")

	// Count active and idle functions
	activeCount := 0
	idleCount := 0
	for _, function := range functions {
		if function.IsIdle {
			idleCount++
		} else {
			activeCount++
		}
	}

	// Print status summary
	fmt.Fprintf(w, "Active\t%d\n", activeCount)
	fmt.Fprintf(w, "Idle\t%d\n", idleCount)

	w.Flush()

	// Print runtime distribution
	fmt.Println("\n## Lambda Runtime Distribution")

	w = tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "RUNTIME\tCOUNT")

	for _, runtime := range runtimes {
		fmt.Fprintf(w, "%s\t%d\n", runtime, runtimeCounts[runtime])
	}

	w.Flush()
}

// truncateString truncates a string to the given max length and adds "..." if necessary
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

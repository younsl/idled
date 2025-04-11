package formatter

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"

	"github.com/younsl/idled/internal/models"
)

// FormatLambdaTable formats and prints Lambda functions info in a table
func FormatLambdaTable(functions []models.LambdaFunctionInfo) {
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

	// Use tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)

	// Print header
	fmt.Fprintln(w, "FUNCTION\tRUNTIME\tMEMORY\tLAST INVOCATION\tINVOCATIONS (30d)\tERRORS (30d)\tIDLE DAYS\tESTIMATED COST\tSTATUS")
	fmt.Fprintln(w, "--------\t-------\t------\t---------------\t----------------\t-----------\t---------\t--------------\t------")

	// Loop through each function
	for _, function := range functions {
		// Format last invocation
		lastInvocation := "Unknown"
		if function.LastInvocation != nil {
			lastInvocation = function.LastInvocation.Format("2006-01-02")
		}

		// Format memory size
		memorySize := fmt.Sprintf("%d MB", function.MemorySize)

		// Format invocations and errors
		invocations := fmt.Sprintf("%d", function.InvocationsLast30Days)
		errors := fmt.Sprintf("%d", function.ErrorsLast30Days)

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
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			truncateString(function.FunctionName, 50),
			function.Runtime,
			memorySize,
			lastInvocation,
			invocations,
			errors,
			idleDays,
			cost,
			status,
		)
	}

	// Flush the tabwriter buffer
	w.Flush()

	// Print summary
	idleCount := 0
	for _, function := range functions {
		if function.IsIdle {
			idleCount++
		}
	}

	fmt.Printf("\nFound %d Lambda functions, %d of which are idle.\n", len(functions), idleCount)
}

// truncateString truncates a string to the given max length and adds "..." if necessary
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

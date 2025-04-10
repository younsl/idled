package formatter

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/younsl/idled/internal/models"
)

// MAX_NAME_WIDTH defines the maximum width for Name column
const MAX_NAME_WIDTH = 20

// PrintVolumesTable prints a formatted table of available EBS volumes
func PrintVolumesTable(volumes []models.VolumeInfo, scanTime time.Time, scanDuration time.Duration) {
	if len(volumes) == 0 {
		fmt.Println("No available EBS volumes found.")
		return
	}

	// Sort volumes by estimated savings (highest first)
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].EstimatedSavings > volumes[j].EstimatedSavings
	})

	// kubectl 스타일 tabwriter 설정
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header as requested
	fmt.Fprintln(w, "NAME\tVOLUME ID\tTYPE\tSIZE\tSTATUS\tMONTHLY SAVINGS\tPRICING")

	// Pre-process names to handle Korean and get max string width
	processedNames := make([]string, len(volumes))
	for i, volume := range volumes {
		// Handle empty name case
		name := volume.Name
		if name == "" {
			name = "N/A"
		}

		// Limit name length and ensure proper width for Korean
		if StringWidth(name) > MAX_NAME_WIDTH {
			// Truncate name if it's too long while preserving proper display width
			truncated := ""
			currentWidth := 0
			for _, r := range name {
				charWidth := RuneWidth(r)
				if currentWidth+charWidth > MAX_NAME_WIDTH-2 { // -2 for ".."
					break
				}
				truncated += string(r)
				currentWidth += charWidth
			}
			name = truncated + ".."
		}

		// Pad with spaces to ensure consistent column width
		nameWidth := StringWidth(name)
		paddingNeeded := MAX_NAME_WIDTH - nameWidth
		if paddingNeeded > 0 {
			name = name + strings.Repeat(" ", paddingNeeded)
		}

		processedNames[i] = name
	}

	// Print each volume
	for i, volume := range volumes {
		// Format the monthly cost and savings with 2 decimal places
		var savings string
		if volume.PricingSource == "N/A" {
			savings = "N/A"
		} else {
			savings = fmt.Sprintf("$%.2f", volume.EstimatedSavings)
		}

		// Add a marker for pricing source
		pricingMarker := GetPricingMarker(volume.PricingSource)

		// Use pre-processed name with proper spacing
		fmt.Fprintf(w, "%s\t%s\t%s\t%d GB\t%s\t%s\t%s\n",
			processedNames[i],
			volume.VolumeID,
			volume.VolumeType,
			volume.Size,
			volume.State,
			savings,
			pricingMarker,
		)
	}

	// Print totals
	printVolumeTotals(w, volumes)

	w.Flush()
}

// printVolumeTotals prints the summary information at the bottom of the table
func printVolumeTotals(w *tabwriter.Writer, volumes []models.VolumeInfo) {
	totalSize := 0

	// Calculate total potential savings
	var totalSavings float64

	for _, volume := range volumes {
		totalSavings += volume.EstimatedSavings
		totalSize += volume.Size
	}

	// Format totals with 2 decimal places
	formattedSavings := fmt.Sprintf("$%.2f", totalSavings)

	// Print summary with kubernetes style alignment
	fmt.Fprintf(w, "Total:\t\t\t%d GB\t\t%s\n",
		totalSize,
		formattedSavings,
	)
}

// PrintVolumesSummary displays summary information about volumes
func PrintVolumesSummary(volumes []models.VolumeInfo) {
	if len(volumes) == 0 {
		return
	}

	// Group by volume type
	volumeTypes := make(map[string]struct {
		count   int
		size    int
		savings float64
	})

	for _, volume := range volumes {
		typeInfo := volumeTypes[volume.VolumeType]
		typeInfo.count++
		typeInfo.size += volume.Size
		typeInfo.savings += volume.EstimatedSavings
		volumeTypes[volume.VolumeType] = typeInfo
	}

	fmt.Println("\n## Available EBS Volumes Summary")

	// kubectl 스타일 tabwriter 설정
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "VOLUME TYPE\tCOUNT\tTOTAL SIZE\tPOTENTIAL MONTHLY SAVINGS")

	// Sort volume types for consistent output
	types := make([]string, 0, len(volumeTypes))
	for volumeType := range volumeTypes {
		types = append(types, volumeType)
	}
	sort.Strings(types)

	// Print each type
	for _, volumeType := range types {
		info := volumeTypes[volumeType]
		fmt.Fprintf(w, "%s\t%d\t%d GB\t$%.2f\n",
			volumeType,
			info.count,
			info.size,
			info.savings,
		)
	}

	w.Flush()
}

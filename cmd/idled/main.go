package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/younsl/idled/internal/models"
	"github.com/younsl/idled/pkg/aws"
	"github.com/younsl/idled/pkg/formatter"
	"github.com/younsl/idled/pkg/pricing"
	"github.com/younsl/idled/pkg/utils"
)

// Version information
const (
	Version   = "0.1.0"
	BuildDate = "2023-04-09"
)

var (
	regions           []string
	services          []string
	showVersion       bool
	supportedServices = map[string]bool{
		"ec2": true,
		"ebs": true,
		"s3":  true,
	}
)

// startResourceSpinner creates and starts a spinner with a message for the given service
func startResourceSpinner(service string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[9], 200*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Analyzing %s resources...", service)
	// Don't set FinalMSG here as it will be set dynamically based on scan time
	s.Start()
	return s
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "idled",
		Short: "CLI tool to find idle AWS resources",
		Long: `idled is a CLI tool that searches for idle AWS resources
and displays the results in a table format.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If version flag is set, print version info and exit
			if showVersion {
				fmt.Printf("idled version %s (built: %s)\n", Version, BuildDate)
				return
			}

			// Use default region if none specified
			if len(regions) == 0 {
				regions = []string{utils.GetDefaultRegion()}
			}

			// Validate regions
			var validRegions []string
			for _, region := range regions {
				if utils.IsValidRegion(region) {
					validRegions = append(validRegions, region)
				} else {
					fmt.Printf("Warning: Skipping invalid region '%s'\n", region)
				}
			}

			if len(validRegions) == 0 {
				fmt.Println("No valid regions specified. Exiting.")
				return
			}

			// Use default service if none specified
			if len(services) == 0 {
				services = []string{"ec2"}
			}

			// Validate services
			for _, service := range services {
				supported, exists := supportedServices[service]
				if !exists {
					fmt.Printf("Warning: Unknown service '%s'\n", service)
					continue
				}
				if !supported {
					fmt.Printf("Warning: Service '%s' is not yet implemented\n", service)
				}
			}

			// Only process supported services
			var activeServices []string
			for _, service := range services {
				if supported, exists := supportedServices[service]; exists && supported {
					activeServices = append(activeServices, service)
				}
			}

			if len(activeServices) == 0 {
				fmt.Println("No supported services specified. Exiting.")
				return
			}

			// Process each service
			for _, service := range activeServices {
				switch service {
				case "ec2":
					processEC2(validRegions)
				case "ebs":
					processEBS(validRegions)
				case "s3":
					processS3(validRegions)
				// Add more services here in the future
				default:
					// This should never happen due to earlier checks
					fmt.Printf("Skipping unsupported service: %s\n", service)
				}
			}

			// Print combined pricing API statistics once after all services are processed
			formatter.PrintPricingAPIStats()
		},
	}

	// Version flag
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	// Initialize default regions
	defaultRegions := []string{utils.GetDefaultRegion()}

	// Region flags (long and short forms)
	rootCmd.Flags().StringSliceVarP(&regions, "regions", "r", nil,
		fmt.Sprintf("AWS regions to check (comma separated, default: %s)", strings.Join(defaultRegions, ", ")))

	// Initialize default services
	defaultServices := []string{"ec2"}

	// Service flags (long and short forms)
	rootCmd.Flags().StringSliceVarP(&services, "services", "s", nil,
		fmt.Sprintf("AWS services to check (comma separated, default: %s)", strings.Join(defaultServices, ", ")))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// processEC2 handles the scanning of EC2 instances
func processEC2(regions []string) {
	fmt.Println("Starting EC2 scan...")
	scanStartTime := time.Now()

	// Start the spinner
	s := startResourceSpinner("EC2")

	// Slice to store results
	allInstances := make([]struct {
		instances []models.InstanceInfo
		err       error
		region    string
	}, len(regions))

	// Process each region in parallel
	var wg sync.WaitGroup
	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()

			client, err := aws.NewEC2Client(r)
			if err != nil {
				allInstances[idx].err = err
				allInstances[idx].region = r
				return
			}

			instances, err := client.GetStoppedInstances()
			allInstances[idx].instances = instances
			allInstances[idx].err = err
			allInstances[idx].region = r
		}(i, region)
	}

	wg.Wait()

	// Calculate scan duration
	scanDuration := time.Since(scanStartTime)

	// Set completion message with scan time
	s.FinalMSG = fmt.Sprintf("✓ EC2 resources analyzed - Completed in %.2f seconds\n", scanDuration.Seconds())
	s.Stop() // Stop the spinner when done

	// Display API init message if any
	if msg := pricing.GetInitMessage(); msg != "" {
		fmt.Println(msg)
	}

	// Process results
	var allStoppedInstances []models.InstanceInfo

	// Process results from each region
	for _, result := range allInstances {
		if result.err != nil {
			fmt.Printf("Error in region %s: %v\n", result.region, result.err)
			continue
		}
		allStoppedInstances = append(allStoppedInstances, result.instances...)
	}

	// Display as table
	formatter.PrintInstancesTable(allStoppedInstances, scanStartTime, scanDuration)
	formatter.PrintInstancesSummary(allStoppedInstances)
}

// processEBS handles the scanning of available EBS volumes
func processEBS(regions []string) {
	fmt.Println("Starting EBS scan...")
	scanStartTime := time.Now()

	// Start the spinner
	s := startResourceSpinner("EBS")

	// Slice to store results
	allVolumes := make([]struct {
		volumes []models.VolumeInfo
		err     error
		region  string
	}, len(regions))

	// Process each region in parallel
	var wg sync.WaitGroup
	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()

			client, err := aws.NewEBSClient(r)
			if err != nil {
				allVolumes[idx].err = err
				allVolumes[idx].region = r
				return
			}

			volumes, err := client.GetAvailableVolumes()
			allVolumes[idx].volumes = volumes
			allVolumes[idx].err = err
			allVolumes[idx].region = r
		}(i, region)
	}

	wg.Wait()

	// Calculate scan duration
	scanDuration := time.Since(scanStartTime)

	// Set completion message with scan time
	s.FinalMSG = fmt.Sprintf("✓ EBS resources analyzed - Completed in %.2f seconds\n", scanDuration.Seconds())
	s.Stop() // Stop the spinner when done

	// Display API init message if any
	if msg := pricing.GetInitMessage(); msg != "" {
		fmt.Println(msg)
	}

	// Process results
	var allAvailableVolumes []models.VolumeInfo

	// Process results from each region
	for _, result := range allVolumes {
		if result.err != nil {
			fmt.Printf("Error in region %s: %v\n", result.region, result.err)
			continue
		}
		allAvailableVolumes = append(allAvailableVolumes, result.volumes...)
	}

	// Display as table with the requested format
	formatter.PrintVolumesTable(allAvailableVolumes, scanStartTime, scanDuration)
	formatter.PrintVolumesSummary(allAvailableVolumes)
}

// processS3 handles the scanning of idle S3 buckets
func processS3(regions []string) {
	fmt.Println("Starting S3 scan...")
	scanStartTime := time.Now()

	// Start the spinner
	s := startResourceSpinner("S3")

	// Slice to store results
	allBuckets := make([]struct {
		buckets []models.BucketInfo
		err     error
		region  string
	}, len(regions))

	// Process each region in parallel
	var wg sync.WaitGroup
	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()

			client, err := aws.NewS3Client(r)
			if err != nil {
				allBuckets[idx].err = err
				allBuckets[idx].region = r
				return
			}

			buckets, err := client.GetIdleBuckets()
			allBuckets[idx].buckets = buckets
			allBuckets[idx].err = err
			allBuckets[idx].region = r
		}(i, region)
	}

	wg.Wait()

	// Calculate scan duration
	scanDuration := time.Since(scanStartTime)

	// Set completion message with scan time
	s.FinalMSG = fmt.Sprintf("✓ S3 resources analyzed - Completed in %.2f seconds\n", scanDuration.Seconds())
	s.Stop() // Stop the spinner when done

	// Display API init message if any
	if msg := pricing.GetInitMessage(); msg != "" {
		fmt.Println(msg)
	}

	// Process results
	var allIdleBuckets []models.BucketInfo

	// Process results from each region
	for _, result := range allBuckets {
		if result.err != nil {
			fmt.Printf("Error in region %s: %v\n", result.region, result.err)
			continue
		}
		allIdleBuckets = append(allIdleBuckets, result.buckets...)
	}

	// Calculate scan duration
	scanDuration = time.Since(scanStartTime)

	// Display as table
	formatter.PrintBucketsTable(allIdleBuckets, scanStartTime, scanDuration)
	formatter.PrintBucketsSummary(allIdleBuckets)
}

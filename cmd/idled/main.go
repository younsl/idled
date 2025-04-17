package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
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
	Version        = "0.6.1"
	BuildDate      = "2025-04-17"
	DefaultService = "ec2"
)

var (
	regions           []string
	services          []string
	showVersion       bool
	supportedServices = map[string]bool{
		"ec2":    true,
		"ebs":    true,
		"s3":     true,
		"lambda": true,
		"eip":    true,
		"iam":    true,
		"config": true,
		"elb":    true,
		"logs":   true,
	}
)

// Define service descriptions for help text
var serviceDescriptions = map[string]string{
	"ec2":    "Find stopped EC2 instances",
	"ebs":    "Find unattached EBS volumes",
	"s3":     "Find idle S3 buckets",
	"lambda": "Find idle Lambda functions",
	"eip":    "Find unattached Elastic IP addresses",
	"iam":    "Find idle IAM users, roles, and policies",
	"config": "Find idle AWS Config rules, recorders, and delivery channels",
	"elb":    "Find idle Elastic Load Balancers (ALB, NLB)",
	"logs":   "Find idle CloudWatch Log Groups",
}

// startResourceSpinner creates and starts a spinner with a message for the given service and regions
func startResourceSpinner(service string, regions []string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[9], 200*time.Millisecond)
	regionStr := "Global"
	if len(regions) > 0 {
		regionStr = strings.Join(regions, ", ")
		if len(regions) > 5 { // Limit displayed regions for conciseness
			regionStr = fmt.Sprintf("%s, ... (%d total)", strings.Join(regions[:5], ", "), len(regions))
		}
	}
	s.Suffix = fmt.Sprintf(" Analyzing %s resources in %s ...", service, regionStr)
	// Don't set FinalMSG here as it will be set dynamically based on scan time
	s.Start()
	return s
}

// Common function to start scan
func startScan(serviceName string, regions []string) (time.Time, *spinner.Spinner) {
	// fmt.Printf("Starting %s scan in regions: %s ...\n", serviceName, strings.Join(regions, ", ")) // Keep console clean, spinner shows info
	scanStartTime := time.Now()
	s := startResourceSpinner(serviceName, regions) // Pass regions to spinner
	return scanStartTime, s
}

// Common result structure
type ScanResult[T any] struct {
	Data   []T
	Err    error
	Region string
}

// Common function to process results
func processResults[T any](results []ScanResult[T], scanStartTime time.Time, s *spinner.Spinner, printTable func([]T, time.Time, time.Duration), printSummary func([]T)) {
	scanDuration := time.Since(scanStartTime)
	var allData []T
	for _, result := range results {
		if result.Err == nil {
			allData = append(allData, result.Data...)
		}
	}
	s.FinalMSG = fmt.Sprintf("✓ [%d items found] resources analyzed - Completed in %.2f seconds\n",
		len(allData), scanDuration.Seconds())
	s.Stop()

	// Display API init message if any (moved here for consistency)
	if msg := pricing.GetInitMessage(); msg != "" {
		fmt.Println(msg)
	}

	allData = []T{} // Reset to re-process for error display and final table
	for _, result := range results {
		if result.Err != nil {
			fmt.Printf("Error in region %s: %v\n", result.Region, result.Err)
			continue
		}
		allData = append(allData, result.Data...)
	}
	printTable(allData, scanStartTime, scanDuration)
	printSummary(allData)
}

// Common function to handle errors
func handleErrors(errChan <-chan error) []string {
	var allErrors []string
	for err := range errChan {
		allErrors = append(allErrors, err.Error())
	}
	return allErrors
}

// Generic function to handle service-specific scan logic
func processService[T any](
	serviceName string, // Service name (for spinner message)
	regions []string, // List of regions to scan
	getDataForRegion func(region string) ([]T, error), // Function to get data for a specific region
	printTable func([]T, time.Time, time.Duration), // Function to print results as a table
	printSummary func([]T), // Function to print result summary
) {
	scanStartTime, s := startScan(serviceName, regions)
	results := make([]ScanResult[T], len(regions))
	var wg sync.WaitGroup

	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()
			results[idx].Region = r
			// Execute service-specific data fetching logic
			data, err := getDataForRegion(r)
			results[idx].Data = data
			results[idx].Err = err
		}(i, region)
	}

	wg.Wait()
	// Call common result processing function
	processResults(results, scanStartTime, s, printTable, printSummary)
}

// Refactor processEC2 function (using processService)
func processEC2(regions []string) {
	getData := func(region string) ([]models.InstanceInfo, error) {
		client, err := aws.NewEC2Client(region)
		if err != nil {
			return nil, err
		}
		return client.GetStoppedInstances()
	}
	processService("EC2", regions, getData, formatter.PrintInstancesTable, formatter.PrintInstancesSummary)
}

// Refactor processEBS function (using processService)
func processEBS(regions []string) {
	getData := func(region string) ([]models.VolumeInfo, error) {
		client, err := aws.NewEBSClient(region)
		if err != nil {
			return nil, err
		}
		return client.GetAvailableVolumes()
	}
	processService("EBS", regions, getData, formatter.PrintVolumesTable, formatter.PrintVolumesSummary)
}

// Refactor processS3 function (using processService)
func processS3(regions []string) {
	getData := func(region string) ([]models.BucketInfo, error) {
		client, err := aws.NewS3Client(region)
		if err != nil {
			return nil, err
		}
		return client.GetIdleBuckets()
	}
	processService("S3", regions, getData, formatter.PrintBucketsTable, formatter.PrintBucketsSummary)
}

// Refactor processLambda function (using processService)
func processLambda(regions []string) {
	getData := func(region string) ([]models.LambdaFunctionInfo, error) {
		client, err := aws.NewLambdaClient(region)
		if err != nil {
			return nil, err
		}
		return client.GetIdleFunctions()
	}
	processService("Lambda", regions, getData, formatter.PrintLambdaTable, formatter.PrintLambdaSummary)
}

// Refactor processEIP function (using processService)
func processEIP(regions []string) {
	getData := func(region string) ([]models.EIPInfo, error) {
		client, err := aws.NewEIPClient(region)
		if err != nil {
			return nil, err
		}
		return client.GetUnattachedEIPs()
	}
	processService("Elastic IP", regions, getData, formatter.PrintEIPsTable, formatter.PrintEIPsSummary)
}

// processIAM handles the scanning of IAM resources
func processIAM(regions []string) {
	// Pass nil for regions as IAM is global
	scanStartTime, _ := startScan("IAM", nil)
	// region := regions[0] // Keep original logic for client init region
	// fmt.Printf("Note: IAM is a global service. Region parameter '%s' will be used for configuration only.\n", region)
	client, err := aws.NewIAMClient(regions[0]) // Use the first region for client init
	if err != nil {
		fmt.Printf("Error initializing IAM client: %v\n", err)
		return
	}
	users, err := client.GetIdleUsers()
	if err != nil {
		fmt.Printf("Error getting IAM users: %v\n", err)
	} else {
		fmt.Println("\nIAM Users:")
		formatter.FormatIAMUserTable(os.Stdout, users)
	}
	roles, err := client.GetIdleRoles()
	if err != nil {
		fmt.Printf("Error getting IAM roles: %v\n", err)
	} else {
		fmt.Println("\nIAM Roles:")
		formatter.FormatIAMRoleTable(os.Stdout, roles)
	}
	policies, err := client.GetIdlePolicies()
	if err != nil {
		fmt.Printf("Error getting IAM policies: %v\n", err)
	} else {
		fmt.Println("\nIAM Policies:")
		formatter.FormatIAMPolicyTable(os.Stdout, policies)
	}
	scanDuration := time.Since(scanStartTime)
	fmt.Printf("\n✓ IAM resources analyzed - Completed in %.2f seconds\n\n", scanDuration.Seconds())
}

// processConfig handles the scanning of AWS Config resources
func processConfig(regions []string) {
	scanStartTime, s := startScan("Config", regions)
	results := make([]struct {
		rules     []models.ConfigRuleInfo
		recorders []models.ConfigRecorderInfo
		channels  []models.ConfigDeliveryChannelInfo
		region    string
		err       error
	}, len(regions))
	var wg sync.WaitGroup
	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()
			client, err := aws.NewConfigClient(r)
			if err != nil {
				fmt.Printf("Error initializing AWS Config client for region %s: %v\n", r, err)
				results[idx].err = err
				results[idx].region = r
				return
			}
			rules, err := client.GetAllConfigRules()
			if err != nil {
				fmt.Printf("Error getting AWS Config rules for region %s: %v\n", r, err)
			}
			results[idx].rules = rules
			recorders, err := client.GetAllConfigRecorders()
			if err != nil {
				fmt.Printf("Error getting AWS Config recorders for region %s: %v\n", r, err)
			}
			results[idx].recorders = recorders
			channels, err := client.GetAllConfigDeliveryChannels()
			if err != nil {
				fmt.Printf("Error getting AWS Config delivery channels for region %s: %v\n", r, err)
			}
			results[idx].channels = channels
			results[idx].region = r
		}(i, region)
	}
	wg.Wait()

	scanDuration := time.Since(scanStartTime)

	var allRules []models.ConfigRuleInfo
	var allRecorders []models.ConfigRecorderInfo
	var allChannels []models.ConfigDeliveryChannelInfo
	for _, result := range results {
		if result.err == nil {
			allRules = append(allRules, result.rules...)
			allRecorders = append(allRecorders, result.recorders...)
			allChannels = append(allChannels, result.channels...)
		}
	}
	totalCount := len(allRules) + len(allRecorders) + len(allChannels)
	s.FinalMSG = fmt.Sprintf("✓ [%d resources found] AWS Config resources analyzed - Completed in %.2f seconds\n",
		totalCount, scanDuration.Seconds())
	s.Stop()
	allRules = []models.ConfigRuleInfo{}
	allRecorders = []models.ConfigRecorderInfo{}
	allChannels = []models.ConfigDeliveryChannelInfo{}
	for _, result := range results {
		if result.err != nil {
			fmt.Printf("Error in region %s: %v\n", result.region, result.err)
			continue
		}
		allRules = append(allRules, result.rules...)
		allRecorders = append(allRecorders, result.recorders...)
		allChannels = append(allChannels, result.channels...)
	}
	if len(allRules) > 0 {
		fmt.Println("\nAWS Config Rules:")
		formatter.FormatConfigRulesTable(os.Stdout, allRules)
	} else {
		fmt.Println("\nNo AWS Config rules found.")
	}
	if len(allRecorders) > 0 {
		fmt.Println("\nAWS Config Recorders:")
		formatter.FormatConfigRecordersTable(os.Stdout, allRecorders)
	} else {
		fmt.Println("\nNo AWS Config recorders found.")
	}
	if len(allChannels) > 0 {
		fmt.Println("\nAWS Config Delivery Channels:")
		formatter.FormatConfigDeliveryChannelsTable(os.Stdout, allChannels)
	} else {
		fmt.Println("\nNo AWS Config delivery channels found.")
	}
	fmt.Printf("\n✓ AWS Config resources analyzed - Completed in %.2f seconds\n\n", scanDuration.Seconds())
}

// Refactor processELB function (using processService)
func processELB(regions []string) {
	getData := func(region string) ([]models.ELBResource, error) {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config for region %s: %w", region, err)
		}
		scanner := aws.NewELBScanner(cfg)
		return scanner.GetIdleELBs(context.TODO(), region)
	}
	// PrintELBTable, PrintELBSummary need os.Stdout -> use anonymous functions
	printTable := func(data []models.ELBResource, _ time.Time, _ time.Duration) {
		formatter.PrintELBTable(os.Stdout, data)
	}
	printSummary := func(data []models.ELBResource) {
		formatter.PrintELBSummary(os.Stdout, data)
	}
	processService("ELB (v2)", regions, getData, printTable, printSummary)
}

// processLogs handles the scanning of CloudWatch Log Groups, aligned with EC2 flow
func processLogs(regions []string) {
	scanStartTime, s := startScan("Logs", regions)
	var allLogGroups []models.LogGroupInfo
	var mu sync.Mutex
	errChan := make(chan error, len(regions)*2)
	var wg sync.WaitGroup
	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(r))
			if err != nil {
				errChan <- fmt.Errorf("failed to load config for region %s: %w", r, err)
				return
			}
			idleThreshold := 90
			logGroups, scanErrs := aws.ScanLogGroups(cfg, idleThreshold)
			if len(logGroups) > 0 {
				mu.Lock()
				allLogGroups = append(allLogGroups, logGroups...)
				mu.Unlock()
			}
			if len(scanErrs) > 0 {
				for _, scanErr := range scanErrs {
					errChan <- fmt.Errorf("region %s: %w", r, scanErr)
				}
			}
		}(region)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()
	allErrors := handleErrors(errChan)
	scanDuration := time.Since(scanStartTime)
	s.FinalMSG = fmt.Sprintf("✓ [%d Log Groups found] Logs resources analyzed - Completed in %.2f seconds\n",
		len(allLogGroups), scanDuration.Seconds())
	s.Stop()
	if len(allErrors) > 0 {
		fmt.Printf("\nErrors during CloudWatch Logs scan:\n")
		for _, errMsg := range allErrors {
			fmt.Printf(" - %s\n", errMsg)
		}
		fmt.Println()
	}
	formatter.PrintLogGroupsTable(allLogGroups)
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func main() {
	var showServiceList bool

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

			// If list services flag is set, show available services and exit
			if showServiceList {
				fmt.Println("Available services:")

				// Get a sorted list of supported services for consistent output
				var serviceList []string
				for service, isSupported := range supportedServices {
					if isSupported {
						serviceList = append(serviceList, service)
					}
				}
				sort.Strings(serviceList)

				// Define default services here as well for checking
				defaultServices := []string{DefaultService}

				// Print each service with its description
				for _, service := range serviceList {
					description, ok := serviceDescriptions[service]
					if !ok {
						description = "No description available"
					}
					// Check if the service is a default service
					isDefault := false
					for _, ds := range defaultServices {
						if service == ds {
							isDefault = true
							break
						}
					}

					if isDefault {
						fmt.Printf("  %-8s - %s (default)\n", service, description)
					} else {
						fmt.Printf("  %-8s - %s\n", service, description)
					}
				}

				fmt.Println("\nExample usage:")
				fmt.Printf("  %s --services %s\n", os.Args[0], strings.Join(serviceList[:min(3, len(serviceList))], ","))
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
				services = []string{DefaultService}
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
				case "lambda":
					processLambda(validRegions)
				case "eip":
					processEIP(validRegions)
				case "iam":
					processIAM(validRegions)
				case "config":
					processConfig(validRegions)
				case "elb":
					processELB(validRegions)
				case "logs":
					processLogs(validRegions)
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

	// Service list flag (show available services)
	rootCmd.Flags().BoolVarP(&showServiceList, "list-services", "l", false, "List available services")

	// Initialize default regions
	defaultRegions := []string{utils.GetDefaultRegion()}

	// Region flags (long and short forms)
	rootCmd.Flags().StringSliceVarP(&regions, "regions", "r", nil,
		fmt.Sprintf("AWS regions to check (comma separated, default: %s)", strings.Join(defaultRegions, ", ")))

	// Initialize default services
	defaultServices := []string{DefaultService}

	// Service flags (long and short forms)
	rootCmd.Flags().StringSliceVarP(&services, "services", "s", nil,
		fmt.Sprintf("AWS services to check (comma separated, default: %s)", strings.Join(defaultServices, ", ")))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

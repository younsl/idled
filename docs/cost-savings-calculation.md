# Cost Savings Calculation in Idled

This document explains how Idled calculates cost savings for different AWS resources.

## EC2 Instance Savings

### How It Works

For EC2 instances, Idled calculates:

1. **Monthly Cost (`COST/MO`)**: The cost to run the instance for a full month.
   - Formula: `hourly price × 730 hours` (730 = average hours in a month)
   - This is the monthly cost if the instance were running.

2. **Total Savings (`TOTAL SAVED`)**: The accumulated savings since the instance was stopped.
   - Formula: `monthly cost × (days stopped ÷ 30)`
   - This counts all savings from the stop date until now.
   - For example, an instance stopped for 90 days saves 3 times its monthly cost.

### Example

If an instance:
- Costs $100 per month to run
- Has been stopped for 60 days

The calculation shows:
- Monthly Cost: $100
- Total Saved: $100 × (60 ÷ 30) = $200

## EBS Volume Savings

> [!WARNING]
> Current calculation does not include costs for IOPS or throughput. For io1, io2, and gp3 volumes with custom IOPS or throughput, actual savings may be higher than reported.

### How It Works

For EBS volumes, Idled calculates:

1. **Monthly Savings (`MONTHLY SAVINGS`)**: The potential savings per month if the unused volume were deleted.
   - Formula: `volume size × price per GB-month`
   - This shows how much you would save each month by removing this volume.
   - **Note**: Unlike EC2 instances, EBS volumes show only the monthly cost, not accumulated savings.

2. **Potential Monthly Savings (Summary)**: The total monthly savings across all volumes of a specific type.
   - This is the sum of monthly savings for all volumes, grouped by type.

### Example

If a volume:
- Is 100 GB in size
- Costs $0.10 per GB-month

The calculation shows:
- Monthly Savings: 100 × $0.10 = $10.00 per month
- This is regardless of how long the volume has been unused

## Price Data Sources

Idled collects pricing information from:

1. **AWS Pricing API**: Real-time prices direct from AWS
2. **Cache**: Previously retrieved prices stored for better performance
3. **Default**: Built-in fallback prices when API data is unavailable

The source is marked in the `PRICING` column: API, CACHE, or N/A. 
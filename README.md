# idled

[![GitHub release](https://img.shields.io/github/v/release/younsl/idled?style=flat-square&color=black&logo=github&logoColor=white&label=release)](https://github.com/younsl/idled/releases)

<img src="https://github.com/younsl/box/blob/main/box/assets/pink-container-84x84.png" alt="pink container logo" width="84" height="84">

idled stands for "idle finder". idled is a CLI tool that finds idle AWS resources across regions and shows the results in a table format.

## Features

- Scan multiple AWS regions for idle resources
- Currently supports stopped EC2 instances, unattached EBS volumes, and idle S3 buckets (RDS, ELB planned)
- Show resource details and idle time
- Display results in a clean table format (Kubernetes style)
- Sort instances by idle time (longest first)
- Provide summary statistics by idle time
- Follows the Golang Standard Project Layout
- Scan multiple AWS regions in parallel
- Identify stopped EC2 instances, unattached EBS volumes, idle S3 buckets, idle MSK clusters, and more
- Display resource details (ID, type, region, idle time, etc.)
- Sort resources by idle time or potential savings
- Calculate estimated cost savings using real-time pricing data
- Display total estimated cost savings across all resources
- Real-time progress indication for long-running operations

## Installation

### From Source

```bash
git clone https://github.com/younsl/idled.git
cd idled
go build -o bin/idled ./cmd/idled
```

## Build

```bash
# Build the binary
make build

# Just run the application
make run

# Clean, format, test and build
make

# Show all available make commands
make help
```

## Usage

> [!IMPORTANT]
> You need to set the `AWS_PROFILE` environment variable to your AWS profile name before running idled command.

Help command:

```bash
idled --help
```

Basic usage:

```bash
export AWS_PROFILE=your-profile
idled
```

Specify AWS regions:

```bash
idled --regions us-east-1,us-west-2
```

Specify AWS services:

> [!NOTE]
> If the `-s` or `--services` flag is omitted, `idled` defaults to scanning **ec2** only.

```bash
idled --services ebs
idled --services ec2,ebs
idled --services s3
idled --services lambda
idled --services iam
idled --services msk
idled --services config
idled --services secretsmanager
idled --services ec2,ebs,s3,lambda,iam,config
```

When scanning for MSK clusters (`--services msk`), `idled` will:
- List all MSK clusters in the specified regions.
- Check each cluster's connection count and average CPU utilization (System + User) over the last 30 days.
- Identify clusters as idle/underutilized based on:
    - **No Connections:** Maximum `ConnectionCount` is 0.
    - **Low CPU Usage:** Average combined CPU (System + User) is below 30%.
- Display results including Cluster Name, ARN, Region, State, Instance Type, Creation Time, Idle Status (`IS IDLE`), and the Reason for being flagged (`REASON`).

Check CLI version:

```bash
idled --version
idled -v
```

## AWS Credentials

This tool uses the AWS SDK's default credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credential file (`~/.aws/credentials`)
3. EC2 or ECS instance role

## Documentation

For more details about idled, please refer to the following documents:

- [Available Services](docs/available-services.md)
- [Project Structure](docs/project-structure.md)

## Implementation Details

### Real-time Pricing Data

IdleFinder integrates with the AWS Pricing API to retrieve real-time pricing information:

- Uses the AWS Pricing API to fetch accurate pricing data based on instance type, volume type, and region
- Implements caching to minimize API calls and improve performance
- Falls back to estimated pricing when the API is unavailable
- Calculates monthly costs and actual savings for each resource
- Shows total potential cost savings across all resources

This feature helps teams identify idle resources with the highest cost impact, prioritizing cleanup efforts for maximum savings.

## Contributing

Contributions are welcome! Feel free to submit pull requests for new features, bug fixes, or documentation improvements. Your contributions help make idled better for everyone.

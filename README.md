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
- Identify stopped EC2 instances, unattached EBS volumes, and idle S3 buckets
- Display resource details (ID, type, region, stop time, etc.)
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

```bash
idled --services ebs
idled --services ec2,ebs
idled --services s3
idled --services ec2,ebs,s3
```

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

## Supported Services

### Amazon Web Services

| Service    | Status       | Resource Description          | Remark |
|------------|--------------|--------------------------------|--------|
| EC2        | ✅ Supported | Stopped EC2 instances          | -      |
| EBS        | ✅ Supported | Unattached EBS volumes         | Not available IOPS and throughput calculation, Only available volume size |
| S3         | ✅ Supported | Idle S3 buckets                | Empty buckets, buckets with no recent modifications or API activity |
| Lambda     | ✅ Supported | Idle Lambda functions          | Functions with no invocations or minimal usage in last 30 days |
| ELB        | ⏳ Planned   | Load balancers with no targets | -      |
| EC2 EIP    | ⏳ Planned   | Non-attached Elastic IPs       | -      |

## Documentation

Detailed documentation is available in the `docs/` directory:

- [Cost Savings Calculation](docs/cost-savings-calculation.md) - How cost savings are calculated for each resource type
- [Project Structure](docs/project-structure.md) - Layout and organization of the codebase

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

## 지원 서비스

현재 다음 AWS 서비스를 스캔할 수 있습니다:

- **EC2**: 중지된 EC2 인스턴스
- **EBS**: 연결되지 않은 EBS 볼륨
- **S3**: 액세스되지 않은 S3 버킷
- **Lambda**: 호출되지 않은 Lambda 함수 
- **EIP**: 연결되지 않은 Elastic IP 주소

### 특정 서비스 스캔

여러 서비스를 한 번에 스캔할 수 있습니다:

```bash
# 중지된 EC2 인스턴스와 미연결 Elastic IP 주소를 스캔
idled --services ec2,eip
```
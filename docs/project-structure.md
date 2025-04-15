# Project Structure

This project follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) for organizing code and maintaining a clean structure.

## Directory Layout

```
idled/
├── cmd/
│   └── idled/        # Main CLI application
│       └── main.go
├── internal/
│   └── models/       # Internal data models (struct definitions)
│       ├── ec2.go
│       ├── ebs.go
│       ├── s3.go
│       ├── lambda.go
│       ├── eip.go
│       ├── iam.go
│       ├── config.go
│       └── elb.go      # Added ELB model
├── pkg/
│   ├── aws/          # AWS API interaction logic
│   │   ├── ec2.go
│   │   ├── ebs.go
│   │   ├── s3.go
│   │   ├── lambda.go
│   │   ├── eip.go
│   │   ├── iam.go
│   │   ├── config.go
│   │   └── elb.go      # Added ELB logic
│   ├── formatter/    # Output formatting (tables, summaries)
│   │   ├── ec2_table.go
│   │   ├── ebs_table.go
│   │   ├── s3_table.go
│   │   ├── lambda_table.go
│   │   ├── eip_table.go
│   │   ├── iam_table.go
│   │   ├── config_table.go
│   │   ├── elb_table.go  # Added ELB table formatter
│   │   └── common.go   # Common formatting utilities
│   ├── pricing/      # AWS Pricing API interaction (optional, for cost estimation)
│   │   └── pricing.go
│   └── utils/        # General utility functions (e.g., region validation)
│       └── aws_utils.go
├── docs/             # Project documentation
│   ├── aws/          # Per-service documentation (NEW)
│   │   ├── ec2.md
│   │   ├── ebs.md
│   │   ├── s3.md
│   │   ├── lambda.md
│   │   ├── eip.md
│   │   ├── iam.md
│   │   ├── config.md
│   │   └── elb.md
│   └── project-structure.md
├── Makefile          # Build automation
├── go.mod
├── go.sum
└── README.md
```

## Design Principles

The code is organized following these principles:

- **Clear Separation of Concerns**: Each package has a well-defined responsibility (AWS calls, formatting, models, main logic).
- **Modularity**: Components are designed to be relatively independent.
- **Testability**: Structure allows for easier unit testing of individual packages.
- **Maintainability**: Follows consistent patterns and Go standards.

## Code Organization Overview

- **`/cmd/idled`**: Contains the `main.go` file, which handles CLI argument parsing (using Cobra), orchestrates calls to different service scanners based on flags, and manages overall application flow including spinners.
- **`/internal/models`**: Defines the Go structs (e.g., `EC2Instance`, `ELBResource`) used to hold data retrieved from AWS APIs for each service.
- **`/pkg/aws`**: Houses the core logic for interacting with AWS APIs for each supported service. Each service has its own file (e.g., `ec2.go`, `elb.go`) containing functions to fetch resources and determine their idle status based on defined criteria (API calls, CloudWatch checks).
- **`/pkg/formatter`**: Contains functions responsible for taking the collected resource data (slices of model structs) and presenting it to the user in a formatted table (using `text/tabwriter`) or as a summary.
- **`/pkg/pricing`**: (If used) Contains logic to interact with the AWS Pricing API to estimate costs for certain resources (like EBS volumes or EIPs).
- **`/pkg/utils`**: Provides common helper functions used across different packages, such as AWS region validation.
- **`/docs`**: Contains project documentation, including per-service details in the `docs/aws/` subdirectory.

## Implementation Details (Concise)

Specific logic for identifying idle resources resides within the respective files in `pkg/aws/`. Key criteria include:

- **EC2**: Checks for `stopped` state.
- **EBS**: Checks for `available` state (unattached).
- **S3**: Checks CloudTrail for recent access (GetObject, PutObject, etc.).
- **Lambda**: Checks CloudWatch `Invocations` metric for recent activity.
- **EIP**: Checks if the EIP is unassociated.
- **IAM**: Checks last used timestamps for users (login/keys) and roles (assumed), and attachment count for policies.
- **Config**: Checks for `FAILED` evaluation status (rules) or `Failure` status (recorders, channels).
- **ELB (ALB/NLB)**: Checks target health (`DescribeTargetHealth`) and relevant CloudWatch metrics (`RequestCount` for ALB, `ActiveFlowCount` for NLB) for recent activity.

The output formatting for each service is handled by the corresponding `_table.go` file in `pkg/formatter/`.
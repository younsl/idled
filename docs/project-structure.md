# Project Structure

This project follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) for organizing code and maintaining a clean structure.

## Directory Layout

```
idled/
├── cmd/
│   └── idled/        # CLI executable
│       └── main.go
├── internal/
│   └── models/       # Internal data models
│       ├── ec2.go
│       ├── ebs.go
│       ├── s3.go
│       └── iam.go
├── pkg/
│   ├── aws/          # AWS API wrapper
│   │   ├── ec2.go
│   │   ├── ebs.go
│   │   ├── s3.go
│   │   ├── lambda.go
│   │   ├── iam.go
│   │   ├── ec2_pricing.go
│   │   └── ebs_pricing.go
│   ├── formatter/    # Output formatters
│   │   ├── ec2_table.go
│   │   ├── ebs_table.go
│   │   ├── s3_table.go
│   │   ├── lambda_table.go
│   │   └── iam_table.go
│   └── utils/        # Utility functions
│       └── format.go
├── docs/             # Documentation
│   ├── cost-savings-calculation.md
│   ├── project-structure.md
│   └── iam.md
├── Makefile          # Build automation
├── go.mod
├── go.sum
└── README.md
```

## Design Principles

The code is organized following these principles:

### `/cmd`

- Main applications for this project
- Each subdirectory represents an executable application
- Minimal code that initializes and starts the application

### `/internal`

- Private application and library code
- Contains code that should not be imported by other applications
- Models that define the core data structures used in the application

### `/pkg`

- Library code that's safe to use by external applications
- Contains reusable components that could be used by other projects
- Follows clear separation of concerns:
  - AWS API operations in `aws/` package
  - Output formatting in `formatter/` package
  - Pricing and cost calculations in separate files

### `/docs`

- Documentation files for the project
- Includes detailed explanations about specific aspects of the system

## Code Organization

- **Clear Separation of Concerns**: Each component has a well-defined responsibility
- **Modularity**: Components are designed to be independent and reusable
- **Testability**: Code structure allows for easy unit testing
- **Maintainability**: Code follows consistent patterns and style

## S3 Implementation Details

The S3 idle bucket detection is implemented with the following components:

### 1. Data Model (`internal/models/s3.go`)

- Defines the `BucketInfo` struct that holds information about an S3 bucket
- Includes fields for bucket stats, activity metrics, and idle detection

### 2. AWS Client (`pkg/aws/s3.go`)

- Implements S3 API operations using AWS SDK v2
- Provides methods to:
  - List all buckets and filter by region
  - Analyze bucket statistics (object count, size, last modified)
  - Check bucket configurations (website, policy, notifications)
  - Determine if a bucket is idle based on multiple criteria
  - Get CloudWatch metrics for API usage patterns

### 3. Output Formatter (`pkg/formatter/s3.go`)

- Displays S3 bucket information in table format
- Summarizes idle buckets by category
- Shows usage patterns and statistics

### 4. Main CLI (`cmd/idled/main.go`)

- Adds S3 service to supported services
- Implements parallel processing of regions
- Consolidates results for display

### 5. Progress Indication

- Shows real-time progress for S3 operations
- Especially valuable for large buckets or many buckets
- Helps users understand the state of long-running operations

## Lambda Implementation Details

The Lambda idle function detection is implemented with the following components:

### 1. Data Model (`internal/models/lambda.go`)

- Defines the `LambdaFunctionInfo` struct that holds information about a Lambda function
- Includes fields for function configuration, invocation metrics, and idle detection

### 2. AWS Client (`pkg/aws/lambda.go`)

- Implements Lambda and CloudWatch API operations using AWS SDK v2
- Provides methods to:
  - List all Lambda functions in a region
  - Analyze function usage metrics (invocations, errors, duration)
  - Determine the last invocation time from CloudWatch metrics
  - Calculate estimated monthly costs based on memory, duration, and invocation count
  - Determine if a function is idle based on invocation patterns

### 3. Output Formatter (`pkg/formatter/lambda_table.go`)

- Displays Lambda function information in a tabular format
- Shows key metrics like runtime, memory, invocations, errors, and idle days
- Estimates monthly cost for each function
- Provides summary statistics on idle functions

### 4. Progress Indication

- Shows real-time progress for Lambda analysis operations
- Displays current function being analyzed and overall progress percentage
- Provides clear feedback for long-running operations where many Lambda functions are being analyzed

## IAM Implementation Details

The IAM idle resource detection is implemented with the following components:

### 1. Data Model (`internal/models/iam.go`)

- Defines the data structures for IAM users, roles, and policies
- Includes fields for tracking:
  - Resource identifiers (name, ARN, ID)
  - Age and creation dates
  - Last activity timestamps
  - Security configurations (MFA, access keys)
  - Attachment and relationship information
  - Idle status determination

### 2. AWS Client (`pkg/aws/iam.go`)

- Implements IAM API operations using AWS SDK v2
- Provides methods to:
  - List all IAM users, roles, and policies in the account
  - Analyze user activity through login history and access key usage
  - Analyze role usage and determine if service-linked or cross-account
  - Extract policy attachment and version information
  - Calculate idle days based on creation and last activity dates
  - Determine if resources are idle based on configurable criteria

### 3. Output Formatter (`pkg/formatter/iam_table.go`)

- Displays IAM resource information in tabular format
- Shows key metrics like resource age, last activity, and security configurations
- Groups and sorts resources by idle status for better visibility
- Provides summary statistics for each resource type

### 4. Progress Indication

- Shows real-time progress for IAM operations
- Displays the current stage of analysis and overall completion percentage
- Indicates resource count and processing status for each resource type
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
│       └── s3.go
├── pkg/
│   ├── aws/          # AWS API wrapper
│   │   ├── ec2.go
│   │   ├── ebs.go
│   │   ├── s3.go
│   │   ├── ec2_pricing.go
│   │   └── ebs_pricing.go
│   ├── formatter/    # Output formatters
│   │   ├── ec2_table.go
│   │   ├── ebs_table.go
│   │   └── s3.go
│   └── utils/        # Utility functions
│       └── format.go
├── docs/             # Documentation
│   ├── cost-savings-calculation.md
│   └── project-structure.md
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
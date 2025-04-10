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
│       ├── ec2.go    # EC2 instance data model
│       └── ebs.go    # EBS volume data model
├── pkg/
│   ├── aws/          # AWS API clients
│   │   ├── ec2.go    # EC2 resource operations
│   │   └── ebs.go    # EBS volume operations
│   ├── pricing/      # AWS pricing operations
│   │   ├── api.go    # Pricing API client
│   │   ├── common.go # Shared pricing functionality
│   │   ├── types.go  # Type definitions and defaults
│   │   ├── stats.go  # Pricing API statistics
│   │   ├── ec2.go    # EC2 pricing operations
│   │   └── ebs.go    # EBS pricing operations
│   ├── formatter/    # Output formatters
│   │   ├── ec2_table.go     # EC2 table output
│   │   ├── ebs_table.go     # EBS table output
│   │   ├── stats.go         # Statistics formatting
│   │   └── unicode_width.go # Unicode width utilities
│   └── utils/        # Utility functions
│       ├── timeutils.go    # Time-related utilities
│       ├── tagutils.go     # AWS tag utilities
│       ├── jsonutils.go    # JSON utilities
│       └── regionutils.go  # AWS region utilities
├── docs/             # Documentation
│   ├── cost-savings-calculation.md
│   └── project-structure.md
├── Makefile          # Build automation
├── go.mod            # Go modules definition
├── go.sum            # Go modules checksums
└── README.md         # Project overview
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
  - Pricing calculations in the `pricing/` package
  - Output formatting in `formatter/` package
  - Common utilities in `utils/` package. These utility functions centralize common operations, reduce code duplication, and improve maintainability across the codebase.

### `/docs`

- Documentation files for the project
- Includes detailed explanations about specific aspects of the system

## Package Organization

### Pricing Module

The pricing module has been extracted to a dedicated package with clear responsibilities:

- `api.go`: Core pricing API client and AWS API interactions
- `common.go`: Shared functionality for pricing operations
- `types.go`: Type definitions and default price tables
- `ec2.go` & `ebs.go`: Resource-specific pricing logic

## Code Organization

This project follows the following principles as much as possible:

- **Clear Separation of Concerns**: Each component has a well-defined responsibility
- **Modularity**: Components are designed to be independent and reusable
- **Testability**: Code structure allows for easy unit testing
- **Maintainability**: Code follows consistent patterns and style
- **Extensibility**: New services can be added by following established patterns
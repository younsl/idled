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
│       └── ebs.go
├── pkg/
│   ├── aws/          # AWS API wrapper
│   │   ├── ec2.go
│   │   ├── ebs.go
│   │   ├── ec2_pricing.go
│   │   └── ebs_pricing.go
│   └── formatter/    # Output formatters
│       ├── ec2_table.go
│       └── ebs_table.go
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
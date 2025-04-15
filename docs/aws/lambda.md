# Lambda Functions

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Compute  |

AWS Lambda is a core serverless compute service. However, functions created for development/testing or no longer triggered can be left unused. While idle Lambda functions themselves typically don't incur direct costs when not running, they increase management complexity and can pose potential security risks. Associated resources like CloudWatch Logs might also incur costs.

## Scan Criteria

- `idled` identifies Lambda functions as **idle** if they meet the following criteria:
    - **No Recent Invocations:** The function has not been invoked for a certain period (default: 90 days), based on CloudWatch Metrics (`Invocations`).
    - **Exclusion Patterns (Optional):** Functions matching specific name patterns (e.g., `prod-`) could potentially be excluded from scans (this feature is not currently implemented in `idled` but could be added).

### Command

```bash
idled -s lambda -r <REGION>
```

## Cost Model

- Lambda functions incur negligible costs when not invoked (due to the monthly free tier).
- However, associated CloudWatch Logs can incur costs based on storage volume.
- `idled` focuses on identifying idle functions. It does not currently calculate potential cost savings. Costs associated with CloudWatch Logs should be managed separately.
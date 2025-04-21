# Lambda Functions

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Output Table](#output-table)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Compute  |

AWS Lambda is a core serverless compute service. However, functions created for development/testing or no longer triggered can be left unused. While idle Lambda functions themselves typically don't incur direct costs when not running, they increase management complexity and can pose potential security risks if not properly managed. Simply checking for recent invocations might not be sufficient, as a function could be configured with a trigger but not have been activated recently. `idled` helps identify potentially unused functions by checking both invocation history and trigger configurations.
Associated resources like CloudWatch Logs might also incur costs.

## Scan Criteria

- `idled` identifies Lambda functions as **idle** if they meet the following criteria:
    - **No Recent Invocations:** The function has not been invoked for a certain period (default: 30 days, configurable), based on CloudWatch Metrics (`Invocations`).
- `idled` also checks for the presence of **triggers** for each function using:
    - `ListEventSourceMappings` API: For event source mapping triggers (e.g., SQS, Kinesis, DynamoDB Streams).
    - `GetPolicy` API: For resource-based policies indicating triggers from other services (e.g., API Gateway, S3, SNS, EventBridge).

### Command

```bash
idled -s lambda -r <REGION>
```

## Output Table

The command outputs a table with the following columns:

| Column          | Description                                                                      |
|-----------------|----------------------------------------------------------------------------------|
| FUNCTION        | Name of the Lambda function.                                                     |
| RUNTIME         | Runtime environment (e.g., `nodejs18.x`, `python3.10`).                          |
| MEMORY          | Memory allocation in MB.                                                         |
| REGION          | AWS Region where the function resides (e.g., `us-east-1`, `ap-northeast-2`)                                           |
| TRIGGER         | Indicates if any triggers are configured (`Yes`/`No`). Checks event source mappings and resource policies. |
| LAST INVOKE     | Date of the last invocation (YYYY-MM-DD), based on CloudWatch metrics. 'Unknown' if never invoked or no data. |
| IDLE DAYS       | Number of days since the last invocation. '-' if invoked recently or unknown.    |
| COST/MO         | Estimated monthly cost (highly approximate, based on recent usage).              |
| STATUS          | `Idle` if no invocations within the threshold period(30 days), `Active` otherwise. |

## Cost Model

- Lambda functions incur negligible costs when not invoked (due to the monthly free tier).
- The `TRIGGER` column helps identify functions that are configured to run but haven't recently, differentiating them from functions that are likely completely unused.
- However, associated CloudWatch Logs can incur costs based on storage volume.
- `idled` focuses on identifying idle functions and trigger presence. It provides a basic cost estimate but does not calculate precise potential savings. Costs associated with CloudWatch Logs should be managed separately.
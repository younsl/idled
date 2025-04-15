# EC2 Instances

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Compute  |

EC2 instances are common AWS compute resources. However, instances often remain in a stopped state without being used. These instances still incur costs for their attached EBS volumes. From a FinOps perspective, identifying and potentially terminating these idle instances is crucial for cost savings.

## Scan Criteria

- `idled` identifies EC2 instances that are in the **stopped** state.
- Instances that have been stopped for an extended period can be considered potential candidates for deletion (the specific duration depends on organizational policy).

### Command

```bash
idled -s ec2 -r <REGION>
```

## Cost Model

- Stopped EC2 instances themselves do not incur compute costs.
- However, the attached EBS volumes continue to incur storage costs.
- `idled` calculates the estimated monthly cost of the EBS volumes attached to the identified stopped instances using the AWS Pricing API. (Note: EBS cost calculation logic is in `pkg/pricing`, `pkg/aws/ebs.go`, etc.)
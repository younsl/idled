# EBS Volumes

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Storage  |

EBS volumes provide block storage for EC2 instances. Even after an instance is terminated or a volume is detached, the EBS volume might remain and continue to incur costs. Identifying and deleting unused ('available' state) EBS volumes is important for reducing storage expenses.

## Scan Criteria

- `idled` identifies EBS volumes that are in the **available** state, meaning they are not attached to any EC2 instance.

### Command

```bash
idled -s ebs -r <REGION>
```

## Cost Model

- EBS volumes in the `available` state incur monthly costs based on the provisioned storage size and type (gp2, gp3, io1, etc.).
- `idled` calculates the estimated monthly cost for the identified `available` volumes using the AWS Pricing API. (Note: EBS cost calculation logic is in `pkg/pricing` and `pkg/aws/ebs.go`, etc.)

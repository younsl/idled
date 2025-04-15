# Elastic IP (EIP) Addresses

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Network  |

Elastic IP (EIP) addresses are static IPv4 addresses for dynamic cloud computing. However, an EIP incurs a small hourly charge if it's allocated but not associated with a running resource like an EC2 instance or an Elastic Network Interface (ENI). Finding and releasing unused EIPs is a simple way to reduce unnecessary costs.

## Scan Criteria

- `idled` identifies EIP addresses that are **unassociated** (not attached to any resource).

### Command

```bash
idled -s eip -r <REGION>
```

## Cost Model

- EIP addresses not associated with a running resource incur an hourly charge.
- EIP addresses associated with a running instance generally do not have additional charges (with some exceptions, e.g., if the instance is stopped or has only one EIP).
- `idled` can calculate the estimated monthly cost for identified unassociated EIPs based on the AWS Pricing API or known fixed costs. (Note: EIP costs vary slightly by region; refer to `pkg/pricing` or related code for calculation logic.) 
# Elastic Load Balancing (ALB/NLB)

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Network  |

Elastic Load Balancing (ELB) automatically distributes incoming application traffic across multiple targets, such as EC2 instances or containers. Application Load Balancers (ALB) and Network Load Balancers (NLB) are commonly used. However, ELBs that have no registered targets or receive no traffic can incur unnecessary costs. (Note: Classic Load Balancers are currently excluded from `idled` scans).

## Scan Criteria

- `idled` identifies ALBs and NLBs as **idle** if they meet one or more of the following criteria:
    - **No Healthy Targets:** No targets in a 'Healthy' state are registered with the associated target groups (checked via `DescribeTargetHealth` API).
        - Specific reasons: `No targets registered` or `No healthy targets registered`.
    - **No Recent Traffic:** Even if healthy targets exist, there has been zero relevant traffic over a certain period (default: 14 days).
        - ALB: `RequestCount` (Sum) = 0
        - NLB: `ActiveFlowCount` (Average) = 0 (checked via CloudWatch Metrics)

### Command

```bash
idled -s elb -r <REGION>
```

## Cost Model

- ALBs and NLBs are charged based on the hours they run and the Load Balancer Capacity Units (LCUs) consumed.
- Even with zero traffic, an hourly charge applies as long as the load balancer is running.
- Deleting idle ELBs saves the hourly running cost and potential LCU costs.
- `idled` currently identifies idle ELBs to highlight cost-saving opportunities but does not estimate the specific cost savings.
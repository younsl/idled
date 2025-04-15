# AWS Config

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category   |
|----------|-------------------|------------|
| AWS      | Regional          | Compliance |

AWS Config is a service that enables you to assess, audit, and evaluate the configurations of your AWS resources. While crucial for compliance and security, unused or disabled Config rules, recorders, or delivery channels can still require management attention. Config rules, in particular, can incur costs based on the number of evaluations. Identifying and cleaning up idle or misconfigured Config resources helps with cost optimization and management efficiency.

## Scan Criteria

- `idled` identifies AWS Config resources in the following states (currently, the focus is more on **configuration errors** or **inactive states** rather than a strict definition of 'idle'):
    - **Config Rules:**
        - Rules where `EvaluationStatus` is `FAILED` (potential configuration error).
        - *Future Enhancement:* Rules that are consistently `NON_COMPLIANT` for a long period without remediation.
        - *Future Enhancement:* Rules that are disabled (`RuleState` is `INACTIVE` - not currently checked by `idled`).
    - **Configuration Recorders:** Where `lastStatus` is `Failure`.
    - **Delivery Channels:** Where `lastStatus` is `Failure`.
- *Note:* Defining 'idle' for AWS Config can be subjective. `idled` primarily focuses on identifying configuration errors or potentially unnecessary resources (like failed recorders/channels).

### Command

```bash
idled -s config -r <REGION>
```

## Cost Model

- AWS Config costs are mainly driven by the number of Configuration Items recorded and the number of active Config rule evaluations.
- Fixing failed recorders or delivery channels can reduce costs associated with error logging or retries.
- Disabling or deleting unused or failing Config rules can save on rule evaluation costs.
- `idled` does not currently estimate direct cost savings for Config resources.
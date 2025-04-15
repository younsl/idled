# IAM (Identity and Access Management)

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Global            | Security |

IAM is crucial for managing access to AWS resources. However, unused IAM users, roles, and policies can accumulate over time. While these idle entities don't incur direct costs, they increase management overhead and can pose security risks by violating the principle of least privilege. Regularly identifying and cleaning up idle IAM entities is important for security and governance.

## Scan Criteria

- `idled` identifies IAM resources as **idle** based on the following criteria:
    - **IAM User:** No console login or API key usage for a certain period (default: 90 days), based on the IAM Credential Report and `GetUser` API.
    - **IAM Role:** Not assumed by any service or user for a certain period (default: 90 days), based on `RoleLastUsed` information from the `GetRole` API.
    - **IAM Policy:** Managed policies that are not currently attached to any IAM user, group, or role, based on `AttachmentCount` from the `ListPolicies` API.

### Command

> [!INFO]
> IAM service is global, so region option `-r <region>` is not supported.

```bash
idled -s iam
```

## Cost Model

- The IAM service itself is free. Therefore, removing idle IAM entities does not directly reduce costs.
- However, it provides significant benefits by reducing security risks and improving management efficiency.

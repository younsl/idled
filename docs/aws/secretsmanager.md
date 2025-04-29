# AWS Secrets Manager

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category |
|----------|-------------------|----------|
| AWS      | Regional          | Security |

AWS Secrets Manager helps you protect access to your applications, services, and IT resources without the upfront investment and on-going maintenance costs of operating your own infrastructure. However, secrets created for testing, temporary purposes, or for applications that no longer exist might remain unused. While Secrets Manager charges per secret stored and per API call, identifying and deleting unused secrets helps maintain security hygiene (reducing attack surface) and can lead to minor cost savings, especially if many secrets are stored.

## Scan Criteria

`idled` identifies Secrets Manager secrets as **idle** based on the following criterion:

-   **Last Accessed Date:** The secret has not been accessed (retrieved via API) in the last **90 days**. This is determined by checking the `LastAccessedDate` field returned by the `ListSecrets` API call.
    -   Note: Secrets that have never been accessed (`LastAccessedDate` is null) are currently **not** flagged as idle by this scanner.

## Command

To scan for idle Secrets Manager secrets, use the `--services secretsmanager` flag. You must also specify the region(s) to scan.

```bash
idled -s secretsmanager -r <REGION>
```

Example:

```bash
export AWS_PROFILE=your-profile
idled --services secretsmanager --regions us-east-1,eu-west-1
```

## Cost Model

- Secrets Manager charges primarily based on the number of secrets stored per month and the number of API calls made.
- Deleting idle secrets reduces the monthly storage cost for those secrets.
- `idled` identifies potentially idle secrets based on access patterns but **does not currently calculate** the specific cost savings. 
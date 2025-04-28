# AWS Managed Streaming for Kafka (MSK)

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category           |
|----------|-------------------|--------------------|
| AWS      | Regional          | Analytics/Streaming |

AWS Managed Streaming for Kafka (MSK) makes it easy to build and run applications that use Apache Kafka to process streaming data. However, MSK clusters incur costs based on broker instance hours, storage, and data transfer. Clusters that are provisioned but unused (no connections) or significantly underutilized (very low CPU usage) represent unnecessary expenses. Identifying these clusters helps optimize costs.

## Scan Criteria

`idled` identifies MSK clusters as **idle or underutilized** if they meet either of the following criteria over the last 30 days (based on CloudWatch metrics):

1.  **No Connections:** The maximum `ConnectionCount` metric for the cluster was 0.
2.  **Low CPU Usage:** The average combined CPU utilization (`CpuUser` + `CpuSystem`) metric was below 30%.

## Command

To scan for idle/underutilized MSK clusters, use the `--services msk` flag. You must also specify the region(s) to scan.

```bash
idled -s msk -r <REGION>
```

Example:

```bash
export AWS_PROFILE=your-profile
idled --services msk --regions us-east-1,us-west-2
```

## Cost Model

- MSK clusters incur costs based on broker instance type and running hours, provisioned storage, and data transfer fees.
- Even idle or underutilized clusters incur charges for the provisioned instances and storage.
- Deleting identified idle/underutilized clusters can lead to significant cost savings.
- `idled` identifies potentially idle/underutilized MSK clusters based on activity metrics but **does not currently calculate** the specific cost savings for each cluster.

## Output Details

The output for MSK scans includes the following information:

-   **CLUSTER NAME:** The name of the MSK cluster.
-   **ARN:** The Amazon Resource Name (ARN) of the cluster.
-   **REGION:** The AWS region where the cluster is located.
-   **STATE:** The current state of the cluster (e.g., ACTIVE, CREATING).
-   **INSTANCE TYPE:** The EC2 instance type used for the broker nodes.
-   **CREATION TIME:** The timestamp when the cluster was created.
-   **IS IDLE:** Indicates if the cluster is identified as idle/underutilized (`true` or `false`).
-   **REASON:** Explains why the cluster was flagged (e.g., "No connections in the last 30 days", "Average CPU utilization < 30% in the last 30 days"). 
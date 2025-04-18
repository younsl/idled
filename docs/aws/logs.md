# AWS CloudWatch Logs

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Output Columns](#output-columns)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category      |
|----------|-------------------|---------------|
| AWS      | Regional          | Observability |

CloudWatch Log Groups can accumulate significant amounts of log data over time, leading to increased storage costs. Log groups that haven't received new log events for an extended period might indicate unused applications, inactive resources, or leftover logging configurations. Identifying these idle log groups helps in cleaning up unnecessary resources and potentially reducing AWS costs associated with log storage and management.

## Scan Criteria

`idled` determines if a CloudWatch Log Group is potentially idle based on the **timestamp of the last ingested log event**.

1.  It lists all log groups using the `DescribeLogGroups` API.
2.  For each log group, it attempts to find the timestamp of the most recent log event using the `FilterLogEvents` API (searching for 1 event).
3.  **Primary Check:** If a last event timestamp is found, it's compared against the idle threshold (e.g., 90 days). If the last event is older than the threshold, the log group is flagged as idle.
4.  **Fallback Check:** If no log events are found (e.g., the group is empty or new) or an error occurs during the event check, the **log group's creation time** is used as a fallback for the idleness comparison.
5.  Log groups where the effective timestamp (last event or creation time) is older than the threshold are included in the results.

**Note:** Using `FilterLogEvents` provides more accuracy but increases scan time and API calls compared to only checking creation time or stored bytes.

### Command

```bash
idled -s logs -r <REGION>
```

The output table includes the following columns:

- **LOG GROUP NAME:** The name of the CloudWatch Log Group.
- **RETENTION:** The retention period configured (days or "Never expire").
- **SIZE:** The total stored size of log data (e.g., KB, MB, GB).
- **CREATED:** The date the log group was created (YYYY-MM-DD).
- **LAST EVENT:** The date of the last recorded log event (YYYY-MM-DD), or "N/A (Created: YYYY-MM-DD)" if using creation time as fallback.

## Cost Model

CloudWatch Logs costs are primarily based on three factors:

1. **Ingestion:** Charges per GB of log data sent to CloudWatch Logs.
2. **Storage:** Charges per GB per month for log data stored. Pricing can vary based on the storage class (e.g., Standard) and the configured **retention period**. Longer retention means higher storage costs.
3. **Analysis:** Charges for running queries using CloudWatch Logs Insights (per GB of data scanned).

Identifying and removing idle log groups primarily helps reduce **storage costs**. If a log group is no longer receiving logs (no ingestion cost), the existing stored data will continue to incur charges until it expires based on the retention policy or the log group is deleted.

`idled` helps identify potentially unused log groups based on the last event time or creation time. However, **it does not calculate potential cost savings** for CloudWatch Logs due to the complexity of tiered storage pricing and variable ingestion patterns. The provided information (Size, Retention) should be used for manual cost assessment before deciding to delete a log group.

# ECR Repositories

## Table of Contents

- [Scan Rationale](#scan-rationale)
- [Scan Criteria](#scan-criteria)
- [Command](#command)
- [Cost Model](#cost-model)

## Scan Rationale

| Provider | Regional / Global | Category  |
|----------|-------------------|-----------|
| AWS      | Regional          | Container |

Amazon Elastic Container Registry (ECR) is a fully managed container registry service. Over time, repositories may become inactive, containing old or unused images. While ECR storage costs are generally lower than EBS, identifying and potentially cleaning up unused repositories helps maintain a clean environment and can contribute to minor cost savings.

## Scan Criteria

- `idled` identifies ECR repositories based on the **last image push time** across all images within that repository.
- It determines the most recent push timestamp by examining the `imagePushedAt` field for all images via the `DescribeImages` API call.
- A repository is considered potentially **idle** if this most recent push time is older than a configurable threshold.
- The default threshold for considering a repository idle is **90 days** without any image push events.
- The tool also displays the total number of images in each repository.

### Command

```bash
idled -s ecr -r <REGION>
```

## Cost Model

- ECR storage costs are based on the amount of data stored in your repositories.
- `idled` **currently does not calculate** the specific storage cost for each identified idle repository. It focuses on identifying inactivity based on the last push time.
- Future enhancements could potentially integrate ECR storage pricing.
# Available Services

- [Supported Services](#supported-services)
- [Usage](#usage)

## Supported Services

### Amazon Web Services

The following AWS services are currently supported by `idled`:

| Service | Status    | Resource | Remarks |
|---------|-----------|----------|---------|
| [EC2](docs/aws/ec2.md) | ✅ Supported | Stopped EC2 instances (Default) | Detects stopped EC2 instances |
| [EBS](docs/aws/ebs.md) | ✅ Supported | Unattached EBS volumes | Detects unattached EBS volumes |
| [S3](docs/aws/s3.md) | ✅ Supported | Idle S3 buckets | Detects idle S3 buckets |
| [Lambda](docs/aws/lambda.md) | ✅ Supported | Idle Lambda functions | Detects idle Lambda functions |
| [EIP](docs/aws/eip.md) | ✅ Supported | Unattached Elastic IPs | Detects unattached Elastic IPs |
| [IAM](docs/aws/iam.md) | ✅ Supported | Idle IAM users, roles, and policies | Detects unused IAM resources |
| [Config](docs/aws/config.md) | ✅ Supported | Idle Config rules, recorders, and delivery channels | Detects unused Config resources |
| [ELB](docs/aws/elb.md) | ✅ Supported | Idle ALBs and NLBs with no targets or zero traffic in the last 14 days | Detects idle ALBs and NLBs |
| [Logs](docs/aws/logs.md) | ✅ Supported | Idle CloudWatch Log Groups | Detects idle CloudWatch Log Groups |

## Command Usage

You can use the `--list-services` (or `-l`) flag to list all supported services.

```bash
idled --list-services
```

You can specify which services to scan using the `--services` (or `-s`) flag, providing a comma-separated list (e.g., `-s ec2,ebs,s3`). If no services are specified, `idled` will scan the default service (`ec2`).

```bash
idled -s ec2,ebs,s3
```
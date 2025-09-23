# AWS RDS IAM Authentication Support for PostgreSQL Scaler

## Overview

This enhancement adds support for AWS RDS IAM Database Authentication to the KEDA PostgreSQL scaler. It allows KEDA to automatically generate and refresh temporary authentication tokens when connecting to AWS RDS PostgreSQL instances, eliminating the need for static database passwords.

## Features

- **Automatic Detection**: Detects when running on AWS with IRSA (IAM Roles for Service Accounts) and connecting to an RDS endpoint
- **Token Generation**: Automatically generates RDS IAM authentication tokens valid for 15 minutes
- **Token Refresh**: Refreshes tokens before expiry, ensuring continuous operation
- **Seamless Integration**: Works alongside existing authentication methods (password-based and Azure Workload Identity)

## Prerequisites

1. **AWS RDS Instance**: PostgreSQL instance with IAM database authentication enabled
2. **IRSA Setup**: KEDA operator running with a ServiceAccount that has an IAM role attached
3. **IAM Permissions**: The IAM role must have `rds-db:connect` permission for the target database user
4. **Database User**: PostgreSQL user created with IAM authentication enabled

## Configuration

### 1. Enable IAM Authentication on RDS Instance

```bash
aws rds modify-db-instance \
  --db-instance-identifier your-instance \
  --enable-iam-database-authentication \
  --apply-immediately
```

### 2. Create IAM Role for KEDA

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "rds-db:connect",
      "Resource": "arn:aws:rds-db:region:account-id:dbuser:db-resource-id/database-username"
    }
  ]
}
```

### 3. Annotate KEDA ServiceAccount

```bash
kubectl annotate serviceaccount keda-operator -n keda \
  eks.amazonaws.com/role-arn=arn:aws:iam::account-id:role/keda-rds-access
```

### 4. Create Database User

```sql
CREATE USER keda_reader WITH LOGIN;
GRANT rds_iam TO keda_reader;
GRANT SELECT ON your_table TO keda_reader;
```

### 5. Configure ScaledObject

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: postgres-scaler
spec:
  triggers:
  - type: postgresql
    metadata:
      host: "your-instance.region.rds.amazonaws.com"
      port: "5432"
      userName: "keda_reader"  # No password needed!
      dbName: "your_database"
      sslmode: "require"
      query: "SELECT COUNT(*) FROM your_table"
      targetQueryValue: "5"
```

## How It Works

1. **Detection**: When no password is provided and the host matches an RDS endpoint pattern, the scaler checks for IRSA credentials
2. **Token Generation**: Uses the AWS SDK to generate a temporary authentication token
3. **Connection**: Uses the token as the password in the PostgreSQL connection string
4. **Refresh**: Monitors token expiry and generates new tokens as needed (14 minutes after generation)

## Supported RDS Endpoints

The scaler recognizes the following RDS endpoint patterns:
- `*.rds.amazonaws.com` - Standard RDS endpoints
- `*.rds.amazonaws.com.cn` - China region endpoints
- `*.rds-fips.amazonaws.com` - FIPS-compliant endpoints

## Environment Variables

The scaler respects standard AWS environment variables:
- `AWS_WEB_IDENTITY_TOKEN_FILE` - Set by IRSA
- `AWS_ROLE_ARN` - Set by IRSA
- `AWS_REGION` - Optional, extracted from endpoint if not set
- `AWS_DEFAULT_REGION` - Fallback region

## Troubleshooting

### Common Issues

1. **No IRSA credentials found**
   - Ensure the ServiceAccount has the correct IAM role annotation
   - Verify the pod has the IRSA environment variables set

2. **Permission denied**
   - Check the IAM role has `rds-db:connect` permission
   - Verify the database user has `rds_iam` role granted

3. **Connection refused**
   - Ensure RDS instance has IAM authentication enabled
   - Verify SSL/TLS is enabled (required for IAM auth)

### Debug Logging

Enable verbose logging to see token generation details:

```yaml
spec:
  triggers:
  - type: postgresql
    metadata:
      # ... other metadata ...
    authenticationRef:
      name: postgres-trigger-auth
```

## Compatibility

- Works with all PostgreSQL versions supported by RDS
- Compatible with existing KEDA features (metrics API, scaling behaviors, etc.)
- Coexists with password-based and Azure Workload Identity authentication

## Security Benefits

- **No Static Passwords**: Eliminates hardcoded database passwords
- **Short-lived Tokens**: Tokens expire after 15 minutes, reducing exposure window
- **IAM Integration**: Leverages AWS IAM for centralized access control
- **Audit Trail**: All database access is logged via CloudTrail

## Migration Guide

To migrate from password-based authentication:

1. Enable IAM authentication on your RDS instance
2. Create a new database user with IAM authentication
3. Grant necessary permissions to the new user
4. Setup IRSA for KEDA operator
5. Update ScaledObject to remove password and use IAM user
6. Monitor logs to ensure successful authentication

## Example Use Cases

### Auto-scaling based on job queue

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: job-processor-scaler
spec:
  scaleTargetRef:
    name: job-processor
  triggers:
  - type: postgresql
    metadata:
      host: "postgres.us-west-2.rds.amazonaws.com"
      port: "5432"
      userName: "keda_reader"
      dbName: "jobs_db"
      sslmode: "require"
      query: "SELECT COUNT(*) FROM job_queue WHERE status = 'pending'"
      targetQueryValue: "10"
```

### Multi-tenant scaling

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: tenant-scaler
spec:
  scaleTargetRef:
    name: tenant-processor
  triggers:
  - type: postgresql
    metadata:
      host: "shared-db.us-east-1.rds.amazonaws.com"
      port: "5432"
      userName: "tenant_monitor"
      dbName: "tenants"
      sslmode: "require"
      query: "SELECT SUM(pending_tasks) FROM tenant_metrics"
      targetQueryValue: "100"
```

## Contributing

For issues or improvements related to AWS RDS IAM authentication support, please open an issue or PR at [github.com/kedacore/keda](https://github.com/kedacore/keda).
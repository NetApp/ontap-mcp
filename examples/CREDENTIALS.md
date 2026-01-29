# Credentials Management

ontap-mcp supports multiple methods for managing authentication credentials with a priority-based system. This allows you to choose the best approach for your security requirements.

## Authentication Priority Order

Credentials are resolved in the following priority order:

1. **credentials_script** - Dynamic credentials from an external script (highest priority)
2. **credentials_file** - Static credentials from a separate YAML file
3. **Inline config** - Username/password directly in ontap.yml (lowest priority)

## Method 1: Credentials File

The credentials_file feature allows you to store credentials in a separate YAML file, keeping sensitive information out of your main configuration.

### Configuration

In your `ontap.yml`:

```yaml
Pollers:
  cluster1:
    addr: 10.193.48.11
    credentials_file: secrets/cluster1.yml
```

In your `secrets/cluster1.yml`:

```yaml
Pollers:
  cluster1:
    username: harvest
    password: mySecurePassword
```

### Features

- **Separate credentials**: Keep secrets in a different file from your main config
- **Multiple clusters**: One credentials file can contain credentials for multiple clusters
- **Optional username**: If username is not in the credentials file, it will use the one from ontap.yml
- **File permissions**: Set restrictive permissions on the credentials file (e.g., `chmod 600 secrets/cluster1.yml`)

### Example: Multiple Clusters in One File

You can store credentials for multiple clusters in a single credentials file:

`ontap.yml`:
```yaml
Pollers:
  prod-cluster:
    addr: 10.1.1.100
    credentials_file: secrets/credentials.yml
  
  dev-cluster:
    addr: 10.1.2.100
    credentials_file: secrets/credentials.yml
```

`secrets/credentials.yml`:
```yaml
Pollers:
  prod-cluster:
    username: prod-admin
    password: prodSecurePass123
  
  dev-cluster:
    username: dev-admin
    password: devSecurePass456
```

### Best Practices

1. **Store credentials files outside the main config directory**
2. **Use restrictive file permissions**: `chmod 600 credentials.yml`
3. **Don't commit credentials files to version control**: Add to `.gitignore`
4. **Use separate files for different environments**: `prod-creds.yml`, `dev-creds.yml`

## Method 2: Credentials Script

The credentials_script feature allows you to fetch authentication credentials dynamically via an external script.

### Use Cases

- Integrating with secrets managers (AWS Secrets Manager, HashiCorp Vault, etc.)
- Implementing password rotation policies
- Fetching temporary credentials from external auth systems
- Dynamic token generation

### Configuration

```yaml
Pollers:
  ontap1:
    addr: 10.1.1.1
    username: admin  # Optional: can be provided by the script instead
    credentials_script:
      path: ./get_credentials.sh
      schedule: 3h
      timeout: 10s
```

### Configuration Options

- **path** (required): Path to the executable script
- **schedule** (optional, default: `24h`): How often to refresh credentials
  - Set to `always` to call the script on every request
  - Use duration format: `1h`, `30m`, `24h`, etc.
- **timeout** (optional, default: `10s`): Maximum time to wait for the script to complete

### Script Requirements

#### Arguments

Your script will be called with 1 or 2 arguments:

```bash
./script $addr           # When no username is in the config
./script $addr $username # When username is in the config
```

- `$1` = Cluster address from the `addr` field
- `$2` = Username from the `username` field (if present)

#### Output Format

The script must output valid YAML to stdout. Three response formats are supported:

**1. Password Only**

```yaml
password: MySecretPassword
```

The username from the config will be used.

**2. Password and Username**

```yaml
username: admin
password: MySecretPassword
```

**3. Auth Token (Bearer Token)**

```yaml
authToken: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

When an auth token is provided, it will be used for Bearer token authentication.

### Example Scripts

#### AWS Secrets Manager Integration

```bash
#!/bin/bash
CLUSTER_ADDR="$1"

# Fetch from AWS Secrets Manager
SECRET=$(aws secretsmanager get-secret-value \
  --secret-id "ontap/${CLUSTER_ADDR}" \
  --query SecretString \
  --output text)

# Parse JSON secret
USERNAME=$(echo "$SECRET" | jq -r .username)
PASSWORD=$(echo "$SECRET" | jq -r .password)

echo "username: $USERNAME"
echo "password: $PASSWORD"
```

#### HashiCorp Vault Integration

```bash
#!/bin/bash
CLUSTER_ADDR="$1"

# Fetch from Vault
vault kv get -format=json "secret/ontap/${CLUSTER_ADDR}" | \
  jq -r '"password: " + .data.data.password'
```

#### Token-Based Authentication

```bash
#!/bin/bash
CLUSTER_ADDR="$1"

# Get a temporary auth token
TOKEN=$(curl -s "https://auth.example.com/api/tokens/create?cluster=${CLUSTER_ADDR}")

echo "authToken: $TOKEN"
```

### Credential Caching

Credentials are cached based on the `schedule` setting:

- **First request**: Script is executed, credentials are cached
- **Subsequent requests**: Cached credentials are used until the schedule expires
- **After expiration**: Script is executed again to refresh credentials

## Method 3: Inline Configuration

The simplest method is to include credentials directly in your `ontap.yml`:

```yaml
Pollers:
  cluster1:
    addr: 10.1.1.1
    username: admin
    password: myPassword
```

**Note**: This is the least secure method and should only be used in development environments.

## Priority Examples

### Example 1: Script Overrides Everything

```yaml
Pollers:
  cluster1:
    addr: 10.1.1.1
    username: config-user
    password: config-pass
    credentials_file: secrets/creds.yml
    credentials_script:
      path: ./get_creds.sh
      schedule: 1h
```

**Result**: Script credentials are used (highest priority)

### Example 2: File Overrides Inline

```yaml
Pollers:
  cluster1:
    addr: 10.1.1.1
    username: config-user
    password: config-pass
    credentials_file: secrets/creds.yml
```

**Result**: Credentials from file are used

### Example 3: Inline Only

```yaml
Pollers:
  cluster1:
    addr: 10.1.1.1
    username: config-user
    password: config-pass
```

**Result**: Inline credentials are used

## Security Best Practices

1. **Restrict file permissions**: `chmod 600` for credentials files
2. **Never commit secrets to version control**: Use `.gitignore`
3. **Use environment-specific credentials**: Separate prod/dev/test
4. **Rotate credentials regularly**: Implement automatic rotation
5. **Monitor access**: Log and audit credential usage
6. **Use script-based credentials for production**: Integrate with secrets managers
7. **Validate script execution**: Ensure scripts are owned by trusted users

## Troubleshooting

### Credentials File Issues

**File not found:**
- Check the path is relative to where ontap-mcp is running
- Use absolute paths if needed: `/opt/ontap-mcp/secrets/creds.yml`

**Cluster not found in file:**
- Ensure the cluster name in the file matches the ontap.yml poller name exactly
- Check YAML formatting (proper indentation, no tabs)

**Permission denied:**
- Check file permissions: `ls -l credentials.yml`
- Ensure the user running ontap-mcp can read the file

### Script Issues

**Script not executing:**
- Check file permissions: `chmod +x script.sh`
- Test manually: `./script.sh 10.1.1.1 admin`

**Invalid YAML errors:**
- Test script output: `./script.sh 10.1.1.1 admin | cat -A`
- Ensure proper YAML formatting

**Timeout issues:**
- Test script execution time: `time ./script.sh 10.1.1.1 admin`
- Increase timeout if needed: `timeout: 30s`

## Example Configuration

Complete example showing all authentication methods:

```yaml
Pollers:
  # Production: Use script with secrets manager
  prod-cluster:
    addr: 10.1.1.100
    username: admin
    credentials_script:
      path: /opt/ontap-mcp/scripts/get_prod_credentials.sh
      schedule: 1h
      timeout: 10s

  # Staging: Use credentials file
  staging-cluster:
    addr: 10.1.2.100
    credentials_file: /opt/ontap-mcp/secrets/staging-creds.yml

  # Development: Use inline (least secure)
  dev-cluster:
    addr: 10.1.3.100
    username: dev-admin
    password: devpass123
```

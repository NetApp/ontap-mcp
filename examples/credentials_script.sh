#!/bin/bash
# Example credentials script for ontap-mcp
# This script receives the cluster address as $1 and optionally the username as $2
# It must output YAML to stdout with password (required) and optionally username or authToken

# Arguments:
CLUSTER_ADDR="$1"
USERNAME="$2"

# Example 1: Return password only (username will be taken from config)
# echo "password: MySecretPassword"

# Example 2: Return both username and password
# echo "username: admin"
# echo "password: MySecretPassword"

# Example 3: Return auth token (preferred for token-based auth)
# echo "authToken: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Example 4: Fetch from environment variables
# echo "password: ${ONTAP_PASSWORD}"

# Example 5: Fetch from a secrets manager (e.g., AWS Secrets Manager, HashiCorp Vault, etc.)
# This example assumes you have a secrets manager CLI installed
# SECRET=$(secrets-manager get "ontap/${CLUSTER_ADDR}")
# echo "password: ${SECRET}"

# Example 6: Generate a temporary password from an external auth system
# TEMP_PASS=$(curl -s "https://auth.example.com/get-temp-password?host=${CLUSTER_ADDR}&user=${USERNAME}")
# echo "password: ${TEMP_PASS}"

# For this example, we'll just return a hardcoded password
# In production, fetch from a secure location!
echo "password: your-secure-password-here"

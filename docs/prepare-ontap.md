## Prepare ONTAP cDot clusters for ONTAP-MCP

ONTAP-MCP requires login credentials to access monitored hosts. An admin account is needed to create, update, and delete cluster resources. If you don't want to use the default admin account, please create a generic admin account to be used by the ONTAP-MCP.

If you want to limit the ONTAP-MCP's access to specific SVMs or read-only action, you can create a role with the appropriate permissions and assign it to the user.


## ontap.yaml configuration

The ONTAP-MCP server uses a configuration file to store cluster connection details.

!!! note "Cluster names in your ontap.yaml"

    The cluster names you specify in the `ontap.yaml` file will be used as identifiers for those clusters in the MCP API, so choose them wisely.

Below is an example `ontap.yaml` configuration file for two clusters, `sar` and `sar2`. 
If you use Harvest, this file shares the same format as the `harvest.yml`, so you can use your `harvest.yml` with to the ONTAP-MCP.

```yaml
Pollers:
  sar:
    addr: 10.193.48.11
    use_insecure_tls: true
    username: admin
    credentials_script:
      path: /path/to/credentials_script
      schedule: always
  sar2:
    addr: 10.195.15.41
    use_insecure_tls: true
    username: admin
    password: password
```

Below is a table describing the configuration options:

| Option               | Type              | Description                                                                                                                                                                    | Default |
|----------------------|-------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| poller-name          | required          | The IP address or hostname of the ONTAP cluster.                                                                                                                               | -       |
| `addr`               | required          | The IPv4, IPv6 or FQDN of the ONTAP cluster.                                                                                                                                   | -       |
| `username`           |                   | The username for authentication.                                                                                                                                               | -       |
| `password`           |                   | The password for authentication. Not recommended for production use. Use `credentials_script` or `credentials_file` instead. See [authentication](#authentication) for details | -       |
| `use_insecure_tls`   | optional, bool    | Set to `true` to allow insecure TLS connections (e.g., self-signed certificates). Not recommended for production use.                                                          | false   |
| `credentials_file`   | optional, string  | Path to a yaml file that contains cluster credentials. The file should have the same shape as ontap.yaml. Path can be relative to ontap.yaml or absolute.                      |         |
| `credentials_script` | optional, section | Section that defines how ONTAP-MCP should fetch credentials via external script. See [here](#credentials-script) for details. 	                                                |         |
 
# Authentication

The ONTAP-MCP server supports multiple methods for managing authentication credentials with a priority-based system. 
This allows you to choose the best approach for your security requirements.

## Authentication Priority Order

When multiple authentication methods are configured for a cluster, the ONTAP-MCP server will resolve them in the following priority order:

1. **Static Credentials**: If static credentials are defined directly in the `ontap.yaml` file, they will be used.
2. **Credentials Script**: If a credentials script is defined, the server will execute the script to retrieve credentials.
3. **Credentials File**: If a credentials file is specified, the server will read credentials from the file.

If no authentication method is configured, the server will not be able to connect to the cluster and will log an error.
 
## Static Credentials 

Store the `username` and `password` directly in the `ontap.yaml` file. 
Storing the password in your `ontap.yaml` is not recommended for production use due to security risks.

## Credentials File

The `credentials_file` feature allows you to store credentials in a separate YAML file, keeping sensitive information out of your main configuration.

At runtime, the `credentials_file` will be read and the included credentials will be used to authenticate with the matching cluster(s).

The format of the `credentials_file` is similar to `ontap.yaml` and can contain multiple cluster credentials. 

Example:

Snippet from `ontap.yaml`:

```yaml
Pollers:
  cluster1:
    addr: sar
    credentials_file: secrets/cluster1.yml
```

Example `secrets/cluster1.yml`:

```yaml
Pollers:
  cluster1:
    username: admin
    password: admin-password
```

Set restrictive permissions on the credentials file (e.g., `chmod 600 secrets/cluster1.yml`)

## Credentials Script

The `credentials_script` feature allows you to fetch authentication credentials dynamically via an external script.
This can be configured in the `Pollers` section of your `ontap.yaml` file, as shown in the example below.

At runtime, ONTAP-MCP will invoke the script specified in the `credentials_script` `path` section. 
ONTAP-MCP will call the script with one or two arguments depending on how your poller is configured
in the `ontap.yaml` file. The script will be called like this: `./script $addr` or `./script $addr $username`.

- The first argument `$addr` is the address of the cluster taken from the `addr` field under the `Pollers` section of your `ontap.yaml` file.
- The second argument `$username` is the username for the cluster taken from the `username` field under the `Pollers` section of your `ontap.yaml` file.
  If your `ontap.yaml` does not include a username, nothing will be passed.

The script should return the credentials to ONTAP-MCP by writing the response to the script's standard output (stdout) as YAML.

### YAML format

If the script outputs a YAML object with `username` and `password` keys, ONTAP-MCP will use both the `username` and `password` from the output. 
For example, if the script writes the following, ONTAP-MCP will use `myuser` and `mypassword` for the cluster's credentials.

   ```yaml
   username: myuser
   password: mypassword
   ```
If only the `password` is provided, ONTAP-MCP will use the `username` from the `ontap.yaml` file, if available. 
If your username or password contains spaces, `#`, or other characters with special meaning in YAML, make sure you quote the value like so:
`password: "my password with spaces"`

If the script outputs a YAML object containing an `authToken`, ONTAP will use this `authToken` when communicating with the ONTAP cluster.
ONTAP-MCP will include the `authToken` in the HTTP request's authorization header using the Bearer authentication scheme.

   ```yaml
   authToken: eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJEcEVkRmgyODlaTXpYR25OekFvaWhTZ0FaUnBtVlVZSDJ3R3dXb0VIWVE0In0.eyJleHAiOjE3MjE4Mj
   ```
When using `authToken`, the `username` and `password` fields are ignored if they are present in the script's output.

If the script doesn't finish within the specified `timeout`, ONTAP-MCP will terminate the script and any spawned processes.

Credential scripts are defined under the `credentials_script` section within `Pollers` in your `ontap.yaml`.
Below are the options for the `credentials_script` section:

| parameter | type                    | description                                                                                                                                                                  | default |
|-----------|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| path      | string                  | Absolute path to the script that takes two arguments: `addr` and `username`, in that order.                                                                                  |         |
| schedule  | go duration or `always` | Schedule for calling the authentication script. If set to `always`, the script is called every time a password is requested; otherwise, the previously cached value is used. | 24h     |
| timeout   | go duration             | Maximum time ONTAP-MCP will wait for the script to finish before terminating it and its descendants.                                                                         | 10s     |

### Example

Here is an example of how to configure the `credentials_script` in the `ontap.yaml` file:

```yaml
Pollers:
  ontap1:
    addr: 10.1.1.1
    username: admin # Optional: if not provided, the script must return the username
    credentials_script:
      path: ./get_credentials
      schedule: 3h
      timeout: 10s
```

In this example, the `get_credentials` script should be located in the same directory as the `ontap.yaml` file and should be executable.
It should output the credentials in a YAML format. Here are two example scripts:

`get_credentials` that outputs username and password in YAML format:
```bash
#!/bin/bash
cat << EOF
username: myuser
password: mypassword
EOF
```

`get_credentials` that outputs authToken in YAML format:
```bash
#!/bin/bash
# script requests an access token from the authorization server
# authorization returns an access token to the script
# script writes the YAML formatted authToken like so:
cat << EOF
authToken: $authToken
EOF
```

Below are a couple of OAuth2 credential script examples for authenticating with ONTAP OAuth2-enabled clusters.

??? note "These are examples that you will need to adapt to your environment."

    Example OAuth2 script authenticating with the Keycloak auth provider via `curl`. Uses [jq](https://github.com/jqlang/jq) to extract the token. This script outputs the authToken in YAML format.

    ```bash
    #!/bin/bash

    response=$(curl --silent "http://{KEYCLOAK_IP:PORT}/realms/{REALM_NAME}/protocol/openid-connect/token" \
      --header "Content-Type: application/x-www-form-urlencoded" \
      --data-urlencode "grant_type=password" \
      --data-urlencode "username={USERNAME}" \
      --data-urlencode "password={PASSWORD}" \
      --data-urlencode "client_id={CLIENT_ID}" \
      --data-urlencode "client_secret={CLIENT_SECRET}")

    access_token=$(echo "$response" | jq -r '.access_token')

    cat << EOF
    authToken: $access_token
    EOF
    ```

    Example OAuth2 script authenticating with the Auth0 auth provider via `curl`. Uses [jq](https://github.com/jqlang/jq) to extract the token. This script outputs the authToken in YAML format.

    ```bash
    #!/bin/bash
    response=$(curl --silent https://{AUTH0_TENANT_URL}/oauth/token \
      --header 'content-type: application/json' \
      --data '{"client_id":"{CLIENT_ID}","client_secret":"{CLIENT_SECRET}","audience":"{ONTAP_CLUSTER_IP}","grant_type":"client_credentials"')

    access_token=$(echo "$response" | jq -r '.access_token')

    cat << EOF
    authToken: $access_token
    EOF
    ```

### Troubleshooting

* Make sure your script is executable
* When running ONTAP-MCP from a container, ensure that you have mounted the credential script so that it is available inside the container and that you have updated the path in the `ontap.yaml` file to reflect the path inside the container.
* If running ONTAP-MCP from a container, ensure that your [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)) references an interpreter that exists inside the container. ONTAP-MCP containers are built from [Distroless](https://github.com/GoogleContainerTools/distroless) images, so you may need to use `#!/busybox/sh`.
* Ensure the user/group that executes your poller also has read and execute permissions on the script.
  One way to test this is to `su` to the user/group that runs ONTAP-MCP
  and ensure that the `su`-ed user/group can execute the script too.
* Make sure that your script emits valid YAML. You can use [YAML Lint](http://www.yamllint.com/) to check your output. Test script output with `./script.sh 10.1.1.1 admin | yamllint -` to ensure the output is valid YAML.
* When you want to include debug logging from your script, make sure to redirect the debug output to `stderr` instead of `stdout`, or write the debug output as YAML comments prefixed with `#.`
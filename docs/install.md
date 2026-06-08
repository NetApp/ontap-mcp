# Installation

The ONTAP MCP Server is distributed as a Docker container image, or you can build it from source.

## Container Images

ONTAP MCP Server is available as pre-built container images:

| Image                              | Description               |
|------------------------------------|---------------------------|
| `ghcr.io/netapp/ontap-mcp:latest`  | Stable release version    |
| `ghcr.io/netapp/ontap-mcp:nightly` | Latest development builds |

## MCP Client Integration

To integrate the ONTAP MCP server with your MCP client (e.g., GitHub Copilot, Claude Desktop), configure your `mcp.json` file to connect to the MCP server.

Start the ONTAP MCP server:

```bash
docker run -d \
  --name ontap-mcp-server \
  -p 8083:8083 \
  -v /path/to/your/ontap.yaml:/opt/mcp/ontap.yaml \
  ghcr.io/netapp/ontap-mcp:latest \
  start --port 8083 --host 0.0.0.0
```

If you only want to bind to localhost, omit the `--host` option.

See [Configuration File](#configuration-file) below for details on preparing `ontap.yaml`.

Then configure your mcp.json:

```json
{
  "servers": {
    "ontap-mcp": {
      "type": "http",
      "url": "http://your-server-ip:8083"
    }
  }
}
```

## Building from Source

### Prerequisites

- Go(check `.go.env` in the repository root for the exact required version)
- Git
- [Just](https://just.systems/)
- Docker (optional, for building Docker images)

### Clone the Repository

First, clone the ontap-mcp repository:

```bash
git clone https://github.com/NetApp/ontap-mcp.git
cd ontap-mcp
```

### Build Docker Image

Build your own Docker image from source:

```bash
# Build the Docker image using make (creates ontap-mcp:latest by default)
just docker-build

# Or specify a custom tag
just docker-build DOCKER_TAG=ontap-mcp:local
```

Alternatively, build directly with Docker:

```bash
# From the ontap-mcp repository root
docker build -f Dockerfile -t ontap-mcp:local .
```

### Running the Built Docker Image

After building, use your local image in your MCP client configuration. See [MCP Client Integration](#mcp-client-integration) above for configuration examples - just replace `ghcr.io/netapp/ontap-mcp:latest` with your local image tag (e.g., `ontap-mcp:local`).

## Configuration File

The server requires a YAML config file that defines which ONTAP clusters to connect to.
By default, it looks for `ontap.yaml` in its working directory (`/opt/mcp` inside the container).

Create your config file based on the [ontap-example.yaml](https://github.com/NetApp/ontap-mcp/blob/main/ontap-example.yaml) template:

```yaml
Pollers:
  cluster1:
    addr: 10.0.0.1
    username: admin
    password: password
    use_insecure_tls: true
```

Mount it into the container using `-v`:

```bash
docker run -d \
  --name ontap-mcp-server \
  -p 8083:8083 \
  -v /path/to/your/ontap.yaml:/opt/mcp/ontap.yaml \
  ghcr.io/netapp/ontap-mcp:latest \
  start --port 8083 --host 0.0.0.0
```

## OAuth Authentication for ONTAP MCP server

If you want to configure OAuth authentication to prevent unauthorized access to your MCP server. Add `McpAuth` section in your config file based on the [ontap-example.yaml](https://github.com/NetApp/ontap-mcp/blob/main/ontap-example.yaml) template:
By default, it looks for `ontap.yaml` in its working directory (`/opt/mcp` inside the container).

sample contents of `ontap.yaml`
```yaml
McpAuth:
  issuer: http://localhost:9090/realms/REALM
  alg: RS256
  audience_required: http://localhost:8080

Pollers:
  cluster1:
    addr: 10.0.0.1
    username: admin
    password: password
    use_insecure_tls: true
```

Below is a table describing the configuration options in `McpAuth` section:

| Option              | Type             | Description                                                                                                                                                                                                                                                                                                                                                           | Default        |
|---------------------|------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------|
| `issuer`            | required, string | Name of the Issuer of the auth token to generate the jwks_URI to fetch public keys.                                                                                                                                                                                                                                                                                   |                |
| `alg`               | optional, string | Algorithm used to generate the token, which would be used to validate the given Bearer token with public keys. Supported asymmetric algorithms are: <br/>RSA Digital Signatures[RS256, RS384, RS512], <br/>RSA-PSS Digital Signatures[PS256, PS384, PS512], <br/>ECDSA (Elliptic Curve) Signatures[ES256, ES384, ES512], <br/>EdDSA (Edwards-curve) Signatures[EdDSA] | RS256          |
| `audience_required` | optional, string | Expected Audience which are allowed to access the mcp tools. Default would be the MCP server URL, http://localhost:8080                                                                                                                                                                                                                                               | MCP_SERVER_URL |

To integrate the ONTAP MCP server with your MCP client (e.g., GitHub Copilot, Claude Desktop), configure your `mcp.json` file with Authentication header with bearer `AUTH_TOKEN` as below to connect to the MCP server with OAuth authentication.

sample contents of `mcp.json`
```json
{
  "servers": {
    "ontap-mcp": {
      "type": "http",
      "url": "http://your-server-ip:8083",
       "headers": {
           "Authorization": "Bearer AUTH_TOKEN"
       }
    }
  }
}
```

## Logs

To view the MCP server logs:

```bash
docker logs <container-id>
```

## Configuration

For complete configuration options and environment variables, run:

```bash
docker run --rm ghcr.io/netapp/ontap-mcp:latest start --help
```

This displays all available environment variables with descriptions, authentication options, and advanced settings.

Some commonly used flags:

| Flag | Description |
|---|---|
| `--read-only` | Disable all mutating operations. Only read-only tools are registered. |
| `--stateless` | Disable mcp-session-id header validation. Required when deploying behind proxies or gateways that do not preserve session headers, e.g. on-premises data gateways. |
| `--json-response` | Respond with application/json instead of text/event-stream. Required when deploying behind proxies or gateways that do not relay SSE/chunked responses, e.g. on-premises data gateways. |
| `--inspect-traffic` | Log all MCP HTTP request and response bodies for debugging. |

## Next Steps

- Explore [Prepare ONTAP](prepare-ontap.md)
- Explore [Usage Examples](examples.md)

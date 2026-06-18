## OAuth for ONTAP MCP server

### Prerequisites

- Issuer: Defined in the `McpAuth` section of the `ontap.yaml` [Configuration File](#configuration-file).
- Auth Token: Either provided in the `mcp.json` file for [MCP Client Integration](#mcp-client-integration) or included in any HTTP/HTTPS request using the `Authorization` header.

### Configuration File

To configure OAuth authentication and restrict unauthorized access to your MCP server, add the `McpAuth` section to your configuration file. Use the [ontap-example.yaml](https://github.com/NetApp/ontap-mcp/blob/main/ontap-example.yaml) template as a reference.

By default, the system looks for the `ontap.yaml` file in its working directory (`/opt/mcp` inside the container).

Below is a sample content of the `ontap.yaml` file:

```yaml
McpAuth:
  issuer: http://localhost:9090/realms/REALM
  alg: RS256
  audience: http://localhost:8080
  scope: mcp:tools

Pollers:
  cluster1:
    addr: 10.0.0.1
    username: admin
    password: password
    use_insecure_tls: true
```

Below is a table describing the configuration options in the `McpAuth` section:

| Option     | Type             | Description                                                                                                                                                                                                                                                                                                                                                                                   | Default |
|------------|------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `issuer`   | required, string | The name of the issuer of the authentication token, used to generate the `jwks_uri` for fetching public keys. Examples: <br/> - https://AUTH0_DOMAIN (for Auth0 configuration) <br/> - http://KEYCLOAK_URL/realms/REALM (for Keycloak configuration)                                                                                                                                          |         |
| `alg`      | optional, string | The algorithm used to generate the token, which will validate the given Bearer token against public keys. Supported asymmetric algorithms include: <br/> - RSA Digital Signatures: `[RS256, RS384, RS512]` <br/> - RSA-PSS Digital Signatures: `[PS256, PS384, PS512]` <br/> - ECDSA (Elliptic Curve) Signatures: `[ES256, ES384, ES512]` <br/> - EdDSA (Edwards-curve) Signatures: `[EdDSA]` | `RS256` |
| `audience` | required, string | The expected audience allowed to access MCP tools.                                                                                                                                                                                                                                                                                                                                            |         |
| `scope`    | optional, string | The expected scope allowed to access MCP tools. The default scope is empty                                                                                                                                                                                                                                                                                                                    |         |


### MCP Client Integration

To integrate the ONTAP MCP server with your MCP client (e.g., GitHub Copilot, Claude Desktop), configure the `mcp.json` file with the `Authorization` header containing the Bearer `AUTH_TOKEN`. This enables the MCP client to connect to the MCP server using OAuth authentication.

Below is a sample content for the `mcp.json` file:

```json
{
  "servers": {
    "ontap-mcp": {
      "type": "http",
      "url": "http://your-mcp-server-ip:8080",
       "headers": {
           "Authorization": "Bearer AUTH_TOKEN"
       }
    }
  }
}
```

### MCP Specification Compliance

The ONTAP MCP server adheres to most of the mandatory requirements outlined in the MCP Authorization Specification.

#### Unique key offerings
- Discovery Endpoints - Fully compliant with RFC 9728 implementation.
- Token Audience Validation - Enforces strict security boundaries.
- Token Algorithm Validation - Enforces strict security boundaries.
- Token Scope Validation - Enforces strict security boundaries.
- WWW-Authenticate Headers - Provides proper handling of `401 Unauthorized` responses.
- Dynamic Client Registration - Supports RFC 7591 for seamless onboarding of MCP clients.

The OAuth Discovery endpoint is available at:
`/.well-known/oauth-protected-resource`

This endpoint provides Protected Resource Metadata for OAuth compliance.

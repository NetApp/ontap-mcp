# Integration Tests

## Prerequisites

- `ontap.yaml` in `integration/test/`
- `.ontap-mcp.env` in `integration/test/` — LLM credentials (**do not commit**):
  ```
  LLM_USER=<your-username>
  LLM_TOKEN=<your-llm-token>
  ```
  Get your token from https://llm-proxy-api.ai.openeng.netapp.com/ui/
- Start MCP server `go run main.go start --inspect-traffic --port 8083`

## Running

```bash
just ci
```

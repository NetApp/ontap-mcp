@echo off
REM Start ONTAP MCP server on port 8083
start /b "" "C:\Users\tasos\AppData\Local\mcp-manager\ontap-mcp\ontap-mcp.exe" start --port 8083 --config "C:\Users\tasos\AppData\Local\mcp-manager\ontap-mcp\ontap.yaml"
exit

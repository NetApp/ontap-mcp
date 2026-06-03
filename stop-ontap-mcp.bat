@echo off
taskkill /IM ontap-mcp.exe /F >nul 2>&1
echo ONTAP MCP server stopped.

# ONTAP Model Context Protocol Server

The ONTAP Model Context Protocol (MCP) server provides MCP clients like GitHub Copilot, Claude Desktop, and other large language models (LLMs) access to your NetApp ONTAP storage systems: create automation tools enabling self-service provisioning and lifecycle management. 

Supports multi-cluster management, volume lifecycle operations, data protection policies, and NFS/CIFS access control.  

:star2: Use Cases

- **AI-Assisted Storage Management**: Direct integration with AI assistants via MCP
- **Self-Service Provisioning**: Allow users to provision and manage storage resources through natural language commands.
- **Data Protection**: Automated snapshot policy management
- **Multi-Cluster Management**: Manage multiple ONTAP clusters through a single MCP interface. 
- **Custom Workflows**: Create custom workflows and automations tailored to your specific storage environment.

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io/docs/getting-started/intro) (MCP) is an open standard that enables interactions between MCP clients and external data sources.
The ONTAP MCP server provides APIs for large language models (LLMs) to query and provision ONTAP cluster objects. This allows you to leverage AI assistants like GitHub Copilot and Claude Desktop to manage your ONTAP storage systems through natural language commands.

## What You Can Ask

The ONTAP MCP server allows you to ask natural language questions about your ONTAP clusters and:

- provision and manage storage volumes
- configure NFS and CIFS export policies
- manage snapshot policies and data protection
- configure QoS policies
- and much more! See more [examples](examples.md)

## Architecture

The ONTAP MCP server supports streamable HTTP that can handle multiple client connections. The MCP server can talk with multi-clusters with centralized credential management and unified tool interfaces. All tools operate through registered cluster configurations rather than requiring credentials for each API call, providing enhanced security and simplified multi-cluster workflows. 

```mermaid
graph LR
    A[MCP Client<br/>GitHub Copilot<br/>Claude Desktop] --> B[ONTAP MCP Server]
    B --> C[ONTAP Cluster 1]
    B --> D[ONTAP Cluster 2]
    B --> E[ONTAP Cluster 3]
    B --> F[ONTAP Cluster ...]
```

## Prerequisites

- NetApp ONTAP cluster(s) with admin credentials
- Docker environment for running the MCP server
- Network connectivity from MCP server to your ONTAP cluster(s)

## Next Steps

- [Install the MCP Server](install.md)
- [Configure Environment Variables](install.md#configuration)
- Try the [Usage Examples](examples.md)
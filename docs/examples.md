# Usage Examples

This section provides example queries you can ask your MCP client (GitHub Copilot, Claude Desktop, etc.) when using the ONTAP MCP Server. For more examples and community discussions about MCP usage, see: [ONTAP MCP examples discussion](https://github.com/NetApp/ontap-mcp/discussions/12) 

Higher-capability language models provide better analysis and insights. When possible use the latest model versions with large context windows. You will get better results when using flagship models like GPT-5.X, Sonnet 4.X, Gemini 3, etc. 

The following examples were run with Claude Sonnet 4.5 large language model. 

## Reference Questions

Below are example questions that work well with the ONTAP MCP Server:

### Volume Provisioning

**Create a Volume**

- On the sar cluster, create a 100MB volume named docs on the marketing svm and the umeng_aff300_aggr2 aggregate with thin provisioning enabled.

**Resize a Volume**

- On the sar cluster, resize the docs volume on the marketing svm to 500MB.
- On the sar cluster, increase the size of the docs volume on the marketing svm by 200MB.

**Enable Autogrowth on a Volume**

- On the sar cluster, enable autogrowth on the docs volume

---

### Manage QoS Policies

**Create a QoS Policy**

- On the sar cluster, create a fixed QoS policy named gold on the marketing svm with a max throughput of 5000 iops/s
 
- On the sar cluster, set the qos policy of the docs volume on the marketing svm to 'gold'

---

### Manage Snapshot Policies

- On the sar cluster, create a snapshot policy named every4hours on the gold SVM. The schedule is 4 hours and keep the last 5 snapshots.

---

## MCP Clients

Common MCP clients that work with ONTAP MCP Server:

- **GitHub Copilot**: Integrated in VS Code, supports MCP connections
- **Claude Desktop**: Anthropic's desktop application with MCP support
- **Custom MCP Clients**: Any application implementing the MCP standard

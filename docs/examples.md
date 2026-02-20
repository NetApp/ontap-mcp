# Usage Examples

This section provides example queries you can ask your MCP client (GitHub Copilot, Claude Desktop, etc.) when using the ONTAP MCP Server. For more examples and community discussions about MCP usage, see: [ONTAP MCP examples discussion](https://github.com/NetApp/ontap-mcp/discussions/12)

Higher-capability language models provide better analysis and insights. When possible use the latest model versions with large context windows. You will get better results when using flagship models like GPT-5.X, Sonnet 4.X, Gemini 3, etc.

The following examples were run with Claude Sonnet 4.5 large language model.

## Reference Questions

Below are example questions that work well with the ONTAP MCP Server:

### Volume Provisioning

**Create a Volume**

- On the umeng-aff300-05-06 cluster, create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate

Expected Response: Volume "docs" has been created successfully on the umeng-aff300-05-06 cluster with 20MB size on the marketing SVM using the harvest_vc_aggr aggregate.

**Resize a Volume**

- On the umeng-aff300-05-06 cluster, resize the docs volume on the marketing svm to 25MB.

Expected Response: Volume "docs" has been successfully resized to 25MB on the marketing SVM.

- On the umeng-aff300-05-06 cluster, increase the size of the docs volume on the marketing svm by 5MB.

Expected Response: Volume "docs" has been successfully increased by 5MB to 30MB on the marketing SVM.

---

### Manage QoS Policies

**Create a QoS Policy**

- On the umeng-aff300-05-06 cluster, create a fixed QoS policy named gold on the marketing svm with a max throughput of 5000 iops.

Expected Response: The fixed QoS policy "gold" has been successfully created on the marketing SVM with a maximum throughput of 5000 IOPS on the umeng-aff300-05-06 cluster.

---

### NFS Export Policy Provisioning

**Create an NFS Export policy**

- On the umeng-aff300-05-06 cluster, create an NFS export policy name nfsEngPolicy on the marketing svm

Expected Response: NFS Export Policy created successfully.

**Rename an NFS Export policy**

- On the umeng-aff300-05-06 cluster, rename the NFS export policy from nfsEngPolicy to nfsMgrPolicy on the marketing svm.

Expected Response: NFS Export Policy updated successfully.

---

### NFS Export Policy Rules Provisioning

**Create an NFS Export policy rule**

- On the umeng-aff300-05-06 cluster, create an NFS export policy rule as client match 0.0.0.0/0, ro rule any, rw rule any in nfsMgrPolicy on the marketing svm

Expected Response: NFS Export Policy Rule created successfully.

**Update an NFS Export policy rule**

- On the umeng-aff300-05-06 cluster, update the NFS export policy rule for nfsMgrPolicy export policy on the marketing svm ro rule from any to never.

Expected Response: NFS Export Policy Rules updated successfully.

---

### CIFS share Provisioning

**Create a CIFS share**

- On the umeng-aff300-05-06 cluster, create a CIFS share named cifsFin at the path / on the marketing svm

Expected Response: CIFS share created successfully.

**Update a CIFS share**

- On the umeng-aff300-05-06 cluster, update the CIFS share named cifsFin. Change it's path to /cifsFin on the marketing svm

Expected Response: CIFS share updated successfully.

---

### Manage Snapshot Policies

- On the umeng-aff300-05-06 cluster, create a snapshot policy named every4hours on the gold SVM. The schedule is 4 hours and keeps the last 5 snapshots.

Expected Response: The snapshot policy "every4hours" has been successfully created on the gold SVM with a schedule of every 4 hours, retaining the last 5 snapshots on the umeng-aff300-05-06 cluster.

---

## MCP Clients

Common MCP clients that work with ONTAP MCP Server:

- **GitHub Copilot**: Integrated in VS Code, supports MCP connections
- **Claude Desktop**: Anthropic's desktop application with MCP support
- **Custom MCP Clients**: Any application implementing the MCP standard

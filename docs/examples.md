# Usage Examples

This section provides example queries you can ask your MCP client (GitHub Copilot, Claude Desktop, etc.) when using the ONTAP MCP Server. For more examples and community discussions about MCP usage, see: [ONTAP MCP examples discussion](https://github.com/NetApp/ontap-mcp/discussions/12)

Higher-capability language models provide better analysis and insights. When possible use the latest model versions with large context windows. You will get better results when using flagship models like GPT-5.X, Sonnet 4.X, Gemini 3, etc.

The following examples were run with Claude Sonnet 4.5 large language model.

## Reference Questions

Below are example questions that work well with the ONTAP MCP Server:

### Volume Provisioning

**Create a Volume**

- On the umeng-aff300-05-06 cluster, create a 20MB volume named docs on the marketing svm and the harvest_vc_aggr aggregate
- <span class="key">Expected Response</span>: Volume "docs" has been created successfully on the umeng-aff300-05-06 cluster with 20MB size on the marketing SVM using the harvest_vc_aggr aggregate.

**Resize a Volume**

- On the umeng-aff300-05-06 cluster, resize the docs volume on the marketing svm to 25MB.

- <span class="key">Expected Response</span>: Volume "docs" has been successfully resized to 25MB on the marketing SVM.

- On the umeng-aff300-05-06 cluster, increase the size of the docs volume on the marketing svm by 5MB.

- <span class="key">Expected Response</span>: Volume "docs" has been successfully increased by 5MB to 30MB on the marketing SVM.

**Volume Autogrowth**

- On the umeng-aff300-05-06 cluster, enable autogrowth on the docs volume in the marketing svm

- <span class="key">Expected Response</span>: Autogrowth has been enabled on the "docs" volume in the marketing SVM on the umeng-aff300-05-06 cluster.

- Increase the docs volume autogrow percent to 62

- <span class="key">Expected Response</span>: The autogrow threshold for the docs volume has been successfully increased to 62%. The volume will now automatically grow when it reaches 62% capacity.

---

### Manage QoS Policies

**Create a QoS Policy**

- On the umeng-aff300-05-06 cluster, create a fixed QoS policy named gold on the marketing svm with a max throughput of 5000 iops.

- <span class="key">Expected Response</span>: The fixed QoS policy "gold" has been successfully created on the marketing SVM with a maximum throughput of 5000 IOPS on the umeng-aff300-05-06 cluster.

---

### NFS Export Policy Provisioning

**Create a NFS Export policy**

- On the umeng-aff300-05-06 cluster, create NFS export policy as nfsEngPolicy on the marketing svm

Expected Response: NFS Export Policy created successfully.

**Update a NFS Export policy**

- On the umeng-aff300-05-06 cluster, update NFS export policy from nfsEngPolicy to nfsMgrPolicy on the marketing svm and client match to 1.1.1.1/32, ro rule to any, rw rule to any.

Expected Response: NFS Export Policy updated successfully.

---

### NFS Export Policy Rules Provisioning

**Create a NFS Export policy rule**

- On the umeng-aff300-05-06 cluster, create NFS export policy rule as client match 0.0.0.0/0, ro rule any, rw rule any in nfsMgrPolicy on the marketing svm

Expected Response: NFS Export Policy Rule created successfully.

**Update a NFS Export policy rule**

- On the umeng-aff300-05-06 cluster, update NFS export policy rule for nfsMgrPolicy export policy on the marketing svm ro rule from any to never.

Expected Response: NFS Export Policy Rules updated successfully.

---

### CIFS share Provisioning

**Create a CIFS share**

- On the umeng-aff300-05-06 cluster, create CIFS share named cifsFin with path as / on the marketing svm

Expected Response: CIFS share created successfully.

**Update a CIFS share**

- On the umeng-aff300-05-06 cluster, update CIFS share cifsFin path to /cifsFin on the marketing svm

Expected Response: CIFS share updated successfully.

---

### Manage Snapshot Policies

- On the umeng-aff300-05-06 cluster, create a snapshot policy named every4hours on the gold SVM. The schedule is 4 hours and keeps the last 5 snapshots.

- <span class="key">Expected Response</span>: The snapshot policy "every4hours" has been successfully created on the gold SVM with a schedule of every 4 hours, retaining the last 5 snapshots on the umeng-aff300-05-06 cluster.

- On the umeng-aff300-05-06 cluster, create a snapshot policy named biweekly on the vs_test SVM. The schedule would be 2 weekday 12 hour 30 minutes and keeps the last 3 snapshots.

Expected Response: The snapshot policy has been successfully created.

- On the umeng-aff300-05-06 cluster, create a snapshot policy named every5days on the vs_test SVM. The schedule is P5D and keeps the last 2 snapshots.

Expected Response: The snapshot policy has been successfully created.

---

### Manage Schedule

- On the umeng-aff300-05-06 cluster, create a cron schedule with 5 * * * * named as 5minutes

Expected Response: The schedule has been successfully created.

- On the umeng-aff300-05-06 cluster, create a cron schedule with * * 11 1-2 * named as 11dayjantofeb

Expected Response: The schedule has been successfully created.

---

## MCP Clients

Common MCP clients that work with ONTAP MCP Server:

- **GitHub Copilot**: Integrated in VS Code, supports MCP connections
- **Claude Desktop**: Anthropic's desktop application with MCP support
- **Custom MCP Clients**: Any application implementing the MCP standard
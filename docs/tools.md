# Tools

The following tools are provided by the ONTAP MCP server.

ONTAP MCP provides a set of tools that can be used to interact with the ONTAP API. These tools are designed to help users discover and manage their ONTAP clusters more efficiently. The tools are categorized based on their functionality, such as API discovery, volume management, data protection, CIFS/SMB integration, NFS export policy management, performance management, SVM management, qtree management, network interface management, LUN and igroup management, iSCSI management, FCP management, NVMe management, and multi-cluster management.

All ONTAP MCP tools are annotated with hint metadata: `readOnlyHint`, `idempotentHint`, and `destructiveHint`. The `readOnlyHint` indicates that the tool does not modify any data and is safe to use for discovery and information retrieval. The `destructiveHint` indicates that the tool performs actions that can modify or delete data, and should be used with caution.

If you want to run the ONTAP MCP server in read-only mode, you can start the server with the `--read-only` flag. In this mode, only tools with the `readOnlyHint` will be available for use, ensuring that no modifications can be made to the ONTAP cluster. See the [configuration documentation](install.md#configuration) for more details on how to start the server in read-only mode.

## API Discovery

- `list_ontap_endpoints` (available when the API catalog is loaded)
- `search_ontap_endpoints` (available when the API catalog is loaded)
- `describe_ontap_endpoint` (available when the API catalog is loaded)
- `ontap_get`

## Volume Management

- `create_volume`
- `update_volume`
- `delete_volume`

## Data Protection

- `create_snapshot`
- `delete_snapshot`
- `restore_snapshot`
- `create_snapshot_policy`
- `update_snapshot_policy`
- `delete_snapshot_policy`
- `create_schedule`
- `add_schedule_in_snapshot_policy`
- `update_schedule_in_snapshot_policy`
- `remove_schedule_in_snapshot_policy`

## CIFS/SMB Integration

- `create_cifs_share`
- `update_cifs_share`
- `delete_cifs_share`

## NFS Export Policy Management

- `create_nfs_export_policies`
- `update_nfs_export_policies`
- `delete_nfs_export_policies`
- `create_nfs_export_policies_rules`
- `update_nfs_export_policies_rules`
- `delete_nfs_export_policies_rules`

## Performance Management

- `list_qos_policies`
- `create_qos_policy`
- `update_qos_policy`
- `delete_qos_policy`

## SVM Management

- `create_svm`
- `update_svm`
- `delete_svm`

## Qtree Management

- `create_qtree`
- `update_qtree`
- `delete_qtree`

## Network Interface Management

- `create_network_ip_interface`
- `update_network_ip_interface`
- `delete_network_ip_interface`

## LUN and igroup Management

- `create_lun`
- `update_lun`
- `delete_lun`
- `create_igroup`
- `update_igroup`
- `delete_igroup`
- `add_igroup_initiator`
- `remove_igroup_initiator`
- `create_lun_map`
- `delete_lun_map`

## iSCSI Management

- `create_iscsi_service`
- `update_iscsi_service`
- `delete_iscsi_service`

## FCP Management

- `create_fcp_service`
- `update_fcp_service`
- `delete_fcp_service`
- `create_fc_interface`
- `update_fc_interface`
- `delete_fc_interface`

## NVMe Management

- `create_nvme_service`
- `update_nvme_service`
- `delete_nvme_service`
- `create_nvme_subsystem`
- `update_nvme_subsystem`
- `delete_nvme_subsystem`
- `add_nvme_subsystem_host`
- `remove_nvme_subsystem_host`
- `create_nvme_namespace`
- `update_nvme_namespace`
- `delete_nvme_namespace`
- `create_nvme_subsystem_map`
- `delete_nvme_subsystem_map`

## Multi-Cluster Management

- `list_registered_clusters`
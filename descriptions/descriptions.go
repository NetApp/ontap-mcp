package descriptions

const Instructions = `You are a NetApp storage infrastructure expert specializing in ONTAP storage management.

WORKFLOW — ALWAYS START HERE:
1. Call list_registered_clusters to get cluster names AND their ONTAP versions.
2. Use the cluster_name in all subsequent tool calls.
3. When calling describe_ontap_endpoint, compare each field/filter "since" value against the
   cluster's ontap_version — omit any field or filter introduced in a newer ONTAP version.

## Querying ONTAP data (read operations)
Use the swagger-catalog tools + ontap_get for any listing or read request:

  list_ontap_endpoints   → discover available REST paths (optionally filter by tag)
  search_ontap_endpoints → find paths by keyword (e.g. "snapshot", "lun", "nfs")
  describe_ontap_endpoint→ get filterable fields for one path
  ontap_get              → execute the GET and return raw JSON results

### ONTAP REST query syntax (for ontap_get filters)
- Exact match:     "svm.name": "vs1"
- Wildcard:        "name": "vol*"
- Greater than:    "space.size": ">1073741824"
- Less than:       "space.size": "<1073741824"
- OR values:       "state": "online|offline"
- NOT:             "state": "!offline"
- Comma list for fields param: "name,svm.name,space.size,state"

## Write operations
Create/update/delete operations remain as dedicated typed tools
`

const ListClusters = `List all ONTAP clusters registered in the server configuration.
USE THIS FIRST: Always call this before any other tool to discover valid cluster names.`

const CreateVolume = `Create a volume on a cluster by cluster name.`
const DeleteVolume = `Delete a volume on a cluster by cluster name.`
const UpdateVolume = `Update volume name, size, state, nfs export policy of volume on a cluster by cluster name.`

const CreateSnapshot = `Create a snapshot of a volume on a cluster by cluster name.`
const DeleteSnapshot = `Delete a snapshot of a volume on a cluster by cluster name.`
const RestoreSnapshot = `Restore a volume to a snapshot on a cluster by cluster name.`

const CreateSnapshotPolicy = `Create a snapshot policy on a cluster by cluster name.`
const UpdateSnapshotPolicy = `Update a snapshot policy on a cluster by cluster name.`
const DeleteSnapshotPolicy = `Delete a snapshot policy on a cluster by cluster name.`
const CreateSchedule = `Create a cron schedule on a cluster by cluster name. Ex: 5 1 * * *, this cron expression indicates schedule would be triggered at 01:05 AM for every day`

const AddScheduleInSnapshotPolicy = `Add a schedule entry to an existing snapshot policy on a cluster by cluster name.`
const UpdateScheduleInSnapshotPolicy = `Update a schedule entry within an existing snapshot policy on a cluster by cluster name. At least one of count or snapmirror_label must be provided.`
const RemoveScheduleInSnapshotPolicy = `Remove a schedule entry from an existing snapshot policy on a cluster by cluster name.`

const ListQoSPolicies = `List QoS policies from an ONTAP cluster — includes both SVM-scoped and cluster-scoped (admin SVM) policies.

The response is split into two sections:
- svm_policies: policies scoped to a specific SVM
- cluster_policies: policies that govern every workload on the cluster, regardless of SVM

Cluster-scoped policies (cluster_policies) must ALWAYS be included in your response — never omit them, even when the user asks about a specific SVM.
When svm_name is provided, the response also includes a "message" field that explains both counts.

Units:
- Adaptive: expected_iops and peak_iops are in IOPS/TB. absolute_min_iops is in IOPS.
- Fixed: *_iops fields are in IOPS; *_mbps fields are in MB/s.`

const CreateQoSPolicy = `Create a QoS policy on a cluster by cluster name.`
const UpdateQoSPolicy = `Update a QoS policy on a cluster by cluster name.`
const DeleteQoSPolicy = `Delete a QoS policy on a cluster by cluster name.`

const CreateNFSExportPolicy = `Create NFS Export policies on a cluster by cluster name.`
const UpdateNFSExportPolicy = `Update NFS Export policies on a cluster by cluster name.`
const DeleteNFSExportPolicy = `Delete NFS Export policies on a cluster by cluster name.`
const CreateNFSExportPolicyRules = `Create NFS Export policies rules on a cluster by cluster name.`
const UpdateNFSExportPolicyRules = `Update NFS Export policies rules on a cluster by cluster name.`
const DeleteNFSExportPolicyRules = `Delete NFS Export policies rules on a cluster by cluster name.`

const CreateCIFSShare = `Create CIFS share on a cluster by cluster name.`
const UpdateCIFSShare = `Update CIFS share on a cluster by cluster name.`
const DeleteCIFSShare = `Delete CIFS share on a cluster by cluster name.`

const CreateQtree = `Create Qtree on a cluster by cluster name.`
const UpdateQtree = `Update Qtree on a cluster by cluster name.`
const DeleteQtree = `Delete Qtree on a cluster by cluster name.`

const CreateNVMeService = `Create NVMe service on a cluster by cluster name.`
const UpdateNVMeService = `Update NVMe service on a cluster by cluster name.`
const DeleteNVMeService = `Delete NVMe service on a cluster by cluster name.`

const CreateIscsiService = `Create iSCSI service on a cluster by cluster name.`
const UpdateIscsiService = `Update iSCSI service on a cluster by cluster name.`
const DeleteIscsiService = `Delete iSCSI service on a cluster by cluster name.`

const CreateNetworkIPInterface = `Create Network IP interface on a cluster by cluster name.`
const UpdateNetworkIPInterface = `Update Network IP interface on a cluster by cluster name.`
const DeleteNetworkIPInterface = `Delete Network IP interface on a cluster by cluster name.`

const CreateNVMeSubsystem = `Create NVMe subsystem on a cluster by cluster name.`
const UpdateNVMeSubsystem = `Update NVMe subsystem on a cluster by cluster name.`
const DeleteNVMeSubsystem = `Delete NVMe subsystem on a cluster by cluster name.`

const AddNVMeSubsystemHost = `Add a host NQN to an NVMe subsystem on a cluster by cluster name.`
const RemoveNVMeSubsystemHost = `Remove a host NQN from an NVMe subsystem on a cluster by cluster name.`

const CreateNVMeNamespace = `Create NVMe namespace on a cluster by cluster name.`
const UpdateNVMeNamespace = `Update NVMe namespace on a cluster by cluster name.`
const DeleteNVMeNamespace = `Delete NVMe namespace on a cluster by cluster name.`

const CreateNVMeSubsystemMap = `Create NVMe subsystem map on a cluster by cluster name.`
const DeleteNVMeSubsystemMap = `Delete NVMe subsystem map on a cluster by cluster name.`

const CreateLUN = `Create a LUN on a specified volume and SVM with a given size and OS type.`
const UpdateLUN = `Update a LUN: resize, rename, or toggle enabled/disabled state (online/offline).`
const DeleteLUN = `Delete a LUN from a specified volume and SVM.`

const CreateFCPService = `Create FCP service on a cluster by cluster name.`
const UpdateFCPService = `Update FCP service on a cluster by cluster name.`
const DeleteFCPService = `Delete FCP service on a cluster by cluster name.`

const CreateFCInterface = `Create FC interface on a cluster by cluster name.`
const UpdateFCInterface = `Update FC interface on a cluster by cluster name.`
const DeleteFCInterface = `Delete FC interface on a cluster by cluster name.`

const CreateIGroup = `Create an igroup (initiator group) on a cluster by cluster name.`
const UpdateIGroup = `Update an igroup on a cluster by cluster name.`
const DeleteIGroup = `Delete an igroup on a cluster by cluster name.`
const AddIGroupInitiator = `Add an initiator to an igroup on a cluster by cluster name.`
const RemoveIGroupInitiator = `Remove an initiator from an igroup on a cluster by cluster name.`

const CreateLunMap = `Create a LUN map on a cluster by cluster name. Maps a LUN to an igroup, making the LUN accessible to the initiators in the igroup.`
const DeleteLunMap = `Delete a LUN map on a cluster by cluster name. Removes the mapping between a LUN and an igroup.`

const ListOntapEndpoints = `List ONTAP REST collection endpoints in the catalog.
The catalog contains all endpoints — can be large. Prefer search_ontap_endpoints for targeted discovery.
Use the optional 'match' parameter to filter by substring or regex pattern (e.g. "snapshot", "lun", ".*nfs.*export.*").
Omit 'match' only when you want a full overview of all available endpoints.`

const SearchOntapEndpoints = `Search the catalog by keyword across endpoint paths, summaries, and tags (e.g. "snapshot", "lun", "nfs").`

const DescribeOntapEndpoint = `Get filterable query params for an endpoint. Call before ontap_get to learn valid filter names and which sub-objects need explicit fields (e.g. "space.*", "efficiency.*").
Pass cluster_name to automatically filter out fields and filters not available in that cluster's ONTAP version.`

const CreateSVM = `Create an SVM on a cluster by cluster name.`
const UpdateSVM = `Update an SVM on a cluster by cluster name.`
const DeleteSVM = `Delete an SVM on a cluster by cluster name.`

const OntapGet = `Execute a read-only GET against any ONTAP REST endpoint.

RULES:
1. Always specify 'fields' — omitting it returns 50+ fields per record and floods the context window.
   BAD:  {"cluster_name":"dc1","path":"/storage/volumes"}
   GOOD: {"cluster_name":"dc1","path":"/storage/volumes","fields":"name,state,svm.name"}
2. Always use 'filters' to narrow results — NEVER fetch all records to find one item. This is wasteful and will exceed the context window.
   WRONG: {"cluster_name":"dc1","path":"/storage/volumes","fields":"name,uuid"}  ← fetches everything
   RIGHT: {"cluster_name":"dc1","path":"/storage/volumes","fields":"uuid","filters":{"name":"<vol>","svm.name":"<svm>"}}
3. To query a sub-resource with a UUID path (e.g. /storage/volumes/{volume.uuid}/snapshots):
   a. {"cluster_name":"dc1","path":"/storage/volumes","fields":"uuid","filters":{"name":"<vol>","svm.name":"<svm>"}}
   b. {"cluster_name":"dc1","path":"/storage/volumes/{volume.uuid}/snapshots","path_params":{"volume.uuid":"<uuid-from-a>"},"fields":"name,create_time"}

Parameters:
- fields:      comma-separated fields, e.g. "name,svm.name,space.size" — use "space.*" to expand sub-objects
- filters:     key-value object using ONTAP query syntax (see system instructions for syntax reference)
- path_params: values for {param} placeholders in templated paths, e.g. {"volume.uuid":"abc-123"}
- max_records: cap result count; omit to return all

Example — collection:
{"cluster_name":"dc1","path":"/storage/volumes","fields":"name,uuid,svm.name,state","filters":{"svm.name":"vs1"}}

Example — snapshots for a volume (2 calls):
Call 1: {"cluster_name":"dc1","path":"/storage/volumes","fields":"uuid","filters":{"name":"vol1","svm.name":"vs1"}}
Call 2: {"cluster_name":"dc1","path":"/storage/volumes/{volume.uuid}/snapshots","path_params":{"volume.uuid":"<uuid-from-call-1>"},"fields":"name,create_time,comment"}`

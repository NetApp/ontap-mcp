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

const CreateSnapshotPolicy = `Create a snapshot policy on a cluster by cluster name.`
const DeleteSnapshotPolicy = `Delete a snapshot policy on a cluster by cluster name.`
const CreateSchedule = `Create a cron schedule on a cluster by cluster name. Ex: 5 1 * * *, this cron expression indicates schedule would be triggered at 01:05 AM for every day`

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

const ListOntapEndpoints = `List ONTAP REST collection endpoints in the catalog.
The catalog contains all endpoints — can be large. Prefer search_ontap_endpoints for targeted discovery.
Use the optional 'match' parameter to filter by substring or regex pattern (e.g. "snapshot", "lun", ".*nfs.*export.*").
Omit 'match' only when you want a full overview of all available endpoints.`

const SearchOntapEndpoints = `Search the catalog by keyword across endpoint paths, summaries, and tags (e.g. "snapshot", "lun", "nfs").`

const DescribeOntapEndpoint = `Get filterable query params for an endpoint. Call before ontap_get to learn valid filter names and which sub-objects need explicit fields (e.g. "space.*", "efficiency.*").
Pass cluster_name to automatically filter out fields and filters not available in that cluster's ONTAP version.`

const OntapGet = `Execute a read-only GET against any ONTAP REST endpoint.

CRITICAL: ALWAYS pass the 'fields' parameter with only the specific fields you need.
Omitting 'fields' returns 50+ fields per record and floods the context window with noise.
BAD:  {"cluster_name": "dc1", "path": "/storage/volumes"}                              ← returns everything
GOOD: {"cluster_name": "dc1", "path": "/storage/volumes", "fields": "name,state,svm.name,space.used"}

- path: endpoint path without /api prefix — collection (e.g. /storage/volumes) or resource template (e.g. /storage/volumes/{volume.uuid}/snapshots)
- path_params: (OBJECT) values for {param} placeholders in the path, e.g. {"volume.uuid": "abc-123"}.
  Obtain the UUID/key first by querying the collection endpoint (e.g. GET /storage/volumes with fields="uuid,name").
- fields: (STRING) comma-separated dot-notation fields to return, e.g. "name,svm.name,space.size" — use "space.*" to expand sub-objects
- filters: (OBJECT) ONTAP query syntax — exact:"vs1" wildcard:"vol*" range:">1073741824" OR:"online|offline" NOT:"!offline"
- max_records: limit results; omit to return all

Examples:
  Collection:
  {
    "cluster_name": "dc1",
    "path": "/storage/volumes",
    "fields": "name,uuid,svm.name,space.size,state",
    "filters": {"svm.name": "vs1", "state": "online"}
  }

  Resource (snapshots for a volume — first get the volume UUID from /storage/volumes):
  {
    "cluster_name": "dc1",
    "path": "/storage/volumes/{volume.uuid}/snapshots",
    "path_params": {"volume.uuid": "abc-1234-5678"},
    "fields": "name,create_time,comment"
  }

Call describe_ontap_endpoint first to learn valid field, filter, and path-param names.`

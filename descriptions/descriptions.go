package descriptions

const ListClusters = `List registered clusters`
const Instructions = `You are a NetApp storage infrastructure expert specializing in ONTAP.`

const ListVolumes = `List volumes on a cluster by cluster name.`
const CreateVolume = `Create a volume on a cluster by cluster name.`
const DeleteVolume = `Delete a volume on a cluster by cluster name.`
const UpdateVolume = `Update volume name, size, state, nfs export policy of volume on a cluster by cluster name.`

const ListSnapshotPolicy = `List snapshot policies on a cluster by cluster name.`
const CreateSnapshotPolicy = `Create a snapshot policy on a cluster by cluster name.`
const DeleteSnapshotPolicy = `Delete a snapshot policy on a cluster by cluster name.`
const CreateSchedule = `Create a cron schedule on a cluster by cluster name. Ex: 5 1 * * *, this cron expression indicates schedule would be triggered at 01:05 AM for every day`

const ListQoSPolicy = `List QoS policies on a cluster by cluster name.`
const CreateQoSPolicy = `Create a QoS policy on a cluster by cluster name.`
const UpdateQoSPolicy = `Update a QoS policy on a cluster by cluster name.`
const DeleteQoSPolicy = `Delete a QoS policy on a cluster by cluster name.`

const ListNFSExportPolicy = `List NFS Export policies on a cluster by cluster name.`
const CreateNFSExportPolicy = `Create NFS Export policies on a cluster by cluster name.`
const UpdateNFSExportPolicy = `Update NFS Export policies on a cluster by cluster name.`
const DeleteNFSExportPolicy = `Delete NFS Export policies on a cluster by cluster name.`
const CreateNFSExportPolicyRules = `Create NFS Export policies rules on a cluster by cluster name.`
const UpdateNFSExportPolicyRules = `Update NFS Export policies rules on a cluster by cluster name.`
const DeleteNFSExportPolicyRules = `Delete NFS Export policies rules on a cluster by cluster name.`

const ListCIFSShare = `List CIFS share on a cluster by cluster name.`
const CreateCIFSShare = `Create CIFS share a cluster by cluster name.`
const UpdateCIFSShare = `Update CIFS share on a cluster by cluster name.`
const DeleteCIFSShare = `Delete CIFS share on a cluster by cluster name.`

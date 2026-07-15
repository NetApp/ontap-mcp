package tool

type ListVolume struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name,omitzero" jsonschema:"SVM name"`
}

type ListClusterParams struct{}

type VolumeCreate struct {
	Cluster      string    `json:"cluster_name" jsonschema:"cluster name"`
	SVM          string    `json:"svm_name" jsonschema:"SVM name"`
	Volume       string    `json:"volume_name" jsonschema:"volume name"`
	Aggregate    string    `json:"aggregate_name" jsonschema:"aggregate name"`
	JunctionPath string    `json:"nas.path,omitzero" jsonschema:"junction path"`
	Size         string    `json:"size,omitzero" jsonschema:"size of the volume (e.g., '100GB', '1TB')"`
	ExportPolicy string    `json:"nas.export_policy.name,omitzero" jsonschema:"nfs export policy name. Will be created if it doesn't exist"`
	QoS          VolumeQoS `json:"qos,omitzero" jsonschema:"QoS settings: use policy_name to assign an existing policy, or max_iops/min_iops/max_mbps/min_mbps for inline limits (mutually exclusive)"`
	Type         string    `json:"type,omitzero" jsonschema:"type of volume (e.g., 'rw', 'dp', 'ls')"`
}

type Volume struct {
	Cluster      string    `json:"cluster_name" jsonschema:"cluster name"`
	SVM          string    `json:"svm_name" jsonschema:"SVM name"`
	Volume       string    `json:"volume_name" jsonschema:"volume name"`
	Aggregate    string    `json:"aggregate_name,omitzero" jsonschema:"aggregate name"`
	JunctionPath string    `json:"nas.path,omitzero" jsonschema:"junction path"`
	NewVolume    string    `json:"new_volume_name,omitzero" jsonschema:"new volume name"`
	Size         string    `json:"size,omitzero" jsonschema:"size of the volume (e.g., '100GB', '1TB')"`
	State        string    `json:"state,omitzero" jsonschema:"state of the volume (e.g., 'online', 'offline')"`
	ExportPolicy string    `json:"nas.export_policy.name,omitzero" jsonschema:"nfs export policy name. Will be created if it doesn't exist"`
	Autosize     Autosize  `json:"autosize,omitzero" jsonschema:"autosize"`
	QoS          VolumeQoS `json:"qos,omitzero" jsonschema:"QoS settings: use policy_name to assign an existing policy, or max_iops/min_iops/max_mbps/min_mbps for inline limits (mutually exclusive)"`
	Type         string    `json:"type,omitzero" jsonschema:"type of volume (e.g., 'rw', 'dp', 'ls')"`
}

type VolumeModify struct {
	Cluster      string       `json:"cluster_name" jsonschema:"cluster name"`
	Operation    string       `json:"operation" jsonschema:"volume operation type (e.g., update, delete)"`
	SVM          string       `json:"svm_name" jsonschema:"SVM name"`
	Volume       string       `json:"volume_name" jsonschema:"volume name"`
	VolumeUpdate VolumeUpdate `json:"volume_update,omitzero" jsonschema:"update volume operation"`
}

type VolumeUpdate struct {
	NewVolume    string    `json:"new_volume_name,omitzero" jsonschema:"new volume name for rename operation"`
	Size         string    `json:"size,omitzero" jsonschema:"size of the volume (e.g., '100GB', '1TB')"`
	State        string    `json:"state,omitzero" jsonschema:"state of the volume (e.g., 'online', 'offline')"`
	JunctionPath string    `json:"nas.path,omitzero" jsonschema:"junction path"`
	ExportPolicy string    `json:"nas.export_policy.name,omitzero" jsonschema:"nfs export policy name"`
	Autosize     Autosize  `json:"autosize,omitzero" jsonschema:"autosize"`
	QoS          VolumeQoS `json:"qos,omitzero" jsonschema:"QoS settings"`
}

type VolumeQoS struct {
	RemovePolicy bool   `json:"remove_qos_policy,omitzero" jsonschema:"set to true to remove the QoS policy from the volume"`
	PolicyName   string `json:"policy_name,omitzero" jsonschema:"name of an existing QoS policy to assign. Mutually exclusive with inline throughput fields and remove_qos_policy"`
	MaxIOPS      string `json:"max_iops,omitzero" jsonschema:"inline: max throughput in IOPS (\"0\" = none, removes limit). Mutually exclusive with policy_name"`
	MinIOPS      string `json:"min_iops,omitzero" jsonschema:"inline: min throughput in IOPS (\"0\" = none, removes limit, AFF only). Mutually exclusive with policy_name"`
	MaxMBPS      string `json:"max_mbps,omitzero" jsonschema:"inline: max throughput in MB/s (\"0\" = none, removes limit). Mutually exclusive with policy_name"`
	MinMBPS      string `json:"min_mbps,omitzero" jsonschema:"inline: min throughput in MB/s (\"0\" = none, removes limit). Mutually exclusive with policy_name"`
}

type Autosize struct {
	MaxSize         string `json:"maximum,omitzero" jsonschema:"maximum size a volume grows"`
	MinSize         string `json:"minimum,omitzero" jsonschema:"minimum size a volume shrinks"`
	Mode            string `json:"mode,omitzero" jsonschema:"autosize mode (e.g., 'grow', 'grow_shrink', 'off')"`
	GrowThreshold   string `json:"grow_threshold,omitzero" jsonschema:"percentage of auto growth"`
	ShrinkThreshold string `json:"shrink_threshold,omitzero" jsonschema:"percentage of auto shrinkage"`
}

type SnapshotPolicyCreate struct {
	Cluster  string `json:"cluster_name" jsonschema:"cluster name"`
	SVM      string `json:"svm_name" jsonschema:"SVM name"`
	Name     string `json:"name" jsonschema:"snapshot policy name"`
	Schedule string `json:"schedule" jsonschema:"schedule of snapshot policy"`
	Count    int    `json:"count" jsonschema:"number of snapshots"`
}

type SnapshotPolicy struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Name    string `json:"name" jsonschema:"snapshot policy name"`
	Enabled string `json:"enabled,omitzero" jsonschema:"the state of snapshot policy"`
	Comment string `json:"comment,omitzero" jsonschema:"comment associated with the snapshot policy"`
}

type SnapshotPolicyModify struct {
	Cluster              string               `json:"cluster_name" jsonschema:"cluster name"`
	Operation            string               `json:"operation" jsonschema:"snapshot policy operation type (e.g., update, delete)"`
	SVM                  string               `json:"svm_name" jsonschema:"SVM name"`
	Name                 string               `json:"name" jsonschema:"snapshot policy name"`
	SnapshotPolicyUpdate SnapshotPolicyUpdate `json:"snapshot_policy_update,omitzero" jsonschema:"update snapshot policy operation"`
}

type SnapshotPolicyUpdate struct {
	Comment string `json:"comment,omitzero" jsonschema:"comment associated with the snapshot policy"`
	Enabled string `json:"enabled,omitzero" jsonschema:"the state of snapshot policy"`
}

type Schedule struct {
	Cluster        string `json:"cluster_name" jsonschema:"cluster name"`
	Name           string `json:"name"`
	CronExpression string `json:"cron_expression" jsonschema:"cron_expression"`
}

type SnapshotPolicySchedule struct {
	Cluster         string `json:"cluster_name" jsonschema:"cluster name"`
	SVM             string `json:"svm_name" jsonschema:"SVM name"`
	PolicyName      string `json:"policy_name" jsonschema:"snapshot policy name"`
	ScheduleName    string `json:"schedule_name" jsonschema:"name of the schedule"`
	Count           int    `json:"count,omitzero" jsonschema:"number of snapshots to keep for this schedule"`
	SnapmirrorLabel string `json:"snapmirror_label,omitzero" jsonschema:"SnapMirror label for this schedule"`
}

type SnapshotPolicyScheduleModify struct {
	Cluster                      string                       `json:"cluster_name" jsonschema:"cluster name"`
	Operation                    string                       `json:"operation" jsonschema:"snapshot policy schedule operation type (e.g., update, remove)"`
	SVM                          string                       `json:"svm_name" jsonschema:"SVM name"`
	PolicyName                   string                       `json:"policy_name" jsonschema:"snapshot policy name"`
	ScheduleName                 string                       `json:"schedule_name" jsonschema:"name of the schedule"`
	SnapshotPolicyScheduleUpdate SnapshotPolicyScheduleUpdate `json:"snapshot_policy_schedule_update,omitzero" jsonschema:"update snapshot policy schedule operation"`
}

type SnapshotPolicyScheduleUpdate struct {
	Count           int    `json:"count,omitzero" jsonschema:"number of snapshots to keep for this schedule"`
	SnapmirrorLabel string `json:"snapmirror_label,omitzero" jsonschema:"SnapMirror label for this schedule"`
}

type Snapshot struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Volume  string `json:"volume_name" jsonschema:"volume name"`
	Name    string `json:"name" jsonschema:"snapshot name"`
}

type SnapshotModify struct {
	Cluster   string `json:"cluster_name" jsonschema:"cluster name"`
	Operation string `json:"operation" jsonschema:"snapshot operation type (e.g., restore, delete)"`
	SVM       string `json:"svm_name" jsonschema:"SVM name"`
	Volume    string `json:"volume_name" jsonschema:"volume name"`
	Name      string `json:"name" jsonschema:"snapshot name"`
}

type Cron struct {
	Days     string `json:"days,omitzero"`
	Hours    string `json:"hours,omitzero"`
	Minutes  string `json:"minutes,omitzero"`
	Months   string `json:"months,omitzero"`
	Weekdays string `json:"weekdays,omitzero"`
}

type QoSPolicy struct {
	Cluster         string `json:"cluster_name" jsonschema:"cluster name"`
	SVM             string `json:"svm_name" jsonschema:"SVM name"`
	Name            string `json:"name" jsonschema:"qos policy name"`
	NewName         string `json:"new_name,omitzero" jsonschema:"new qos policy name"`
	MaxThIOPS       string `json:"max_throughput_iops,omitzero" jsonschema:"max throughput of fixed qos policy"`
	MinThIOPS       string `json:"min_throughput_iops,omitzero" jsonschema:"min throughput of fixed qos policy"`
	ExpectedIOPS    string `json:"expected_iops,omitzero" jsonschema:"expected iops of adaptive qos policy"`
	PeakIOPS        string `json:"peak_iops,omitzero" jsonschema:"peak iops of adaptive qos policy"`
	AbsoluteMinIOPS string `json:"absolute_min_iops,omitzero" jsonschema:"absolute min iops of adaptive qos policy"`
	CapacityShared  bool   `json:"capacity_shared,omitzero" jsonschema:"whether the capacities are shared across all objects that use this QoS policy-group. Default is false."`
}

type QoSPolicyModify struct {
	Cluster         string          `json:"cluster_name" jsonschema:"cluster name"`
	Operation       string          `json:"operation" jsonschema:"QoS policy operation type (e.g., update, delete)"`
	SVM             string          `json:"svm_name" jsonschema:"SVM name"`
	Name            string          `json:"name" jsonschema:"QoS policy name"`
	QoSPolicyUpdate QoSPolicyUpdate `json:"qos_policy_update,omitzero" jsonschema:"update QoS policy operation"`
}

type QoSPolicyUpdate struct {
	NewName         string `json:"new_name,omitzero" jsonschema:"new QoS policy name"`
	MaxThIOPS       string `json:"max_throughput_iops,omitzero" jsonschema:"max throughput of fixed QoS policy"`
	MinThIOPS       string `json:"min_throughput_iops,omitzero" jsonschema:"min throughput of fixed QoS policy"`
	ExpectedIOPS    string `json:"expected_iops,omitzero" jsonschema:"expected iops of adaptive QoS policy"`
	PeakIOPS        string `json:"peak_iops,omitzero" jsonschema:"peak iops of adaptive QoS policy"`
	AbsoluteMinIOPS string `json:"absolute_min_iops,omitzero" jsonschema:"absolute min iops of adaptive QoS policy"`
}

type NFSExportPolicyCreate struct {
	Cluster      string `json:"cluster_name" jsonschema:"cluster name"`
	SVM          string `json:"svm_name" jsonschema:"SVM name"`
	ExportPolicy string `json:"export_policy" jsonschema:"nfs export policy name"`
	ClientMatch  string `json:"client_match" jsonschema:"list of clients"`
	ROrule       string `json:"ro_rule" jsonschema:"read only rules"`
	RWrule       string `json:"rw_rule" jsonschema:"read write rules"`
}

type NFSExportPolicy struct {
	Cluster         string `json:"cluster_name" jsonschema:"cluster name"`
	SVM             string `json:"svm_name,omitzero" jsonschema:"SVM name"`
	ExportPolicy    string `json:"export_policy,omitzero" jsonschema:"nfs export policy name"`
	NewExportPolicy string `json:"new_export_policy,omitzero" jsonschema:"new nfs export policy name"`
	ClientMatch     string `json:"client_match,omitzero" jsonschema:"list of clients"`
	ROrule          string `json:"ro_rule,omitzero" jsonschema:"read only rules"`
	RWrule          string `json:"rw_rule,omitzero" jsonschema:"read write rules"`
}

type NFSExportPolicyModify struct {
	Cluster               string                `json:"cluster_name" jsonschema:"cluster name"`
	Operation             string                `json:"operation" jsonschema:"NFS export policy operation type (e.g., update, delete)"`
	ExportPolicy          string                `json:"export_policy" jsonschema:"nfs export policy name"`
	NFSExportPolicyUpdate NFSExportPolicyUpdate `json:"nfs_export_policy_update,omitzero" jsonschema:"update NFS export policy operation"`
}

type NFSExportPolicyUpdate struct {
	NewExportPolicy string `json:"new_export_policy,omitzero" jsonschema:"new nfs export policy name"`
	ClientMatch     string `json:"client_match,omitzero" jsonschema:"list of clients"`
	ROrule          string `json:"ro_rule,omitzero" jsonschema:"read only rules"`
	RWrule          string `json:"rw_rule,omitzero" jsonschema:"read write rules"`
}

type NFSExportPolicyRulesCreate struct {
	Cluster      string `json:"cluster_name" jsonschema:"cluster name"`
	ExportPolicy string `json:"export_policy" jsonschema:"nfs export policy name"`
	ClientMatch  string `json:"client" jsonschema:"list of clients"`
	ROrule       string `json:"ro_rule" jsonschema:"read only rules"`
	RWrule       string `json:"rw_rule" jsonschema:"read write rules"`
}

type NFSExportPolicyRules struct {
	Cluster        string `json:"cluster_name" jsonschema:"cluster name"`
	ExportPolicy   string `json:"export_policy" jsonschema:"nfs export policy name"`
	OldClientMatch string `json:"old_client,omitzero" jsonschema:"old list of clients"`
	ClientMatch    string `json:"client,omitzero" jsonschema:"list of clients"`
	OldROrule      string `json:"old_ro_rule,omitzero" jsonschema:"old read only rules"`
	ROrule         string `json:"ro_rule,omitzero" jsonschema:"read only rules"`
	OldRWrule      string `json:"old_rw_rule,omitzero" jsonschema:"old read write rules"`
	RWrule         string `json:"rw_rule,omitzero" jsonschema:"read write rules"`
}

type NFSExportPolicyRulesModify struct {
	Cluster                    string                     `json:"cluster_name" jsonschema:"cluster name"`
	Operation                  string                     `json:"operation" jsonschema:"NFS export policy rules operation type (e.g., update, delete)"`
	ExportPolicy               string                     `json:"export_policy" jsonschema:"nfs export policy name"`
	NFSExportPolicyRulesUpdate NFSExportPolicyRulesUpdate `json:"nfs_export_policy_rules_update,omitzero" jsonschema:"update NFS export policy rules operation"`
}

type NFSExportPolicyRulesUpdate struct {
	OldClientMatch string `json:"old_client,omitzero" jsonschema:"old list of clients (required to identify rule for update/delete)"`
	OldROrule      string `json:"old_ro_rule,omitzero" jsonschema:"old read only rules (required to identify rule for update/delete)"`
	OldRWrule      string `json:"old_rw_rule,omitzero" jsonschema:"old read write rules (required to identify rule for update/delete)"`
	ClientMatch    string `json:"client,omitzero" jsonschema:"new list of clients"`
	ROrule         string `json:"ro_rule,omitzero" jsonschema:"new read only rules"`
	RWrule         string `json:"rw_rule,omitzero" jsonschema:"new read write rules"`
}

type CIFSShareCreate struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Name    string `json:"name" jsonschema:"cifs share name"`
	Path    string `json:"path" jsonschema:"cifs share path"`
}

type CIFSShare struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name,omitzero" jsonschema:"SVM name"`
	Name    string `json:"name,omitzero" jsonschema:"cifs share name"`
	Path    string `json:"path,omitzero" jsonschema:"cifs share path"`
}

type CIFSShareModify struct {
	Cluster         string          `json:"cluster_name" jsonschema:"cluster name"`
	Operation       string          `json:"operation" jsonschema:"CIFS share operation type (e.g., update, delete)"`
	SVM             string          `json:"svm_name" jsonschema:"SVM name"`
	Name            string          `json:"name" jsonschema:"CIFS share name"`
	CIFSShareUpdate CIFSShareUpdate `json:"cifs_share_update,omitzero" jsonschema:"update CIFS share operation"`
}

type CIFSShareUpdate struct {
	Path string `json:"path,omitzero" jsonschema:"new CIFS share path"`
}

type QtreeCreate struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Volume  string `json:"volume_name" jsonschema:"Volume name"`
	Name    string `json:"name" jsonschema:"qtree name"`
}

type Qtree struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Volume  string `json:"volume_name" jsonschema:"Volume name"`
	Name    string `json:"name" jsonschema:"qtree name"`
	NewName string `json:"new_name,omitzero" jsonschema:"new qtree name"`
}

type QtreeModify struct {
	Cluster     string      `json:"cluster_name" jsonschema:"cluster name"`
	Operation   string      `json:"operation" jsonschema:"qtree operation type (e.g., update, delete)"`
	SVM         string      `json:"svm_name" jsonschema:"SVM name"`
	Volume      string      `json:"volume_name" jsonschema:"Volume name"`
	Name        string      `json:"name" jsonschema:"qtree name"`
	QtreeUpdate QtreeUpdate `json:"qtree_update,omitzero" jsonschema:"update qtree operation"`
}

type QtreeUpdate struct {
	NewName string `json:"new_name,omitzero" jsonschema:"new qtree name"`
}

type LUNCreate struct {
	Cluster                 string `json:"cluster_name" jsonschema:"cluster name"`
	SVM                     string `json:"svm_name" jsonschema:"SVM name"`
	Volume                  string `json:"volume_name" jsonschema:"volume name where the LUN resides"`
	Name                    string `json:"lun_name" jsonschema:"LUN name"`
	Size                    string `json:"size" jsonschema:"size of the LUN (e.g., '10GB', '1TB')"`
	OsType                  string `json:"os_type" jsonschema:"OS type (e.g., linux, windows, windows_2008, windows_gpt, aix, esxi, hyper_v, solaris, vmware, xen)"`
	SpaceGuaranteeRequested bool   `json:"space_guarantee_requested,omitzero" jsonschema:"set to true to request thick provisioning (space guarantee) for the LUN"`
}

type LUN struct {
	Cluster                string `json:"cluster_name" jsonschema:"cluster name"`
	SVM                    string `json:"svm_name" jsonschema:"SVM name"`
	Volume                 string `json:"volume_name" jsonschema:"volume name where the LUN resides"`
	Name                   string `json:"lun_name" jsonschema:"LUN name"`
	NewName                string `json:"new_lun_name,omitzero" jsonschema:"new LUN name for rename operation"`
	Size                   string `json:"size,omitzero" jsonschema:"size of the LUN (e.g., '10GB', '1TB')"`
	Enabled                string `json:"enabled,omitzero" jsonschema:"LUN state: 'true' to enable (online) or 'false' to disable (offline) the LUN"`
	AllowDeleteWhileMapped bool   `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows deletion of a mapped LUN. This parameter should be used with caution"`
}

type LUNModify struct {
	Cluster                string    `json:"cluster_name" jsonschema:"cluster name"`
	Operation              string    `json:"operation" jsonschema:"LUN operation type (e.g., update, delete)"`
	SVM                    string    `json:"svm_name" jsonschema:"SVM name"`
	Volume                 string    `json:"volume_name" jsonschema:"volume name where the LUN resides"`
	Name                   string    `json:"lun_name" jsonschema:"LUN name"`
	AllowDeleteWhileMapped bool      `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows deletion of a mapped LUN. This parameter should be used with caution"`
	LUNUpdate              LUNUpdate `json:"lun_update,omitzero" jsonschema:"update LUN operation"`
}

type LUNUpdate struct {
	NewName string `json:"new_lun_name,omitzero" jsonschema:"new LUN name for rename operation"`
	Size    string `json:"size,omitzero" jsonschema:"size of the LUN (e.g., '10GB', '1TB')"`
	Enabled string `json:"enabled,omitzero" jsonschema:"LUN state: 'true' to enable (online) or 'false' to disable (offline) the LUN"`
}

type NVMeService struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Enabled string `json:"enabled,omitzero" jsonschema:"admin state of the NVMe service"`
}

type NVMeServiceModify struct {
	Cluster           string            `json:"cluster_name" jsonschema:"cluster name"`
	Operation         string            `json:"operation" jsonschema:"NVMe service operation type (e.g., update, delete)"`
	SVM               string            `json:"svm_name" jsonschema:"SVM name"`
	NVMeServiceUpdate NVMeServiceUpdate `json:"nvme_service_update,omitzero" jsonschema:"update NVMe service operation"`
}

type NVMeServiceUpdate struct {
	Enabled string `json:"enabled,omitzero" jsonschema:"admin state of the NVMe service"`
}

type IscsiService struct {
	Cluster     string `json:"cluster_name" jsonschema:"cluster name"`
	SVM         string `json:"svm_name" jsonschema:"SVM name"`
	Enabled     string `json:"enabled,omitzero" jsonschema:"admin state of the iSCSI service"`
	TargetAlias string `json:"target.alias,omitzero" jsonschema:"iSCSI target alias of the iSCSI service"`
}

type IscsiServiceModify struct {
	Cluster            string             `json:"cluster_name" jsonschema:"cluster name"`
	Operation          string             `json:"operation" jsonschema:"iSCSI service operation type (e.g., update, delete)"`
	SVM                string             `json:"svm_name" jsonschema:"SVM name"`
	IscsiServiceUpdate IscsiServiceUpdate `json:"iscsi_service_update,omitzero" jsonschema:"update iSCSI service operation"`
}

type IscsiServiceUpdate struct {
	Enabled string `json:"enabled,omitzero" jsonschema:"admin state of the iSCSI service"`
}

type NetworkIPInterface struct {
	Cluster         string `json:"cluster_name" jsonschema:"cluster name"`
	SVM             string `json:"svm_name,omitzero" jsonschema:"SVM name"`
	IPSpace         string `json:"ipspace_name,omitzero" jsonschema:"ipspace name"`
	Name            string `json:"name" jsonschema:"name of the interface"`
	Scope           string `json:"scope" jsonschema:"scope of network interface(e.g., 'cluster', 'svm')"`
	IPAddress       string `json:"ip.address,omitzero" jsonschema:"IP address for the interface"`
	IPNetmask       string `json:"ip.netmask,omitzero" jsonschema:"IP netmask of the interface"`
	Subnet          string `json:"subnet_name,omitzero" jsonschema:"subnet name"`
	HomeNode        string `json:"location.home_node,omitzero" jsonschema:"home node"`
	BroadcastDomain string `json:"location.broadcast_domain,omitzero" jsonschema:"broadcast domain"`
	AutoRevert      string `json:"location.auto_revert,omitzero" jsonschema:"auto_revert"`
	ServicePolicy   string `json:"service_policy,omitzero" jsonschema:"service policy (e.g., default-data-files, default-data-blocks, default-data-iscsi, default-management, default-intercluster, default-route-announce)"`
}

type NetworkIPInterfaceModify struct {
	Cluster                  string                   `json:"cluster_name" jsonschema:"cluster name"`
	Operation                string                   `json:"operation" jsonschema:"network IP interface operation type (e.g., update, delete)"`
	SVM                      string                   `json:"svm_name,omitzero" jsonschema:"SVM name"`
	Scope                    string                   `json:"scope" jsonschema:"scope of network interface(e.g., 'cluster', 'svm')"`
	Name                     string                   `json:"name" jsonschema:"name of the interface"`
	NetworkIPInterfaceUpdate NetworkIPInterfaceUpdate `json:"network_ip_interface_update,omitzero" jsonschema:"update network IP interface operation"`
}

type NetworkIPInterfaceUpdate struct {
	AutoRevert    string `json:"location.auto_revert,omitzero" jsonschema:"auto_revert"`
	ServicePolicy string `json:"service_policy,omitzero" jsonschema:"service policy"`
}

type NVMeSubsystem struct {
	Cluster                string   `json:"cluster_name" jsonschema:"cluster name"`
	SVM                    string   `json:"svm_name" jsonschema:"SVM name"`
	Name                   string   `json:"name" jsonschema:"name for NVMe subsystem"`
	OSType                 string   `json:"os_type,omitzero" jsonschema:"operating system of the NVMe subsystem's hosts (e.g., aix, linux, vmware, windows)"`
	HostNQNs               []string `json:"hosts_nqns,omitzero" jsonschema:"array of NVMe qualified name (NQN) used to identify the NVMe hosts"`
	Comment                string   `json:"comment,omitzero" jsonschema:"configurable comment for the NVMe subsystem"`
	AllowDeleteWhileMapped bool     `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows for the deletion of a mapped NVMe subsystem. This parameter should be used with caution."`
	AllowDeleteWithHosts   bool     `json:"allow_delete_with_hosts,omitzero" jsonschema:"Allows for the deletion of an NVMe subsystem with NVMe hosts. This parameter should be used with caution."`
}

type NVMeSubsystemModify struct {
	Cluster                string              `json:"cluster_name" jsonschema:"cluster name"`
	Operation              string              `json:"operation" jsonschema:"NVMe subsystem operation type (e.g., update, delete)"`
	SVM                    string              `json:"svm_name" jsonschema:"SVM name"`
	Name                   string              `json:"name" jsonschema:"name for NVMe subsystem"`
	AllowDeleteWhileMapped bool                `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows for the deletion of a mapped NVMe subsystem. This parameter should be used with caution."`
	AllowDeleteWithHosts   bool                `json:"allow_delete_with_hosts,omitzero" jsonschema:"Allows for the deletion of an NVMe subsystem with NVMe hosts. This parameter should be used with caution."`
	NVMeSubsystemUpdate    NVMeSubsystemUpdate `json:"nvme_subsystem_update,omitzero" jsonschema:"update NVMe subsystem operation"`
}

type NVMeSubsystemUpdate struct {
	Comment string `json:"comment,omitzero" jsonschema:"configurable comment for the NVMe subsystem"`
}

type NVMeSubsystemHost struct {
	Cluster string   `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string   `json:"svm_name" jsonschema:"SVM name"`
	Name    string   `json:"name" jsonschema:"name for NVMe subsystem"`
	NQN     string   `json:"nqn,omitzero" jsonschema:"NVMe qualified name (NQN) used to identify the NVMe host"`
	Records []string `json:"records_nqns,omitzero" jsonschema:"array of NVMe hosts specified to add multiple NVMe hosts to an NVMe subsystem"`
}

type NVMeNamespace struct {
	Cluster                string `json:"cluster_name" jsonschema:"cluster name"`
	SVM                    string `json:"svm_name" jsonschema:"SVM name"`
	Name                   string `json:"name" jsonschema:"name for NVMe namespace"`
	OSType                 string `json:"os_type,omitzero" jsonschema:"operating system type of the NVMe namespace (e.g., aix, linux, vmware, windows)"`
	Size                   string `json:"space.size,omitzero" jsonschema:"total provisioned size of the NVMe namespace (e.g., '100GB', '1TB')"`
	AllowDeleteWhileMapped bool   `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows deletion of a mapped NVMe namespace. This parameter should be used with caution."`
}

type NVMeNamespaceModify struct {
	Cluster                string              `json:"cluster_name" jsonschema:"cluster name"`
	Operation              string              `json:"operation" jsonschema:"NVMe namespace operation type (e.g., update, delete)"`
	SVM                    string              `json:"svm_name" jsonschema:"SVM name"`
	Name                   string              `json:"name" jsonschema:"name for NVMe namespace"`
	AllowDeleteWhileMapped bool                `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows deletion of a mapped NVMe namespace. This parameter should be used with caution."`
	NVMeNamespaceUpdate    NVMeNamespaceUpdate `json:"nvme_namespace_update,omitzero" jsonschema:"update NVMe namespace operation"`
}

type NVMeNamespaceUpdate struct {
	Size string `json:"space.size,omitzero" jsonschema:"total provisioned size of the NVMe namespace"`
}

type NVMeSubsystemMap struct {
	Cluster   string `json:"cluster_name" jsonschema:"cluster name"`
	SVM       string `json:"svm_name" jsonschema:"SVM name"`
	Subsystem string `json:"subsystem_name" jsonschema:"name for NVMe subsystem"`
	Namespace string `json:"namespace_name" jsonschema:"name for NVMe namespace"`
}

type FCPService struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
	Enabled string `json:"enabled,omitzero" jsonschema:"admin state of the FCP service"`
}

type FCPServiceModify struct {
	Cluster          string           `json:"cluster_name" jsonschema:"cluster name"`
	Operation        string           `json:"operation" jsonschema:"FCP service operation type (e.g., update, delete)"`
	SVM              string           `json:"svm_name" jsonschema:"SVM name"`
	FCPServiceUpdate FCPServiceUpdate `json:"fcp_service_update,omitzero" jsonschema:"update FCP service operation"`
}

type FCPServiceUpdate struct {
	Enabled string `json:"enabled,omitzero" jsonschema:"admin state of the FCP service"`
}

type FCInterfaceCreate struct {
	Cluster      string `json:"cluster_name" jsonschema:"cluster name"`
	SVM          string `json:"svm_name" jsonschema:"SVM name"`
	Name         string `json:"name" jsonschema:"FC interface name"`
	DataProtocol string `json:"data_protocol" jsonschema:"data protocol of the FC interface (e.g. fcp)"`
	Enabled      string `json:"enabled,omitzero" jsonschema:"admin state of the FC interface"`
	HomeNodeName string `json:"location.home_port.node.name" jsonschema:"name of the home node for the FC interface"`
	HomePortName string `json:"location.home_port.name" jsonschema:"name of the home port on the home node for the FC interface"`
}

type FCInterface struct {
	Cluster      string `json:"cluster_name" jsonschema:"cluster name"`
	SVM          string `json:"svm_name" jsonschema:"SVM name"`
	Name         string `json:"name" jsonschema:"FC interface name"`
	Enabled      string `json:"enabled,omitzero" jsonschema:"admin state of the FC interface"`
	HomeNodeName string `json:"location.home_port.node.name,omitzero" jsonschema:"name of the home node for the FC interface"`
	HomePortName string `json:"location.home_port.name,omitzero" jsonschema:"name of the home port on the home node for the FC interface"`
}

type FCInterfaceModify struct {
	Cluster           string            `json:"cluster_name" jsonschema:"cluster name"`
	Operation         string            `json:"operation" jsonschema:"FC interface operation type (e.g., update, delete)"`
	SVM               string            `json:"svm_name" jsonschema:"SVM name"`
	Name              string            `json:"name" jsonschema:"FC interface name"`
	FCInterfaceUpdate FCInterfaceUpdate `json:"fc_interface_update,omitzero" jsonschema:"update FC interface operation"`
}

type FCInterfaceUpdate struct {
	Enabled      string `json:"enabled,omitzero" jsonschema:"admin state of the FC interface"`
	HomeNodeName string `json:"location.home_port.node.name,omitzero" jsonschema:"name of the home node for the FC interface"`
	HomePortName string `json:"location.home_port.name,omitzero" jsonschema:"name of the home port on the home node for the FC interface"`
}

type IGroupCreate struct {
	Cluster  string `json:"cluster_name" jsonschema:"cluster name"`
	SVM      string `json:"svm_name" jsonschema:"SVM name"`
	Name     string `json:"name" jsonschema:"igroup name"`
	OSType   string `json:"os_type" jsonschema:"OS type (aix, hpux, hyper_v, linux, netware, openvms, solaris, vmware, windows, xen)"`
	Protocol string `json:"protocol" jsonschema:"protocol (fcp, iscsi, mixed)"`
	Comment  string `json:"comment,omitzero" jsonschema:"comment"`
}
type IGroup struct {
	Cluster                string `json:"cluster_name" jsonschema:"cluster name"`
	SVM                    string `json:"svm_name" jsonschema:"SVM name"`
	Name                   string `json:"name" jsonschema:"igroup name"`
	NewName                string `json:"new_name,omitzero" jsonschema:"new igroup name"`
	OSType                 string `json:"os_type,omitzero" jsonschema:"OS type (aix, hpux, hyper_v, linux, netware, openvms, solaris, vmware, windows, xen)"`
	Comment                string `json:"comment,omitzero" jsonschema:"comment"`
	AllowDeleteWhileMapped bool   `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows the deletion of a mapped initiator group. This parameter should be used with caution"`
}

type IGroupModify struct {
	Cluster                string       `json:"cluster_name" jsonschema:"cluster name"`
	Operation              string       `json:"operation" jsonschema:"igroup operation type (e.g., update, delete)"`
	SVM                    string       `json:"svm_name" jsonschema:"SVM name"`
	Name                   string       `json:"name" jsonschema:"igroup name"`
	AllowDeleteWhileMapped bool         `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows the deletion of a mapped initiator group. This parameter should be used with caution"`
	IGroupUpdate           IGroupUpdate `json:"igroup_update,omitzero" jsonschema:"update igroup operation"`
}

type IGroupUpdate struct {
	NewName string `json:"new_name,omitzero" jsonschema:"new igroup name"`
	OSType  string `json:"os_type,omitzero" jsonschema:"OS type (aix, hpux, hyper_v, linux, netware, openvms, solaris, vmware, windows, xen)"`
	Comment string `json:"comment,omitzero" jsonschema:"comment"`
}

type IGroupInitiator struct {
	Cluster                string   `json:"cluster_name" jsonschema:"cluster name"`
	SVM                    string   `json:"svm_name" jsonschema:"SVM name"`
	IGroupName             string   `json:"igroup_name" jsonschema:"igroup name"`
	InitiatorName          string   `json:"initiator_name,omitzero" jsonschema:"initiator name (IQN for iSCSI or WWPN for FC)"`
	Comment                string   `json:"comment,omitzero" jsonschema:"comment"`
	Records                []string `json:"records,omitzero" jsonschema:"An array of initiators specified to add multiple initiators to an initiator group in a single API call"`
	AllowDeleteWhileMapped bool     `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows the deletion of an initiator from a mapped initiator group. This parameter should be used with caution."`
}

type LunMap struct {
	Cluster    string `json:"cluster_name" jsonschema:"cluster name"`
	SVM        string `json:"svm_name" jsonschema:"SVM name"`
	LunName    string `json:"lun_name" jsonschema:"LUN name (full path, e.g. /vol/vol1/lun1)"`
	IGroupName string `json:"igroup_name" jsonschema:"igroup name to map the LUN to"`
}

type SnapMirrorCreate struct {
	Cluster         string `json:"cluster_name" jsonschema:"cluster name"`
	SourcePath      string `json:"source.path" jsonschema:"SnapMirror source endpoint path (format: <svm>:<volume>, e.g. vs1:vol1)"`
	DestinationPath string `json:"destination.path" jsonschema:"SnapMirror destination endpoint path (format: <svm>:<volume>, e.g. vs2:vol2)"`
	PolicyName      string `json:"policy_name" jsonschema:"SnapMirror policy name"`
}
type SnapMirror struct {
	Cluster              string `json:"cluster_name" jsonschema:"cluster name"`
	DestinationPath      string `json:"destination.path" jsonschema:"SnapMirror destination endpoint path (format: <svm>:<volume>, e.g. vs2:vol2)"`
	PolicyName           string `json:"policy_name,omitzero" jsonschema:"SnapMirror policy name"`
	TransferScheduleName string `json:"transfer_schedule.name,omitzero" jsonschema:"SnapMirror transfer schedule name"`
	State                string `json:"state,omitzero" jsonschema:"State of the relationship (e.g., broken_off, paused, snapmirrored, uninitialized, in_sync, out_of_sync, synchronizing, expanding)"`
}

type SnapMirrorModify struct {
	Cluster          string           `json:"cluster_name" jsonschema:"cluster name"`
	Operation        string           `json:"operation" jsonschema:"SnapMirror operation type (e.g., update, delete)"`
	DestinationPath  string           `json:"destination.path" jsonschema:"SnapMirror destination endpoint path (format: <svm>:<volume>, e.g. vs2:vol2)"`
	SnapMirrorUpdate SnapMirrorUpdate `json:"snapmirror_update,omitzero" jsonschema:"update SnapMirror relationship operation"`
}

type SnapMirrorUpdate struct {
	PolicyName           string `json:"policy_name,omitzero" jsonschema:"SnapMirror policy name"`
	TransferScheduleName string `json:"transfer_schedule_name,omitzero" jsonschema:"SnapMirror transfer schedule name"`
	SnapMirrorOperation  string `json:"snapmirror_operation,omitzero" jsonschema:"SnapMirror relationship operations (e.g., initialize, break, resync)"`
	State                string `json:"state,omitzero" jsonschema:"State of the relationship (e.g., broken_off, paused, snapmirrored, uninitialized, in_sync, out_of_sync, synchronizing, expanding)"`
}

type OntapGetParams struct {
	Cluster    string            `json:"cluster_name" jsonschema:"cluster name, from list_registered_clusters"`
	Fields     string            `json:"fields,omitzero" jsonschema:"comma-separated dot-notation fields to return, e.g. \"name,svm.name,space.size\" — use space.* to expand all space sub-fields"`
	Path       string            `json:"path" jsonschema:"ONTAP REST API path without /api prefix. May be a collection (e.g. /storage/volumes) or a resource template (e.g. /storage/volumes/{uuid})"`
	PathParams map[string]string `json:"path_params,omitzero" jsonschema:"values for path placeholders when the path contains {param} segments, e.g. {\"volume.uuid\":\"abc-123\"}. Get the UUID first from the collection endpoint."`
	Filters    map[string]string `json:"filters,omitzero" jsonschema:"filter key-value pairs using ONTAP query syntax as JSON object, e.g. {\"svm.name\":\"vs1\",\"state\":\"online\"}"`
	MaxRecords int               `json:"max_records,omitzero" jsonschema:"limit results. omit to return all records"`
}

type ListEndpointsParams struct {
	Match string `json:"match,omitzero" jsonschema:"optional substring or regex to filter endpoint paths and summaries; omit to return all"`
}

type SearchEndpointsParams struct {
	Query string `json:"query" jsonschema:"keyword to search across endpoint paths, summaries and tags"`
}

type DescribeEndpointParams struct {
	Path    string `json:"path" jsonschema:"ONTAP REST API path, e.g. /storage/volumes"`
	Cluster string `json:"cluster_name,omitzero" jsonschema:"cluster name — if provided, filters out fields and filters not available in that cluster's ONTAP version"`
}

type NFSService struct {
	Cluster    string `json:"cluster_name" jsonschema:"cluster name"`
	SVM        string `json:"svm_name" jsonschema:"SVM name"`
	Enabled    string `json:"enabled,omitzero" jsonschema:"admin state of the NFS service (true/false, default: true)"`
	V3Enabled  string `json:"v3_enabled,omitzero" jsonschema:"enable NFSv3 (true/false)"`
	V40Enabled string `json:"v40_enabled,omitzero" jsonschema:"enable NFSv4.0 (true/false)"`
	V41Enabled string `json:"v41_enabled,omitzero" jsonschema:"enable NFSv4.1 (true/false)"`
}

type NFSServiceModify struct {
	Cluster          string           `json:"cluster_name" jsonschema:"cluster name"`
	Operation        string           `json:"operation" jsonschema:"NFS service operation type (e.g., update, delete)"`
	SVM              string           `json:"svm_name" jsonschema:"SVM name"`
	NFSServiceUpdate NFSServiceUpdate `json:"nfs_service_update,omitzero" jsonschema:"update NFS service operation"`
}

type NFSServiceUpdate struct {
	Enabled    string `json:"enabled,omitzero" jsonschema:"admin state of the NFS service (true/false)"`
	V3Enabled  string `json:"v3_enabled,omitzero" jsonschema:"enable NFSv3 (true/false)"`
	V40Enabled string `json:"v40_enabled,omitzero" jsonschema:"enable NFSv4.0 (true/false)"`
	V41Enabled string `json:"v41_enabled,omitzero" jsonschema:"enable NFSv4.1 (true/false)"`
}

type CIFSServiceCreate struct {
	Cluster    string `json:"cluster_name" jsonschema:"cluster name"`
	SVM        string `json:"svm_name" jsonschema:"SVM name"`
	Name       string `json:"cifs_server_name" jsonschema:"CIFS server name (NetBIOS name, max 15 chars)"`
	ADDomain   string `json:"ad_domain" jsonschema:"Active Directory domain FQDN to join"`
	ADUser     string `json:"ad_user" jsonschema:"AD admin username with domain join privileges"`
	ADPassword string `json:"ad_password" jsonschema:"AD admin password"`
	ADOu       string `json:"ad_ou,omitzero" jsonschema:"AD organizational unit (e.g., CN=Computers)"`
}

type CIFSService struct {
	Cluster    string `json:"cluster_name" jsonschema:"cluster name"`
	SVM        string `json:"svm_name" jsonschema:"SVM name"`
	Name       string `json:"cifs_server_name,omitzero" jsonschema:"CIFS server name (NetBIOS name, max 15 chars)"`
	ADUser     string `json:"ad_user,omitzero" jsonschema:"AD admin username with domain join privileges"`
	ADPassword string `json:"ad_password,omitzero" jsonschema:"AD admin password"`
}

type CIFSServiceModify struct {
	Cluster           string            `json:"cluster_name" jsonschema:"cluster name"`
	Operation         string            `json:"operation" jsonschema:"CIFS service operation type (e.g., update, delete)"`
	SVM               string            `json:"svm_name" jsonschema:"SVM name"`
	ADUser            string            `json:"ad_user,omitzero" jsonschema:"AD admin username (required with ad_password for clean AD unjoin on delete)"`
	ADPassword        string            `json:"ad_password,omitzero" jsonschema:"AD admin password (required with ad_user for clean AD unjoin on delete)"`
	CIFSServiceUpdate CIFSServiceUpdate `json:"cifs_service_update,omitzero" jsonschema:"update CIFS service operation"`
}

type CIFSServiceUpdate struct {
	Name string `json:"cifs_server_name,omitzero" jsonschema:"new CIFS server name (NetBIOS name, max 15 chars)"`
}

type DNSServiceCreate struct {
	Cluster              string   `json:"cluster_name" jsonschema:"cluster name"`
	SVM                  string   `json:"svm_name" jsonschema:"SVM name"`
	Domains              []string `json:"domains" jsonschema:"list of DNS domain names (e.g., [\"example.com\"])"`
	Servers              []string `json:"servers" jsonschema:"list of DNS server IP addresses (e.g., [\"10.0.0.1\"])"`
	SkipConfigValidation bool     `json:"skip_config_validation,omitzero" jsonschema:"Indicates whether or not the validation for the specified DNS configuration is disabled."`
}

type DNSServiceDelete struct {
	Cluster              string   `json:"cluster_name" jsonschema:"cluster name"`
	SVM                  string   `json:"svm_name" jsonschema:"SVM name"`
	Domains              []string `json:"domains,omitzero" jsonschema:"list of DNS domain names (e.g., [\"example.com\"])"`
	Servers              []string `json:"servers,omitzero" jsonschema:"list of DNS server IP addresses (e.g., [\"10.0.0.1\"])"`
	SkipConfigValidation bool     `json:"skip_config_validation,omitzero" jsonschema:"Indicates whether or not the validation for the specified DNS configuration is disabled."`
}

type SVMCreate struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	Name    string `json:"svm_name" jsonschema:"SVM name"`
}

type SVM struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	Name    string `json:"svm_name" jsonschema:"SVM name"`
	NewName string `json:"new_name,omitzero" jsonschema:"new name of SVM"`
	State   string `json:"state,omitzero" jsonschema:"state of SVM (e.g., starting, running, stopping, stopped, deleting, initializing)"`
	Comment string `json:"comment,omitzero" jsonschema:"comment"`
}

type SVMModify struct {
	Cluster   string    `json:"cluster_name" jsonschema:"cluster name"`
	Operation string    `json:"operation" jsonschema:"SVM operation type (e.g., update, delete)"`
	Name      string    `json:"svm_name" jsonschema:"SVM name"`
	SVMUpdate SVMUpdate `json:"svm_update,omitzero" jsonschema:"update SVM operation"`
}

type SVMUpdate struct {
	NewName string `json:"new_name,omitzero" jsonschema:"new name of SVM"`
	State   string `json:"state,omitzero" jsonschema:"state of SVM (e.g., starting, running, stopping, stopped, deleting, initializing)"`
	Comment string `json:"comment,omitzero" jsonschema:"comment"`
}

type SVMPeer struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name" jsonschema:"SVM name"`
}

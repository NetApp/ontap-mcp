package tool

type Volume struct {
	Cluster      string `json:"cluster_name" jsonschema:"cluster name"`
	SVM          string `json:"svm_name" jsonschema:"SVM name"`
	Volume       string `json:"volume_name" jsonschema:"volume name"`
	Aggregate    string `json:"aggregate_name,omitzero" jsonschema:"aggregate name"`
	NewVolume    string `json:"new_volume_name,omitzero" jsonschema:"new volume name"`
	Size         string `json:"size,omitzero" jsonschema:"size of the volume (e.g., '100GB', '1TB')"`
	NewSize      string `json:"new_size,omitzero" jsonschema:"new size of the volume (e.g., '100GB', '1TB')"`
	NewState     string `json:"new_state,omitzero" jsonschema:"new state of the volume (e.g., 'online', 'offline')"`
	ExportPolicy string `json:"export_policy,omitzero" jsonschema:"nfs export policy name. Will be created if it doesn't exist"`
}

type SnapshotPolicy struct {
	Cluster  string `json:"cluster_name" jsonschema:"cluster name"`
	SVM      string `json:"svm_name" jsonschema:"SVM name"`
	Name     string `json:"name,omitzero" jsonschema:"snapshot policy name"`
	Schedule string `json:"schedule,omitzero" jsonschema:"schedule of snapshot policy"`
	Count    int    `json:"count,omitzero" jsonschema:"number of snapshots"`
}

type QoSPolicy struct {
	Cluster         string `json:"cluster_name" jsonschema:"cluster name"`
	SVM             string `json:"svm_name,omitzero" jsonschema:"SVM name"`
	Name            string `json:"name,omitzero" jsonschema:"qos policy name"`
	NewName         string `json:"new_name,omitzero" jsonschema:"new qos policy name"`
	MaxThIOPS       string `json:"max_throughput_iops,omitzero" jsonschema:"max throughput of fixed qos policy"`
	MinThIOPS       string `json:"min_throughput_iops,omitzero" jsonschema:"min throughput of fixed qos policy"`
	ExpectedIOPS    string `json:"expected_iops,omitzero" jsonschema:"expected iops of adaptive qos policy"`
	PeakIOPS        string `json:"peak_iops,omitzero" jsonschema:"peak iops of adaptive qos policy"`
	AbsoluteMinIOPS string `json:"absolute_min_iops,omitzero" jsonschema:"absolute min iops of adaptive qos policy"`
	CapacityShared  bool   `json:"capacity_shared,omitzero" jsonschema:"whether the capacities are shared across all objects that use this QoS policy-group. Default is false."`
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

type NFSExportPolicyRules struct {
	Cluster        string `json:"cluster_name" jsonschema:"cluster name"`
	ExportPolicy   string `json:"export_policy" jsonschema:"nfs export policy name"`
	OldClientMatch string `json:"old_client,omitzero" jsonschema:"old list of clients"`
	ClientMatch    string `json:"client,omitzero" jsonschema:"new list of clients"`
	OldROrule      string `json:"old_ro_rule,omitzero" jsonschema:"old read only rules"`
	ROrule         string `json:"ro_rule,omitzero" jsonschema:"new read only rules"`
	OldRWrule      string `json:"old_rw_rule,omitzero" jsonschema:"old read write rules"`
	RWrule         string `json:"rw_rule,omitzero" jsonschema:"new read write rules"`
}

type CIFSShare struct {
	Cluster string `json:"cluster_name" jsonschema:"cluster name"`
	SVM     string `json:"svm_name,omitzero" jsonschema:"SVM name"`
	Name    string `json:"name,omitzero" jsonschema:"cifs share name"`
	Path    string `json:"path,omitzero" jsonschema:"cifs share path"`
}

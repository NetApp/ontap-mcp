package ontap

import (
	"fmt"
	"time"
)

type OErr struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Target  string `json:"target"`
}

type ClusterError struct {
	Err        OErr `json:"error"`
	StatusCode int
}

func (o ClusterError) Error() string {
	return fmt.Sprintf("message: %s code: %s", o.Err.Message, o.Err.Code)
}

type JobResponse struct {
	UUID        string    `json:"uuid"`
	Description string    `json:"description"`
	State       string    `json:"state"` // enum: queued, running, paused, success, failure
	Message     string    `json:"message"`
	Code        int       `json:"code"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Svm         struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"svm,omitzero"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitzero"`
}

type PostJob struct {
	Job struct {
		UUID string `json:"uuid"`
	} `json:"job"`
}

type GetData struct {
	Records []struct {
		ID       int          `json:"id,omitzero"`
		UUID     string       `json:"uuid,omitzero"`
		Index    int          `json:"index,omitzero"`
		Name     string       `json:"name,omitzero"`
		Svm      NameAndUUID  `json:"svm,omitzero"`
		Volume   NameAndUUID  `json:"volume,omitzero"`
		RoRule   []string     `json:"ro_rule,omitzero"`
		RwRule   []string     `json:"rw_rule,omitzero"`
		Clients  []ClientData `json:"clients,omitzero"`
		Nas      NAS          `json:"nas,omitzero"`
		Schedule NameAndUUID  `json:"schedule,omitzero"`
	} `json:"records"`
	NumRecords int `json:"num_records"`
}

type NASExportPolicy struct {
	Name string `json:"name"`
}

type Target struct {
	Alias string `json:"alias"`
}

type IP struct {
	Address string `json:"address" jsonschema:"IP address for the interface"`
	Netmask string `json:"netmask" jsonschema:"IP netmask for the interface"`
}

type Location struct {
	HomeNode        NameAndUUID `json:"home_node,omitzero" jsonschema:"home node"`
	BroadcastDomain NameAndUUID `json:"broadcast_domain,omitzero" jsonschema:"broadcast domain"`
	AutoRevert      string      `json:"auto_revert,omitzero" jsonschema:"auto revert"`
}

type NAS struct {
	ExportPolicy NASExportPolicy `json:"export_policy,omitzero"`
	Path         string          `json:"path,omitzero"`
}

type Autosize struct {
	MaxSize         string `json:"maximum,omitzero"`
	MinSize         string `json:"minimum,omitzero"`
	Mode            string `json:"mode"` // enum: grow, grow_shrink, off
	GrowThreshold   string `json:"grow_threshold,omitzero"`
	ShrinkThreshold string `json:"shrink_threshold,omitzero"`
}

type VolumeQoSPolicy struct {
	Name           string `json:"name,omitzero"`
	MaxThroughIOPS *int   `json:"max_throughput_iops,omitzero"`
	MinThroughIOPS *int   `json:"min_throughput_iops,omitzero"`
	MaxThroughMBPS *int   `json:"max_throughput_mbps,omitzero"`
	MinThroughMBPS *int   `json:"min_throughput_mbps,omitzero"`
}

type VolumeQoS struct {
	Policy VolumeQoSPolicy `json:"policy,omitzero"`
}

type Volume struct {
	SVM        NameAndUUID   `json:"svm,omitzero"`
	Name       string        `json:"name,omitzero"`
	Aggregates []NameAndUUID `json:"aggregates,omitzero"`
	State      string        `json:"state,omitempty"` // enum: error, mixed, offline, online, restricted
	Style      string        `json:"style,omitempty"` // enum: flexvol, flexgroup, flexgroup_constituent
	Size       int64         `json:"size,omitempty"`
	Nas        NAS           `json:"nas,omitzero"`
	Autosize   Autosize      `json:"autosize,omitzero"`
	QoS        VolumeQoS     `json:"qos,omitzero"`
}

type NameAndUUID struct {
	Name string `json:"name,omitempty"`
	UUID string `json:"uuid,omitempty"`
}

type OnlyCount struct {
	NumRecords int `json:"num_records"`
}

type NameAndSVM struct {
	Name string      `json:"name"`
	Svm  NameAndUUID `json:"svm,omitzero"`
}

type SnapshotPolicy struct {
	SVM     NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Name    string      `json:"name,omitzero" jsonschema:"snapshot policy name"`
	Copies  []Copy      `json:"copies,omitzero" jsonschema:"snapshot copies"`
	Enabled string      `json:"enabled,omitzero" jsonschema:"the state of snapshot policy"`
	Comment string      `json:"comment,omitzero" jsonschema:"comment associated with the snapshot policy"`
}

type Copy struct {
	Count    int      `json:"count"`
	Schedule Schedule `json:"schedule"`
}

type Schedule struct {
	Name string `json:"name"`
	Cron Cron   `json:"cron,omitzero"`
}

type SnapshotPolicySchedule struct {
	Count           int         `json:"count,omitzero" jsonschema:"number of snapshots to keep for this schedule"`
	Schedule        NameAndUUID `json:"schedule,omitzero" jsonschema:"name of the schedule"`
	SnapmirrorLabel string      `json:"snapmirror_label,omitzero" jsonschema:"SnapMirror label for this schedule"`
}

type Cron struct {
	Days     []int `json:"days,omitzero"`
	Hours    []int `json:"hours,omitzero"`
	Minutes  []int `json:"minutes"`
	Months   []int `json:"months,omitzero"`
	Weekdays []int `json:"weekdays,omitzero"`
}

type QoSPolicy struct {
	SVM      NameAndUUID `json:"svm,omitzero"`
	Name     string      `json:"name,omitzero"`
	Fixed    QoSFixed    `json:"fixed,omitzero"`
	Adaptive QoSAdaptive `json:"adaptive,omitzero"`
}

type QoSFixed struct {
	MaxThIOPS      int64 `json:"max_throughput_iops"`
	MinThIOPS      int64 `json:"min_throughput_iops"`
	CapacityShared bool  `json:"capacity_shared"`
}

type QoSAdaptive struct {
	ExpectedIOPS    int64 `json:"expected_iops"`
	PeakIOPS        int64 `json:"peak_iops"`
	AbsoluteMinIOPS int64 `json:"absolute_min_iops"`
}

type ExportPolicy struct {
	SVM   NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Name  string      `json:"name,omitzero" jsonschema:"export policy name"`
	Rules []Rule      `json:"rules,omitzero" jsonschema:"rules of export policy"`
}

type ClientData struct {
	Match string `json:"match,omitempty"`
}

type Rule struct {
	Clients    []ClientData `json:"clients,omitzero" jsonschema:"list of clients"`
	ROrule     []string     `json:"ro_rule,omitzero" jsonschema:"read only rules"`
	RWrule     []string     `json:"rw_rule,omitzero" jsonschema:"read write rules"`
	ClientsStr string       `json:"clients_string,omitzero" jsonschema:"list of clients string"`
	ROruleStr  string       `json:"ro_rule_string,omitzero" jsonschema:"read only rules string"`
	RWruleStr  string       `json:"rw_rule_string,omitzero" jsonschema:"read write rules string"`
}

type CIFSShare struct {
	SVM  NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Name string      `json:"name,omitzero" jsonschema:"cifs share name"`
	Path string      `json:"path,omitzero" jsonschema:"cifs share path"`
}

type SVM struct {
	Name string `json:"name" jsonschema:"svm name"`
}

type Qtree struct {
	SVM    NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Volume NameAndUUID `json:"volume,omitzero" jsonschema:"volume name"`
	Name   string      `json:"name,omitzero" jsonschema:"qtree name"`
}

type NVMeService struct {
	SVM     NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Enabled string      `json:"enabled,omitzero" jsonschema:"admin state of the NVMe service"`
}

type IscsiService struct {
	SVM     NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Enabled string      `json:"enabled,omitzero" jsonschema:"admin state of the iSCSI service"`
	Target  Target      `json:"target,omitzero" jsonschema:"target of iSCSI service"`
}

type NetworkIPInterface struct {
	SVM           NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	IPSpace       NameAndUUID `json:"ipspace,omitzero" jsonschema:"ipspace name"`
	Name          string      `json:"name" jsonschema:"name of the interface"`
	Scope         string      `json:"scope,omitzero" jsonschema:"scope"` // enum: cluster, svm
	IP            IP          `json:"ip,omitzero" jsonschema:"ip address"`
	Subnet        NameAndUUID `json:"subnet,omitzero" jsonschema:"subnet name"`
	Location      Location    `json:"location,omitzero" jsonschema:"location name"`
	ServicePolicy NameAndUUID `json:"service_policy,omitzero" jsonschema:"service policy"`
}

type NVMeSubsystem struct {
	SVM                    NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Name                   string      `json:"name,omitzero" jsonschema:"name for NVMe subsystem"`
	OSType                 string      `json:"os_type,omitzero" jsonschema:"operating system of the NVMe subsystem's hosts"`
	Hosts                  []NVMeHost  `json:"hosts,omitzero" jsonschema:"NVMe hosts configured for access to the NVMe subsystem"`
	Comment                string      `json:"comment,omitzero" jsonschema:"configurable comment for the NVMe subsystem"`
	AllowDeleteWhileMapped bool        `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows for the deletion of a mapped NVMe subsystem. This parameter should be used with caution."`
	AllowDeleteWithHosts   bool        `json:"allow_delete_with_hosts,omitzero" jsonschema:"Allows for the deletion of an NVMe subsystem with NVMe hosts. This parameter should be used with caution."`
}

type NVMeHost struct {
	NQN string `json:"nqn,omitzero" jsonschema:"NVMe qualified name (NQN) used to identify the NVMe host"`
}

type NVMeSubsystemHost struct {
	NQN     string     `json:"nqn,omitzero" jsonschema:"NVMe qualified name (NQN) used to identify the NVMe host"`
	Records []NVMeHost `json:"records,omitzero" jsonschema:"array of NVMe hosts specified to add multiple NVMe hosts to an NVMe subsystem"`
}

type NVMeNamespace struct {
	SVM                    NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Name                   string      `json:"name,omitzero" jsonschema:"name for NVMe namespace"`
	OSType                 string      `json:"os_type,omitzero" jsonschema:"operating system type of the NVMe namespace"`
	Space                  Space       `json:"space,omitzero" jsonschema:"space of NVMe namespace"`
	AllowDeleteWhileMapped bool        `json:"allow_delete_while_mapped,omitzero" jsonschema:"Allows deletion of a mapped NVMe namespace. This parameter should be used with caution."`
}

type Space struct {
	Size string `json:"size,omitzero" jsonschema:"total provisioned size of the NVMe namespace (e.g., '100GB', '1TB')"`
}

type NVMeSubsystemMap struct {
	SVM       NameAndUUID `json:"svm" jsonschema:"svm name"`
	Subsystem NameAndUUID `json:"subsystem" jsonschema:"subsystem name"`
	Namespace NameAndUUID `json:"namespace" jsonschema:"namespace name"`
}

const (
	ASAr2 = "asar2"
	CDOT  = "cdot"
)

type Cluster struct {
	Name          string  `json:"name"`
	UUID          string  `json:"uuid"`
	Version       Version `json:"version"`
	SanOptimized  bool    `json:"san_optimized"`
	Disaggregated bool    `json:"disaggregated"`
}

type Version struct {
	Full       string `json:"full"`
	Generation int    `json:"generation"`
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
}

type Remote struct {
	Name            string
	Model           string
	UUID            string
	Version         Version
	Serial          string
	IsSanOptimized  bool
	IsDisaggregated bool
	ZAPIsExist      bool
	ZAPIsChecked    bool
	HasREST         bool
	IsClustered     bool
}

func (r Remote) IsZero() bool {
	return r.Name == "" && r.Model == "" && r.UUID == ""
}

func (r Remote) IsKeyPerf() bool {
	return r.IsDisaggregated
}

func (r Remote) IsAFX() bool {
	return r.IsDisaggregated && !r.IsSanOptimized
}

func (r Remote) IsASAr2() bool {
	return r.Model == ASAr2
}

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
		ID      int          `json:"id,omitzero"`
		UUID    string       `json:"uuid,omitzero"`
		Index   int          `json:"index,omitzero"`
		Name    string       `json:"name,omitzero"`
		Svm     NameAndUUID  `json:"svm,omitzero"`
		Volume  NameAndUUID  `json:"volume,omitzero"`
		RoRule  []string     `json:"ro_rule,omitzero"`
		RwRule  []string     `json:"rw_rule,omitzero"`
		Clients []ClientData `json:"clients,omitzero"`
		Nas     NAS          `json:"nas,omitzero"`
		Lun     NameAndUUID  `json:"lun,omitzero"`
		IGroup  NameAndUUID  `json:"igroup,omitzero"`
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
	SVM    NameAndUUID `json:"svm,omitzero" jsonschema:"svm name"`
	Name   string      `json:"name,omitzero" jsonschema:"snapshot policy name"`
	Copies []Copy      `json:"copies,omitzero" jsonschema:"snapshot copies"`
}

type Copy struct {
	Count    int      `json:"count"`
	Schedule Schedule `json:"schedule"`
}

type Schedule struct {
	Name string `json:"name"`
	Cron Cron   `json:"cron,omitzero"`
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

type InitiatorName struct {
	Name string `json:"name,omitzero" jsonschema:"The FC WWPN, iSCSI IQN, or iSCSI EUI that identifies the host initiator."`
}

type IGroupInitiator struct {
	Name    string          `json:"name,omitzero"`
	Comment string          `json:"comment,omitzero"`
	Records []InitiatorName `json:"records,omitzero" jsonschema:"An array of initiators specified to add multiple initiators to an initiator group in a single API call. Not allowed when the name property is used."`
}

type IGroup struct {
	SVM        NameAndUUID       `json:"svm,omitzero"`
	Name       string            `json:"name,omitzero"`
	OSType     string            `json:"os_type,omitzero"`
	Protocol   string            `json:"protocol,omitzero"`
	Comment    string            `json:"comment,omitzero"`
	Initiators []IGroupInitiator `json:"initiators,omitzero"`
}

type LunMap struct {
	SVM    NameAndUUID `json:"svm,omitzero"`
	Lun    NameAndUUID `json:"lun,omitzero"`
	IGroup NameAndUUID `json:"igroup,omitzero"`
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

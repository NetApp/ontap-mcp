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
		RoRule  []string     `json:"ro_rule,omitzero"`
		RwRule  []string     `json:"rw_rule,omitzero"`
		Clients []ClientData `json:"clients,omitzero"`
		Nas     NAS          `json:"nas,omitzero"`
	} `json:"records"`
	NumRecords int `json:"num_records"`
}

type NASExportPolicy struct {
	Name string `json:"name"`
}

type NAS struct {
	ExportPolicy NASExportPolicy `json:"export_policy"`
}

type Volume struct {
	SVM        NameAndUUID   `json:"svm,omitzero"`
	Name       string        `json:"name,omitzero"`
	Aggregates []NameAndUUID `json:"aggregates,omitzero"`
	State      string        `json:"state,omitempty"` // enum: error, mixed, offline, online, restricted
	Style      string        `json:"style,omitempty"` // enum: flexvol, flexgroup, flexgroup_constituent
	Size       int64         `json:"size,omitempty"`
	Nas        NAS           `json:"nas,omitzero"`
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
}

type QoSPolicy struct {
	SVM      NameAndUUID `json:"svm,omitzero"`
	Name     string      `json:"name,omitzero"`
	Fixed    QoSFixed    `json:"fixed,omitzero"`
	Adaptive QoSAdaptive `json:"adaptive,omitzero"`
}

type QoSFixed struct {
	MaxThIOPS int64 `json:"max_throughput_iops"`
	MinThIOPS int64 `json:"min_throughput_iops"`
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

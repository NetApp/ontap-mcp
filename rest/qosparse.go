package rest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	iopsPerUnitRe = regexp.MustCompile(`(?i)^(\d+)iops/(tb|gb)$`)
	iopsRe        = regexp.MustCompile(`(?i)^(\d+)iops$`)
	bpsRe         = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)(b|kb|mb|gb|tb)/s$`)
)

var unitToMbps = map[string]float64{
	"b":  1.0 / (1000 * 1000),
	"kb": 1.0 / 1000,
	"mb": 1,
	"gb": 1000,
	"tb": 1000 * 1000,
}

type Xput struct {
	IOPS string
	Mbps string
}

func parseXput(s string) (Xput, error) {
	lower := strings.ToLower(strings.TrimSpace(s))
	empty := Xput{}
	if lower == "" || lower == "inf" || lower == "0" {
		return empty, nil
	}

	before, after, found := strings.Cut(lower, ",")
	if found {
		l, e1 := parseXput(before)
		r, e2 := parseXput(after)
		if e1 != nil || e2 != nil {
			return empty, fmt.Errorf("parseXput %q: %w %w", s, e1, e2)
		}
		return Xput{IOPS: l.IOPS, Mbps: r.Mbps}, nil
	}

	if m := iopsPerUnitRe.FindStringSubmatch(lower); len(m) == 3 {
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return empty, fmt.Errorf("parseXput %q: %w", s, err)
		}
		if strings.EqualFold(m[2], "gb") {
			v *= 1000 // normalise to per-TB (ONTAP default)
		}
		return Xput{IOPS: strconv.FormatInt(v, 10)}, nil
	}

	if m := iopsRe.FindStringSubmatch(lower); len(m) == 2 {
		return Xput{IOPS: m[1]}, nil
	}

	if m := bpsRe.FindStringSubmatch(lower); len(m) == 3 {
		num, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return empty, fmt.Errorf("parseXput %q: %w", s, err)
		}
		mult, ok := unitToMbps[strings.ToLower(m[2])]
		if !ok {
			return empty, fmt.Errorf("parseXput %q: unknown unit %q", s, m[2])
		}
		mbps := num * mult
		mbpsStr := strconv.FormatFloat(mbps, 'f', 2, 64)
		mbpsStr = strings.TrimRight(mbpsStr, "0")
		mbpsStr = strings.TrimRight(mbpsStr, ".")
		return Xput{Mbps: mbpsStr}, nil
	}

	return empty, fmt.Errorf("parseXput %q: unrecognised format", s)
}

func xputIOPS(x Xput) (int64, error) {
	if x.IOPS == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(x.IOPS, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("xputIOPS: invalid IOPS value %q: %w", x.IOPS, err)
	}
	return v, nil
}

func xputMbps(x Xput) (float64, error) {
	if x.Mbps == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(x.Mbps, 64)
	if err != nil {
		return 0, fmt.Errorf("xputMbps: invalid Mbps value %q: %w", x.Mbps, err)
	}
	return v, nil
}

type xputField string

func (x *xputField) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*x = xputField(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*x = xputField(n.String())
	return nil
}

type cliFixedRecord struct {
	PolicyGroup   string    `json:"policy_group"`
	Vserver       string    `json:"vserver"`
	Class         string    `json:"class"`
	MaxThroughput xputField `json:"max_throughput"`
	MinThroughput xputField `json:"min_throughput"`
	NumWorkloads  int       `json:"num_workloads"`
	IsShared      bool      `json:"is_shared"`
}

type cliAdaptiveRecord struct {
	PolicyGroup            string    `json:"policy_group"`
	Vserver                string    `json:"vserver"`
	ExpectedIOPS           xputField `json:"expected_iops"`
	PeakIOPS               xputField `json:"peak_iops"`
	AbsoluteMinIOPS        xputField `json:"absolute_min_iops"`
	ExpectedIOPSAllocation string    `json:"expected_iops_allocation"`
	PeakIOPSAllocation     string    `json:"peak_iops_allocation"`
	BlockSize              string    `json:"block_size"`
	NumWorkloads           int       `json:"num_workloads"`
}

type clusterSVMRef struct {
	Name string `json:"name"`
}

type clusterFixed struct {
	MaxThroughputIOPS int64   `json:"max_throughput_iops,omitempty"`
	MaxThroughputMbps float64 `json:"max_throughput_mbps,omitempty"`
	MinThroughputIOPS int64   `json:"min_throughput_iops,omitempty"`
	MinThroughputMbps float64 `json:"min_throughput_mbps,omitempty"`
	CapacityShared    bool    `json:"capacity_shared"`
}

type clusterAdaptive struct {
	ExpectedIOPS           int64  `json:"expected_iops,omitempty"`
	PeakIOPS               int64  `json:"peak_iops,omitempty"`
	AbsoluteMinIOPS        int64  `json:"absolute_min_iops,omitempty"`
	BlockSize              string `json:"block_size,omitempty"`
	ExpectedIOPSAllocation string `json:"expected_iops_allocation,omitempty"`
	PeakIOPSAllocation     string `json:"peak_iops_allocation,omitempty"`
}

type clusterQoSRecord struct {
	Name        string           `json:"name"`
	SVM         clusterSVMRef    `json:"svm"`
	Scope       string           `json:"scope"`
	ObjectCount int              `json:"object_count,omitempty"`
	Fixed       *clusterFixed    `json:"fixed,omitempty"`
	Adaptive    *clusterAdaptive `json:"adaptive,omitempty"`
}

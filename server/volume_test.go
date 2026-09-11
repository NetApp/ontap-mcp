package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateVolume(t *testing.T) {
	tests := []struct {
		name            string
		volume          string
		svm             string
		aggregate       string
		model           string
		size            string
		path            string
		expectedErr     string
		expectedVolJSON ontap.Volume
	}{
		{
			name:            "Normal volume in cdot",
			volume:          "volume1",
			svm:             "svm1",
			aggregate:       "aggr1",
			model:           ontap.CDOT,
			size:            "100mb",
			path:            "/volume1",
			expectedErr:     "",
			expectedVolJSON: ontap.Volume{SVM: ontap.NameAndUUID{Name: "svm1"}, Name: "volume1", Aggregates: []ontap.NameAndUUID{{Name: "aggr1"}}, Size: 104857600, Nas: ontap.NAS{Path: "/volume1"}},
		},
		{
			name:            "Normal volume in afx",
			volume:          "volume2",
			svm:             "svm2",
			aggregate:       "",
			model:           ontap.AFX,
			size:            "10GB",
			path:            "/volume2",
			expectedErr:     "",
			expectedVolJSON: ontap.Volume{SVM: ontap.NameAndUUID{Name: "svm2"}, Name: "volume2", Size: 10737418240, Nas: ontap.NAS{Path: "/volume2"}},
		},
		{
			name:            "Volume with error in cdot",
			volume:          "volume3",
			svm:             "svm3",
			aggregate:       "",
			model:           ontap.CDOT,
			size:            "100mb",
			path:            "/volume3",
			expectedErr:     "aggregate name is required",
			expectedVolJSON: ontap.Volume{},
		},
		{
			name:            "Volume with error in afx",
			volume:          "volume4",
			svm:             "svm4",
			aggregate:       "aggr4",
			model:           ontap.AFX,
			size:            "100mb",
			path:            "/volume4",
			expectedErr:     "aggregate name must not be provided for AFX clusters",
			expectedVolJSON: ontap.Volume{},
		},
		{
			name:            "Volume with error in asar2",
			volume:          "volume5",
			svm:             "svm5",
			aggregate:       "aggr5",
			model:           ontap.ASAr2,
			size:            "100mb",
			path:            "/volume5",
			expectedErr:     "volume creation is not supported on ASAr2 clusters, use storage units instead",
			expectedVolJSON: ontap.Volume{},
		},
		{
			name:            "Volume without model in cdot",
			volume:          "volume6",
			svm:             "svm6",
			aggregate:       "",
			model:           "",
			size:            "100mb",
			path:            "/volume6",
			expectedErr:     "aggregate name is required",
			expectedVolJSON: ontap.Volume{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newCreateVolume(tool.VolumeCreate{SVM: tt.svm, Aggregate: tt.aggregate, Volume: tt.volume, Size: tt.size, JunctionPath: tt.path}, tt.model)
			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.expectedErr, err)
				}
			}
		})
	}
}

func classicASA() ontap.Remote {
	return ontap.Remote{Model: ontap.CDOT, IsSanOptimized: true}
}

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

func marshalVolume(t *testing.T, v ontap.Volume) string {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(body)
}

func TestCreateVolumeMapsCDOTFlexVolWithoutStyle(t *testing.T) {
	got, err := newCreateVolume(tool.VolumeCreate{
		SVM:       "vs1",
		Volume:    "vol1",
		Aggregate: "aggr1",
	}, ontap.CDOT)
	if err != nil {
		t.Fatalf("newCreateVolume() error = %v", err)
	}

	want := `{"svm":{"name":"vs1"},"name":"vol1","aggregates":[{"name":"aggr1"}]}`
	if body := marshalVolume(t, got); body != want {
		t.Errorf("flexvol create body = %s, want %s", body, want)
	}
}

func TestCreateVolumeMapsFlexGroupWithTwoAggregates(t *testing.T) {
	got, err := newCreateVolume(tool.VolumeCreate{
		SVM:                      "vs1",
		Volume:                   "fg1",
		Style:                    "flexgroup",
		AggregateNames:           []string{"aggr1", "aggr2"},
		ConstituentsPerAggregate: intPtr(10),
		Size:                     "1TB",
		JunctionPath:             "/fg1",
		GuaranteeType:            "none",
	}, ontap.CDOT)
	if err != nil {
		t.Fatalf("newCreateVolume() error = %v", err)
	}

	want := `{"svm":{"name":"vs1"},"name":"fg1","aggregates":[{"name":"aggr1"},{"name":"aggr2"}],"style":"flexgroup","size":1099511627776,"nas":{"path":"/fg1"},"guarantee":{"type":"none"},"constituents_per_aggregate":10}`
	if body := marshalVolume(t, got); body != want {
		t.Errorf("flexgroup create body = %s, want %s", body, want)
	}
}

func TestCreateVolumeRejectsBothAggregateNameAndAggregateNames(t *testing.T) {
	_, err := newCreateVolume(tool.VolumeCreate{
		SVM:            "vs1",
		Volume:         "fg1",
		Style:          "flexgroup",
		Aggregate:      "aggr1",
		AggregateNames: []string{"aggr1", "aggr2"},
	}, ontap.CDOT)
	if err == nil {
		t.Fatal("newCreateVolume() error = nil, want aggregate name conflict")
	}
	if !strings.Contains(err.Error(), "cannot set both") {
		t.Errorf("error = %q, want conflict about both aggregate fields", err)
	}
}

func TestCreateVolumeRejectsFlexGroupWithoutAggregateNames(t *testing.T) {
	_, err := newCreateVolume(tool.VolumeCreate{
		SVM:    "vs1",
		Volume: "fg1",
		Style:  "flexgroup",
	}, ontap.CDOT)
	if err == nil {
		t.Fatal("newCreateVolume() error = nil, want missing aggregate_names")
	}
	if !strings.Contains(err.Error(), "aggregate_names") {
		t.Errorf("error = %q, want aggregate_names required", err)
	}
}

func TestCreateVolumeRejectsFlexGroupConstituentStyle(t *testing.T) {
	_, err := newCreateVolume(tool.VolumeCreate{
		SVM:            "vs1",
		Volume:         "fg1",
		Style:          "flexgroup_constituent",
		AggregateNames: []string{"aggr1"},
	}, ontap.CDOT)
	if err == nil {
		t.Fatal("newCreateVolume() error = nil, want unsupported style")
	}
	if !strings.Contains(err.Error(), "flexgroup_constituent") {
		t.Errorf("error = %q, want flexgroup_constituent rejected", err)
	}
}

func TestCreateVolumeRejectsFlexGroupFieldsOnFlexVol(t *testing.T) {
	tests := []struct {
		name string
		in   tool.VolumeCreate
	}{
		{
			name: "aggregate_names",
			in: tool.VolumeCreate{
				SVM:            "vs1",
				Volume:         "vol1",
				Aggregate:      "aggr1",
				AggregateNames: []string{"aggr2"},
			},
		},
		{
			name: "constituents_per_aggregate",
			in: tool.VolumeCreate{
				SVM:                      "vs1",
				Volume:                   "vol1",
				Aggregate:                "aggr1",
				ConstituentsPerAggregate: intPtr(4),
			},
		},
		{
			name: "optimize_aggr_list",
			in: tool.VolumeCreate{
				SVM:              "vs1",
				Volume:           "vol1",
				Aggregate:        "aggr1",
				OptimizeAggrList: boolPtr(true),
			},
		},
		{
			name: "granular_data",
			in: tool.VolumeCreate{
				SVM:          "vs1",
				Volume:       "vol1",
				Aggregate:    "aggr1",
				GranularData: "basic",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newCreateVolume(tt.in, ontap.CDOT)
			if err == nil {
				t.Fatal("newCreateVolume() error = nil, want FlexGroup fields rejected on FlexVol")
			}
		})
	}
}

func TestCreateVolumeOmitsOptimizeAggregatesUnlessTrue(t *testing.T) {
	base := tool.VolumeCreate{
		SVM:            "vs1",
		Volume:         "fg1",
		Style:          "flexgroup",
		AggregateNames: []string{"aggr1", "aggr2"},
	}

	omitted, err := newCreateVolume(base, ontap.CDOT)
	if err != nil {
		t.Fatalf("omitted optimize_aggr_list: %v", err)
	}
	if strings.Contains(marshalVolume(t, omitted), "optimize_aggregates") {
		t.Errorf("omitted optimize_aggr_list body = %s, want no optimize_aggregates", marshalVolume(t, omitted))
	}

	falseVal := base
	falseVal.OptimizeAggrList = boolPtr(false)
	gotFalse, err := newCreateVolume(falseVal, ontap.CDOT)
	if err != nil {
		t.Fatalf("optimize_aggr_list=false: %v", err)
	}
	if strings.Contains(marshalVolume(t, gotFalse), "optimize_aggregates") {
		t.Errorf("false optimize_aggr_list body = %s, want no optimize_aggregates", marshalVolume(t, gotFalse))
	}

	trueVal := base
	trueVal.OptimizeAggrList = boolPtr(true)
	gotTrue, err := newCreateVolume(trueVal, ontap.CDOT)
	if err != nil {
		t.Fatalf("optimize_aggr_list=true: %v", err)
	}
	if !strings.Contains(marshalVolume(t, gotTrue), `"optimize_aggregates":true`) {
		t.Errorf("true optimize_aggr_list body = %s, want optimize_aggregates true", marshalVolume(t, gotTrue))
	}
}

func TestCreateVolumeMapsGranularDataModes(t *testing.T) {
	base := tool.VolumeCreate{
		SVM:            "vs1",
		Volume:         "fg1",
		Style:          "flexgroup",
		AggregateNames: []string{"aggr1", "aggr2"},
	}

	for _, mode := range []string{"", "disabled"} {
		in := base
		in.GranularData = mode
		got, err := newCreateVolume(in, ontap.CDOT)
		if err != nil {
			t.Fatalf("granular_data=%q: %v", mode, err)
		}
		body := marshalVolume(t, got)
		if strings.Contains(body, "granular_data") {
			t.Errorf("granular_data=%q body = %s, want no granular_data fields", mode, body)
		}
	}

	for _, mode := range []string{"basic", "advanced"} {
		in := base
		in.GranularData = mode
		got, err := newCreateVolume(in, ontap.CDOT)
		if err != nil {
			t.Fatalf("granular_data=%q: %v", mode, err)
		}
		body := marshalVolume(t, got)
		if !strings.Contains(body, `"granular_data":true`) || !strings.Contains(body, `"granular_data_mode":"`+mode+`"`) {
			t.Errorf("granular_data=%q body = %s, want enabled mode", mode, body)
		}
	}

	in := base
	in.GranularData = "weird"
	if _, err := newCreateVolume(in, ontap.CDOT); err == nil {
		t.Fatal("granular_data=weird: error = nil, want rejection")
	}
}

func TestCreateVolumeRejectsFlexGroupOnClassicASA(t *testing.T) {
	_, err := newCreateVolumeRemote(tool.VolumeCreate{
		SVM:            "vs1",
		Volume:         "fg1",
		Style:          "flexgroup",
		AggregateNames: []string{"aggr1", "aggr2"},
	}, classicASA())
	if err == nil {
		t.Fatal("newCreateVolumeRemote() error = nil, want ASA FlexGroup refusal")
	}
	if !strings.Contains(err.Error(), "All SAN Arrays") {
		t.Errorf("error = %q, want All SAN Arrays", err)
	}
}

func TestCreateVolumeRejectsFlexGroupFieldsOnAFX(t *testing.T) {
	_, err := newCreateVolume(tool.VolumeCreate{
		SVM:            "vs1",
		Volume:         "fg1",
		Style:          "flexgroup",
		AggregateNames: []string{"saz0"},
	}, ontap.AFX)
	if err == nil {
		t.Fatal("newCreateVolume() error = nil, want AFX FlexGroup refusal")
	}
	if !strings.Contains(err.Error(), "AFX") {
		t.Errorf("error = %q, want AFX placement message", err)
	}
}

func TestCreateVolumeAllowsFlexVolOnClassicASA(t *testing.T) {
	got, err := newCreateVolumeRemote(tool.VolumeCreate{
		SVM:       "vs1",
		Volume:    "vol1",
		Aggregate: "aggr1",
	}, classicASA())
	if err != nil {
		t.Fatalf("newCreateVolumeRemote() error = %v", err)
	}
	want := `{"svm":{"name":"vs1"},"name":"vol1","aggregates":[{"name":"aggr1"}]}`
	if body := marshalVolume(t, got); body != want {
		t.Errorf("classic ASA flexvol body = %s, want %s", body, want)
	}
}

func TestCreateVolumeMapsAFXFlexVolWithoutAggregate(t *testing.T) {
	got, err := newCreateVolume(tool.VolumeCreate{
		SVM:    "vs1",
		Volume: "vol1",
		Size:   "10GB",
	}, ontap.AFX)
	if err != nil {
		t.Fatalf("newCreateVolume() error = %v", err)
	}
	want := `{"svm":{"name":"vs1"},"name":"vol1","size":10737418240}`
	if body := marshalVolume(t, got); body != want {
		t.Errorf("AFX flexvol body = %s, want %s", body, want)
	}
}

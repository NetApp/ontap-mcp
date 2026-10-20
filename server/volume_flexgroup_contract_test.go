package server

import (
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestFlexGroupVolumeContract(t *testing.T) {
	const (
		firstAggr  = "harvest_vc_aggr"
		secondAggr = "umeng_aff300_aggr1"
		accepted   = `{"job":{"uuid":"fg-job"}}`
		// 100GB, the size every FlexGroup step in the integration suite uses.
		size = `107374182400`

		cdot  = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.16.1","generation":9,"major":16,"minor":1},"san_optimized":false,"disaggregated":false}`
		afx   = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.17.1","generation":9,"major":17,"minor":1},"san_optimized":false,"disaggregated":true}`
		asar2 = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.17.1","generation":9,"major":17,"minor":1},"san_optimized":true,"disaggregated":true}`
		asa   = `{"name":"contract-cluster","uuid":"c-uuid","version":{"full":"NetApp Release 9.16.1","generation":9,"major":16,"minor":1},"san_optimized":true,"disaggregated":false}`
	)
	cluster := func(resp string) exchange {
		return exchange{method: http.MethodGet, path: "/api/cluster?fields=*", resp: resp}
	}
	jobDone := exchange{method: http.MethodGet, path: "/api/cluster/jobs/fg-job", resp: `{"uuid":"fg-job","state":"success"}`}

	// post expects a FlexGroup create; fields are the body members after svm, name, style, and size.
	post := func(fields string) exchange {
		return exchange{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusAccepted, resp: accepted,
			body: `{"svm":{"name":"fgmarketing"},"name":"fgdocs","style":"flexgroup","size":` + size + `,` + fields + `}`}
	}
	// args mirrors the integration suite's flexGroupCreateArgs.
	args := func(extra map[string]any) map[string]any {
		a := map[string]any{"cluster_name": contractCluster, "svm_name": "fgmarketing", "volume_name": "fgdocs", "style": "flexgroup", "size": "100GB"}
		maps.Copy(a, extra)
		return a
	}
	bothAggrs := []string{firstAggr, secondAggr}

	tests := []struct {
		name     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			name: "create with placement fields",
			args: args(map[string]any{
				"aggregate_names": []string{firstAggr}, "constituents_per_aggregate": 1, "optimize_aggr_list": false,
				"granular_data": "disabled", "nas.path": "/fgdocs",
			}),
			ontap: []exchange{
				cluster(cdot),
				post(`"aggregates":[{"name":"harvest_vc_aggr"}],"constituents_per_aggregate":1,"nas":{"path":"/fgdocs"}`),
				jobDone,
			},
			wantText: "Volume created successfully",
		},
		{
			// Aggregate order is the caller's unless optimize_aggr_list is true.
			name:     "create preserves aggregate order",
			args:     args(map[string]any{"style": "FlexGroup", "aggregate_names": []string{secondAggr, firstAggr}}),
			ontap:    []exchange{cluster(cdot), post(`"aggregates":[{"name":"umeng_aff300_aggr1"},{"name":"harvest_vc_aggr"}]`), jobDone},
			wantText: "Volume created successfully",
		},
		{
			name:     "create with basic granular data",
			args:     args(map[string]any{"aggregate_names": bothAggrs, "granular_data": "basic"}),
			ontap:    []exchange{cluster(cdot), post(`"aggregates":[{"name":"harvest_vc_aggr"},{"name":"umeng_aff300_aggr1"}],"granular_data":true,"granular_data_mode":"basic"`), jobDone},
			wantText: "Volume created successfully",
		},
		{
			name:     "create with advanced granular data",
			args:     args(map[string]any{"aggregate_names": bothAggrs, "granular_data": " Advanced "}),
			ontap:    []exchange{cluster(cdot), post(`"aggregates":[{"name":"harvest_vc_aggr"},{"name":"umeng_aff300_aggr1"}],"granular_data":true,"granular_data_mode":"advanced"`), jobDone},
			wantText: "Volume created successfully",
		},
		{
			name: "create with constituents per aggregate and optimized aggregate list",
			args: args(map[string]any{"aggregate_names": bothAggrs, "constituents_per_aggregate": 2, "optimize_aggr_list": true}),
			ontap: []exchange{
				cluster(cdot),
				post(`"aggregates":[{"name":"harvest_vc_aggr"},{"name":"umeng_aff300_aggr1"}],"constituents_per_aggregate":2,"optimize_aggregates":true`),
				jobDone,
			},
			wantText: "Volume created successfully",
		},
		{
			// ONTAP 9.9.1 has no granular_data; the integration suite expects this rejection.
			name: "create surfaces ONTAP unexpected argument",
			args: args(map[string]any{"aggregate_names": bothAggrs, "granular_data": "basic"}),
			ontap: []exchange{
				cluster(cdot),
				{method: http.MethodPost, path: "/api/storage/volumes", status: http.StatusBadRequest,
					body: `{"svm":{"name":"fgmarketing"},"name":"fgdocs","style":"flexgroup","size":` + size + `,
						"aggregates":[{"name":"harvest_vc_aggr"},{"name":"umeng_aff300_aggr1"}],"granular_data":true,"granular_data_mode":"basic"}`,
					resp: `{"error":{"message":"Unexpected argument \"granular_data\".","code":"262179"}}`},
			},
			wantText: `Unexpected argument "granular_data".`,
			wantErr:  true,
		},
		{
			// When the model lookup fails the create proceeds as CDOT.
			name: "create falls back to CDOT when cluster lookup fails",
			args: args(map[string]any{"aggregate_names": []string{firstAggr}}),
			ontap: []exchange{
				{method: http.MethodGet, path: "/api/cluster?fields=*", status: http.StatusForbidden, resp: `{"error":{"message":"not authorized","code":"6"}}`},
				post(`"aggregates":[{"name":"harvest_vc_aggr"}]`),
				jobDone,
			},
			wantText: "Volume created successfully",
		},
		{
			name:     "reject FlexGroup without aggregates",
			args:     args(nil),
			ontap:    []exchange{cluster(cdot)},
			wantText: "aggregate_names is required when style is flexgroup",
			wantErr:  true,
		},
		{
			name:     "reject both aggregate fields",
			args:     args(map[string]any{"aggregate_name": firstAggr, "aggregate_names": bothAggrs}),
			ontap:    []exchange{cluster(cdot)},
			wantText: "cannot set both aggregate_name and aggregate_names",
			wantErr:  true,
		},
		{
			name:     "reject duplicate aggregates",
			args:     args(map[string]any{"aggregate_names": []string{firstAggr, " " + firstAggr}}),
			ontap:    []exchange{cluster(cdot)},
			wantText: `aggregate_names contains duplicate aggregate "harvest_vc_aggr"`,
			wantErr:  true,
		},
		{
			name:     "reject empty aggregate entry",
			args:     args(map[string]any{"aggregate_names": []string{firstAggr, " "}}),
			ontap:    []exchange{cluster(cdot)},
			wantText: "aggregate_names entries must not be empty",
			wantErr:  true,
		},
		{
			name:     "reject FlexGroup constituent style",
			args:     args(map[string]any{"style": "flexgroup_constituent", "aggregate_names": bothAggrs}),
			ontap:    []exchange{cluster(cdot)},
			wantText: "style flexgroup_constituent is not supported when creating a volume",
			wantErr:  true,
		},
		{
			name:     "reject unknown style",
			args:     args(map[string]any{"style": "flexcache", "aggregate_names": bothAggrs}),
			ontap:    []exchange{cluster(cdot)},
			wantText: `unsupported volume style "flexcache"; use flexvol or flexgroup`,
			wantErr:  true,
		},
		{
			name:     "reject invalid granular data mode",
			args:     args(map[string]any{"aggregate_names": bothAggrs, "granular_data": "weird"}),
			ontap:    []exchange{cluster(cdot)},
			wantText: `unsupported granular_data "weird"; use disabled, basic, or advanced`,
			wantErr:  true,
		},
		{
			name:     "reject placement fields on a FlexVol",
			args:     args(map[string]any{"style": "flexvol", "aggregate_name": firstAggr, "constituents_per_aggregate": 2}),
			ontap:    []exchange{cluster(cdot)},
			wantText: "aggregate_names, constituents_per_aggregate, optimize_aggr_list, and granular_data are only valid when style is flexgroup",
			wantErr:  true,
		},
		{
			name:     "reject FlexGroup on AFX",
			args:     args(map[string]any{"aggregate_names": bothAggrs}),
			ontap:    []exchange{cluster(afx)},
			wantText: "AFX volume placement omits aggregates",
			wantErr:  true,
		},
		{
			name:     "reject aggregate name on AFX",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "fgmarketing", "volume_name": "fgdocs", "aggregate_name": firstAggr},
			ontap:    []exchange{cluster(afx)},
			wantText: "aggregate name must not be provided for AFX clusters",
			wantErr:  true,
		},
		{
			name:     "reject space guarantee on AFX",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "fgmarketing", "volume_name": "fgdocs", "guarantee.type": "volume"},
			ontap:    []exchange{cluster(afx)},
			wantText: "space guarantee type must not be provided for AFX clusters",
			wantErr:  true,
		},
		{
			name:     "reject any volume on ASAr2",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "fgmarketing", "volume_name": "fgdocs", "aggregate_name": firstAggr},
			ontap:    []exchange{cluster(asar2)},
			wantText: "volume creation is not supported on ASAr2 clusters, use storage units instead",
			wantErr:  true,
		},
		{
			name:     "reject FlexGroup on SAN-optimized cluster",
			args:     args(map[string]any{"aggregate_names": bothAggrs}),
			ontap:    []exchange{cluster(asa)},
			wantText: "FlexGroup is not supported on All SAN Arrays",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, "create_volume", tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("got: %q; want to contain: %q", text, tt.wantText)
			}
		})
	}
}

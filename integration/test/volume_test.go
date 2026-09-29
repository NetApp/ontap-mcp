package main

import (
	"context"
	"crypto/tls"
	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/ontap"
	"log/slog"
	"maps"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestVolume(t *testing.T) {
	SkipIfMissing(t, CheckTools)
	tests := []struct {
		name             string
		input            string
		afxInput         string
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create SVM",
			input:            ClusterStr + "create " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docsnew") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Create volume",
			input:            ClusterStr + "create a 20MB volume named " + rn("docs") + " on the " + rn("marketing") + " svm and use harvest_vc_aggr aggregate if required",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "resize the " + rn("docs") + " volume on the " + rn("marketing") + " svm to 25MB",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume size",
			input:            ClusterStr + "update junction path of the " + rn("docs") + " volume on the " + rn("marketing") + " svm to empty",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Enable volume autogrowth",
			input:            ClusterStr + "enable autogrowth and grow percent to 62 on the " + rn("docs") + " volume in the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Rename volume",
			input:            ClusterStr + "rename the " + rn("docs") + " volume on the " + rn("marketing") + " svm to " + rn("docsnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm to offline",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume state",
			input:            ClusterStr + "update state of the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm to online",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume junction path",
			input:            ClusterStr + "update junction path of the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm to /" + rn("docsnew"),
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume files maximum",
			input:            ClusterStr + "increase maximum number of files to 2000 on the " + rn("docsnew") + " volume on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			// Maximum number of files would not be increase exactly same as requested, the discrepancy happens due to how ONTAP calculates and allocates internal file structures (inodes).
			verifyAPI: ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing") + "&fields=files.maximum", validationFunc: verifyFilesMax(2000)},
		},
		{
			name:             "Create thick-provisioned volume",
			input:            ClusterStr + "create a 50MB thick-provisioned volume named " + rn("thick") + " on the " + rn("marketing") + " svm and use harvest_vc_aggr aggregate if required with space guarantee type volume and snapshot reserve 5 percent",
			afxInput:         ClusterStr + "create a 50MB volume named " + rn("thick") + " on the " + rn("marketing") + " svm and use harvest_vc_aggr aggregate if required with snapshot reserve 5 percent",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("thick") + "&svm=" + rn("marketing"), validationFunc: createObject},
		},
		{
			name:             "Update volume snapshot policy",
			input:            ClusterStr + "set the snapshot policy of the " + rn("thick") + " volume on the " + rn("marketing") + " svm to none",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Update volume efficiency",
			input:            ClusterStr + "disable compression and deduplication on the " + rn("thick") + " volume on the " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{},
		},
		{
			name:             "Clean thick volume",
			input:            ClusterStr + "delete volume " + rn("thick") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("thick") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docs") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docs") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean volume",
			input:            ClusterStr + "delete volume " + rn("docsnew") + " in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/volumes?name=" + rn("docsnew") + "&svm=" + rn("marketing"), validationFunc: deleteObject},
		},
		{
			name:             "Clean SVM",
			input:            ClusterStr + "delete " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + rn("marketing"), validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}

	poller := cfg.Pollers[Cluster]
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: poller.InsecureTLS(), // #nosec G402
		},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	model := fetchModel("api/cluster?fields=san_optimized,disaggregated", poller, client)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if model == ontap.AFX && tt.afxInput != "" {
				tt.input = tt.afxInput
			}
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if _, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
		})
	}
}

func TestFlexGroupVolume(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	const (
		firstAggregate  = "harvest_vc_aggr"
		secondAggregate = "umeng_aff300_aggr1"
	)
	flexGroupName := rn("fgdocs")
	svmName := rn("fgmarketing")
	flexGroupAPI := "api/storage/volumes?name=" + flexGroupName + "&svm.name=" + svmName + "&fields=style,aggregates.name,constituents.name,nas.path"

	tests := []struct {
		name             string
		input            string
		toolArgs         map[string]any
		expectedOntapErr string
		verifyAPI        ontapVerifier
	}{
		{
			name:             "Clean FlexGroup SVM",
			input:            ClusterStr + "delete " + svmName + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/svm/svms?name=" + svmName, validationFunc: deleteObject},
		},
		{
			name:      "Create FlexGroup SVM",
			input:     ClusterStr + "create " + svmName + " svm",
			verifyAPI: ontapVerifier{api: "api/svm/svms?name=" + svmName, validationFunc: createObject},
		},
		{
			name:             "Reject FlexGroup without aggregates",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, nil),
			expectedOntapErr: "aggregate_names is required",
		},
		{
			name:             "Reject FlexGroup with both aggregate fields",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"aggregate_name": firstAggregate, "aggregate_names": []string{firstAggregate, secondAggregate}}),
			expectedOntapErr: "cannot set both aggregate_name and aggregate_names",
		},
		{
			name:             "Reject FlexGroup with duplicate aggregates",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"aggregate_names": []string{firstAggregate, firstAggregate}}),
			expectedOntapErr: "contains duplicate aggregate",
		},
		{
			name:             "Reject FlexGroup constituent style",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"style": "flexgroup_constituent", "aggregate_names": []string{firstAggregate, secondAggregate}}),
			expectedOntapErr: "flexgroup_constituent is not supported",
		},
		{
			name:             "Reject invalid granular data mode",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"aggregate_names": []string{firstAggregate, secondAggregate}, "granular_data": "weird"}),
			expectedOntapErr: "unsupported granular_data",
		},
		{
			name:             "Reject basic granular data on ONTAP 9.9.1",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"aggregate_names": []string{firstAggregate, secondAggregate}, "granular_data": "basic"}),
			expectedOntapErr: "Unexpected argument",
		},
		{
			name:             "Reject advanced granular data on ONTAP 9.9.1",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"aggregate_names": []string{firstAggregate, secondAggregate}, "granular_data": "advanced"}),
			expectedOntapErr: "Unexpected argument",
		},
		{
			name:             "Reject optimized aggregates on ONTAP 9.9.1",
			toolArgs:         flexGroupCreateArgs(svmName, flexGroupName, map[string]any{"aggregate_names": []string{firstAggregate, secondAggregate}, "constituents_per_aggregate": 2, "optimize_aggr_list": true}),
			expectedOntapErr: "Unexpected argument",
		},
		{
			name:  "Create FlexGroup with placement fields",
			input: ClusterStr + "create a 100GB FlexGroup volume named " + flexGroupName + " on the " + svmName + " svm using aggregate_names " + firstAggregate + " with 1 constituent per aggregate, optimize_aggr_list false, granular_data disabled, and junction path /" + flexGroupName,
			verifyAPI: ontapVerifier{
				api:            flexGroupAPI,
				validationFunc: verifyFlexGroup([]string{firstAggregate}, 1, "/"+flexGroupName),
			},
		},
		{
			name:      "Clear FlexGroup volume",
			input:     ClusterStr + "delete volume " + flexGroupName + " in " + svmName + " svm",
			verifyAPI: ontapVerifier{api: "api/storage/volumes?name=" + flexGroupName + "&svm.name=" + svmName, validationFunc: deleteObject},
		},
		{
			name:      "Clean FlexGroup SVM",
			input:     ClusterStr + "delete " + svmName + " svm",
			verifyAPI: ontapVerifier{api: "api/svm/svms?name=" + svmName, validationFunc: deleteObject},
		},
	}

	cfg, err := config.ReadConfig(ConfigFile)
	if err != nil {
		t.Fatalf("Error parsing the config: %v", err)
	}
	poller := cfg.Pollers[Cluster]
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: poller.InsecureTLS()}} // #nosec G402
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.toolArgs != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()
				_, err := testAgent.callMCPTool(ctx, "create_volume", tt.toolArgs)
				if err == nil || !strings.Contains(err.Error(), tt.expectedOntapErr) {
					t.Fatalf("create_volume error = %v, want error containing %q", err, tt.expectedOntapErr)
				}
				return
			}
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			if _, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr); err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
		})
	}
}

func flexGroupCreateArgs(svmName, volumeName string, extra map[string]any) map[string]any {
	args := map[string]any{
		"cluster_name": Cluster,
		"svm_name":     svmName,
		"volume_name":  volumeName,
		"style":        "flexgroup",
		"size":         "100GB",
	}
	maps.Copy(args, extra)
	return args
}

func fetchModel(api string, poller *config.Poller, client *http.Client) string {
	type response struct {
		Name          string `json:"name"`
		SanOptimized  bool   `json:"san_optimized"`
		Disaggregated bool   `json:"disaggregated"`
	}

	var data response
	err := requests.URL("https://"+poller.Addr+"/"+api).
		BasicAuth(poller.Username, poller.Password).
		Client(client).
		ToJSON(&data).
		Fetch(context.Background())
	if err != nil {
		slog.Warn("fetchModel: request failed", slog.String("err", err.Error()))
		return ontap.CDOT
	}

	switch {
	case data.Disaggregated && data.SanOptimized:
		return ontap.ASAr2
	case data.Disaggregated && !data.SanOptimized:
		return ontap.AFX
	default:
		return ontap.CDOT
	}
}

func verifyFilesMax(expectedFilesMax int) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		possibleVariation := 10
		type Files struct {
			Maximum *int `json:"maximum,omitzero"`
		}
		type Volume struct {
			Files Files `json:"files,omitzero"`
		}
		type response struct {
			NumRecords int      `json:"num_records"`
			Records    []Volume `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifyFilesMax: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyFilesMax: expected 1 record, got %d", data.NumRecords)
			return false
		}

		gotVolume := data.Records[0]
		if gotVolume.Files.Maximum == nil {
			t.Errorf("verifyFilesMax: nil files.maximum found")
			return false
		}

		if v := *gotVolume.Files.Maximum; v < (expectedFilesMax-possibleVariation) || v >= expectedFilesMax {
			t.Errorf("verifyFilesMax: files.maximum value is not in range %d - %d, got %d", expectedFilesMax-possibleVariation, expectedFilesMax, v)
			return false
		}
		return true
	}
}

func verifyFlexGroup(expectedAggregates []string, expectedConstituents int, expectedJunctionPath string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type volume struct {
			Style        string              `json:"style"`
			Aggregates   []ontap.NameAndUUID `json:"aggregates"`
			Constituents []ontap.NameAndUUID `json:"constituents"`
			NAS          struct {
				Path string `json:"path"`
			} `json:"nas"`
		}
		type response struct {
			NumRecords int      `json:"num_records"`
			Records    []volume `json:"records"`
		}

		var data response
		if err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background()); err != nil {
			t.Errorf("verifyFlexGroup: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifyFlexGroup: expected 1 record, got %d", data.NumRecords)
			return false
		}

		got := data.Records[0]
		if got.Style != "flexgroup" {
			t.Errorf("verifyFlexGroup: style = %q, want flexgroup", got.Style)
			return false
		}
		expectedTotalConstituents := len(expectedAggregates) * expectedConstituents
		if len(got.Constituents) != expectedTotalConstituents {
			t.Errorf("verifyFlexGroup: constituent count = %d, want %d", len(got.Constituents), expectedTotalConstituents)
			return false
		}
		if expectedJunctionPath != "" && got.NAS.Path != expectedJunctionPath {
			t.Errorf("verifyFlexGroup: nas.path = %q, want %q", got.NAS.Path, expectedJunctionPath)
			return false
		}

		gotAggregates := make(map[string]bool, len(got.Aggregates))
		for _, aggregate := range got.Aggregates {
			gotAggregates[aggregate.Name] = true
		}
		for _, aggregate := range expectedAggregates {
			if !gotAggregates[aggregate] {
				t.Errorf("verifyFlexGroup: aggregates = %v, missing %q", got.Aggregates, aggregate)
				return false
			}
		}
		if len(gotAggregates) != len(expectedAggregates) {
			t.Errorf("verifyFlexGroup: aggregates = %v, want %v", got.Aggregates, expectedAggregates)
			return false
		}
		return true
	}
}

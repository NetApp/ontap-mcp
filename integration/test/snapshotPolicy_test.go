package main

import (
	"context"
	"crypto/tls"
	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/ontap"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestSnapshot(t *testing.T) {
	SkipIfMissing(t, CheckTools)

	tests := []struct {
		name             string
		input            string
		expectedOntapErr string
		verifyAPI        ontapVerifier
		mustContain      []string
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
			name:             "Clean snapshot policy every4hours",
			input:            ClusterStr + "delete " + rn("every4hours") + " snapshot policy in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours"), validationFunc: deleteObject},
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            ClusterStr + "Delete " + rn("every5min") + " snapshot policy in " + rn("marketing") + " svm",
			expectedOntapErr: "because it does not exist",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every5min"), validationFunc: deleteObject},
		},
		{
			name:             "Create snapshot policy every4hours",
			input:            ClusterStr + "create a snapshot policy named " + rn("every4hours") + " on the " + rn("marketing") + " SVM. The schedule is 4hours and keeps the last 5 snapshots",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours"), validationFunc: createObject},
		},
		{
			name:             "Create snapshot policy every5min",
			input:            ClusterStr + "create a snapshot policy named " + rn("every5min") + " on the " + rn("marketing") + " SVM. The schedule is 5minutes and keeps the last 2 snapshots",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every5min"), validationFunc: createObject},
		},
		{
			name:             "Add schedule to snapshot policy every4hours",
			input:            ClusterStr + "add schedule 2hours in a snapshot policy named " + rn("every4hours") + " on the " + rn("marketing") + " SVM and keeps the last 6 snapshots",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours") + "&svm.name=" + rn("marketing") + "&fields=copies", validationFunc: verifySchedule(true, "2hours", "-", 6)},
		},
		{
			name:             "Update schedule in snapshot policy every4hours",
			input:            ClusterStr + "update schedule named 2hours in a snapshot policy named " + rn("every4hours") + " on the " + rn("marketing") + " SVM with snapmirror label as `sm2`",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours") + "&svm.name=" + rn("marketing") + "&fields=copies", validationFunc: verifySchedule(true, "2hours", "sm2", 6)},
		},
		{
			name:             "Remove schedule from snapshot policy every4hours",
			input:            ClusterStr + "remove schedule named 2hours from a snapshot policy named " + rn("every4hours") + " on the " + rn("marketing") + " SVM",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours") + "&svm.name=" + rn("marketing") + "&fields=copies", validationFunc: verifySchedule(false, "2hours", "", 0)},
		},
		{
			name:             "Update snapshot policy every4hours",
			input:            ClusterStr + "disable a snapshot policy named " + rn("every4hours") + " on the " + rn("marketing") + " SVM with comment as `4_hour_policy`",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours") + "&svm.name=" + rn("marketing") + "&fields=enabled,comment", validationFunc: verifySnapshotPolicy(false, "4_hour_policy")},
		},
		{
			name:             "Clean snapshot policy every4hours",
			input:            ClusterStr + "delete " + rn("every4hours") + " snapshot policy in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every4hours"), validationFunc: deleteObject},
		},
		{
			name:             "Clean snapshot policy every5min",
			input:            ClusterStr + "Delete " + rn("every5min") + " snapshot policy in " + rn("marketing") + " svm",
			expectedOntapErr: "",
			verifyAPI:        ontapVerifier{api: "api/storage/snapshot-policies?name=" + rn("every5min"), validationFunc: deleteObject},
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
			InsecureSkipVerify: poller.UseInsecureTLS, // #nosec G402
		},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slog.Debug("", slog.String("Input", tt.input))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			response, err := testAgent.ChatWithResponse(ctx, t, tt.input, tt.expectedOntapErr)
			if err != nil {
				t.Fatalf("Error processing input %q: %v", tt.input, err)
			}
			if tt.verifyAPI.api != "" && !tt.verifyAPI.validationFunc(t, tt.verifyAPI.api, poller, client) {
				t.Errorf("Error while accessing the object via prompt %q", tt.input)
			}
			lower := strings.ToLower(response)
			for _, want := range tt.mustContain {
				if !strings.Contains(lower, strings.ToLower(want)) {
					t.Errorf("response missing expected text %q \nfull response: %s", want, response)
				}
			}
		})
	}
}

func verifySchedule(exist bool, scheduleName string, expectedSMLabel string, expectedCount int) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type Copy struct {
			Count           int               `json:"count,omitzero" jsonschema:"number of snapshots to keep for this schedule"`
			Schedule        ontap.NameAndUUID `json:"schedule,omitzero" jsonschema:"name of the schedule"`
			SnapmirrorLabel string            `json:"snapmirror_label,omitzero" jsonschema:"SnapMirror label for this schedule"`
		}
		type SnapshotPolicy struct {
			Copies []Copy `json:"copies,omitzero" jsonschema:"snapshot copies"`
		}
		type response struct {
			NumRecords int              `json:"num_records"`
			Records    []SnapshotPolicy `json:"records"`
		}

		var data response
		var scheduleFound bool
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifySchedule: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifySchedule: expected 1 record, got %d", data.NumRecords)
			return false
		}

		for _, ssCopy := range data.Records[0].Copies {
			if ssCopy.Schedule.Name != scheduleName {
				continue
			}
			if !exist {
				t.Errorf("verifySchedule: schedule should not be exist")
				return false
			}
			scheduleFound = true

			if expectedSMLabel != ssCopy.SnapmirrorLabel {
				t.Errorf("verifySchedule: got = %s, want %s", ssCopy.SnapmirrorLabel, expectedSMLabel)
				return false
			}
			if expectedCount != ssCopy.Count {
				t.Errorf("verifySchedule: got = %d, want %d", ssCopy.Count, expectedCount)
				return false
			}
		}

		if !scheduleFound && exist {
			t.Errorf("verifySchedule: schedule must be exist")
			return false
		}

		return true
	}
}

func verifySnapshotPolicy(expectedState bool, expectedComment string) func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
	return func(t *testing.T, api string, poller *config.Poller, client *http.Client) bool {
		type SnapshotPolicy struct {
			Enabled bool   `json:"enabled"`
			Comment string `json:"comment"`
		}
		type response struct {
			NumRecords int              `json:"num_records"`
			Records    []SnapshotPolicy `json:"records"`
		}

		var data response
		err := requests.URL("https://"+poller.Addr+"/"+api).
			BasicAuth(poller.Username, poller.Password).
			Client(client).
			ToJSON(&data).
			Fetch(context.Background())
		if err != nil {
			t.Errorf("verifySnapshotPolicy: request failed: %v", err)
			return false
		}
		if data.NumRecords != 1 {
			t.Errorf("verifySnapshotPolicy: expected 1 record, got %d", data.NumRecords)
			return false
		}

		gotSPolicy := data.Records[0]
		if expectedState != gotSPolicy.Enabled {
			t.Errorf("verifySnapshotPolicy: got = %v, want %v", gotSPolicy.Enabled, expectedState)
			return false
		}
		if expectedComment != gotSPolicy.Comment {
			t.Errorf("verifySnapshotPolicy: got = %s, want %s", gotSPolicy.Comment, expectedComment)
			return false
		}

		return true
	}
}

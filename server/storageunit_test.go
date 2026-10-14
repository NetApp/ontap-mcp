package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateStorageUnit(t *testing.T) {
	tests := []struct {
		name        string
		storageUnit string
		svm         string
		remote      ontap.Remote
		cloneSource string
		snapshot    string
		expectedErr string
	}{
		{
			name:        "storage unit in cdot",
			svm:         "svm1",
			storageUnit: "sunit1",
			remote:      ontap.Remote{Model: ontap.CDOT},
			cloneSource: "source1",
			expectedErr: "storage unit creation is only supported on ASAr2 clusters",
		},
		{
			name:        "storage unit in asar2",
			svm:         "svm2",
			storageUnit: "sunit2",
			remote:      ontap.Remote{Model: ontap.ASAr2},
			cloneSource: "source2",
			snapshot:    "snapshot2",
			expectedErr: "",
		},
		{
			name:        "storage unit in afx",
			svm:         "svm4",
			storageUnit: "sunit4",
			remote:      ontap.Remote{Model: ontap.AFX},
			cloneSource: "source4",
			expectedErr: "storage unit creation is only supported on ASAr2 clusters",
		},
		{
			name:        "storage unit clone source is required",
			svm:         "svm5",
			storageUnit: "sunit5",
			remote:      ontap.Remote{Model: ontap.ASAr2},
			expectedErr: "source storage unit name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newCreateStorageUnit(tool.StorageUnitCreate{
				SVM: tt.svm, Name: tt.storageUnit,
				CloneSource: tt.cloneSource, CloneSnapshot: tt.snapshot,
			}, tt.remote)
			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Clone.Source.StorageUnit.Name != tt.cloneSource {
					t.Errorf("expected clone source %q, got %q", tt.cloneSource, got.Clone.Source.StorageUnit.Name)
				}
				if got.Clone.Source.Snapshot.Name != tt.snapshot {
					t.Errorf("expected clone snapshot %q, got %q", tt.snapshot, got.Clone.Source.Snapshot.Name)
				}
				if tt.snapshot != "" {
					payload, err := json.Marshal(got)
					if err != nil {
						t.Fatalf("failed to marshal clone request: %v", err)
					}
					expectedClone := `"clone":{"source":{"storage_unit":{"name":"` + tt.cloneSource + `"},"snapshot":{"name":"` + tt.snapshot + `"}}}`
					if !strings.Contains(string(payload), expectedClone) {
						t.Errorf("expected clone request payload to contain %s, got %s", expectedClone, payload)
					}
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

func TestStorageUnitNameRejectsInvalidCharacters(t *testing.T) {
	for _, character := range []string{"*", "|", "!", "<", ">", "{", "}"} {
		t.Run(character, func(t *testing.T) {
			err := validateStorageUnitName("storage"+character+"unit", "storage unit name")
			if err == nil {
				t.Fatalf("expected name containing %q to be rejected", character)
			}
			if !strings.Contains(err.Error(), "storage unit name contains an invalid character") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	if err := validateStorageUnitName("storage-unit_1", "storage unit name"); err != nil {
		t.Errorf("unexpected error for valid name: %v", err)
	}
}

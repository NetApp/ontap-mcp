package server

import (
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
		size        string
		osType      string
		expectedErr string
	}{
		{
			name:        "storage unit in cdot",
			svm:         "svm1",
			storageUnit: "sunit1",
			remote:      ontap.Remote{Model: ontap.CDOT},
			size:        "100mb",
			osType:      "linux",
			expectedErr: "storage unit creation is only supported on ASAr2 clusters",
		},
		{
			name:        "storage unit in asar2",
			svm:         "svm2",
			storageUnit: "sunit2",
			remote:      ontap.Remote{Model: ontap.ASAr2},
			size:        "100mb",
			osType:      "windows",
			expectedErr: "",
		},
		{
			name:        "storage unit with error in asar2",
			svm:         "svm3",
			storageUnit: "sunit3",
			remote:      ontap.Remote{Model: ontap.ASAr2},
			size:        "",
			osType:      "vmware",
			expectedErr: "invalid storage unit size: size is empty",
		},
		{
			name:        "storage unit in afx",
			svm:         "svm4",
			storageUnit: "sunit4",
			remote:      ontap.Remote{Model: ontap.AFX},
			size:        "200mb",
			osType:      "",
			expectedErr: "storage unit creation is only supported on ASAr2 clusters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newCreateStorageUnit(tool.StorageUnitCreate{SVM: tt.svm, Name: tt.storageUnit, Size: tt.size, OsType: tt.osType}, tt.remote)
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

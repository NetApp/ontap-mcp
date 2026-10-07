package server

import (
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateStorageUnit(t *testing.T) {
	tests := []struct {
		name            string
		storageUnit     string
		svm             string
		remote          ontap.Remote
		size            string
		osType          string
		expectedErr     string
		expectedVolJSON ontap.StorageUnit
	}{
		{
			name:            "storage unit in cdot",
			svm:             "svm1",
			storageUnit:     "sunit1",
			remote:          ontap.Remote{Model: ontap.CDOT},
			size:            "100mb",
			osType:          "linux",
			expectedErr:     "storage unit creation is only supported on ASAr2 clusters",
			expectedVolJSON: ontap.StorageUnit{},
		},
		{
			name:            "storage unit in asar2",
			svm:             "svm2",
			storageUnit:     "sunit2",
			remote:          ontap.Remote{Model: ontap.ASAr2},
			size:            "100mb",
			osType:          "windows",
			expectedErr:     "",
			expectedVolJSON: ontap.StorageUnit{SVM: ontap.NameAndUUID{Name: "svm2"}, Name: "sunit2", Space: ontap.StorageUnitSpace{Size: 104857600}},
		},
		{
			name:            "storage unit with error in asar2",
			svm:             "svm3",
			storageUnit:     "sunit3",
			remote:          ontap.Remote{Model: ontap.ASAr2},
			size:            "",
			osType:          "vmware",
			expectedErr:     "invalid storage unit size: size is empty",
			expectedVolJSON: ontap.StorageUnit{},
		},
		{
			name:            "storage unit in afx",
			svm:             "svm4",
			storageUnit:     "sunit4",
			remote:          ontap.Remote{Model: ontap.AFX},
			size:            "200mb",
			osType:          "",
			expectedErr:     "storage unit creation is only supported on ASAr2 clusters",
			expectedVolJSON: ontap.StorageUnit{},
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

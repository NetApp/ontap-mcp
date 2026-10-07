package server

import (
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateLun(t *testing.T) {
	tests := []struct {
		name            string
		volume          string
		svm             string
		lun             string
		remote          ontap.Remote
		size            string
		osType          string
		expectedErr     string
		expectedVolJSON ontap.LUN
	}{
		{
			name:            "Normal lun in cdot",
			volume:          "volume1",
			svm:             "svm1",
			lun:             "lun1",
			remote:          ontap.Remote{Model: ontap.CDOT},
			size:            "100mb",
			osType:          "linux",
			expectedErr:     "",
			expectedVolJSON: ontap.LUN{SVM: ontap.NameAndUUID{Name: "svm1"}, Name: "lun1", Space: ontap.LUNSpace{Size: 104857600}},
		},
		{
			name:            "lun with error in cdot",
			volume:          "volume2",
			svm:             "svm2",
			lun:             "",
			remote:          ontap.Remote{Model: ontap.CDOT},
			size:            "100mb",
			osType:          "windows",
			expectedErr:     "LUN name is required",
			expectedVolJSON: ontap.LUN{},
		},
		{
			name:            "lun with error in asar2",
			volume:          "volume3",
			svm:             "svm3",
			lun:             "lun3",
			remote:          ontap.Remote{Model: ontap.ASAr2},
			size:            "100mb",
			osType:          "exsi",
			expectedErr:     "lun creation is not supported on ASAr2 clusters, use storage units instead",
			expectedVolJSON: ontap.LUN{},
		},
		{
			name:            "lun without model in cdot",
			volume:          "volume4",
			svm:             "svm4",
			lun:             "lun4",
			remote:          ontap.Remote{},
			size:            "100mb",
			osType:          "",
			expectedErr:     "OS type is required",
			expectedVolJSON: ontap.LUN{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newCreateLUN(tool.LUNCreate{SVM: tt.svm, Name: tt.lun, Volume: tt.volume, Size: tt.size, OsType: tt.osType}, tt.remote)
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

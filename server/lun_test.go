package server

import (
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateLun(t *testing.T) {
	tests := []struct {
		name        string
		volume      string
		svm         string
		lun         string
		remote      ontap.Remote
		size        string
		osType      string
		expectedErr string
	}{
		{
			name:        "Normal lun in cdot",
			volume:      "volume1",
			svm:         "svm1",
			lun:         "lun1",
			remote:      ontap.Remote{Model: ontap.CDOT},
			size:        "100mb",
			osType:      "linux",
			expectedErr: "",
		},
		{
			name:        "lun with error in cdot",
			volume:      "volume2",
			svm:         "svm2",
			lun:         "",
			remote:      ontap.Remote{Model: ontap.CDOT},
			size:        "100mb",
			osType:      "windows",
			expectedErr: "LUN name is required",
		},
		{
			name:        "lun creation in asar2",
			volume:      "volume3",
			svm:         "svm3",
			lun:         "lun3",
			remote:      ontap.Remote{Model: ontap.ASAr2},
			size:        "100mb",
			osType:      "esxi",
			expectedErr: "",
		},
		{
			name:        "lun without model in cdot",
			volume:      "volume4",
			svm:         "svm4",
			lun:         "lun4",
			remote:      ontap.Remote{},
			size:        "100mb",
			osType:      "",
			expectedErr: "OS type is required",
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

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "lun_1.v2-prod"},
		{name: ""},
		{name: "name with spaces", wantErr: true},
		{name: "name/with/slashes", wantErr: true},
		{name: "name..with-dots", wantErr: true},
		{name: "name.with.dots"},
		{name: "name+with+plus", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName(%q) error = %v, wantErr %t", tt.name, err, tt.wantErr)
			}
		})
	}
}

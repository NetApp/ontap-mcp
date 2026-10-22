package server

import (
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateLun(t *testing.T) {
	tests := []struct {
		name         string
		volume       string
		svm          string
		lun          string
		remote       ontap.Remote
		size         string
		osType       string
		expectedErr  string
		expectedName string
	}{
		{
			name:         "Normal lun in cdot",
			volume:       "volume1",
			svm:          "svm1",
			lun:          "lun1",
			remote:       ontap.Remote{Model: ontap.CDOT},
			size:         "100mb",
			osType:       "linux",
			expectedName: "/vol/volume1/lun1",
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
			name:         "lun creation in asar2",
			volume:       "volume3",
			svm:          "svm3",
			lun:          "lun3",
			remote:       ontap.Remote{Model: ontap.ASAr2},
			size:         "100mb",
			osType:       "esxi",
			expectedName: "lun3",
		},
		{
			name:         "lun creation without volume in asar2",
			svm:          "svm3",
			lun:          "lun3",
			remote:       ontap.Remote{Model: ontap.ASAr2},
			size:         "100mb",
			osType:       "esxi",
			expectedName: "lun3",
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
			out, err := newCreateLUN(tool.LUNCreate{SVM: tt.svm, Name: tt.lun, Volume: tt.volume, Size: tt.size, OsType: tt.osType}, tt.remote)
			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if out.Name != tt.expectedName {
					t.Errorf("expected LUN name %q, got %q", tt.expectedName, out.Name)
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

func TestNewUpdateLUNRenamePath(t *testing.T) {
	tests := []struct {
		name         string
		volume       string
		remote       ontap.Remote
		expectedName string
	}{
		{
			name:         "cDOT rename uses volume path",
			volume:       "volume1",
			remote:       ontap.Remote{Model: ontap.CDOT},
			expectedName: "/vol/volume1/lun2",
		},
		{
			name:         "ASAr2 rename uses bare name without volume",
			remote:       ontap.Remote{Model: ontap.ASAr2},
			expectedName: "lun2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := newUpdateLUN(tool.LUN{SVM: "svm1", Volume: tt.volume, Name: "lun1", NewName: "lun2"}, tt.remote)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Name != tt.expectedName {
				t.Errorf("expected LUN name %q, got %q", tt.expectedName, out.Name)
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

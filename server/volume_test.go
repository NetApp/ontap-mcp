package server

import (
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

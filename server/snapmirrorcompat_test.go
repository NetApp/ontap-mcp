package server

import (
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/ontap"
)

func TestXDPCompatibilityMatrixShape(t *testing.T) {
	if len(xdpCompatibilityRows) != len(xdpReleaseOrder) {
		t.Fatalf("matrix has %d rows, want %d", len(xdpCompatibilityRows), len(xdpReleaseOrder))
	}
	for _, release := range xdpReleaseOrder {
		row, ok := xdpCompatibilityRows[release]
		if !ok {
			t.Errorf("matrix is missing source release %s", release)
			continue
		}
		row = strings.ReplaceAll(row, " ", "")
		if len(row) != len(xdpReleaseOrder) {
			t.Errorf("matrix row %s has %d cells, want %d", release, len(row), len(xdpReleaseOrder))
		}
		index := releaseIndex(release)
		if index < 0 || row[index] != '1' {
			t.Errorf("matrix diagonal for %s must be compatible", release)
		}
	}
}

func TestCheckXDPCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		source      ontap.Remote
		destination ontap.Remote
		wantKnown   bool
		wantOK      bool
		wantReason  string
	}{
		{
			name:        "supported release pair cdot",
			source:      remoteAt(9, 16, 1, ontap.CDOT),
			destination: remoteAt(9, 15, 1, ontap.CDOT),
			wantKnown:   true,
			wantOK:      true,
		},
		{
			name:        "supported release pair afx",
			source:      remoteAt(9, 16, 1, ontap.AFX),
			destination: remoteAt(9, 15, 1, ontap.AFX),
			wantKnown:   true,
			wantOK:      true,
		},
		{
			name:        "supported release pair asar2",
			source:      remoteAt(9, 16, 1, ontap.ASAr2),
			destination: remoteAt(9, 15, 1, ontap.ASAr2),
			wantKnown:   true,
			wantOK:      true,
		},
		{
			name:        "unsupported release pair",
			source:      remoteAt(9, 19, 1, ontap.CDOT),
			destination: remoteAt(9, 5, 0, ontap.CDOT),
			wantKnown:   true,
			wantReason:  "not supported",
		},
		{
			name:        "unknown future release",
			source:      remoteAt(9, 20, 1, ontap.CDOT),
			destination: remoteAt(9, 19, 1, ontap.CDOT),
			wantReason:  "not available",
		},
		{
			name:        "ASA r2 mismatch",
			source:      remoteAt(9, 16, 0, ontap.ASAr2),
			destination: remoteAt(9, 16, 1, ontap.CDOT),
			wantKnown:   true,
			wantReason:  "only with other ASA r2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := checkXDPCompatibility(test.source, test.destination)
			if got.Known != test.wantKnown || got.Compatible != test.wantOK {
				t.Errorf("result = known %t compatible %t, want known %t compatible %t", got.Known, got.Compatible, test.wantKnown, test.wantOK)
			}
			if test.wantReason != "" && !strings.Contains(got.Reason, test.wantReason) {
				t.Errorf("reason %q does not contain %q", got.Reason, test.wantReason)
			}
		})
	}
}

// nolint:unparam
func remoteAt(generation, major, minor int, model string) ontap.Remote {
	return ontap.Remote{
		Version: ontap.Version{Generation: generation, Major: major, Minor: minor},
		Model:   model,
	}
}

package server

import (
	"testing"

	"github.com/netapp/ontap-mcp/tool"
)

func TestNewCreateLUN_SpaceGuarantee(t *testing.T) {
	base := tool.LUNCreate{
		SVM:     "vs1",
		Volume:  "vol1",
		Name:    "lun1",
		Size:    "10GB",
		OsType:  "linux",
	}

	tests := []struct {
		name                    string
		spaceGuaranteeRequested bool
		wantGuaranteeNil        bool
		wantGuaranteeRequested  bool
	}{
		{
			name:                    "thin provisioning (default): Guarantee struct omitted",
			spaceGuaranteeRequested: false,
			wantGuaranteeNil:        true,
		},
		{
			name:                    "thick provisioning: Guarantee.Requested is true",
			spaceGuaranteeRequested: true,
			wantGuaranteeNil:        false,
			wantGuaranteeRequested:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			in.SpaceGuaranteeRequested = tt.spaceGuaranteeRequested

			out, err := newCreateLUN(in)
			if err != nil {
				t.Fatalf("newCreateLUN returned unexpected error: %v", err)
			}

			if tt.wantGuaranteeNil {
				if out.Space.Guarantee.Requested != nil {
					t.Errorf("expected Guarantee.Requested to be nil for thin provisioning, got %v", *out.Space.Guarantee.Requested)
				}
			} else {
				if out.Space.Guarantee.Requested == nil {
					t.Errorf("expected Guarantee.Requested to be non-nil for thick provisioning")
				} else if *out.Space.Guarantee.Requested != tt.wantGuaranteeRequested {
					t.Errorf("Guarantee.Requested: want %v, got %v", tt.wantGuaranteeRequested, *out.Space.Guarantee.Requested)
				}
			}
		})
	}
}

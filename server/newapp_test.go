package server

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/config"
)

func TestNewApp_CaseCollision(t *testing.T) {
	tests := []struct {
		name        string
		pollers     map[string]*config.Poller
		wantErr     bool
		errContains string
	}{
		{
			name: "no collision",
			pollers: map[string]*config.Poller{
				"DC1": {},
				"DC2": {},
			},
			wantErr: false,
		},
		{
			name: "identical names",
			pollers: map[string]*config.Poller{
				"dc1": {},
			},
			wantErr: false,
		},
		{
			name: "case collision upper vs lower",
			pollers: map[string]*config.Poller{
				"DC1": {},
				"dc1": {},
			},
			wantErr:     true,
			errContains: "differ only by case",
		},
		{
			name: "case collision mixed case",
			pollers: map[string]*config.Poller{
				"Cluster1": {},
				"CLUSTER1": {},
			},
			wantErr:     true,
			errContains: "differ only by case",
		},
	}

	logger := slog.Default()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ONTAP{Pollers: tt.pollers}
			app, err := NewApp(cfg, Options{}, logger)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				if app != nil {
					t.Fatal("expected nil *App on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if app == nil {
					t.Fatal("expected non-nil *App, got nil")
				}
			}
		})
	}
}

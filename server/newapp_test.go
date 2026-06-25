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

func TestNewApp_TLSValidation(t *testing.T) {
	tests := []struct {
		name        string
		tls         *config.TLS
		wantErr     bool
		errContains string
		wantCert    string
		wantKey     string
	}{
		{
			name: "no tls",
			tls:  nil,
		},
		{
			name:     "both set",
			tls:      &config.TLS{CertFile: "cert/admin-cert.pem", KeyFile: "cert/admin-key.pem"},
			wantCert: "cert/admin-cert.pem",
			wantKey:  "cert/admin-key.pem",
		},
		{
			name:     "both set with surrounding whitespace",
			tls:      &config.TLS{CertFile: "  cert/admin-cert.pem  ", KeyFile: "\tcert/admin-key.pem\n"},
			wantCert: "cert/admin-cert.pem",
			wantKey:  "cert/admin-key.pem",
		},
		{
			name:        "empty tls struct",
			tls:         &config.TLS{},
			wantErr:     true,
			errContains: "both cert_file and key_file",
		},
		{
			name:        "only cert file",
			tls:         &config.TLS{CertFile: "cert/admin-cert.pem"},
			wantErr:     true,
			errContains: "both cert_file and key_file",
		},
		{
			name:        "only key file",
			tls:         &config.TLS{KeyFile: "cert/admin-key.pem"},
			wantErr:     true,
			errContains: "both cert_file and key_file",
		},
		{
			name:        "key set but cert is whitespace only",
			tls:         &config.TLS{CertFile: "   ", KeyFile: "cert/admin-key.pem"},
			wantErr:     true,
			errContains: "both cert_file and key_file",
		},
	}

	logger := slog.Default()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ONTAP{TLS: tt.tls}
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
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if app == nil {
				t.Fatal("expected non-nil *App, got nil")
			}
			if app.CertFile != tt.wantCert {
				t.Fatalf("CertFile = %q, want %q", app.CertFile, tt.wantCert)
			}
			if app.KeyFile != tt.wantKey {
				t.Fatalf("KeyFile = %q, want %q", app.KeyFile, tt.wantKey)
			}
		})
	}
}

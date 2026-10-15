package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

// TestDNSContract mirrors integration/test/dns_test.go: clean a missing DNS configuration,
// create one with validation disabled, delete it, and create one with validation enabled that
// ONTAP rejects. The SVM create and delete steps are covered by TestSVMContract.
func TestDNSContract(t *testing.T) {
	const (
		svmFound   = `{"num_records":1,"records":[{"uuid":"dns-svm-uuid"}]}`
		svmMissing = `{"num_records":0,"records":[]}`
	)
	dnsSVMLookup := func(resp string) exchange {
		return exchange{method: http.MethodGet, path: "/api/svm/svms?fields=uuid&name=dnsSvc", resp: resp}
	}

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		{
			// The integration "Clean DNS" step expects ONTAP's "entry doesn't exist" to reach the caller.
			name: "delete of missing dns configuration surfaces ONTAP error",
			tool: "delete_dns",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc"},
			ontap: []exchange{
				dnsSVMLookup(svmFound),
				{method: http.MethodDelete, path: "/api/name-services/dns/dns-svm-uuid", status: http.StatusNotFound,
					resp: `{"error":{"message":"entry doesn't exist","code":"4"}}`},
			},
			wantText: "entry doesn't exist",
			wantErr:  true,
		},
		{
			name: "create with validation disabled",
			tool: "create_dns",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc", "domains": []string{"example.com"},
				"servers": []string{"10.10.10.10", "10.10.10.30", "10.10.10.20"}, "skip_config_validation": true},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/name-services/dns", status: http.StatusCreated,
					body: `{"svm":{"name":"dnsSvc"},"domains":["example.com"],"servers":["10.10.10.10","10.10.10.30","10.10.10.20"],"skip_config_validation":true}`},
			},
			wantText: "DNS configuration created successfully",
		},
		{
			name: "delete dns configuration",
			tool: "delete_dns",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc"},
			ontap: []exchange{
				dnsSVMLookup(svmFound),
				{method: http.MethodDelete, path: "/api/name-services/dns/dns-svm-uuid"},
			},
			wantText: "DNS configuration deleted successfully",
		},
		{
			// With validation enabled the body omits skip_config_validation, and ONTAP's
			// validation failure must reach the caller.
			name: "create with validation enabled surfaces ONTAP validation error",
			tool: "create_dns",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc", "domains": []string{"example.com"},
				"servers": []string{"10.10.10.10"}},
			ontap: []exchange{
				{method: http.MethodPost, path: "/api/name-services/dns", status: http.StatusBadRequest,
					body: `{"svm":{"name":"dnsSvc"},"domains":["example.com"],"servers":["10.10.10.10"]}`,
					resp: `{"error":{"message":"Failed to validate the DNS configuration. Verify that the network configuration is correct and that DNS servers are available.","code":"8847365"}}`},
			},
			wantText: "Verify that the network configuration is correct and that DNS servers are available.",
			wantErr:  true,
		},
		{
			name:     "create without domains is rejected without calling ONTAP",
			tool:     "create_dns",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc", "domains": []string{}, "servers": []string{"10.10.10.10"}},
			wantText: "at least one DNS domain is required",
			wantErr:  true,
		},
		{
			name:     "create without servers is rejected without calling ONTAP",
			tool:     "create_dns",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc", "domains": []string{"example.com"}, "servers": []string{}},
			wantText: "at least one DNS server is required",
			wantErr:  true,
		},
		{
			name:     "create without svm is rejected without calling ONTAP",
			tool:     "create_dns",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "", "domains": []string{"example.com"}, "servers": []string{"10.10.10.10"}},
			wantText: "SVM name is required",
			wantErr:  true,
		},
		{
			name:     "delete on missing svm",
			tool:     "delete_dns",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "dnsSvc"},
			ontap:    []exchange{dnsSVMLookup(svmMissing)},
			wantText: "failed to get details of SVM dnsSvc because it does not exist",
			wantErr:  true,
		},
		{
			name:     "delete without svm is rejected without calling ONTAP",
			tool:     "delete_dns",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			assert.Equal(t, isErr, tt.wantErr)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("tool output %q does not contain %q", text, tt.wantText)
			}
		})
	}
}

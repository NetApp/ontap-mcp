package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/assert"
	"github.com/netapp/ontap-mcp/catalog"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/tool"
)

var interfacesCatalog = catalog.APICatalog{
	"/network/ip/interfaces": {
		Filters: map[string]catalog.FilterInfo{"location.node.name": {}, "recommend.ip.family": {}},
		Fields: map[string]catalog.FieldInfo{
			"name":           {Desc: "Interface name"},
			"ip":             {},
			"location":       {Desc: "Current or home location"},
			"rdma_protocols": {Since: "9.17"},
		},
	},
}

func TestDescribeOntapEndpointFields(t *testing.T) {
	a := &App{catalog: interfacesCatalog}
	res, _, err := a.DescribeOntapEndpoint(t.Context(), nil, tool.DescribeEndpointParams{Path: "/network/ip/interfaces"})
	assert.Nil(t, err)

	text := res.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Filters (a name whose part before the first dot is listed under Fields is also valid in 'fields'):\n  location.node.name\n  recommend.ip.family\n")
	assert.Contains(t, text, "\n  ip\n")
}

func TestNormalizePath(t *testing.T) {
	path, query := normalizePath("/api/network/ip/interfaces/")
	assert.Equal(t, path, "/network/ip/interfaces")
	assert.Equal(t, query, "")

	path, query = normalizePath("/network/ip/interfaces?fields=name&state=up")
	assert.Equal(t, path, "/network/ip/interfaces")
	assert.Equal(t, query, "fields=name&state=up")

	p := tool.OntapGetParams{Filters: map[string]string{"state": "down"}}
	assert.Nil(t, applyPathQuery(&p, query+"&max_records=2"))
	assert.Equal(t, p.Fields, "name")
	assert.Equal(t, p.MaxRecords, 2)
	assert.Equal(t, p.Filters, map[string]string{"state": "down"})
}

func TestLargeResultHint(t *testing.T) {
	many := json.RawMessage(`{"num_records":23,"records":[]}`)
	few := json.RawMessage(`{"num_records":3,"records":[]}`)
	assert.Contains(t, largeResultHint("*", 0, many), "Large response: 23 full records")
	assert.Equal(t, largeResultHint("*", 0, few), "")
	assert.Equal(t, largeResultHint("*", 1, many), "")
	assert.Equal(t, largeResultHint("name,svm.name", 0, many), "")
	assert.Equal(t, largeResultHint("name,space.*", 0, many), "")
}

func TestWithPathSuggestions(t *testing.T) {
	a := &App{catalog: interfacesCatalog}
	notFound := ontap.ClusterError{StatusCode: http.StatusNotFound, Err: ontap.OErr{Message: "API not found", Code: "3"}}
	tests := []struct {
		name    string
		err     error
		suggest bool
	}{
		{"api not found", fmt.Errorf("get failed: %w", notFound), true},
		{"other code", ontap.ClusterError{StatusCode: http.StatusNotFound, Err: ontap.OErr{Code: "4"}}, false},
		{"other status", ontap.ClusterError{StatusCode: http.StatusInternalServerError, Err: ontap.OErr{Code: "3"}}, false},
		{"not a cluster error", errors.New("timeout"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.withPathSuggestions(tt.err, "/network/interfaces")
			if tt.suggest {
				assert.Contains(t, got.Error(), "Did you mean: /network/ip/interfaces")
				assert.ErrorIs(t, got, notFound)
			} else {
				assert.Equal(t, got, tt.err)
			}
		})
	}
}

func TestAvailableFieldsHint(t *testing.T) {
	a := &App{catalog: interfacesCatalog}
	assert.Contains(t, a.availableFieldsHint("/network/ip/interfaces", "9.16"), "Available fields: ip, location, name.")
	assert.Equal(t, a.availableFieldsHint("/storage/volumes", "9.16"), "")
}

func TestUnknownFields(t *testing.T) {
	a := &App{catalog: interfacesCatalog, catalogVersion: "9.19"}
	tests := []struct {
		name   string
		path   string
		fields string
		ver    string
		want   []string
	}{
		{"known fields and dot-paths", "/network/ip/interfaces", "name, ip.address,location.node.name", "9.16", nil},
		{"unknown fields", "/network/ip/interfaces", "name,address,node.name", "9.16", []string{"address", "node.name"}},
		{"query-only filter is not a field", "/network/ip/interfaces", "name,recommend", "9.16", []string{"recommend"}},
		{"wildcard skipped", "/network/ip/interfaces", "*", "9.16", nil},
		{"path not in catalog", "/storage/volumes", "bogus", "9.16", nil},
		{"field newer than cluster", "/network/ip/interfaces", "name,rdma_protocols", "9.16", []string{"rdma_protocols"}},
		{"cluster newer than catalog", "/network/ip/interfaces", "bogus", "9.20", nil},
		{"version unknown", "/network/ip/interfaces", "bogus", "", nil},
		{"before 9.11", "/network/ip/interfaces", "bogus", "9.10", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, a.unknownFields(tt.path, tt.fields, tt.ver), tt.want)
		})
	}
}

package server

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/assert"
	"github.com/netapp/ontap-mcp/catalog"
	"github.com/netapp/ontap-mcp/tool"
)

var interfacesCatalog = catalog.APICatalog{
	"/network/ip/interfaces": {
		Filters: map[string]catalog.FilterInfo{"location.node.name": {}},
		Fields: map[string]catalog.FieldInfo{
			"name":     {Desc: "Interface name"},
			"ip":       {},
			"location": {Desc: "Current or home location"},
		},
	},
}

func TestDescribeOntapEndpointFields(t *testing.T) {
	a := &App{catalog: interfacesCatalog}
	res, _, err := a.DescribeOntapEndpoint(t.Context(), nil, tool.DescribeEndpointParams{Path: "/network/ip/interfaces"})
	assert.Nil(t, err)

	text := res.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Filters (each name is also valid in 'fields'):\n  location.node.name\n")
	assert.Contains(t, text, "\n  ip\n")
}

func TestUnknownFields(t *testing.T) {
	a := &App{catalog: interfacesCatalog}
	tests := []struct {
		name   string
		path   string
		fields string
		want   []string
	}{
		{"known fields and dot-paths", "/network/ip/interfaces", "name, ip.address,location.node.name", nil},
		{"unknown fields", "/network/ip/interfaces", "name,address,node.name", []string{"address", "node.name"}},
		{"wildcard skipped", "/network/ip/interfaces", "*", nil},
		{"path not in catalog", "/storage/volumes", "bogus", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, a.unknownFields(tt.path, tt.fields), tt.want)
		})
	}
}

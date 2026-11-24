package catalog

import (
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestSuggestPaths(t *testing.T) {
	cat := APICatalog{
		"/network/ip/interfaces":              {},
		"/network/fc/interfaces":              {},
		"/network/ipspaces":                   {},
		"/storage/volumes":                    {},
		"/storage/volume-efficiency-policies": {},
	}
	tests := []struct {
		name string
		path string
		want []string
	}{
		{"missing segment", "/network/interfaces", []string{"/network/fc/interfaces", "/network/ip/interfaces", "/network/ipspaces"}},
		{"singular matches plural", "/network/ip/interface", []string{"/network/ip/interfaces", "/network/fc/interfaces", "/network/ipspaces"}},
		{"ties prefer fewer path words", "/storage/volume", []string{"/storage/volumes", "/storage/volume-efficiency-policies"}},
		{"no match", "/foo/bar", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, cat.SuggestPaths(tt.path, 5), tt.want)
		})
	}
}

func TestSearchWordFallback(t *testing.T) {
	cat := APICatalog{
		"/network/ip/interfaces":                   {},
		"/storage/volumes":                         {},
		"/storage/volumes/{volume.uuid}/snapshots": {},
	}
	tests := []struct {
		query string
		want  string
		exact bool
	}{
		{"interfaces", "/network/ip/interfaces", true},
		{"network interfaces", "/network/ip/interfaces", false},
		{"list interfaces", "/network/ip/interfaces", false},
		{"volumes snapshots", "/storage/volumes/{volume.uuid}/snapshots", false},
		{"lif", "/network/ip/interfaces", false},
		{"list all lifs", "/network/ip/interfaces", false},
		{"xyz", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			rs, exact := cat.Search(tt.query)
			first := ""
			if len(rs) > 0 {
				first = rs[0].Path
			}
			assert.Equal(t, first, tt.want)
			assert.Equal(t, exact, tt.exact)
		})
	}
}

func TestPathWordsAliases(t *testing.T) {
	assert.Equal(t, pathWords("list all LIFs on vserver"), []string{"list", "all", "interfaces", "on", "svm"})
	assert.Equal(t, pathWords("/storage/aggregates/{uuid}"), []string{"storage", "aggregates"})
}

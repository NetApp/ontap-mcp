package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	goversion "github.com/netapp/ontap-mcp/third_party/go-version"
)

type FilterInfo struct {
	Type  string `json:"type,omitempty"`
	Since string `json:"since,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

type FieldInfo struct {
	Since string `json:"since,omitempty"`
	Desc  string `json:"desc"`
}

type APIEndpoint struct {
	Summary    string                `json:"summary"`
	Tags       []string              `json:"tags"`
	Introduced string                `json:"api_introduced,omitempty"`
	Filters    map[string]FilterInfo `json:"filters,omitempty"`
	Fields     map[string]FieldInfo  `json:"fields,omitempty"`
}

// ExcludedPathPrefixes lists ONTAP REST path prefixes that are covered by dedicated typed tools.
// These paths are excluded from the generated catalog so the LLM uses the typed tools instead.
var ExcludedPathPrefixes = []string{
	"/storage/qos",
}

type APICatalog map[string]APIEndpoint

type File struct {
	ONTAPVersion string     `json:"ontap_version"`
	GeneratedAt  string     `json:"generated_at"`
	Endpoints    APICatalog `json:"endpoints"`
}

func Load(path string) (APICatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog %s: %w", path, err)
	}
	var cf File
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parse catalog %s: %w", path, err)
	}
	return cf.Endpoints, nil
}

func Save(cat APICatalog, ontapVersion string, path string) error {
	if err := os.MkdirAll(dirOf(path), 0750); err != nil {
		return err
	}
	cf := File{
		ONTAPVersion: ontapVersion,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Endpoints:    cat,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0600)
}

type SearchResult struct {
	Path     string      `json:"path"`
	Endpoint APIEndpoint `json:"endpoint"`
}

func (c APICatalog) Search(query string) []SearchResult {
	lower := strings.ToLower(query)
	seen := make(map[string]bool)
	var results []SearchResult
	for path, ep := range c {
		if seen[path] {
			continue
		}
		matched := strings.Contains(strings.ToLower(path), lower) ||
			strings.Contains(strings.ToLower(ep.Summary), lower)
		if !matched {
			for _, tag := range ep.Tags {
				if strings.Contains(strings.ToLower(tag), lower) {
					matched = true
					break
				}
			}
		}
		if matched {
			seen[path] = true
			results = append(results, SearchResult{Path: path, Endpoint: ep})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results
}

func (c APICatalog) ListAll() []SearchResult {
	results := make([]SearchResult, 0, len(c))
	for path, ep := range c {
		results = append(results, SearchResult{Path: path, Endpoint: ep})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results
}

func (ep APIEndpoint) FilterByVersion(ontapVersion string) APIEndpoint {
	out := APIEndpoint{
		Summary:    ep.Summary,
		Tags:       ep.Tags,
		Introduced: ep.Introduced,
	}
	out.Filters = make(map[string]FilterInfo, len(ep.Filters))
	for k, v := range ep.Filters {
		since := v.Since
		if since == "" {
			since = "9.6"
		}
		if compareVersions(since, ontapVersion) <= 0 {
			out.Filters[k] = v
		}
	}
	out.Fields = make(map[string]FieldInfo, len(ep.Fields))
	for k, v := range ep.Fields {
		since := v.Since
		if since == "" {
			since = "9.6"
		}
		if compareVersions(since, ontapVersion) <= 0 {
			out.Fields[k] = v
		}
	}
	return out
}

func CompareVersions(a, b string) int {
	return compareVersions(a, b)
}

func compareVersions(a, b string) int {
	va, err := goversion.NewVersion(a)
	if err != nil {
		return 0
	}
	vb, err := goversion.NewVersion(b)
	if err != nil {
		return 0
	}
	return va.Compare(vb)
}

func dirOf(path string) string {
	idx := strings.LastIndexByte(path, '/')
	if idx < 0 {
		return "."
	}
	return path[:idx]
}

package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
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
	Desc  string `json:"desc,omitempty"`
}

// PathParamInfo describes a path parameter (e.g. {uuid}) in an endpoint template.
type PathParamInfo struct {
	Desc string `json:"desc,omitempty"`
}

type APIEndpoint struct {
	Summary    string                   `json:"summary"`
	Tags       []string                 `json:"tags"`
	Introduced string                   `json:"api_introduced,omitempty"`
	PathParams map[string]PathParamInfo `json:"path_params,omitempty"`
	Filters    map[string]FilterInfo    `json:"filters,omitempty"`
	Fields     map[string]FieldInfo     `json:"fields,omitempty"`
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

// Load returns the catalog endpoints and the ONTAP version the catalog was generated from.
func Load(path string) (APICatalog, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read catalog %s: %w", path, err)
	}
	var cf File
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, "", fmt.Errorf("parse catalog %s: %w", path, err)
	}
	return cf.Endpoints, cf.ONTAPVersion, nil
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

// Search returns endpoints whose path, summary or tag contains query. If none do, it falls back to
// the closest paths by word match and reports exact=false.
func (c APICatalog) Search(query string) ([]SearchResult, bool) {
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
	if len(results) == 0 {
		for _, p := range c.SuggestPaths(query, 10) {
			results = append(results, SearchResult{Path: p, Endpoint: c[p]})
		}
		return results, false
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, true
}

// SuggestPaths returns up to limit catalog paths that share the most words with path.
func (c APICatalog) SuggestPaths(path string, limit int) []string {
	want := pathWords(path)
	type hit struct {
		path  string
		score int
		words int
	}
	var hits []hit
	for p := range c {
		have := pathWords(p)
		score := 0
		for _, w := range want {
			if slices.ContainsFunc(have, func(h string) bool { return wordsMatch(w, h) }) {
				score++
			}
		}
		if score > 0 {
			hits = append(hits, hit{p, score, len(have)})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		if hits[i].words != hits[j].words {
			return hits[i].words < hits[j].words
		}
		return hits[i].path < hits[j].path
	})
	out := make([]string, 0, min(limit, len(hits)))
	for _, h := range hits[:min(limit, len(hits))] {
		out = append(out, h.path)
	}
	return out
}

// wordAliases maps ONTAP CLI terms to the words used in REST paths, so a search for "lif" finds interfaces.
var wordAliases = map[string]string{
	"lif":      "interfaces",
	"lifs":     "interfaces",
	"vserver":  "svm",
	"vservers": "svm",
	"aggr":     "aggregates",
	"aggrs":    "aggregates",
}

// pathWords splits a REST path or search query into lowercase words, skipping {placeholders}
// and mapping CLI aliases (see wordAliases) to their REST equivalents.
func pathWords(path string) []string {
	var words []string
	for w := range strings.FieldsFuncSeq(strings.ToLower(path), func(r rune) bool { return r == '/' || r == '-' || r == '_' || r == ' ' }) {
		if strings.HasPrefix(w, "{") {
			continue
		}
		if alias, ok := wordAliases[w]; ok {
			w = alias
		}
		words = append(words, w)
	}
	return words
}

// wordsMatch treats singular and plural forms as the same word. Short words must match exactly.
func wordsMatch(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) < 4 || len(b) < 4 {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
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
		PathParams: ep.PathParams,
		Filters:    make(map[string]FilterInfo, len(ep.Filters)),
		Fields:     make(map[string]FieldInfo, len(ep.Fields)),
	}
	for k, v := range ep.Filters {
		since := v.Since
		if since == "" {
			since = "9.6"
		}
		if compareVersions(since, ontapVersion) <= 0 {
			out.Filters[k] = v
		}
	}
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
	before, _, found := strings.CutLast(path, "/")
	if !found {
		return "."
	}
	return before
}

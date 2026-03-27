package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/netapp/ontap-mcp/catalog"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/rest"
)

type GenerateCmd struct {
	APICatalog APICatalogCmd `cmd:"api-catalog" help:"Generate API catalog JSON by downloading swagger from an ONTAP cluster"`
}

type APICatalogCmd struct {
	Poller string `required:"" name:"poller" help:"Poller name from config to download swagger from (e.g. dc1)"`
	Output string `name:"output" default:"conf/ontap_api_catalog.json" help:"Destination path for the generated catalog JSON"`
}

func (c *APICatalogCmd) Run(cli *CLI) error {
	cfg, err := config.ReadConfig(cli.ConfigPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cli.ConfigPath, err)
	}
	poller, ok := cfg.Pollers[c.Poller]
	if !ok {
		return fmt.Errorf("poller %q not found in %s", c.Poller, cli.ConfigPath)
	}
	client := rest.New(poller)

	remote, err := client.GetClusterInfo(context.Background())
	if err != nil {
		return fmt.Errorf("get cluster info from %s: %w", poller.Addr, err)
	}
	ontapVersion := fmt.Sprintf("%d.%d", remote.Version.Generation, remote.Version.Major)

	data, err := client.FetchSwagger()
	if err != nil {
		return fmt.Errorf("download swagger from %s: %w", poller.Addr, err)
	}
	fmt.Printf("Downloaded swagger from %s (ONTAP %s, %d bytes)\n", poller.Addr, ontapVersion, len(data))
	return generateAPICatalog(data, ontapVersion, c.Output)
}

type swaggerDoc struct {
	Paths       map[string]swaggerPath `yaml:"paths"`
	Definitions map[string]swaggerDef  `yaml:"definitions"`
}

type swaggerPath struct {
	Get *swaggerOperation `yaml:"get"`
}

type swaggerParameter struct {
	Name        string            `yaml:"name"`
	In          string            `yaml:"in"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"`
	Introduced  string            `yaml:"x-ntap-introduced"`
	Visibility  swaggerVisibility `yaml:"x-ntap-visibility"`
}

type swaggerOperation struct {
	Description string                     `yaml:"description"`
	Tags        []string                   `yaml:"tags"`
	Parameters  []swaggerParameter         `yaml:"parameters"`
	Introduced  string                     `yaml:"x-ntap-introduced"`
	Visibility  swaggerVisibility          `yaml:"x-ntap-visibility"`
	Responses   map[string]swaggerResponse `yaml:"responses"`
}

type swaggerVisibility struct {
	Type string `yaml:"type"`
}

type swaggerResponse struct {
	Schema swaggerSchema `yaml:"schema"`
}

type swaggerSchema struct {
	Ref        string                 `yaml:"$ref"`
	Properties map[string]swaggerProp `yaml:"properties"`
}

type swaggerDef struct {
	Properties map[string]swaggerProp `yaml:"properties"`
}

type swaggerProp struct {
	Description string            `yaml:"description"`
	Ref         string            `yaml:"$ref"`
	Items       *swaggerProp      `yaml:"items"`
	Introduced  string            `yaml:"x-ntap-introduced"`
	Visibility  swaggerVisibility `yaml:"x-ntap-visibility"`
}

// skipParams are query params present on every collection GET.
// They are handled by the ontap_get tool directly and should not be surfaced
// to the LLM as application-level filters.
var skipParams = toSet(
	"fields", "max_records", "return_records", "return_timeout",
	"order_by", "offset", "pretty", "continue_on_failure", "ignore_unknown_fields",
)

// uuidEndpointAllowlist is the set of resource/sub-resource endpoints that
// contain path parameters such as uuid
var uuidEndpointAllowlist = toSet(
	"/storage/volumes/{volume.uuid}/snapshots",
	"/storage/volumes/{volume.uuid}/snapshots/{uuid}",

	"/protocols/san/igroups/{igroup.uuid}/igroups",
	"/protocols/san/igroups/{igroup.uuid}/igroups/{uuid}",
	"/protocols/san/igroups/{igroup.uuid}/initiators",

	"/snapmirror/relationships/{relationship.uuid}/transfers",
)

// skipPrefixes are param name prefixes for performance counters.
var skipPrefixes = []string{
	"statistics.",
	"metric.",
	"statistics_",
}

func toSet(vals ...string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

func isSkippedParam(name string) bool {
	if skipParams[name] {
		return true
	}
	for _, pfx := range skipPrefixes {
		if strings.HasPrefix(name, pfx) {
			return true
		}
	}
	return false
}

func isExcludedPath(path string) bool {
	for _, prefix := range catalog.ExcludedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// stripHTML removes HTML tags and normalises whitespace from swagger descriptions.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteByte(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}
	result := strings.Join(strings.Fields(b.String()), " ")
	return strings.TrimSpace(result)
}

func isGenericFilterDesc(desc string) bool {
	d := strings.ToLower(strings.TrimSpace(desc))
	if !strings.HasPrefix(d, "filter by ") {
		return false
	}
	return !strings.Contains(d[len("filter by "):], " ")
}

func extractFieldDescs(op *swaggerOperation, defs map[string]swaggerDef, apiVersion string) map[string]catalog.FieldInfo {
	respSchema, ok := op.Responses["200"]
	if !ok {
		return nil
	}
	respRef := respSchema.Schema.Ref
	if respRef == "" {
		return nil
	}
	respDefName := strings.TrimPrefix(respRef, "#/definitions/")
	respDef, ok := defs[respDefName]
	if !ok {
		return nil
	}
	recordsProp, ok := respDef.Properties["records"]
	if !ok || recordsProp.Items == nil {
		return nil
	}
	itemRef := recordsProp.Items.Ref
	if itemRef == "" {
		return nil
	}
	itemDefName := strings.TrimPrefix(itemRef, "#/definitions/")
	itemDef, ok := defs[itemDefName]
	if !ok {
		return nil
	}
	result := make(map[string]catalog.FieldInfo, len(itemDef.Properties))
	for name, prop := range itemDef.Properties {
		if strings.HasPrefix(name, "_") {
			continue // skip _links, _tags etc.
		}
		if prop.Visibility.Type == "private" {
			continue
		}
		desc := strings.TrimSpace(prop.Description)
		if desc == "" {
			continue
		}
		fi := catalog.FieldInfo{Desc: stripHTML(desc)}
		if v := prop.Introduced; v != "" && v != "DO_NOT_DISPLAY" && v != apiVersion {
			fi.Since = v
		}
		result[name] = fi
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func extractSummary(description string) string {
	lines := strings.SplitN(strings.TrimSpace(description), "\n", 2)
	s := strings.TrimSpace(lines[0])
	s = strings.TrimRight(s, "\r\n\t .:")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func generateAPICatalog(data []byte, ontapVersion string, outputPath string) error {
	var doc swaggerDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse swagger YAML: %w", err)
	}

	cat := make(catalog.APICatalog, 256)
	skipped := 0

	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		sp := doc.Paths[path]
		if sp.Get == nil {
			continue
		}

		// Skip private/internal endpoints not meant for external use.
		if sp.Get.Visibility.Type == "private" {
			skipped++
			continue
		}

		if isExcludedPath(path) {
			skipped++
			continue
		}

		// For endpoints with path parameters (e.g. {uuid}), only include those
		// in the allowlist.
		if strings.Contains(path, "{") && !uuidEndpointAllowlist[path] {
			skipped++
			continue
		}

		op := sp.Get
		summary := extractSummary(stripHTML(op.Description))

		apiVersion := op.Introduced

		// Collect path parameters (e.g. {uuid}, {name}) for endpoints that have them.
		pathParams := make(map[string]catalog.PathParamInfo)
		filters := make(map[string]catalog.FilterInfo)
		for _, p := range op.Parameters {
			switch p.In {
			case "path":
				ppi := catalog.PathParamInfo{}
				if d := strings.TrimSpace(p.Description); d != "" {
					ppi.Desc = stripHTML(d)
				}
				pathParams[p.Name] = ppi
			case "query":
				if isSkippedParam(p.Name) {
					continue
				}
				if p.Visibility.Type == "private" {
					continue
				}
				t := p.Type
				if t == "string" || t == "" {
					t = ""
				}
				fi := catalog.FilterInfo{Type: t}
				if v := p.Introduced; v != "" && v != "DO_NOT_DISPLAY" && v != apiVersion {
					fi.Since = v
				}
				if d := strings.TrimSpace(p.Description); d != "" && !isGenericFilterDesc(d) {
					fi.Desc = stripHTML(d)
				}
				filters[p.Name] = fi
			}
		}

		ep := catalog.APIEndpoint{
			Summary:    summary,
			Tags:       op.Tags,
			Introduced: apiVersion,
			Fields:     extractFieldDescs(op, doc.Definitions, apiVersion),
		}
		if len(pathParams) > 0 {
			ep.PathParams = pathParams
		}
		if len(filters) > 0 {
			ep.Filters = filters
		}

		cat[path] = ep
	}

	if err := catalog.Save(cat, ontapVersion, outputPath); err != nil {
		return fmt.Errorf("save catalog: %w", err)
	}

	fmt.Printf("Generated catalog: %d endpoints written to %s (%d paths skipped)\n",
		len(cat), outputPath, skipped)
	return nil
}

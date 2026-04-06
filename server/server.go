package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/catalog"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/descriptions"
	"github.com/netapp/ontap-mcp/rest"
	"github.com/netapp/ontap-mcp/server/lock"
	"github.com/netapp/ontap-mcp/tool"
	"github.com/netapp/ontap-mcp/version"
)

type Options struct {
	Host           string
	InspectTraffic bool
	IsTest         bool
	Port           int
	ReadOnly       bool
	TestHTTPClient *http.Client // Optional HTTP client for testing
}

type App struct {
	logger       *slog.Logger
	cfg          *config.ONTAP
	options      Options
	locks        *lock.Map
	catalog      catalog.APICatalog
	versionCache sync.Map
}

type cachedVersion struct {
	version string
	fetched time.Time
}

const versionCacheTTL = 24 * time.Hour

func NewApp(cfg *config.ONTAP, o Options, logger *slog.Logger) *App {
	app := &App{
		cfg:     cfg,
		logger:  logger,
		options: o,
		locks:   lock.New(),
	}

	const catalogPath = "conf/ontap_api_catalog.json"
	if cat, err := catalog.Load(catalogPath); err == nil {
		app.catalog = cat
		logger.Info("loaded API catalog", slog.Int("endpoints", len(cat)), slog.String("path", catalogPath))
	} else {
		logger.Warn("API catalog not found — catalog tools disabled", slog.String("path", catalogPath))
	}

	return app
}

func (a *App) StartServer() {
	if a.options.ReadOnly {
		a.logger.Info("MCP server is running in read-only mode; mutating operations are disabled")
	}
	server := a.createMCPServer()
	a.runHTTPServer(server)
}

func (a *App) createMCPServer() *mcp.Server {
	instructions := "IMPORTANT:" + descriptions.Instructions

	server := mcp.NewServer(&mcp.Implementation{Name: config.AppName, Version: version.Info()}, &mcp.ServerOptions{
		Instructions: instructions,
		Logger:       a.logger,
	})

	addTool(a, server, "list_registered_clusters", descriptions.ListClusters, readOnlyAnnotation, a.ListClusters)

	// operation on Volume object
	addTool(a, server, "create_volume", descriptions.CreateVolume, createAnnotation, a.CreateVolume)
	addTool(a, server, "update_volume", descriptions.UpdateVolume, updateAnnotation, a.UpdateVolume)
	addTool(a, server, "delete_volume", descriptions.DeleteVolume, deleteAnnotation, a.DeleteVolume)

	// operation on Snapshot Policy object
	addTool(a, server, "create_snapshot_policy", descriptions.CreateSnapshotPolicy, createAnnotation, a.CreateSnapshotPolicy)
	addTool(a, server, "delete_snapshot_policy", descriptions.DeleteSnapshotPolicy, deleteAnnotation, a.DeleteSnapshotPolicy)
	addTool(a, server, "create_schedule", descriptions.CreateSchedule, createAnnotation, a.CreateSchedule)

	// operation on QoS Policy object
	addTool(a, server, "list_qos_policies", descriptions.ListQoSPolicies, readOnlyAnnotation, a.ListQoSPolicies)
	addTool(a, server, "create_qos_policy", descriptions.CreateQoSPolicy, createAnnotation, a.CreateQoSPolicy)
	addTool(a, server, "update_qos_policy", descriptions.UpdateQoSPolicy, updateAnnotation, a.UpdateQosPolicy)
	addTool(a, server, "delete_qos_policy", descriptions.DeleteQoSPolicy, deleteAnnotation, a.DeleteQoSPolicy)

	// operation on NFS Export Policy object
	addTool(a, server, "create_nfs_export_policies", descriptions.CreateNFSExportPolicy, createAnnotation, a.CreateNFSExportPolicy)
	addTool(a, server, "update_nfs_export_policies", descriptions.UpdateNFSExportPolicy, updateAnnotation, a.UpdateNFSExportPolicy)
	addTool(a, server, "delete_nfs_export_policies", descriptions.DeleteNFSExportPolicy, deleteAnnotation, a.DeleteNFSExportPolicy)

	// operation on NFS Export Policy rules object
	addTool(a, server, "create_nfs_export_policies_rules", descriptions.CreateNFSExportPolicyRules, createAnnotation, a.CreateNFSExportPoliciesRule)
	addTool(a, server, "update_nfs_export_policies_rules", descriptions.UpdateNFSExportPolicyRules, updateAnnotation, a.UpdateNFSExportPoliciesRule)
	addTool(a, server, "delete_nfs_export_policies_rules", descriptions.DeleteNFSExportPolicyRules, deleteAnnotation, a.DeleteNFSExportPoliciesRule)

	// operation on CIFS share object
	addTool(a, server, "create_cifs_share", descriptions.CreateCIFSShare, createAnnotation, a.CreateCIFSShare)
	addTool(a, server, "update_cifs_share", descriptions.UpdateCIFSShare, updateAnnotation, a.UpdateCIFSShare)
	addTool(a, server, "delete_cifs_share", descriptions.DeleteCIFSShare, deleteAnnotation, a.DeleteCIFSShare)

	// operation on Qtree object
	addTool(a, server, "create_qtree", descriptions.CreateQtree, createAnnotation, a.CreateQtree)
	addTool(a, server, "update_qtree", descriptions.UpdateQtree, updateAnnotation, a.UpdateQtree)
	addTool(a, server, "delete_qtree", descriptions.DeleteQtree, deleteAnnotation, a.DeleteQtree)

	// operation on NVMe service object
	addTool(a, server, "create_nvme_service", descriptions.CreateNVMeService, createAnnotation, a.CreateNVMeService)
	addTool(a, server, "update_nvme_service", descriptions.UpdateNVMeService, updateAnnotation, a.UpdateNVMeService)
	addTool(a, server, "delete_nvme_service", descriptions.DeleteNVMeService, deleteAnnotation, a.DeleteNVMeService)

	if a.catalog != nil {
		addTool(a, server, "list_ontap_endpoints", descriptions.ListOntapEndpoints, readOnlyAnnotation, a.ListOntapEndpoints)
		addTool(a, server, "search_ontap_endpoints", descriptions.SearchOntapEndpoints, readOnlyAnnotation, a.SearchOntapEndpoints)
		addTool(a, server, "describe_ontap_endpoint", descriptions.DescribeOntapEndpoint, readOnlyAnnotation, a.DescribeOntapEndpoint)
	}
	addTool(a, server, "ontap_get", descriptions.OntapGet, readOnlyAnnotation, a.OntapGet)

	return server
}

func (a *App) runHTTPServer(server *mcp.Server) {
	var handler http.Handler

	address := a.options.Host + ":" + strconv.Itoa(a.options.Port)
	a.logger.Info("starting MCP server over HTTP transport",
		slog.String("address", address),
		slog.String("host", a.options.Host),
		slog.Int("port", a.options.Port))

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)

	if a.options.InspectTraffic {
		prevHandler := handler
		loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Example debugging; you could also capture the response.
			body, err := io.ReadAll(req.Body)
			if err != nil {
				a.logger.Error("failed to read request body", slog.Any("error", err))
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			//goland:noinspection GoUnhandledErrorResult
			req.Body.Close() //nolint:gosec
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			fmt.Println(req.Method, string(body))

			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			prevHandler.ServeHTTP(lrw, req)
			fmt.Printf("Response: status=%d, body=\n%s\n", lrw.statusCode, lrw.body.String())
		})
		handler = loggingHandler
	}

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip MCP handler for health endpoint
		if r.URL.Path == "/health" {
			http.DefaultServeMux.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Mcp-Protocol-Version, Mcp-Session-Id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})

	//goland:noinspection HttpUrlsUsage
	a.logger.Info("MCP server endpoint available", slog.String("url", "http://"+address))
	a.logger.Info("Server ready to accept connections")

	httpServer := &http.Server{
		Addr:              address,
		Handler:           wrappedHandler,
		ReadHeaderTimeout: 60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		a.logger.Error("http server failed to start", slog.String("error", err.Error()))
		os.Exit(1)
	}
	a.logger.Info("mcp server shutdown gracefully")
}

type clusterInfo struct {
	Name         string `json:"name"`
	ONTAPVersion string `json:"ontap_version"`
}

func (a *App) getClusterVersion(ctx context.Context, cluster string) (string, error) {
	if cached, ok := a.versionCache.Load(cluster); ok {
		cv := cached.(cachedVersion)
		if time.Since(cv.fetched) < versionCacheTTL {
			return cv.version, nil
		}
	}

	client, err := a.getClient(cluster)
	if err != nil {
		return "", err
	}
	remote, err := client.GetClusterInfo(ctx)
	if err != nil {
		return "", err
	}
	ver := fmt.Sprintf("%d.%d", remote.Version.Generation, remote.Version.Major)
	a.versionCache.Store(cluster, cachedVersion{version: ver, fetched: time.Now()})
	return ver, nil
}

func (a *App) ListClusters(ctx context.Context, _ *mcp.CallToolRequest, _ tool.ListClusterParams) (*mcp.CallToolResult, any, error) {
	clusters := slices.Clone(a.cfg.PollersOrdered)
	slices.Sort(clusters)

	infos := make([]clusterInfo, 0, len(clusters))
	for _, name := range clusters {
		ver, err := a.getClusterVersion(ctx, name)
		if err != nil {
			a.logger.Warn("failed to fetch cluster info", slog.String("cluster", name), slog.String("error", err.Error()))
			infos = append(infos, clusterInfo{Name: name})
			continue
		}
		infos = append(infos, clusterInfo{Name: name, ONTAPVersion: ver})
	}

	data, err := json.Marshal(infos)
	if err != nil {
		return errorResult(err), nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

func (a *App) ListOntapEndpoints(_ context.Context, _ *mcp.CallToolRequest, p tool.ListEndpointsParams) (*mcp.CallToolResult, any, error) {
	var results []catalog.SearchResult
	if p.Match != "" {
		results = a.catalog.Search(p.Match)
	} else {
		results = a.catalog.ListAll()
	}
	if len(results) == 0 {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "No endpoints found"}}}, nil, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d endpoints:\n", len(results))
	for _, r := range results {
		fmt.Fprintf(&sb, "%s — %s\n", r.Path, r.Endpoint.Summary)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}}}, nil, nil
}

func (a *App) SearchOntapEndpoints(_ context.Context, _ *mcp.CallToolRequest, p tool.SearchEndpointsParams) (*mcp.CallToolResult, any, error) {
	if p.Query == "" {
		return errorResult(errors.New("query parameter is required")), nil, nil
	}
	results := a.catalog.Search(p.Query)
	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "No endpoints found matching: " + p.Query}},
		}, nil, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d endpoints matching %q:\n", len(results), p.Query)
	for _, r := range results {
		fmt.Fprintf(&sb, "%s — %s\n", r.Path, r.Endpoint.Summary)
	}
	sb.WriteString("\nCall describe_ontap_endpoint for filters and fields before calling ontap_get.")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}}}, nil, nil
}

func (a *App) DescribeOntapEndpoint(ctx context.Context, _ *mcp.CallToolRequest, p tool.DescribeEndpointParams) (*mcp.CallToolResult, any, error) {
	if p.Path == "" {
		return errorResult(errors.New("path parameter is required")), nil, nil
	}
	ep, ok := a.catalog[p.Path]
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Endpoint not found: " + p.Path + ". Use list_ontap_endpoints or search_ontap_endpoints to discover valid paths."}},
		}, nil, nil
	}

	var versionNote string
	if p.Cluster != "" {
		if ontapVer, err := a.getClusterVersion(ctx, p.Cluster); err == nil {
			ep = ep.FilterByVersion(ontapVer)
			versionNote = ", filtered for ONTAP " + ontapVer
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (since %s%s)\n", p.Path, ep.Introduced, versionNote)

	if len(ep.PathParams) > 0 {
		names := make([]string, 0, len(ep.PathParams))
		for k := range ep.PathParams {
			names = append(names, k)
		}
		sort.Strings(names)
		sb.WriteString("Path parameters required, supply via path_params in ontap_get:\n")
		for _, name := range names {
			pp := ep.PathParams[name]
			if pp.Desc != "" {
				fmt.Fprintf(&sb, "  %s — %s\n", name, pp.Desc)
			} else {
				fmt.Fprintf(&sb, "  %s\n", name)
			}
		}
	}

	if len(ep.Filters) > 0 {
		names := make([]string, 0, len(ep.Filters))
		for k := range ep.Filters {
			names = append(names, k)
		}
		sort.Strings(names)
		sb.WriteString("Filters:\n")
		for _, name := range names {
			f := ep.Filters[name]
			typ := f.Type
			if typ == "" {
				typ = "string"
			}
			var ann string
			switch {
			case typ != "string" && f.Since != "":
				ann = "(" + typ + "," + f.Since + ")"
			case typ != "string":
				ann = "(" + typ + ")"
			case f.Since != "":
				ann = "(" + f.Since + ")"
			}
			if f.Desc != "" {
				fmt.Fprintf(&sb, "  %s%s — %s\n", name, ann, f.Desc)
			} else {
				fmt.Fprintf(&sb, "  %s%s\n", name, ann)
			}
		}
	}

	if len(ep.Fields) > 0 {
		names := make([]string, 0, len(ep.Fields))
		for k := range ep.Fields {
			names = append(names, k)
		}
		sort.Strings(names)
		sb.WriteString("Fields:\n")
		for _, name := range names {
			f := ep.Fields[name]
			if f.Since != "" {
				fmt.Fprintf(&sb, "  %s(%s) — %s\n", name, f.Since, f.Desc)
			} else {
				fmt.Fprintf(&sb, "  %s — %s\n", name, f.Desc)
			}
		}
	}

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}}}, nil, nil
}

func (a *App) OntapGet(ctx context.Context, _ *mcp.CallToolRequest, p tool.OntapGetParams) (*mcp.CallToolResult, any, error) {
	if p.Cluster == "" {
		return errorResult(errors.New("cluster_name is required")), nil, nil
	}
	if p.Path == "" {
		return errorResult(errors.New("path is required")), nil, nil
	}
	if !strings.HasPrefix(p.Path, "/") {
		return errorResult(fmt.Errorf("path must start with /, got %q", p.Path)), nil, nil
	}

	if p.PathParams == nil {
		p.PathParams = make(map[string]string)
	}
	if p.Filters == nil {
		p.Filters = make(map[string]string)
	}

	// Resolve any path-parameter placeholders (e.g. {volume.uuid}) before making the request.
	resolvedPath := p.Path
	for k, v := range p.PathParams {
		escaped := url.PathEscape(v)
		resolvedPath = strings.ReplaceAll(resolvedPath, "{"+k+"}", escaped)
	}
	if strings.Contains(resolvedPath, "{") {
		return errorResult(fmt.Errorf("path %q has unresolved placeholders. Provide their values via path_params (e.g. {\"volume.uuid\": \"<value>\"})", resolvedPath)), nil, nil
	}
	p.Path = resolvedPath

	a.locks.RLock(p.Cluster)
	defer a.locks.RUnlock(p.Cluster)

	client, err := a.getClient(p.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	params := url.Values{}
	for k, v := range p.Filters {
		params.Set(k, v)
	}
	if p.Fields != "" {
		params.Set("fields", p.Fields)
	}
	if p.MaxRecords > 0 {
		params.Set("max_records", strconv.Itoa(p.MaxRecords))
	}
	// ignore_unknown_fields was introduced in ONTAP 9.11. skip for older clusters.
	if ver, err := a.getClusterVersion(ctx, p.Cluster); err != nil || catalog.CompareVersions(ver, "9.11") >= 0 {
		params.Set("ignore_unknown_fields", "true")
	}

	raw, err := client.GenericGet(ctx, p.Path, params, p.MaxRecords)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(stripLinks(raw))}},
	}, nil, nil
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: err.Error()},
		},
		IsError: true,
	}
}

func stripLinks(raw json.RawMessage) json.RawMessage {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	stripped := stripLinksValue(v)
	out, err := json.Marshal(stripped)
	if err != nil {
		return raw
	}
	return out
}

func stripLinksValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, child := range val {
			if k == "_links" {
				continue
			}
			out[k] = stripLinksValue(child)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = stripLinksValue(item)
		}
		return out
	default:
		return v
	}
}

func (a *App) getClient(cluster string) (*rest.Client, error) {
	poller, ok := a.cfg.Pollers[cluster]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", cluster)
	}

	if a.options.TestHTTPClient != nil {
		return rest.NewWithClient(poller, a.options.TestHTTPClient), nil
	}
	return rest.New(poller), nil
}

var (
	readOnlyAnnotation = mcp.ToolAnnotations{
		ReadOnlyHint: true,
	}
	createAnnotation = mcp.ToolAnnotations{
		DestructiveHint: new(false),
	}
	updateAnnotation = mcp.ToolAnnotations{
		DestructiveHint: new(true),
		IdempotentHint:  true,
	}
	deleteAnnotation = mcp.ToolAnnotations{
		DestructiveHint: new(true),
		IdempotentHint:  true,
	}
)

func addTool[In, Out any](a *App, server *mcp.Server, name string, description string, annotations mcp.ToolAnnotations, handler mcp.ToolHandlerFor[In, Out]) {
	if a.options.ReadOnly && !annotations.ReadOnlyHint {
		a.logger.Warn("skipping registration of destructive tool in read-only mode", slog.String("tool", name))
		return
	}
	tt := &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &annotations,
	}

	// Check if the tool handlers in param has any fields. If it doesn't, create a schema with an empty properties
	// Workaround for https://github.com/modelcontextprotocol/go-sdk/issues/693
	typeFor := reflect.TypeFor[In]()

	if typeFor.Kind() == reflect.Ptr {
		typeFor = typeFor.Elem()
	}

	if typeFor.Kind() == reflect.Struct && typeFor.NumField() == 0 {
		tt.InputSchema = json.RawMessage(`{"type":"object","properties":{}}`)
	}

	safeHandler := func(ctx context.Context, req *mcp.CallToolRequest, params In) (res *mcp.CallToolResult, out Out, err error) { //nolint:nonamedreturns
		defer func() {
			if rec := recover(); rec != nil {
				a.logger.Error("panic in tool handler",
					slog.String("tool", name),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())))
				res = errorResult(fmt.Errorf("internal error in tool %s: %v", name, rec))
			}
		}()
		res, out, err = handler(ctx, req, params)
		return
	}

	mcp.AddTool(server, tt, safeHandler)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// parseSizeEmptyAllowed is similar to parseSize but allows empty string as valid input, which will be interpreted as 0 bytes.
func parseSizeEmptyAllowed(size string) (int64, error) {
	trimmed := strings.TrimSpace(size)
	if trimmed == "" {
		return 0, nil
	}
	return parseSize(trimmed)
}

// parseSize parses size strings in multiple formats
// - Raw bytes: "104857600" → 104857600
// - With units: "100MB", "2GB", "1TB"
// - Short units: "100M", "2G", "1T"
// - Case insensitive: "100mb", "2gb", etc.
// Returns size in bytes, or error if format is invalid.
func parseSize(size string) (int64, error) {
	if size == "" {
		return 0, errors.New("size is empty")
	}

	size = strings.TrimSpace(size)

	// Parse with unit suffix first (e.g., "100MB", "2GB", "100M", "2G")
	var num float64
	var unit string

	// Try parsing "100MB", "100GB", "100TB" format
	if n, err := fmt.Sscanf(size, "%f%s", &num, &unit); err == nil && n == 2 {
		// Successfully parsed number with unit
		unit = strings.ToUpper(unit)

		switch unit {
		case "KB", "K":
			return int64(num * 1024), nil
		case "MB", "M":
			return int64(num * 1024 * 1024), nil
		case "GB", "G":
			return int64(num * 1024 * 1024 * 1024), nil
		case "TB", "T":
			return int64(num * 1024 * 1024 * 1024 * 1024), nil
		default:
			return 0, fmt.Errorf("invalid size unit '%s'. Supported: KB, MB, GB, TB (or K, M, G, T), or raw bytes", unit)
		}
	}

	// Try parsing as raw number (bytes) - only if no unit was found
	var sizeBytes int64
	if num, err := fmt.Sscanf(size, "%d", &sizeBytes); err == nil && num == 1 {
		// Successfully parsed as raw bytes
		return sizeBytes, nil
	}

	return 0, fmt.Errorf("invalid size format '%s'. Use '100MB', '2GB', '1TB', or raw bytes", size)
}

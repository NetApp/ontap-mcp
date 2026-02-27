package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/descriptions"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/rest"
	"github.com/netapp/ontap-mcp/server/lock"
	"github.com/netapp/ontap-mcp/tool"
	"github.com/netapp/ontap-mcp/version"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
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
	logger  *slog.Logger
	cfg     *config.ONTAP
	options Options
	locks   *lock.Map
}

func NewApp(cfg *config.ONTAP, o Options, logger *slog.Logger) *App {
	return &App{
		cfg:     cfg,
		logger:  logger,
		options: o,
		locks:   lock.New(),
	}
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
	addTool(a, server, "list_volumes", descriptions.ListVolumes, readOnlyAnnotation, a.ListVolumes)
	addTool(a, server, "create_volume", descriptions.CreateVolume, createAnnotation, a.CreateVolume)
	addTool(a, server, "update_volume", descriptions.UpdateVolume, updateAnnotation, a.UpdateVolume)
	addTool(a, server, "delete_volume", descriptions.DeleteVolume, deleteAnnotation, a.DeleteVolume)

	// operation on Snapshot Policy object
	addTool(a, server, "list_snapshot_policies", descriptions.ListSnapshotPolicy, readOnlyAnnotation, a.ListSnapshotPolicies)
	addTool(a, server, "create_snapshot_policy", descriptions.CreateSnapshotPolicy, createAnnotation, a.CreateSnapshotPolicy)
	addTool(a, server, "delete_snapshot_policy", descriptions.DeleteSnapshotPolicy, deleteAnnotation, a.DeleteSnapshotPolicy)
	addTool(a, server, "create_schedule", descriptions.CreateSchedule, createAnnotation, a.CreateSchedule)

	// operation on QoS Policy object
	addTool(a, server, "list_qos_policies", descriptions.ListQoSPolicy, readOnlyAnnotation, a.ListQoSPolicies)
	addTool(a, server, "create_qos_policy", descriptions.CreateQoSPolicy, createAnnotation, a.CreateQoSPolicy)
	addTool(a, server, "update_qos_policy", descriptions.UpdateQoSPolicy, updateAnnotation, a.UpdateQosPolicy)
	addTool(a, server, "delete_qos_policy", descriptions.DeleteQoSPolicy, deleteAnnotation, a.DeleteQoSPolicy)

	// operation on NFS Export Policy object
	addTool(a, server, "list_nfs_export_policies", descriptions.ListNFSExportPolicy, readOnlyAnnotation, a.ListNFSExportPolicies)
	addTool(a, server, "create_nfs_export_policies", descriptions.CreateNFSExportPolicy, createAnnotation, a.CreateNFSExportPolicy)
	addTool(a, server, "update_nfs_export_policies", descriptions.UpdateNFSExportPolicy, updateAnnotation, a.UpdateNFSExportPolicy)
	addTool(a, server, "delete_nfs_export_policies", descriptions.DeleteNFSExportPolicy, deleteAnnotation, a.DeleteNFSExportPolicy)

	// operation on NFS Export Policy rules object
	addTool(a, server, "create_nfs_export_policies_rules", descriptions.CreateNFSExportPolicyRules, createAnnotation, a.CreateNFSExportPoliciesRule)
	addTool(a, server, "update_nfs_export_policies_rules", descriptions.UpdateNFSExportPolicyRules, updateAnnotation, a.UpdateNFSExportPoliciesRule)
	addTool(a, server, "delete_nfs_export_policies_rules", descriptions.DeleteNFSExportPolicyRules, deleteAnnotation, a.DeleteNFSExportPoliciesRule)

	// operation on CIFS share object
	addTool(a, server, "list_cifs_share", descriptions.ListCIFSShare, readOnlyAnnotation, a.ListCIFSShare)
	addTool(a, server, "create_cifs_share", descriptions.CreateCIFSShare, createAnnotation, a.CreateCIFSShare)
	addTool(a, server, "update_cifs_share", descriptions.UpdateCIFSShare, updateAnnotation, a.UpdateCIFSShare)
	addTool(a, server, "delete_cifs_share", descriptions.DeleteCIFSShare, deleteAnnotation, a.DeleteCIFSShare)

	return server
}

func (a *App) runHTTPServer(server *mcp.Server) {
	var handler http.Handler

	address := a.options.Host + ":" + strconv.Itoa(a.options.Port)
	a.logger.Info("starting MCP server over HTTP transport",
		slog.String("address", address),
		slog.String("host", a.options.Host),
		slog.Int("port", a.options.Port))

	handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)

	if a.options.InspectTraffic {
		prevHandler := handler
		loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Example debugging; you could also capture the response.
			body, err := io.ReadAll(req.Body)
			if err != nil {
				log.Fatal(err)
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

func (a *App) ListClusters(_ context.Context, _ *mcp.CallToolRequest, _ ListClusterParams) (*mcp.CallToolResult, any, error) {
	// Validate params

	clusters := slices.Clone(a.cfg.PollersOrdered)
	slices.Sort(clusters)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(clusters, ",")},
		},
	}, nil, nil
}

func (a *App) ListVolumes(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	a.locks.RLock(parameters.Cluster)
	defer a.locks.RUnlock(parameters.Cluster)

	volumeGet := newGetVolume(parameters)

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	volumes, err := client.GetVolume(ctx, volumeGet)

	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(volumes, ",")},
		},
	}, nil, nil
}

func (a *App) DeleteVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	volumeDelete, err := newDeleteVolume(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.DeleteVolume(ctx, volumeDelete)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Volume deleted successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) CreateVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	volumeCreate, err := newCreateVolume(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.CreateVolume(ctx, volumeCreate)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Volume created successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

func (a *App) UpdateVolume(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.Volume) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	volumeUpdate, err := newUpdateVolume(parameters)
	if err != nil {
		return nil, nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	err = client.UpdateVolume(ctx, volumeUpdate, parameters.Volume, parameters.SVM)

	if err != nil {
		return errorResult(err), nil, err
	}

	responseText := "Volume updated successfully"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
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

// newDeleteVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newDeleteVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Name = in.Volume
	return out, nil
}

// newGetVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newGetVolume(in tool.Volume) ontap.Volume {
	out := ontap.Volume{}
	if in.SVM != "" {
		out.SVM = ontap.NameAndUUID{Name: in.SVM}
	}

	return out
}

// newCreateVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newCreateVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.Aggregate == "" {
		return out, errors.New("aggregate name is required")
	}

	out.SVM = ontap.NameAndUUID{Name: in.SVM}
	out.Aggregates = []ontap.NameAndUUID{
		{Name: in.Aggregate},
	}
	out.Name = in.Volume

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
	}

	if in.ExportPolicy != "" {
		out.Nas = ontap.NAS{
			ExportPolicy: ontap.NASExportPolicy{
				Name: in.ExportPolicy,
			},
		}
	}

	return out, nil
}

// newUpdateVolume validates the customer provided arguments and converts them into
// the corresponding ONTAP object ready to use via the REST API
func newUpdateVolume(in tool.Volume) (ontap.Volume, error) {
	out := ontap.Volume{}
	if in.SVM == "" {
		return out, errors.New("SVM name is required")
	}
	if in.Volume == "" {
		return out, errors.New("volume name is required")
	}
	if in.NewVolume != "" {
		out.Name = in.NewVolume
	}

	if in.Size != "" {
		size, err := parseSize(in.Size)
		if err != nil {
			return ontap.Volume{}, err
		}
		out.Size = size
	}

	if in.NewState != "" {
		out.State = in.NewState
	}

	if in.ExportPolicy != "" {
		out.Nas.ExportPolicy.Name = in.ExportPolicy
	}

	if in.Autosize.Mode != "" {
		out.Autosize.Mode = in.Autosize.Mode
	}
	if in.Autosize.MaxSize != "" {
		out.Autosize.MaxSize = in.Autosize.MaxSize
	}
	if in.Autosize.MinSize != "" {
		out.Autosize.MinSize = in.Autosize.MinSize
	}
	if in.Autosize.GrowThreshold != "" {
		out.Autosize.GrowThreshold = in.Autosize.GrowThreshold
	}
	if in.Autosize.ShrinkThreshold != "" {
		out.Autosize.ShrinkThreshold = in.Autosize.ShrinkThreshold
	}

	return out, nil
}

type ListClusterParams struct{}
type ListVolumeParams struct{}

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
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &annotations,
	}, handler)
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

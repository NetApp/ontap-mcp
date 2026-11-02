package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/tool"
)

type validationRoundTripFunc func(*http.Request) (*http.Response, error)

func (f validationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNewValidateReplication(t *testing.T) {
	tests := []struct {
		name    string
		params  tool.ValidateReplicationParams
		wantErr string
	}{
		{name: "missing source", params: tool.ValidateReplicationParams{DestinationCluster: "b"}, wantErr: "source_cluster is required"},
		{name: "missing destination", params: tool.ValidateReplicationParams{SourceCluster: "a"}, wantErr: "destination_cluster is required"},
		{name: "same cluster", params: tool.ValidateReplicationParams{SourceCluster: "A", DestinationCluster: "a"}, wantErr: "must be different"},
		{name: "only source SVM", params: tool.ValidateReplicationParams{SourceCluster: "a", DestinationCluster: "b", SourceSVM: "svm-a"}, wantErr: "must be provided together"},
		{name: "only destination SVM", params: tool.ValidateReplicationParams{SourceCluster: "a", DestinationCluster: "b", DestinationSVM: "svm-b"}, wantErr: "must be provided together"},
		{name: "clusters only", params: tool.ValidateReplicationParams{SourceCluster: "a", DestinationCluster: "b"}},
		{name: "paired SVMs", params: tool.ValidateReplicationParams{SourceCluster: "a", DestinationCluster: "b", SourceSVM: "svm-a", DestinationSVM: "svm-b"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newValidateReplication(test.params)
			if test.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want error containing %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateReplicationSuccess(t *testing.T) {
	app := newReplicationTestApp(t, false)
	result, response, err := app.ValidateReplication(context.Background(), nil, tool.ValidateReplicationParams{
		SourceCluster:      "source",
		DestinationCluster: "destination",
		SourceSVM:          "svm-source",
		DestinationSVM:     "svm-destination",
	})
	if err != nil {
		t.Fatalf("ValidateReplication() error = %v", err)
	}
	if result != nil {
		t.Fatalf("ValidateReplication() MCP result = %#v, want nil", result)
	}
	if !response.CanReplicate {
		t.Fatalf("CanReplicate = false; missing prerequisites: %v", response.MissingPrerequisites)
	}
	if len(response.Checks) != 9 {
		t.Fatalf("got %d checks, want 9", len(response.Checks))
	}
	if len(response.MissingPrerequisites) != 0 {
		t.Fatalf("missing prerequisites = %v, want none", response.MissingPrerequisites)
	}
}

func TestValidateReplicationAggregatesFailures(t *testing.T) {
	app := newReplicationTestApp(t, true)
	_, response, err := app.ValidateReplication(context.Background(), nil, tool.ValidateReplicationParams{
		SourceCluster:      "source",
		DestinationCluster: "destination",
	})
	if err != nil {
		t.Fatalf("ValidateReplication() error = %v", err)
	}
	if response.CanReplicate {
		t.Fatal("CanReplicate = true, want false")
	}
	if len(response.MissingPrerequisites) != 5 {
		t.Fatalf("got %d missing prerequisites, want 5: %v", len(response.MissingPrerequisites), response.MissingPrerequisites)
	}
	for _, want := range []string{"no intercluster LIFs", "not enabled and up", "no cluster peer relationship", "not supported"} {
		if !strings.Contains(response.Message, want) {
			t.Errorf("message %q does not contain %q", response.Message, want)
		}
	}
}

func newReplicationTestApp(t *testing.T, failing bool) *App {
	t.Helper()
	transport := validationRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", request.Method)
		}
		source := request.URL.Host == "source.test"
		body := replicationFixture(request.URL.Path, source, failing)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	cfg := &config.ONTAP{Pollers: map[string]*config.Poller{
		"source":      {Addr: "source.test", Name: "source"},
		"destination": {Addr: "destination.test", Name: "destination"},
	}}
	app, err := NewApp(cfg, Options{TestHTTPClient: &http.Client{Transport: transport}}, slog.Default())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	return app
}

func replicationFixture(path string, source, failing bool) string {
	clusterName := "destination-real"
	version := `"generation":9,"major":15,"minor":1`
	if source {
		clusterName = "source-real"
		version = `"generation":9,"major":16,"minor":1`
	}
	if failing {
		if source {
			version = `"generation":9,"major":19,"minor":1`
		} else {
			version = `"generation":9,"major":5,"minor":0`
		}
	}

	switch path {
	case "/api/cluster":
		return `{"name":"` + clusterName + `","version":{` + version + `}}`
	case "/api/network/ip/interfaces":
		if failing && source {
			return `{"records":[],"num_records":0}`
		}
		if failing {
			return `{"records":[{"name":"icl1","enabled":true,"state":"down"}],"num_records":1}`
		}
		return `{"records":[{"name":"icl1","enabled":true,"state":"up"}],"num_records":1}`
	case "/api/cluster/peers":
		if failing {
			return `{"records":[],"num_records":0}`
		}
		return `{"records":[{"name":"peer","uuid":"peer-1","status":{"state":"available"}}],"num_records":1}`
	case "/api/svm/peers":
		return `{"records":[{"uuid":"svm-peer-1","state":"peered"}],"num_records":1}`
	default:
		return `{"records":[],"num_records":0}`
	}
}

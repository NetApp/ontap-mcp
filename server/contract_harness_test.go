package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/ontap"
)

// Contract tests drive tools through a real MCP client session against a scripted fake ONTAP.
// They verify the REST requests each tool sends and how it handles ONTAP's responses, without
// a cluster or an LLM.

const (
	contractCluster  = "contract-cluster"
	contractUser     = "admin"
	contractPassword = "contract-secret"
)

// exchange is one expected ONTAP request and the response the fake returns for it.
type exchange struct {
	method string
	path   string // path with optional query, e.g. "/api/storage/volumes?fields=uuid&name=docs"
	body   string // expected request JSON, compared semantically; empty means no body. See matchJSON for placeholders.
	status int    // response status, defaults to 200
	resp   string // response JSON
}

// fakeONTAP serves a fixed script of exchanges in order and fails the test on any deviation.
type fakeONTAP struct {
	t        *testing.T
	cluster  string
	captures *captures
	mu       sync.Mutex
	script   []exchange
	next     int
}

// captures holds placeholder values bound by request bodies. It is shared by every fake in a
// session so a value sent to one cluster can be required on another.
type captures struct {
	mu     sync.Mutex
	values map[string]any
}

// bind records value for name on first use and reports whether it matches on later uses.
func (c *captures) bind(name string, value any) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.values[name]; ok {
		return reflect.DeepEqual(prev, value)
	}
	c.values[name] = value
	return true
}

func (f *fakeONTAP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	got := r.Method + " " + r.URL.RequestURI()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		f.t.Errorf("reading body of %s: %v", got, err)
	}

	if f.next >= len(f.script) {
		f.t.Errorf("%s: unexpected ONTAP request after script ended: %s %s", f.cluster, got, body)
		f.writeError(w, http.StatusInternalServerError, "unexpected request")
		return
	}
	want := f.script[f.next]
	f.next++

	if mismatch := want.mismatch(r, body, f.captures); mismatch != "" {
		f.t.Errorf("%s: ONTAP request #%d: %s\n  got:  %s %s\n  want: %s %s %s", f.cluster, f.next, mismatch, got, body, want.method, want.path, want.body)
		f.writeError(w, http.StatusBadRequest, "contract mismatch")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	status := want.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, want.resp)
}

func (e exchange) mismatch(r *http.Request, body []byte, caps *captures) string {
	if user, pass, ok := r.BasicAuth(); !ok || user != contractUser || pass != contractPassword {
		return "missing or wrong basic auth"
	}
	if !strings.HasPrefix(r.UserAgent(), "ontap-mcp/") {
		return "unexpected User-Agent " + r.UserAgent()
	}
	if r.Method != e.method {
		return "method differs"
	}
	wantURL, err := url.Parse(e.path)
	if err != nil {
		return "bad path in script: " + err.Error()
	}
	if r.URL.Path != wantURL.Path {
		return "path differs"
	}
	if !reflect.DeepEqual(r.URL.Query(), wantURL.Query()) {
		return "query differs"
	}
	if e.body == "" {
		if len(body) != 0 {
			return "unexpected request body"
		}
		return ""
	}
	var got, want any
	if err := json.Unmarshal(body, &got); err != nil {
		return "request body is not JSON"
	}
	if err := json.Unmarshal([]byte(e.body), &want); err != nil {
		return "bad body in script: " + err.Error()
	}
	if !matchJSON(want, got, caps) {
		return "body differs"
	}
	return ""
}

func (f *fakeONTAP) assertDone() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.script[f.next:] {
		f.t.Errorf("%s: ONTAP never received expected request: %s %s %s", f.cluster, e.method, e.path, e.body)
	}
}

// matchJSON compares decoded JSON values. In want, the string "{{*}}" matches any value and
// "{{name}}" matches any value but binds it, so every other "{{name}}" in the session must be equal.
func matchJSON(want, got any, caps *captures) bool {
	switch w := want.(type) {
	case string:
		if name, ok := strings.CutPrefix(w, "{{"); ok {
			if name, ok = strings.CutSuffix(name, "}}"); ok {
				return name == "*" || caps.bind(name, got)
			}
		}
		return w == got
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok || !matchJSON(wv, gv, caps) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !matchJSON(w[i], g[i], caps) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(want, got)
	}
}

func (f *fakeONTAP) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(struct {
		Err ontap.OErr `json:"error"`
	}{Err: ontap.OErr{Message: msg, Code: "0"}})
	if err != nil {
		f.t.Errorf("writing ONTAP error response: %v", err)
	}
}

// newContractSession starts an App wired to a fake ONTAP that expects script, and returns an
// MCP client session connected to it in memory.
func newContractSession(t *testing.T, toolMode string, script []exchange) *mcp.ClientSession {
	t.Helper()
	return newMultiClusterSession(t, toolMode, map[string][]exchange{contractCluster: script})
}

// newMultiClusterSession is newContractSession with one fake ONTAP per cluster name. Each fake
// checks the order of its own requests; ordering across clusters is not checked.
func newMultiClusterSession(t *testing.T, toolMode string, scripts map[string][]exchange) *mcp.ClientSession {
	t.Helper()
	return newOrderedClusterSession(t, toolMode, slices.Sorted(maps.Keys(scripts)), scripts)
}

// newOrderedClusterSession is newMultiClusterSession with an explicit config-file order for the
// clusters, for tools that iterate pollers in that order.
func newOrderedClusterSession(t *testing.T, toolMode string, order []string, scripts map[string][]exchange) *mcp.ClientSession {
	t.Helper()

	caps := &captures{values: make(map[string]any)}
	pool := x509.NewCertPool()
	pollers := make(map[string]*config.Poller, len(scripts))
	for cluster, script := range scripts {
		fake := &fakeONTAP{t: t, cluster: cluster, captures: caps, script: script}
		ts := httptest.NewTLSServer(fake)
		t.Cleanup(ts.Close)
		t.Cleanup(fake.assertDone)
		pool.AddCert(ts.Certificate())
		pollers[cluster] = &config.Poller{Name: cluster, Addr: ts.Listener.Addr().String(), Username: contractUser, Password: contractPassword}
	}

	// The App shares one HTTP client across clusters, so it must trust every fake's certificate.
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}
	t.Cleanup(httpClient.CloseIdleConnections)

	app, err := NewApp(&config.ONTAP{Pollers: pollers, PollersOrdered: order}, Options{ToolMode: toolMode, TestHTTPClient: httpClient}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ss, err := app.createMCPServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "contract-test", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	return cs
}

// callTool invokes a tool and returns its text output and whether it reported a tool error.
func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	var text strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	return text.String(), res.IsError
}

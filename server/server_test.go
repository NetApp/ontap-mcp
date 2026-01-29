package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/netapp/ontap-mcp/tool"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlmjohnson/requests"
	"github.com/carlmjohnson/requests/reqtest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/assert"
	"github.com/netapp/ontap-mcp/config"
	"log/slog"
)

// authStrippingTransport strips Authorization header for replay matching
type authStrippingTransport struct {
	inner http.RoundTripper
}

func (a *authStrippingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request and strip Authorization header
	clonedReq := req.Clone(req.Context())
	clonedReq.Header.Del("Authorization")
	return a.inner.RoundTrip(clonedReq)
}

// authStrippingRecorder wraps reqtest.Record and strips Authorization headers
type authStrippingRecorder struct {
	baseTransport http.RoundTripper
	recordDir     string
}

func (a *authStrippingRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read and preserve the request body
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close() //nolint:gosec
		// Restore body for the actual request
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Make the real request with auth
	resp, err := a.baseTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Read the response body
	var respBodyBytes []byte
	if resp.Body != nil {
		respBodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close() //nolint:gosec
		// Restore for the caller
		resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))
	}

	// Now record a sanitized version
	// Create sanitized request clone without Authorization
	sanitizedReq := &http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           req.Header.Clone(),
		Body:             io.NopCloser(bytes.NewReader(bodyBytes)),
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Close:            req.Close,
		Host:             req.Host,
		Form:             req.Form,
		PostForm:         req.PostForm,
		Trailer:          req.Trailer,
		RemoteAddr:       req.RemoteAddr,
		RequestURI:       req.RequestURI,
	}
	sanitizedReq.Header.Del("Authorization")

	// Create a mock transport that returns our captured response
	mockTransport := &mockResponseTransport{
		resp: &http.Response{
			Status:           resp.Status,
			StatusCode:       resp.StatusCode,
			Proto:            resp.Proto,
			ProtoMajor:       resp.ProtoMajor,
			ProtoMinor:       resp.ProtoMinor,
			Header:           resp.Header.Clone(),
			Body:             io.NopCloser(bytes.NewReader(respBodyBytes)),
			ContentLength:    resp.ContentLength,
			TransferEncoding: resp.TransferEncoding,
			Close:            resp.Close,
			Uncompressed:     resp.Uncompressed,
			Trailer:          resp.Trailer,
			Request:          sanitizedReq,
			TLS:              resp.TLS,
		},
	}

	// Wrap the recorder with our mock transport
	wrappedRecorder := reqtest.Record(mockTransport, a.recordDir)

	// Trigger the recording with sanitized request
	_, _ = wrappedRecorder.RoundTrip(sanitizedReq)

	return resp, nil
}

// mockResponseTransport returns a pre-captured response
type mockResponseTransport struct {
	resp *http.Response
}

func (m *mockResponseTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.resp, nil
}

func TestApp_CreateVolume(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-volume-too-small",
			in: `{
  "aggregate_name" : "umeng_aff300_aggr2",
  "cluster_name" : "sar",
  "size" : "10MB",
  "svm_name" : "osc",
  "volume_name" : "foople"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "is too small",
		},
		{
			name: "create-volume-just-right",
			in: `{
  "aggregate_name" : "umeng_aff300_aggr2",
  "cluster_name" : "sar",
  "size" : "20MB",
  "svm_name" : "osc",
  "volume_name" : "foople"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.Volume
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateVolume(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_UpdateVolume(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "update-volume-missing",
			in: `{
  "cluster_name" : "sar",
  "svm_name" : "osc",
  "volume_name" : "missing",
  "new_volume_name" : "new missing"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "update-volume-present",
			in: `{
  "cluster_name" : "sar",
  "svm_name" : "osc",
  "volume_name" : "volHar22",
  "new_volume_name": "volHar"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.Volume
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.UpdateVolume(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_DeleteVolume(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "delete-volume-missing",
			in: `{
  "cluster_name" : "sar",
  "svm_name" : "osc",
  "volume_name" : "___________________________foople_____"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "delete-volume-present",
			in: `{
  "cluster_name" : "sar",
  "svm_name" : "osc",
  "volume_name" : "foople"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var vd tool.Volume
			err = json.Unmarshal([]byte(tt.in), &vd)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.DeleteVolume(context.Background(), in, vd)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_CreateVolume_WithExportPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-volume-export-policy-exists",
			in: `{
  "aggregate_name" : "umeng_aff300_aggr2",
  "cluster_name" : "sar",
  "size" : "10MB",
  "svm_name" : "osc",
  "volume_name" : "foople",
  "export_policy": "cbg"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "is too small",
		},
		{
			name: "create-volume-export-policy-does-not-exist",
			in: `{
  "aggregate_name" : "umeng_aff300_aggr2",
  "cluster_name" : "sar",
  "size" : "20MB",
  "svm_name" : "osc",
  "volume_name" : "foople",
  "export_policy": "export_foople"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.Volume
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateVolume(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_CreateQoSPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-fixed-qos-policy",
			in: `{
		 "cluster_name": "u2",
		 "name": "qa-fix-1",
		 "svm_name": "vs_test",
		 "max_throughput_iops": "10000",
		 "min_throughput_iops": "10"
		}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
		{
			name: "create-adaptive-qos-policy",
			in: `{
		 "cluster_name": "u2",
		 "name": "qa-adaptive-1",
		 "svm_name": "vs_test",
		 "expected_iops": "1000",
		 "peak_iops": "10000",
		 "absolute_min_iops": "10"
		}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.QoSPolicy
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateQoSPolicy(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_UpdateQoSPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "update-qos-policy-missing",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "name" : "qa-fix-not-exist",
  "new_name" : "new_name"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "update-qos-policy-present",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "name" : "qa-adaptive-11",
  "new_name" : "qa-adaptive-1"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var q tool.QoSPolicy
			err = json.Unmarshal([]byte(tt.in), &q)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.UpdateQosPolicy(context.Background(), in, q)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_DeleteQoSPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "delete-qos-policy-missing",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "name" : "qa-fix-not-exist"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "delete-qos-policy-present",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "name" : "qa-fix-1"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var q tool.QoSPolicy
			err = json.Unmarshal([]byte(tt.in), &q)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.DeleteQoSPolicy(context.Background(), in, q)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_CreateNFSExportPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-nfs-export-policy",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-1",
  "svm_name": "vs_test"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
		{
			name: "create-nfs-export-policy-with-rule",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-2",
  "svm_name": "vs_test",
  "ro_rule": "any",
  "rw_rule": "any",
  "client_match": "0.0.0.0/0"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.NFSExportPolicy
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateNFSExportPolicy(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_UpdateNFSExportPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "update-nfs-export-policy-missing",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-missing",
  "new_export_policy": "export-policy-1",
  "svm_name": "vs_test"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "update-nfs-export-policy-present",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-1",
  "new_export_policy": "export-policy-2",
  "svm_name": "vs_test",
  "ro_rule": "any",
  "rw_rule": "any",
  "client_match": "0.0.0.0/0"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.NFSExportPolicy
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.UpdateNFSExportPolicy(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_DeleteNFSExportPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "delete-nfs_export-policy-missing",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "export_policy" : "export-policy-not-exist"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "delete-nfs-export-policy-present",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "export_policy" : "export-policy-1"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var s tool.NFSExportPolicy
			err = json.Unmarshal([]byte(tt.in), &s)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.DeleteNFSExportPolicy(context.Background(), in, s)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_CreateSnapshotPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-snapshot-policy",
			in: `{
  "cluster_name": "u2",
  "name": "snapshot-policy-1",
  "svm_name": "vs_test",
  "schedule": "daily",
  "count": 2
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.SnapshotPolicy
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateSnapshotPolicy(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_DeleteSnapshotPolicy(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "delete-snapshot-policy-missing",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "name" : "snapshot-policy-not-exist"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "delete-snapshot-policy-present",
			in: `{
  "cluster_name" : "u2",
  "svm_name" : "vs_test",
  "name" : "snapshot-policy-1"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var s tool.SnapshotPolicy
			err = json.Unmarshal([]byte(tt.in), &s)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.DeleteSnapshotPolicy(context.Background(), in, s)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_CreateNFSExportPolicyRule(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-rule-in-non-exist-export-policy",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-not-exist",
  "client": "1.1.1.1/32",
  "ro_rule": "never",
  "rw_rule": "never"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "create-rule-in-export-policy",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-2",
  "client": "1.1.1.1/32",
  "ro_rule": "never",
  "rw_rule": "never"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.NFSExportPolicyRules
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateNFSExportPoliciesRule(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_UpdateNFSExportPolicyRule(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "update-rule-missing",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-not-exist",
  "old_client": "1.1.1.1/32",
  "new_client": "0.0.0.0/0"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "update-rule-present",
			in: `{
  "cluster_name": "u2",
  "export_policy": "export-policy-2",
  "svm_name": "vs_test",
  "old_client": "0.0.0.0/0",
  "new_client": "1.1.1.1/32"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var p tool.NFSExportPolicyRules
			err = json.Unmarshal([]byte(tt.in), &p)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.UpdateNFSExportPoliciesRule(context.Background(), in, p)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_DeleteNFSExportPolicyRule(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "delete-rule-missing",
			in: `{
		 "cluster_name" : "u2",
		 "svm_name" : "vs_test",
		 "export_policy" : "export-policy-not-exist",
		 "client": "1.1.1.1/22"
		}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "delete-rule-present",
			in: `{
		 "cluster_name" : "u2",
		 "svm_name" : "vs_test",
		 "export_policy" : "export-policy-2",
		 "client": "1.1.1.1/32"
		}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var s tool.NFSExportPolicyRules
			err = json.Unmarshal([]byte(tt.in), &s)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.DeleteNFSExportPoliciesRule(context.Background(), in, s)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_CreateCIFSShare(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "create-cifs-share-without-cifs-server",
			in: `{
  "cluster_name": "u2",
  "name": "cifs1",
  "path": "/",
  "svm_name": "vs_test"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "CIFS server doesn't exist",
		},
		{
			name: "create-cifs-share-with-cifs-server",
			in: `{
  "cluster_name": "u2",
  "name": "cifs1",
  "path": "/",
  "svm_name": "vs_test4"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var c tool.CIFSShare
			err = json.Unmarshal([]byte(tt.in), &c)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.CreateCIFSShare(context.Background(), in, c)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_UpdateCIFSShare(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "update-cifs-share-missing",
			in: `{
  "cluster_name": "u2",
  "name": "cifs1-not",
  "path": "//",
  "svm_name": "vs_test"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "update-cifs-share",
			in: `{
  "cluster_name": "u2",
  "name": "cifs1",
  "path": "/",
  "svm_name": "vs_test4"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var c tool.CIFSShare
			err = json.Unmarshal([]byte(tt.in), &c)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.UpdateCIFSShare(context.Background(), in, c)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

func TestApp_DeleteCIFSShare(t *testing.T) {

	tests := []struct {
		name          string
		in            string
		want          *mcp.CallToolResult
		wantErrorLike string
		wantTextLike  string
	}{
		{
			name: "delete-cifs-share-missing",
			in: `{
  "cluster_name": "u2",
  "name": "cifs1",
  "path": "/",
  "svm_name": "vs_test"
}`,
			want:          &mcp.CallToolResult{IsError: true},
			wantErrorLike: "it does not exist",
		},
		{
			name: "delete-cifs-share-present",
			in: `{
  "cluster_name": "u2",
  "name": "cifs1",
  "path": "/",
  "svm_name": "vs_test4"
}`,
			want:         &mcp.CallToolResult{IsError: false},
			wantTextLike: "successful",
		},
	}

	var (
		err error
		cfg *config.ONTAP
	)

	isRecord := os.Getenv("RECORD_HTTP") == "true"
	if isRecord {
		cfg, err = config.ReadConfig("testdata/ontap.yaml")
	} else {
		cfg, err = config.ReadConfig("testdata/replay.yaml")
	}

	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup record/replay directory
			recordDir := filepath.Join("testdata", tt.name)

			// Create base transport with TLS config
			baseTransport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			// Create the appropriate transport
			var transport requests.Transport
			if isRecord {
				transport = &authStrippingRecorder{
					baseTransport: baseTransport,
					recordDir:     recordDir,
				}
			} else {
				// Replay mode: strip auth header to match recorded requests. Otherwise, the hash won't
				// match what was saved.
				transport = &authStrippingTransport{
					inner: reqtest.Replay(recordDir),
				}
			}

			// Create HTTP client with the transport
			testClient := &http.Client{
				Transport: transport,
			}

			// Create app with test HTTP client
			a := NewApp(cfg, Options{TestHTTPClient: testClient}, slog.Default())
			in := &mcp.CallToolRequest{}
			var c tool.CIFSShare
			err = json.Unmarshal([]byte(tt.in), &c)
			assert.Nil(t, err)

			gotCallToolResult, _, err := a.DeleteCIFSShare(context.Background(), in, c)
			if tt.wantErrorLike != "" {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErrorLike))
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, gotCallToolResult.IsError, tt.want.IsError)

			if tt.wantTextLike != "" {
				assert.True(t, strings.Contains((gotCallToolResult.Content[0].(*mcp.TextContent)).Text, tt.wantTextLike))
			}
		})
	}
}

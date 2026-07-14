package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestRefreshJWKSConcurrentSingleflight(t *testing.T) {
	var hits atomic.Int32

	jwksPayload := `{"keys":[{"kty":"RSA","kid":"kid-1","n":"sXchA80BzH2i7US8FGgV2Q4Wzl8Jb0fH0Qf0v2Q0mRwyY1Fh3R1U4nP1kx7t9e0xQ2y9QmMSU4v6Y5f8Yx8k7N7M5Q0f0R4Ujz4jvUQv5N2kWbS5WgQ6iN6K8bV6cQ7x9Qx4xL9o7nJr5xW4i4D8c9Q9e8u8U3X9t2X4n2T7v2Q3R9Q","e":"AQAB"}]}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		// Keep requests in flight long enough to force overlap.
		time.Sleep(75 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jwksPayload))
	}))
	defer ts.Close()

	app := &App{
		logger:     slog.Default(),
		cfg:        &config.ONTAP{},
		jwksURI:    ts.URL,
		jwksKeys:   make(map[string]any),
		httpClient: ts.Client(),
	}

	const workers = 16
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for range workers {
		wg.Go(func() {
			<-start
			errCh <- app.refreshJWKS()
		})
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("refreshJWKS returned error: %v", err)
		}
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("expected exactly 1 upstream JWKS request, got %d", got)
	}

	if _, ok := app.lookupCachedKey("kid-1"); !ok {
		t.Fatal("expected cached key for kid-1 after refresh")
	}
}

func TestRefreshJWKSSingleflightErrorPropagation(t *testing.T) {
	var hits atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer ts.Close()

	app := &App{
		logger:     slog.Default(),
		cfg:        &config.ONTAP{},
		jwksURI:    ts.URL,
		jwksKeys:   make(map[string]any),
		httpClient: ts.Client(),
	}

	const workers = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for range workers {
		wg.Go(func() {
			<-start
			errCh <- app.refreshJWKS()
		})
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err == nil {
			t.Fatal("expected refreshJWKS to return error")
		}
		if got, want := err.Error(), fmt.Sprintf("failed to fetch jwks from %q: status %d", ts.URL, http.StatusBadGateway); got != want {
			t.Fatalf("unexpected error message\nwant: %s\ngot:  %s", want, got)
		}
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("expected exactly 1 upstream JWKS request on error path, got %d", got)
	}
}

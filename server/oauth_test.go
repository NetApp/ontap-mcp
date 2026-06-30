package server

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netapp/ontap-mcp/config"
)

func TestOAuthMiddlewareMissingBearerSetsChallengeFromRequestHost(t *testing.T) {
	app := &App{
		logger: slog.Default(),
		cfg:    &config.ONTAP{},
		scope:  "profile",
	}

	nextCalled := false
	handler := app.OAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://mcp.example.test/", http.NoBody)
	req.Host = "mcp.example.test"
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if nextCalled {
		t.Fatal("next handler should not be called when bearer token is missing")
	}
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}

	wantHeader := `Bearer resource_metadata="http://mcp.example.test/.well-known/oauth-protected-resource", scope="profile"`
	if got := resp.Header().Get("WWW-Authenticate"); got != wantHeader {
		t.Fatalf("unexpected WWW-Authenticate header\nwant: %s\ngot:  %s", wantHeader, got)
	}
}

func TestResourceMetadataURLUsesHTTPSForTLSRequests(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodGet, "https://secure.example.test/", http.NoBody)
	req.Host = "secure.example.test"
	req.TLS = &tls.ConnectionState{}

	want := "https://secure.example.test/.well-known/oauth-protected-resource"
	if got := app.resourceMetadataURL(req); got != want {
		t.Fatalf("unexpected metadata URL\nwant: %s\ngot:  %s", want, got)
	}
}

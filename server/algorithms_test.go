package server

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/netapp/ontap-mcp/config"
)

func TestAlgFromJWK(t *testing.T) {
	tests := []struct {
		name    string
		key     jwkKey
		wantAlg string
		wantOK  bool
	}{
		{"explicit alg wins", jwkKey{Kty: "RSA", Alg: "ps256"}, "PS256", true},
		{"rsa defaults to rs256", jwkKey{Kty: "RSA"}, "RS256", true},
		{"ec p-256", jwkKey{Kty: "EC", Crv: "P-256"}, "ES256", true},
		{"ec p-384", jwkKey{Kty: "EC", Crv: "P-384"}, "ES384", true},
		{"ec p-521", jwkKey{Kty: "EC", Crv: "P-521"}, "ES512", true},
		{"okp ed25519", jwkKey{Kty: "OKP", Crv: "Ed25519"}, "EDDSA", true},
		{"unknown ec curve", jwkKey{Kty: "EC", Crv: "P-999"}, "", false},
		{"unknown kty", jwkKey{Kty: "oct"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAlg, gotOK := algFromJWK(tt.key)
			if gotAlg != tt.wantAlg || gotOK != tt.wantOK {
				t.Fatalf("algFromJWK(%+v) = (%q, %v); want (%q, %v)", tt.key, gotAlg, gotOK, tt.wantAlg, tt.wantOK)
			}
		})
	}
}

func TestNormalizeAlgs(t *testing.T) {
	tests := []struct {
		name string
		in   config.StringSlice
		want []string
	}{
		{"empty", nil, []string{}},
		{"single lowercased", config.StringSlice{"rs256"}, []string{"RS256"}},
		{"list with spaces", config.StringSlice{" rs256 ", "ES256"}, []string{"RS256", "ES256"}},
		{"dedupe", config.StringSlice{"RS256", "rs256", ""}, []string{"RS256"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAlgs(tt.in)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("normalizeAlgs(%v) = %v; want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestAllowedAlgsOverrideTakesPrecedence(t *testing.T) {
	app := &App{
		algOverride: []string{"ES256"},
		derivedAlgs: map[string]bool{"RS256": true},
	}
	if got := app.allowedAlgs(); !slices.Equal(got, []string{"ES256"}) {
		t.Fatalf("allowedAlgs() = %v; want [ES256]", got)
	}
}

func TestAllowedAlgsFallsBackToDerived(t *testing.T) {
	app := &App{
		derivedAlgs: map[string]bool{"RS256": true},
	}
	if got := app.allowedAlgs(); !slices.Equal(got, []string{"RS256"}) {
		t.Fatalf("allowedAlgs() = %v; want [RS256]", got)
	}
}

func TestValidMethodsCanonicalizesEdDSA(t *testing.T) {
	app := &App{algOverride: []string{"EDDSA", "RS256"}}
	got := app.validMethods()
	if !slices.Contains(got, "EdDSA") || slices.Contains(got, "EDDSA") {
		t.Fatalf("validMethods() = %v; expected canonical EdDSA", got)
	}
}

func rsaJWKS(t *testing.T, alg string, pub *rsa.PublicKey) string {
	t.Helper()
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	key := jwkKey{
		Kty: "RSA",
		Kid: "rsa-1",
		Alg: alg,
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}
	body, err := json.Marshal(jwksResponse{Keys: []jwkKey{key}})
	if err != nil {
		t.Fatalf("failed to marshal jwks: %v", err)
	}
	return string(body)
}

func newJWKSApp(t *testing.T, payload string, override []string) *App {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(ts.Close)

	return &App{
		logger:      slog.Default(),
		cfg:         &config.ONTAP{},
		jwksURI:     ts.URL,
		jwksKeys:    make(map[string]any),
		derivedAlgs: make(map[string]bool),
		keyAlg:      make(map[string]string),
		httpClient:  ts.Client(),
		audience:    "mcp-aud",
		issuer:      "https://issuer.test",
		algOverride: override,
	}
}

func signRSAToken(t *testing.T, key *rsa.PrivateKey, method jwt.SigningMethod, kid, aud, iss string) string {
	t.Helper()
	token := jwt.NewWithClaims(method, jwt.MapClaims{
		"aud": aud,
		"iss": iss,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func TestFetchAndCacheJWKSDerivesAlgsAndKeyBinding(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	// JWK carries an explicit alg of RS384.
	app := newJWKSApp(t, rsaJWKS(t, "RS384", &key.PublicKey), nil)
	if err := app.refreshJWKS(); err != nil {
		t.Fatalf("refreshJWKS returned error: %v", err)
	}

	if got := app.allowedAlgs(); !slices.Equal(got, []string{"RS384"}) {
		t.Fatalf("derived allowedAlgs() = %v; want [RS384]", got)
	}
	if alg, ok := app.lookupKeyAlg("rsa-1"); !ok || alg != "RS384" {
		t.Fatalf("lookupKeyAlg(rsa-1) = (%q, %v); want (RS384, true)", alg, ok)
	}
}

func TestVerifyBearerTokenDerivedHappyPath(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	// JWK without an explicit alg -> derived RS256.
	app := newJWKSApp(t, rsaJWKS(t, "", &key.PublicKey), nil)
	if err := app.refreshJWKS(); err != nil {
		t.Fatalf("refreshJWKS returned error: %v", err)
	}

	token := signRSAToken(t, key, jwt.SigningMethodRS256, "rsa-1", "mcp-aud", "https://issuer.test")
	if _, err := app.verifyBearerToken(t.Context(), token, nil); err != nil {
		t.Fatalf("verifyBearerToken rejected a valid RS256 token: %v", err)
	}
}

func TestVerifyBearerTokenRejectsUnlistedAlg(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	// Permit only ES256, but present an RS256 token.
	app := newJWKSApp(t, rsaJWKS(t, "", &key.PublicKey), []string{"ES256"})

	token := signRSAToken(t, key, jwt.SigningMethodRS256, "rsa-1", "mcp-aud", "https://issuer.test")
	if _, err := app.verifyBearerToken(t.Context(), token, nil); err == nil {
		t.Fatal("verifyBearerToken accepted a token with an unlisted algorithm")
	}
}

func TestVerifyBearerTokenPerKeyBindingMismatch(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	// JWK advertises RS384; override permits both RS256 and RS384 so the
	// allow-list passes but the per-key binding must reject the RS256 token.
	app := newJWKSApp(t, rsaJWKS(t, "RS384", &key.PublicKey), []string{"RS256", "RS384"})

	token := signRSAToken(t, key, jwt.SigningMethodRS256, "rsa-1", "mcp-aud", "https://issuer.test")
	if _, err := app.verifyBearerToken(t.Context(), token, nil); err == nil {
		t.Fatal("verifyBearerToken accepted a token whose alg differs from the JWK alg")
	}
}

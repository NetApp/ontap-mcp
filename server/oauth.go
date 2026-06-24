package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/netapp/ontap-mcp/config"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

func (a *App) OAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var scopes []string
		if a.scope != "" {
			scopes = []string{a.scope}
		}
		opts := &auth.RequireBearerTokenOptions{
			Scopes:              scopes,
			ResourceMetadataURL: a.resourceMetadataURL(r),
		}
		auth.RequireBearerToken(a.verifyBearerToken, opts)(next).ServeHTTP(w, r)
	})
}

func (a *App) verifyBearerToken(_ context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
	token, err := jwt.Parse(tokenString, a.keyFunc, jwt.WithValidMethods(a.validMethods()), jwt.WithLeeway(60*time.Second))
	if err != nil {
		slog.Error("failed to parse token", slog.Any("err", err))
		return nil, fmt.Errorf("failed to parse token: %w", errors.Join(auth.ErrInvalidToken, err))
	}

	if !token.Valid {
		slog.Error("token is invalid")
		return nil, auth.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		slog.Error("claim type invalid")
		return nil, fmt.Errorf("claim type invalid: %w", auth.ErrInvalidToken)
	}

	if !a.validateAudience(claims) {
		slog.Error("audience invalid")
		return nil, fmt.Errorf("audience invalid: %w", auth.ErrInvalidToken)
	}

	if !a.validateIssuer(claims) {
		slog.Error("issuer invalid")
		return nil, fmt.Errorf("issuer invalid: %w", auth.ErrInvalidToken)
	}

	if !a.validateScope(claims) {
		slog.Error("scope insufficient")
		return nil, fmt.Errorf("scope insufficient: %w", auth.ErrInvalidToken)
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		slog.Error("token expiration missing or invalid", slog.Any("err", err))
		return nil, fmt.Errorf("token expiration missing or invalid: %w", auth.ErrInvalidToken)
	}

	return &auth.TokenInfo{
		Scopes:     []string{a.scope},
		Expiration: exp.Time,
		Extra: map[string]any{
			"claims": claims,
		},
	}, nil
}

func (a *App) validateAudience(claims jwt.MapClaims) bool {
	aud, ok := claims["aud"]
	if !ok {
		return false
	}

	// aud can be a string or array of strings
	switch v := aud.(type) {
	case string:
		return v == a.audience
	case []any:
		for _, as := range v {
			if audStr, ok := as.(string); ok && audStr == a.audience {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (a *App) validateIssuer(claims jwt.MapClaims) bool {
	iss, ok := claims["iss"].(string)
	if !ok {
		return false
	}
	return strings.TrimSuffix(iss, "/") == strings.TrimSuffix(a.issuer, "/")
}

func (a *App) validateScope(claims jwt.MapClaims) bool {
	if a.scope == "" {
		return true
	}
	scope, _ := claims["scope"].(string)
	return slices.Contains(strings.Fields(scope), a.scope)
}

func (a *App) resourceMetadataURL(r *http.Request) string {
	metadataURL := "http://" + r.Host + "/.well-known/oauth-protected-resource"
	if r.TLS != nil {
		metadataURL = "https://" + r.Host + "/.well-known/oauth-protected-resource"
	}
	return metadataURL
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		slog.Debug("", slog.String("method", r.Method), slog.String("urlPath", r.URL.Path), slog.String("RemoteAddr", r.RemoteAddr))

		if r.Method == http.MethodPost && r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				slog.Error("Error reading body", slog.Any("err", err))
			} else {
				slog.Debug("", slog.String("body", string(bodyBytes)))
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		next.ServeHTTP(w, r)
		slog.Debug("request finished", slog.Any("duration", time.Since(start)))
	})
}

func (a *App) keyFunc(token *jwt.Token) (any, error) {
	alg, _ := token.Header["alg"].(string)
	alg = strings.ToUpper(strings.TrimSpace(alg))

	allowed := a.allowedAlgs()
	if len(allowed) == 0 {
		return nil, errors.New("no permitted signing algorithms available")
	}
	if !slices.Contains(allowed, alg) {
		return nil, fmt.Errorf("unexpected token alg %q; permitted: %v", alg, allowed)
	}

	if err := getSigningKeyAndMethod(token, alg); err != nil {
		return nil, err
	}

	kid, _ := token.Header["kid"].(string)
	aud, _ := token.Claims.GetAudience()
	if !slices.Contains(aud, a.audience) {
		return nil, fmt.Errorf("unexpected audience (aud) %v; expected %q", aud, a.audience)
	}

	// Resolve the key first; this also lazily refreshes the JWKS so the
	// advertised per-key alg is available for the binding check below.
	key, err := a.getAnyPublicKey(kid)
	if err != nil {
		return nil, err
	}

	// When the matching JWK advertises an `alg`, bind the token to it to prevent
	// algorithm-confusion attacks across keys in the JWKS.
	if keyAlg, ok := a.lookupKeyAlg(kid); ok && keyAlg != alg {
		return nil, fmt.Errorf("token alg %q does not match jwk alg %q for kid %q", alg, keyAlg, kid)
	}

	return key, nil
}

// allowedAlgs returns the permitted signing algorithms: the configured override
// when set, otherwise the set derived from the issuer's JWKS.
func (a *App) allowedAlgs() []string {
	if len(a.algOverride) > 0 {
		return a.algOverride
	}
	a.jwksMu.RLock()
	defer a.jwksMu.RUnlock()
	algs := make([]string, 0, len(a.derivedAlgs))
	for alg := range a.derivedAlgs {
		algs = append(algs, alg)
	}
	return algs
}

// validMethods returns the permitted algorithms as canonical JWT signing
// method names (e.g. EdDSA rather than the internal upper-cased EDDSA) for use
// with jwt.WithValidMethods.
func (a *App) validMethods() []string {
	algs := a.allowedAlgs()
	methods := make([]string, len(algs))
	for i, alg := range algs {
		if alg == "EDDSA" {
			methods[i] = "EdDSA"
			continue
		}
		methods[i] = alg
	}
	return methods
}

// lookupKeyAlg returns the alg advertised by the JWK with the given kid, if any.
func (a *App) lookupKeyAlg(kid string) (string, bool) {
	if kid == "" {
		return "", false
	}
	a.jwksMu.RLock()
	defer a.jwksMu.RUnlock()
	alg, ok := a.keyAlg[kid]
	return alg, ok
}

func (a *App) getAnyPublicKey(kid string) (any, error) {
	if key, ok := a.lookupCachedKey(kid); ok {
		return key, nil
	}

	cacheFresh := a.isJWKSCacheFresh()
	if cacheFresh {
		if kid == "" {
			return nil, errors.New("jwt header is missing kid and jwks did not provide a single unambiguous key")
		}
		return nil, fmt.Errorf("no jwks key found for kid %q", kid)
	}

	if err := a.refreshJWKS(); err != nil {
		return nil, err
	}

	if key, ok := a.lookupCachedKey(kid); ok {
		return key, nil
	}

	if kid == "" {
		return nil, errors.New("jwt header is missing kid and jwks did not provide a single unambiguous key")
	}
	return nil, fmt.Errorf("no jwks key found for kid %q", kid)
}

func (a *App) lookupCachedKey(kid string) (any, bool) {
	a.jwksMu.RLock()
	defer a.jwksMu.RUnlock()

	if time.Since(a.jwksFetched) > jwksCacheTTL || len(a.jwksKeys) == 0 {
		return nil, false
	}
	if kid != "" {
		key, ok := a.jwksKeys[kid]
		return key, ok
	}
	if len(a.jwksKeys) == 1 {
		for _, key := range a.jwksKeys {
			return key, true
		}
	}
	return nil, false
}

func (a *App) refreshJWKS() error {
	if a.isJWKSCacheFresh() {
		return nil
	}

	_, err, _ := a.jwksRefresh.Do("jwks_refresh", func() (any, error) {
		if a.isJWKSCacheFresh() {
			return nil, nil
		}
		if err := a.fetchAndCacheJWKS(); err != nil {
			return nil, err
		}
		return nil, nil
	})

	return err
}

func (a *App) isJWKSCacheFresh() bool {
	a.jwksMu.RLock()
	defer a.jwksMu.RUnlock()

	return time.Since(a.jwksFetched) <= jwksCacheTTL && len(a.jwksKeys) > 0
}

func (a *App) fetchAndCacheJWKS() error {
	req, err := http.NewRequest(http.MethodGet, a.jwksURI, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create jwks request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch jwks from %q: %w", a.jwksURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch jwks from %q: status %d", a.jwksURI, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read jwks response: %w", err)
	}

	var doc jwksResponse
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("failed to parse jwks response: %w", err)
	}

	keys := make(map[string]any)
	keyAlg := make(map[string]string)
	derivedAlgs := make(map[string]bool)
	anonymousIndex := 0
	for _, k := range doc.Keys {
		key, err := publicKeyFromJWK(k)
		if err != nil {
			a.logger.Warn("ignoring invalid jwk", slog.String("kid", k.Kid), slog.String("error", err.Error()))
			continue
		}
		kid := strings.TrimSpace(k.Kid)
		if kid == "" {
			kid = fmt.Sprintf("__nokid_%d", anonymousIndex)
			anonymousIndex++
		}
		keys[kid] = key

		if alg, ok := algFromJWK(k); ok {
			derivedAlgs[alg] = true
			if strings.TrimSpace(k.Alg) != "" {
				keyAlg[kid] = alg
			}
		} else {
			a.logger.Warn("could not derive signing algorithm from jwk; set McpAuth.alg to restrict explicitly",
				slog.String("kid", kid), slog.String("kty", k.Kty), slog.String("crv", k.Crv))
		}
	}

	if len(keys) == 0 {
		return errors.New("jwks does not contain any usable keys")
	}

	a.jwksMu.Lock()
	a.jwksKeys = keys
	a.keyAlg = keyAlg
	a.derivedAlgs = derivedAlgs
	a.jwksFetched = time.Now()
	a.jwksMu.Unlock()

	return nil
}

// algFromJWK derives the signing algorithm a JWK is intended for. It prefers the
// key's explicit `alg`, otherwise infers it from the key type and curve. RSA
// keys without an explicit `alg` default to RS256; operators needing PS* or a
// different algorithm must set McpAuth.alg explicitly.
func algFromJWK(k jwkKey) (string, bool) {
	if alg := strings.ToUpper(strings.TrimSpace(k.Alg)); alg != "" {
		return alg, true
	}
	switch strings.ToUpper(strings.TrimSpace(k.Kty)) {
	case "RSA":
		return "RS256", true
	case "EC":
		switch strings.ToUpper(strings.TrimSpace(k.Crv)) {
		case "P-256":
			return "ES256", true
		case "P-384":
			return "ES384", true
		case "P-521":
			return "ES512", true
		}
	case "OKP":
		if strings.EqualFold(strings.TrimSpace(k.Crv), "Ed25519") {
			return "EDDSA", true
		}
	}
	return "", false
}

// normalizeAlgs validates and upper-cases the configured algorithm override.
func normalizeAlgs(in config.StringSlice) []string {
	out := make([]string, 0, len(in))
	for _, alg := range in {
		alg = strings.ToUpper(strings.TrimSpace(alg))
		if alg == "" {
			continue
		}
		if !slices.Contains(out, alg) {
			out = append(out, alg)
		}
	}
	return out
}

func publicKeyFromJWK(k jwkKey) (any, error) {
	switch strings.ToUpper(k.Kty) {
	case "RSA":
		if k.N == "" || k.E == "" {
			return nil, errors.New("RSA key missing required fields (n or e)")
		}
		return rsaPublicKeyFromJWK(k)
	case "EC":
		return ecdsaPublicKeyFromJWK(k)
	case "OKP":
		return edDSAPublicKeyFromJWK(k)
	default:
		return nil, fmt.Errorf("unsupported key type %q", k.Kty)
	}
}

func rsaPublicKeyFromJWK(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("invalid modulus n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("invalid exponent e: %w", err)
	}

	if len(eBytes) == 0 {
		return nil, errors.New("empty exponent e")
	}

	eInt := 0
	for _, b := range eBytes {
		eInt = (eInt << 8) | int(b)
	}
	if eInt <= 0 {
		return nil, errors.New("invalid exponent e")
	}

	n := new(big.Int).SetBytes(nBytes)
	if n.Sign() <= 0 {
		return nil, errors.New("invalid modulus n")
	}

	return &rsa.PublicKey{N: n, E: eInt}, nil
}

func ecdsaPublicKeyFromJWK(k jwkKey) (*ecdsa.PublicKey, error) {
	if k.X == "" || k.Y == "" {
		return nil, errors.New("EC key missing required fields (x or y)")
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("invalid x coordinate: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid y coordinate: %w", err)
	}

	var curve elliptic.Curve
	switch strings.ToUpper(k.Crv) {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve %q", k.Crv)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)
	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("EC point is not on the configured curve")
	}

	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func edDSAPublicKeyFromJWK(k jwkKey) (ed25519.PublicKey, error) {
	if !strings.EqualFold(k.Crv, "Ed25519") {
		return nil, fmt.Errorf("unsupported OKP curve %q", k.Crv)
	}
	if k.X == "" {
		return nil, errors.New("OKP key missing required field x")
	}

	keyBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("invalid OKP x coordinate: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size %d", len(keyBytes))
	}

	return ed25519.PublicKey(keyBytes), nil
}

func getSigningKeyAndMethod(token *jwt.Token, algoUpper string) error {
	var methodDetected jwt.SigningMethod

	switch {
	// --- Asymmetric RSA ---
	case strings.HasPrefix(algoUpper, "RS"):
		switch algoUpper {
		case "RS256":
			methodDetected = jwt.SigningMethodRS256
		case "RS384":
			methodDetected = jwt.SigningMethodRS384
		case "RS512":
			methodDetected = jwt.SigningMethodRS512
		}

	// --- Asymmetric RSA-PSS ---
	case strings.HasPrefix(algoUpper, "PS"):
		switch algoUpper {
		case "PS256":
			methodDetected = jwt.SigningMethodPS256
		case "PS384":
			methodDetected = jwt.SigningMethodPS384
		case "PS512":
			methodDetected = jwt.SigningMethodPS512
		}

	// --- Asymmetric Elliptic Curve (ECDSA) ---
	case strings.HasPrefix(algoUpper, "ES"):
		switch algoUpper {
		case "ES256":
			methodDetected = jwt.SigningMethodES256
		case "ES384":
			methodDetected = jwt.SigningMethodES384
		case "ES512":
			methodDetected = jwt.SigningMethodES512
		}

	// --- Asymmetric EdDSA (Edwards-curve) ---
	case strings.HasPrefix(algoUpper, "ED"):
		if algoUpper == "EDDSA" {
			methodDetected = jwt.SigningMethodEdDSA
		}
	}

	if methodDetected == nil {
		return fmt.Errorf("unsupported signing algorithm %q", algoUpper)
	}

	if token.Method.Alg() != methodDetected.Alg() {
		return fmt.Errorf("unexpected signing method in token header: %v, expected: %v", token.Method.Alg(), methodDetected.Alg())
	}

	if methodDetected == jwt.SigningMethodEdDSA {
		// Detect Method Type Interface: Ensure it resolves to EdDSA's internal structure
		_, isEdDSA := token.Method.(*jwt.SigningMethodEd25519)
		if !isEdDSA {
			return errors.New("token method type assertion mismatch for EdDSA")
		}
	}

	return nil
}

func getJwksURI(issuer string, httpClient *http.Client, logger *slog.Logger) (string, error) {
	if strings.HasPrefix(issuer, "http://") {
		logger.Warn("Issuer starts with http and not https", slog.String("issuer", issuer))
	}
	discoveryEndpoint := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, discoveryEndpoint, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch config, status: %d", resp.StatusCode)
	}

	var OIDCconfigData OIDCConfig
	if err := json.NewDecoder(resp.Body).Decode(&OIDCconfigData); err != nil {
		return "", err
	}

	jwksURI := OIDCconfigData.JwksURI
	if u, e := url.Parse(jwksURI); e != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("jwks_uri must be an http(s) URL, got %q", jwksURI)
	}

	return jwksURI, nil
}

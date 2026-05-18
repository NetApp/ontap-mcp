package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type OAuthConfig struct {
	AuthServerURL string
	JwksURL       string
	ResourceURL   string
	jwks          keyfunc.Keyfunc
}

func (c *OAuthConfig) InitJWKS() error {
	jwks, err := keyfunc.NewDefault([]string{c.JwksURL})
	if err != nil {
		return fmt.Errorf("failed to create client of JWKS: %w", err)
	}
	c.jwks = jwks
	slog.Info("Initialized JWKS", slog.String("JwksURL", c.JwksURL))
	return nil
}

func (c *OAuthConfig) OAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			c.sendUnauthorized(w, r)
			return
		}

		// Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.sendUnauthorized(w, r)
			return
		}

		// Validate JWT token
		token, err := jwt.Parse(tokenString, c.jwks.Keyfunc, jwt.WithValidMethods([]string{"RS256"}))
		if err != nil {
			slog.Error("failed to parse token", slog.Any("err", err))
			c.sendUnauthorized(w, r)
			return
		}

		if !token.Valid {
			slog.Error("token is invalid")
			c.sendUnauthorized(w, r)
			return
		}

		// Get claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.Error("claim type invalid")
			c.sendUnauthorized(w, r)
			return
		}

		slog.Debug("=== JWT Access Token Debug ===")
		slog.Debug("", slog.String("token", tokenString))
		claimsJSON, _ := json.MarshalIndent(claims, "", "  ")
		slog.Debug("", slog.String("claim", string(claimsJSON)))
		slog.Debug("===============================")

		// Validate audience
		if !c.validateAudience(claims) {
			slog.Error("audience invalid")
			c.sendUnauthorized(w, r)
			return
		}

		// Validate issuer
		if !c.validateIssuer(claims) {
			slog.Error("issuer invalid")
			c.sendUnauthorized(w, r)
			return
		}

		// Validate expiration
		// Note: jwt.Parse already validates exp by default, but we explicitly check here for clarity
		if !c.validateExpiration(claims) {
			slog.Error("token has been expired")
			c.sendUnauthorized(w, r)
			return
		}

		// Validate scope
		if !c.validateScope(claims) {
			slog.Error("scope insufficient")
			c.sendUnauthorized(w, r)
			return
		}

		// Authorization successful
		next.ServeHTTP(w, r)
	})
}

func (c *OAuthConfig) validateAudience(claims jwt.MapClaims) bool {
	aud, ok := claims["aud"]
	if !ok {
		return false
	}

	// aud can be a string or array of strings
	switch v := aud.(type) {
	case string:
		return v == c.ResourceURL
	case []any:
		for _, a := range v {
			if audStr, ok := a.(string); ok && audStr == c.ResourceURL {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (c *OAuthConfig) validateIssuer(claims jwt.MapClaims) bool {
	iss, ok := claims["iss"].(string)
	if !ok {
		return false
	}
	return iss == c.AuthServerURL
}

func (c *OAuthConfig) validateExpiration(claims jwt.MapClaims) bool {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false
	}
	// Allow 60 seconds of clock skew
	return time.Now().Unix() < int64(exp)+60
}

func (c *OAuthConfig) validateScope(claims jwt.MapClaims) bool {
	scope, ok := claims["scope"].(string)
	if !ok {
		return false
	}
	// Scope is a space-separated string (OAuth 2.0 standard)
	// Check if "mcp:tools" is present
	s := strings.Split(scope, " ")
	return slices.Contains(s, "mcp:tools")
}

func (c *OAuthConfig) sendUnauthorized(w http.ResponseWriter, _ *http.Request) {
	metadataURL := c.ResourceURL + "/.well-known/oauth-protected-resource"
	w.Header().Set("WWW-Authenticate",
		fmt.Sprintf(`Bearer resource_metadata="%q", scope="openid profile email"`, metadataURL))
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
)

func (a *App) OAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.sendUnauthorized(w, r)
			return
		}

		// Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader || tokenString == "" {
			a.sendUnauthorized(w, r)
			return
		}

		// Validate JWT token
		token, err := jwt.Parse(tokenString, a.keyFunc, jwt.WithValidMethods([]string{a.algUsed}))
		if err != nil {
			slog.Error("failed to parse token", slog.Any("err", err))
			a.sendUnauthorized(w, r)
			return
		}

		if !token.Valid {
			slog.Error("token is invalid")
			a.sendUnauthorized(w, r)
			return
		}

		// Get claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.Error("claim type invalid")
			a.sendUnauthorized(w, r)
			return
		}

		slog.Debug("=== JWT Access Token Debug ===")
		slog.Debug("", slog.String("token", tokenString))
		claimsJSON, _ := json.MarshalIndent(claims, "", "  ")
		slog.Debug("", slog.String("claim", string(claimsJSON)))
		slog.Debug("===============================")

		// Validate audience
		if !a.validateAudience(claims) {
			slog.Error("audience invalid")
			a.sendUnauthorized(w, r)
			return
		}

		// Validate issuer
		if !a.validateIssuer(claims) {
			slog.Error("issuer invalid")
			a.sendUnauthorized(w, r)
			return
		}

		// Validate expiration
		// Note: jwt.Parse already validates exp by default, but we explicitly check here for clarity
		if !a.validateExpiration(claims) {
			slog.Error("token has been expired")
			a.sendUnauthorized(w, r)
			return
		}

		// Validate scope, TODO: scope should be expected as mcp:tools else the validation of scope would be failed ?
		if !a.validateScope(claims) {
			slog.Error("scope insufficient")
			a.sendUnauthorized(w, r)
			return
		}

		// Authorization successful
		next.ServeHTTP(w, r)
	})
}

func (a *App) validateAudience(claims jwt.MapClaims) bool {
	aud, ok := claims["aud"]
	if !ok {
		return false
	}

	// aud can be a string or array of strings
	switch v := aud.(type) {
	case string:
		return v == a.audienceRequired
	case []any:
		for _, as := range v {
			if audStr, ok := as.(string); ok && audStr == a.audienceRequired {
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
	return iss == a.issuer
}

func (a *App) validateExpiration(claims jwt.MapClaims) bool {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false
	}
	// Allow 60 seconds of clock skew
	return time.Now().Unix() < int64(exp)+60
}

func (a *App) validateScope(claims jwt.MapClaims) bool {
	scope, ok := claims["scope"].(string)
	if !ok {
		return false
	}
	// Check if "mcp:tools" is present
	s := strings.Split(scope, " ")
	return slices.Contains(s, "mcp:tools")
}

func (a *App) sendUnauthorized(w http.ResponseWriter, _ *http.Request) {
	metadataURL := a.issuer + "/.well-known/oauth-protected-resource"
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

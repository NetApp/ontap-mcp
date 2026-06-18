package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"io"
	"log/slog"
	"net/http"
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
	token, err := jwt.Parse(tokenString, a.keyFunc, jwt.WithValidMethods([]string{a.algUsed}), jwt.WithLeeway(60*time.Second))
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
		Expiration: exp.Add(60 * time.Second),
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

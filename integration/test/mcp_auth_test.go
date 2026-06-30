package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

type jwkFile struct {
	PublicKeys struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	} `json:"public_keys"`
}

type mcpConfigFile struct {
	Servers map[string]struct {
		Headers map[string]string `json:"headers"`
	} `json:"servers"`
}

func TestMCPAuthBearer(t *testing.T) {
	SkipIfMissing(t, CheckTools)
	keysFixture := loadKeysFixture(t, "mcp_auth_keys.json")
	keyByKid := buildPublicKeyMap(t, keysFixture)

	tests := []struct {
		name             string
		content          string
		wantErr          bool
		expectExpired    bool
		expectBadSig     bool
		expectBadSigAlgo string
	}{
		{
			name: "successful",
			content: `{
				"servers": {
					"ontap-mcp": {
						"type": "http",
						"url": "http://localhost:8080",
						"headers": {
                            "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJKRkZ5c25ub3VGb1FUUE14VDdwZDc1NWVkYVptUDdHbmQ4RnRRQkFHRC1rIn0.eyJleHAiOjE3ODA1NzYyMjgsImlhdCI6MTc4MDU2OTAyOCwianRpIjoidHJydGNjOmM4NjA5MDg4LTM3YjEtOGFhYS0wNTgyLTQ2Y2FmZTg4ZGFlNSIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6OTA5MC9yZWFsbXMvd29yayIsImF1ZCI6ImFjY291bnQiLCJzdWIiOiJiZmY2Y2RmMS1iMDkyLTRhMzUtODdiZS01ZTI0MjYxNWUzNjkiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJ3b3JrLWNsaWVudCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJkZWZhdWx0LXJvbGVzLXdvcmsiLCJvZmZsaW5lX2FjY2VzcyIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJjbGllbnRIb3N0IjoiMTkyLjE2OC42NS4xIiwicHJlZmVycmVkX3VzZXJuYW1lIjoic2VydmljZS1hY2NvdW50LXdvcmstY2xpZW50IiwiY2xpZW50QWRkcmVzcyI6IjE5Mi4xNjguNjUuMSIsImNsaWVudF9pZCI6IndvcmstY2xpZW50In0.gogxEOX64U9p4KCGoaRPjbjCb3RFpVvh4mZtKa0oH7tJ6lwGr_cpoUvEIK8-kivtU5SBm-tWTC2n9lhLUpayP7hAK82pAZzlOOzlQChWIGC0VKEdSfibL8rTr9E0ADV_pmTco9E6xEIDat0xfZB-NjkXm17bS5qNxp8GeuYQA6G_L6CCGDoZLk6HWM1ZGyEb4qHO1qcFGBiNkqhSJf9wrP7kh2ihjMxcA46C1P7Qm_rSh9Zy50F6JcuWdg360CtqUvVzOn3r34NWJ8L6MXaOm1HAcgFF02YCcOyZsXuKh1JiMX3b2gj-Ccs1GqUH0bEnyzrxbpVpWt2nenUzBN3kDA"
						}
					}
				}
			}`,
			wantErr: false,
		},
		{
			name: "token expired",
			content: `{
				"servers": {
					"ontap-mcp": {
						"type": "http",
						"url": "http://localhost:8080",
						"headers": {
                            "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJKRkZ5c25ub3VGb1FUUE14VDdwZDc1NWVkYVptUDdHbmQ4RnRRQkFHRC1rIn0.eyJleHAiOjE3ODAzODk2ODEsImlhdCI6MTc4MDM4OTM4MSwianRpIjoidHJydGNjOmZiNmNjOTc5LTFmZjQtYWU3NS1iNzhkLTRiMWU5NDQ0NzM2ZCIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6OTA5MC9yZWFsbXMvd29yayIsImF1ZCI6ImFjY291bnQiLCJzdWIiOiJiZmY2Y2RmMS1iMDkyLTRhMzUtODdiZS01ZTI0MjYxNWUzNjkiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJ3b3JrLWNsaWVudCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJkZWZhdWx0LXJvbGVzLXdvcmsiLCJvZmZsaW5lX2FjY2VzcyIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJjbGllbnRIb3N0IjoiMTkyLjE2OC42NS4xIiwicHJlZmVycmVkX3VzZXJuYW1lIjoic2VydmljZS1hY2NvdW50LXdvcmstY2xpZW50IiwiY2xpZW50QWRkcmVzcyI6IjE5Mi4xNjguNjUuMSIsImNsaWVudF9pZCI6IndvcmstY2xpZW50In0.u-fHVYcSN4Kge0c8bgS7QUrekEwmW0vMP4yG45v6zsq4gjz8wbs01E-2SCvwpNRMhbCvzaSZUCZ6sISH6xCmSqkg9dVgaheGC-2l2_6Z0DNy9Frddgmuam4pl4BL0VoHP_k9O_sJ8sCkdjiqT4TSMpug8nNKknw98xWd2xZqJyKGwE-_dEE6NW8jKvhob3gSVhU-khyLCgnIdIKMZPe-Ssum751I3GJ-CK94e3DBqQ_maOCh0-tH2sPxfdQmXjxsfGudClC9ueZbAFM0l-ZCC_jSug2-byNNjMPW3nzMOpnU0jktgE63vvO-h4n7pQrv_687izoPd8CQI-3cnd-cXw"
						}
					}
				}
			}`,
			wantErr:       true,
			expectExpired: true,
		},
		{
			name: "wrong token",
			content: `{
				"servers": {
					"ontap-mcp": {
						"type": "http",
						"url": "http://localhost:8080",
						"headers": {
                            "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJKRkZ5c25ub3VGb1FUUE14VDdwZDc1NWVkYVptUDdHbmQ4RnRRQkFHRC1rIn0.eyJleHAiOjE3ODA1NzYyMjgsImlhdCI6MTc4MDU2OTAyOCwianRpIjoidHJydGNjOmM4NjA5MDg4LTM3YjEtOGFhYS0wNTgyLTQ2Y2FmZTg4ZGFlNSIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6OTA5MC9yZWFsbXMvd29yayIsImF1ZCI6ImFjY291bnQiLCJzdWIiOiJiZmY2Y2RmMS1iMDkyLTRhMzUtODdiZS01ZTI0MjYxNWUzNjkiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJ3b3JrLWNsaWVudCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJkZWZhdWx0LXJvbGVzLXdvcmsiLCJvZmZsaW5lX2FjY2VzcyIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJjbGllbnRIb3N0IjoiMTkyLjE2OC42NS4xIiwicHJlZmVycmVkX3VzZXJuYW1lIjoic2VydmljZS1hY2NvdW50LXdvcmstY2xpZW50IiwiY2xpZW50QWRkcmVzcyI6IjE5Mi4xNjguNjUuMSIsImNsaWVudF9pZCI6IndvcmstY2xpZW50In0.gogxEOX64U9p4KCGoaRPjbjCb3RFpVvh4mZtKa0oH7tJ6lwGr_cpoUvEIK8-kivtU5SBm-tWTC2n9lhLUpayP7hAK82pAZzlOOzlQChWIGC0VKEdSfibL8rTr9E0ADV_pmTco9E6xEIDat0xfZB-NjkXm17bS5qNxp8GeuYQA6G_L6CCGDoZLk6HWM1ZGyEb4qHO1qcFGBiNkqhSJf9wrP7kh2ihjMxcA46C1P7Qm_rSh9Zy50F6JcuWdg360CtqUvVzOn3r34NWJ8L6MXaOm1HAcgFF02YCcOyZsXuKh1JiMX3b2gj-Ccs1GqUH0bEnyzrxbpVpWt2nenUzBN3kqwe"
						}
					}
				}
			}`,
			wantErr:      true,
			expectBadSig: true,
		},
		{
			name: "wrong algorithm",
			content: `{
				"servers": {
					"ontap-mcp": {
						"type": "http",
						"url": "http://localhost:8080",
						"headers": {
                            "Authorization": "Bearer eyJhbGciOiJFUzM4NCIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJIaEprQUpYM3FkOEdHb0c1cktReEtzd3JaaUgxTFk5eWtfcHVNNFdXVnY0In0.eyJleHAiOjE3ODA1NzAxNjcsImlhdCI6MTc4MDU2OTU2NywianRpIjoidHJydGNjOmJjYTQ1YzMwLTk0MTgtMjgyYy1mNTg0LTg2OTdjYzc1OTBkMyIsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3Q6OTA5MC9yZWFsbXMvbWNwLXRlc3QiLCJhdWQiOiJhY2NvdW50Iiwic3ViIjoiN2YwM2VhZWMtNTI2OS00ZTFiLWEwYTUtOTJhYjY1ZTdjMjVhIiwidHlwIjoiQmVhcmVyIiwiYXpwIjoidGVzdC1jbGllbnQiLCJhY3IiOiIxIiwiYWxsb3dlZC1vcmlnaW5zIjpbIiJdLCJyZWFsbV9hY2Nlc3MiOnsicm9sZXMiOlsib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImRlZmF1bHQtcm9sZXMtbWNwLXRlc3QiXX0sInJlc291cmNlX2FjY2VzcyI6eyJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJzY29wZSI6InByb2ZpbGUgZW1haWwiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsImNsaWVudEhvc3QiOiIxOTIuMTY4LjY1LjEiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJzZXJ2aWNlLWFjY291bnQtdGVzdC1jbGllbnQiLCJjbGllbnRBZGRyZXNzIjoiMTkyLjE2OC42NS4xIiwiY2xpZW50X2lkIjoidGVzdC1jbGllbnQifQ.wNcMepr6xDrUKsjXf68YpZMujb_mHpF12plvGPY5G9FxNuFur137MOHdarbkr9AVY1QqjduE4MayD4GQ2b-uGz7adRyq9Yb1D_0mpDmfy_R-WvzNhPILchrdHx5fFd9M"
						}
					}
				}
			}`,
			wantErr:          true,
			expectBadSigAlgo: "ES384",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algo := keysFixture.PublicKeys.Keys[0].Alg
			now := time.Unix(1780569028, 0)
			token, err := jwt.Parse(loadBearerFromMCPJSON(t, tt.content), func(token *jwt.Token) (any, error) {
				if token.Method.Alg() != algo {
					return nil, fmt.Errorf("unexpected alg %q", token.Method.Alg())
				}
				kid, _ := token.Header["kid"].(string)
				key, ok := keyByKid[kid]
				if !ok {
					return nil, fmt.Errorf("no key found for kid %q", kid)
				}
				return key, nil
			}, jwt.WithValidMethods([]string{algo}), jwt.WithTimeFunc(func() time.Time { return now }))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected token validation error, got nil")
				}
				if tt.expectExpired && !errors.Is(err, jwt.ErrTokenExpired) {
					t.Fatalf("expected expired-token error, got: %v", err)
				}
				if tt.expectBadSig && !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
					t.Fatalf("expected signature-invalid error, got: %v", err)
				}
				if tt.expectBadSigAlgo != "" && !strings.Contains(err.Error(), tt.expectBadSigAlgo) {
					t.Fatalf("expected signature-invalid error, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected valid token, got error: %v", err)
			}
			if !token.Valid {
				t.Fatal("expected token.Valid=true")
			}
		})
	}
}

func loadKeysFixture(t *testing.T, path string) jwkFile {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var fixture jwkFile
	if err = json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	if len(fixture.PublicKeys.Keys) == 0 || fixture.PublicKeys.Keys[0].Alg == "" {
		t.Fatalf("invalid fixture in %s", path)
	}
	return fixture
}

func loadBearerFromMCPJSON(t *testing.T, scriptContent string) string {
	t.Helper()
	content := []byte(scriptContent)

	var cfg mcpConfigFile
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("failed to parse %s: %v", scriptContent, err)
	}

	serverCfg, ok := cfg.Servers["ontap-mcp"]
	if !ok {
		t.Fatalf("%s missing servers.ontap-mcp", scriptContent)
	}
	auth := strings.TrimSpace(serverCfg.Headers["Authorization"])
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Fatalf("%s does not contain a Bearer Authorization header", scriptContent)
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if token == "" {
		t.Fatalf("%s contains an empty Bearer token", scriptContent)
	}
	return token
}

func buildPublicKeyMap(t *testing.T, fixture jwkFile) map[string]*rsa.PublicKey {
	t.Helper()
	result := make(map[string]*rsa.PublicKey, len(fixture.PublicKeys.Keys))

	for _, k := range fixture.PublicKeys.Keys {
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			t.Fatalf("failed to decode modulus for kid %q: %v", k.Kid, err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			t.Fatalf("failed to decode exponent for kid %q: %v", k.Kid, err)
		}
		e := 0
		for _, b := range eBytes {
			e = (e << 8) | int(b)
		}
		if e <= 0 {
			t.Fatalf("invalid exponent for kid %q", k.Kid)
		}
		result[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}
	}

	if len(result) == 0 {
		t.Fatal("no usable RSA public keys found in fixture")
	}
	return result
}

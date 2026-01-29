package rest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netapp/ontap-mcp/config"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		want     time.Duration
	}{
		{
			name:     "always returns 0",
			schedule: "always",
			want:     0,
		},
		{
			name:     "empty string returns 0",
			schedule: "",
			want:     0,
		},
		{
			name:     "valid duration 3h",
			schedule: "3h",
			want:     3 * time.Hour,
		},
		{
			name:     "valid duration 24h",
			schedule: "24h",
			want:     24 * time.Hour,
		},
		{
			name:     "valid duration 10m",
			schedule: "10m",
			want:     10 * time.Minute,
		},
		{
			name:     "invalid duration defaults to 24h",
			schedule: "invalid",
			want:     24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSchedule(tt.schedule)
			if got != tt.want {
				t.Errorf("parseSchedule(%q) = %v, want %v", tt.schedule, got, tt.want)
			}
		})
	}
}

func TestCredentialsCache_ShouldRefreshCredentials(t *testing.T) {
	tests := []struct {
		name      string
		cache     credentialsCache
		schedule  time.Duration
		want      bool
		setupFunc func(*credentialsCache)
	}{
		{
			name:     "schedule is 0 (always) returns true",
			cache:    credentialsCache{},
			schedule: 0,
			want:     true,
		},
		{
			name:     "cache is empty returns true",
			cache:    credentialsCache{},
			schedule: 24 * time.Hour,
			want:     true,
		},
		{
			name:     "cache not expired returns false",
			cache:    credentialsCache{},
			schedule: 24 * time.Hour,
			want:     false,
			setupFunc: func(c *credentialsCache) {
				c.cachedAt = time.Now()
			},
		},
		{
			name:     "cache expired returns true",
			cache:    credentialsCache{},
			schedule: 1 * time.Millisecond,
			want:     true,
			setupFunc: func(c *credentialsCache) {
				c.cachedAt = time.Now().Add(-2 * time.Millisecond)
			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc(&tt.cache)
			}
			got := tt.cache.shouldRefreshCredentials(tt.schedule)
			if got != tt.want {
				t.Errorf("shouldRefreshCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredentialsCache_UpdateAndGetCredentials(t *testing.T) {
	cache := credentialsCache{}

	username := "testuser"
	password := "testpass"
	authToken := "testtoken"

	cache.updateCache(username, password, authToken)

	creds := cache.getCredentials()

	if creds.Username != username {
		t.Errorf("getCredentials() creds.Username = %v, want %v", creds.Username, username)
	}
	if creds.Password != password {
		t.Errorf("getCredentials() creds.Password = %v, want %v", creds.Password, password)
	}
	if creds.AuthToken != authToken {
		t.Errorf("getCredentials() creds.AuthToken = %v, want %v", creds.AuthToken, authToken)
	}

	if cache.cachedAt.IsZero() {
		t.Error("updateCache() did not set cachedAt time")
	}
}

func TestExecuteCredentialsScript_Success(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "password: mypassword"
echo "username: myuser"
echo "authToken: mytoken"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    scriptPath,
			Timeout: "10s",
		},
	}

	ctx := context.Background()
	response, err := executeCredentialsScript(ctx, poller)

	if err != nil {
		t.Fatalf("executeCredentialsScript() error = %v", err)
	}

	if response.Password != "mypassword" {
		t.Errorf("response.Password = %v, want mypassword", response.Password)
	}
	if response.Username != "myuser" {
		t.Errorf("response.Username = %v, want myuser", response.Username)
	}
	if response.AuthToken != "mytoken" {
		t.Errorf("response.AuthToken = %v, want mytoken", response.AuthToken)
	}
}

func TestExecuteCredentialsScript_OnlyPassword(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "password: onlypass"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    scriptPath,
			Timeout: "10s",
		},
	}

	ctx := context.Background()
	response, err := executeCredentialsScript(ctx, poller)

	if err != nil {
		t.Fatalf("executeCredentialsScript() error = %v", err)
	}

	if response.Password != "onlypass" {
		t.Errorf("response.Password = %v, want onlypass", response.Password)
	}
	if response.Username != "" {
		t.Errorf("response.Username = %v, want empty string", response.Username)
	}
}

func TestExecuteCredentialsScript_OnlyAuthToken(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "authToken: bearer123"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    scriptPath,
			Timeout: "10s",
		},
	}

	ctx := context.Background()
	response, err := executeCredentialsScript(ctx, poller)

	if err != nil {
		t.Fatalf("executeCredentialsScript() error = %v", err)
	}

	if response.AuthToken != "bearer123" {
		t.Errorf("response.AuthToken = %v, want bearer123", response.AuthToken)
	}
}

func TestExecuteCredentialsScript_Timeout(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
sleep 5
echo "password: shouldntsee"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    scriptPath,
			Timeout: "100ms",
		},
	}

	ctx := context.Background()
	_, err = executeCredentialsScript(ctx, poller)

	if err == nil {
		t.Fatal("executeCredentialsScript() expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("executeCredentialsScript() error = %v, want timeout error", err)
	}
}

func TestExecuteCredentialsScript_ScriptNotFound(t *testing.T) {
	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    "/nonexistent/script.sh",
			Timeout: "10s",
		},
	}

	ctx := context.Background()
	_, err := executeCredentialsScript(ctx, poller)

	if err == nil {
		t.Fatal("executeCredentialsScript() expected error for nonexistent script, got nil")
	}
}

func TestExecuteCredentialsScript_InvalidYAML(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "this is not valid yaml: {"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    scriptPath,
			Timeout: "10s",
		},
	}

	ctx := context.Background()
	_, err = executeCredentialsScript(ctx, poller)

	if err == nil {
		t.Fatal("executeCredentialsScript() expected YAML parse error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "parse") && !strings.Contains(errStr, "YAML") && !strings.Contains(errStr, "unmarshal") {
		t.Errorf("executeCredentialsScript() error = %v, want YAML parse error", err)
	}
}

func TestExecuteCredentialsScript_MissingPasswordAndToken(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "username: someuser"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    scriptPath,
			Timeout: "10s",
		},
	}

	ctx := context.Background()
	_, err = executeCredentialsScript(ctx, poller)

	if err == nil {
		t.Fatal("executeCredentialsScript() expected error for missing password/authToken, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "password") && !strings.Contains(errStr, "authToken") {
		t.Errorf("executeCredentialsScript() error = %v, want password/authToken error", err)
	}
}

func TestExecuteCredentialsScript_DefaultTimeout(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "password: testpass"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path: scriptPath,
		},
	}

	ctx := context.Background()
	response, err := executeCredentialsScript(ctx, poller)

	if err != nil {
		t.Fatalf("executeCredentialsScript() error = %v", err)
	}

	if response.Password != "testpass" {
		t.Errorf("response.Password = %v, want testpass", response.Password)
	}
}

func TestExecuteCredentialsScript_InvalidTimeout(t *testing.T) {
	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "admin",
		CredentialsScript: config.CredentialsScript{
			Path:    "/some/script.sh",
			Timeout: "invalid",
		},
	}

	ctx := context.Background()
	_, err := executeCredentialsScript(ctx, poller)

	if err == nil {
		t.Fatal("executeCredentialsScript() expected timeout parse error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("executeCredentialsScript() error = %v, want timeout error", err)
	}
}

func TestClient_GetAuth_NoScript(t *testing.T) {
	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "testuser",
		Password: "testpass",
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)

	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds.Username != "testuser" {
		t.Errorf("getAuth() creds.Username = %v, want testuser", creds.Username)
	}
	if creds.Password != "testpass" {
		t.Errorf("getAuth() creds.Password = %v, want testpass", creds.Password)
	}
	if creds.AuthToken != "" {
		t.Errorf("getAuth() creds.AuthToken = %v, want empty string", creds.AuthToken)
	}
}

func TestClient_GetAuth_WithScript(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "password: scriptpass"
echo "username: scriptuser"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "configuser",
		Password: "configpass",
		CredentialsScript: config.CredentialsScript{
			Path:     scriptPath,
			Timeout:  "10s",
			Schedule: "1h",
		},
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)

	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds.Username != "scriptuser" {
		t.Errorf("getAuth() creds.Username = %v, want scriptuser", creds.Username)
	}
	if creds.Password != "scriptpass" {
		t.Errorf("getAuth() creds.Password = %v, want scriptpass", creds.Password)
	}
	if creds.AuthToken != "" {
		t.Errorf("getAuth() creds.AuthToken = %v, want empty string", creds.AuthToken)
	}
}

func TestClient_GetAuth_ScriptUsernameFromConfig(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "password: scriptpass"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "configuser",
		Password: "configpass",
		CredentialsScript: config.CredentialsScript{
			Path:     scriptPath,
			Timeout:  "10s",
			Schedule: "1h",
		},
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)

	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds.Username != "configuser" {
		t.Errorf("getAuth() creds.Username = %v, want configuser", creds.Username)
	}
	if creds.Password != "scriptpass" {
		t.Errorf("getAuth() creds.Password = %v, want scriptpass", creds.Password)
	}
}

func TestClient_GetAuth_CachingBehavior(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")
	counterFile := filepath.Join(tmpDir, "counter.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
if [ -f "%s" ]; then
    count=$(cat "%s")
else
    count=0
fi
count=$((count + 1))
echo $count > "%s"
echo "password: pass$count"
`, counterFile, counterFile, counterFile)

	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "testuser",
		CredentialsScript: config.CredentialsScript{
			Path:     scriptPath,
			Timeout:  "10s",
			Schedule: "1h",
		},
	}

	client := New(poller)
	ctx := context.Background()

	creds1, err := client.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() first call error = %v", err)
	}

	creds2, err := client.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() second call error = %v", err)
	}

	if creds1.Password != creds2.Password {
		t.Errorf("getAuth() password changed between calls, cache not working: %v != %v", creds1.Password, creds2.Password)
	}

	if creds1.Password != "pass1" {
		t.Errorf("getAuth() first creds.Password = %v, want pass1", creds1.Password)
	}
}

func TestClient_GetAuth_AlwaysRefresh(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")
	counterFile := filepath.Join(tmpDir, "counter.txt")

	scriptContent := fmt.Sprintf(`#!/bin/bash
if [ -f "%s" ]; then
    count=$(cat "%s")
else
    count=0
fi
count=$((count + 1))
echo $count > "%s"
echo "password: pass$count"
`, counterFile, counterFile, counterFile)

	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "testuser",
		CredentialsScript: config.CredentialsScript{
			Path:     scriptPath,
			Timeout:  "10s",
			Schedule: "always",
		},
	}

	client := New(poller)
	ctx := context.Background()

	creds1, err := client.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() first call error = %v", err)
	}

	creds2, err := client.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() second call error = %v", err)
	}

	if creds1.Password == creds2.Password {
		t.Errorf("getAuth() password should change with 'always' schedule: %v == %v", creds1.Password, creds2.Password)
	}

	if creds1.Password != "pass1" {
		t.Errorf("getAuth() first creds.Password = %v, want pass1", creds1.Password)
	}
	if creds2.Password != "pass2" {
		t.Errorf("getAuth() second creds.Password = %v, want pass2", creds2.Password)
	}
}

func TestClient_GetAuth_WithAuthToken(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := `#!/bin/bash
echo "authToken: bearer-token-123"
`
	//nolint:gosec // G306: Script needs to be executable
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	poller := &config.Poller{
		Addr:     "10.1.1.1",
		Username: "testuser",
		CredentialsScript: config.CredentialsScript{
			Path:     scriptPath,
			Timeout:  "10s",
			Schedule: "1h",
		},
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)

	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds.AuthToken != "bearer-token-123" {
		t.Errorf("getAuth() creds.AuthToken = %v, want bearer-token-123", creds.AuthToken)
	}

	if creds.Username != "testuser" {
		t.Errorf("getAuth() creds.Username = %v, want testuser (from config)", creds.Username)
	}
	if creds.Password != "" {
		t.Errorf("getAuth() creds.Password = %v, want empty", creds.Password)
	}
}

func TestLoadCredentialsFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `Pollers:
  cluster1:
    username: fileuser
    password: filepass
  cluster2:
    username: user2
    password: pass2
`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	username, password, err := loadCredentialsFile(credFile, "cluster1")

	if err != nil {
		t.Fatalf("loadCredentialsFile() error = %v", err)
	}

	if username != "fileuser" {
		t.Errorf("loadCredentialsFile() username = %v, want fileuser", username)
	}
	if password != "filepass" {
		t.Errorf("loadCredentialsFile() password = %v, want filepass", password)
	}
}

func TestLoadCredentialsFile_SecondCluster(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `Pollers:
  cluster1:
    username: user1
    password: pass1
  cluster2:
    username: user2
    password: pass2
`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	username, password, err := loadCredentialsFile(credFile, "cluster2")

	if err != nil {
		t.Fatalf("loadCredentialsFile() error = %v", err)
	}

	if username != "user2" {
		t.Errorf("loadCredentialsFile() username = %v, want user2", username)
	}
	if password != "pass2" {
		t.Errorf("loadCredentialsFile() password = %v, want pass2", password)
	}
}

func TestLoadCredentialsFile_OnlyPassword(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `Pollers:
  cluster1:
    password: onlypass
`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	username, password, err := loadCredentialsFile(credFile, "cluster1")

	if err != nil {
		t.Fatalf("loadCredentialsFile() error = %v", err)
	}

	if username != "" {
		t.Errorf("loadCredentialsFile() username = %v, want empty", username)
	}
	if password != "onlypass" {
		t.Errorf("loadCredentialsFile() password = %v, want onlypass", password)
	}
}

func TestLoadCredentialsFile_ClusterNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `Pollers:
  cluster1:
    username: user1
    password: pass1
`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	_, _, err = loadCredentialsFile(credFile, "nonexistent")

	if err == nil {
		t.Fatal("loadCredentialsFile() expected error for nonexistent cluster, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("loadCredentialsFile() error = %v, want 'not found' error", err)
	}
}

func TestLoadCredentialsFile_FileNotFound(t *testing.T) {
	_, _, err := loadCredentialsFile("/nonexistent/file.yml", "cluster1")

	if err == nil {
		t.Fatal("loadCredentialsFile() expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read") {
		t.Errorf("loadCredentialsFile() error = %v, want 'failed to read' error", err)
	}
}

func TestLoadCredentialsFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `this is not valid yaml: {`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	_, _, err = loadCredentialsFile(credFile, "cluster1")

	if err == nil {
		t.Fatal("loadCredentialsFile() expected error for invalid YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("loadCredentialsFile() error = %v, want 'failed to parse' error", err)
	}
}

func TestLoadCredentialsFile_EmptyPath(t *testing.T) {
	_, _, err := loadCredentialsFile("", "cluster1")

	if err == nil {
		t.Fatal("loadCredentialsFile() expected error for empty path, got nil")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("loadCredentialsFile() error = %v, want 'empty' error", err)
	}
}

func TestClient_GetAuth_WithCredentialsFile(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `Pollers:
  testcluster:
    username: fileuser
    password: filepass
`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	poller := &config.Poller{
		Name:            "testcluster",
		Addr:            "10.1.1.1",
		Username:        "configuser",
		Password:        "configpass",
		CredentialsFile: credFile,
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)

	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds.Username != "fileuser" {
		t.Errorf("getAuth() creds.Username = %v, want fileuser", creds.Username)
	}
	if creds.Password != "filepass" {
		t.Errorf("getAuth() creds.Password = %v, want filepass", creds.Password)
	}
	if creds.AuthToken != "" {
		t.Errorf("getAuth() creds.AuthToken = %v, want empty string", creds.AuthToken)
	}
}

func TestClient_GetAuth_CredentialsFileUsernameFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	content := `Pollers:
  testcluster:
    password: filepass
`
	err := os.WriteFile(credFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	poller := &config.Poller{
		Name:            "testcluster",
		Addr:            "10.1.1.1",
		Username:        "configuser",
		Password:        "configpass",
		CredentialsFile: credFile,
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)

	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	// Should use username from config since file doesn't provide one
	if creds.Username != "configuser" {
		t.Errorf("getAuth() creds.Username = %v, want configuser", creds.Username)
	}
	if creds.Password != "filepass" {
		t.Errorf("getAuth() creds.Password = %v, want filepass", creds.Password)
	}
}

func TestClient_GetAuth_PriorityOrder(t *testing.T) {
	if !hasBash(t) {
		return
	}

	tmpDir := t.TempDir()

	// Create credentials file
	credFile := filepath.Join(tmpDir, "credentials.yml")
	//nolint:gosec // G101: Test credentials, not real secrets
	credContent := `Pollers:
  testcluster:
    username: fileuser
    password: filepass
`
	err := os.WriteFile(credFile, []byte(credContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}

	// Create credentials script
	scriptPath := filepath.Join(tmpDir, "get_creds.sh")
	scriptContent := `#!/bin/bash
echo "password: scriptpass"
echo "username: scriptuser"
`
	//nolint:gosec // G306: Script needs to be executable
	err = os.WriteFile(scriptPath, []byte(scriptContent), 0700)
	if err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	// Test 1: Script has highest priority
	poller := &config.Poller{
		Name:            "testcluster",
		Addr:            "10.1.1.1",
		Username:        "configuser",
		Password:        "configpass",
		CredentialsFile: credFile,
		CredentialsScript: config.CredentialsScript{
			Path:     scriptPath,
			Timeout:  "10s",
			Schedule: "1h",
		},
	}

	client := New(poller)
	ctx := context.Background()

	creds, err := client.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds.Username != "scriptuser" {
		t.Errorf("Priority test: script should win, got creds.Username = %v, want scriptuser", creds.Username)
	}
	if creds.Password != "scriptpass" {
		t.Errorf("Priority test: script should win, got creds.Password = %v, want scriptpass", creds.Password)
	}

	// Test 2: File has second priority (no script)
	poller2 := &config.Poller{
		Name:            "testcluster",
		Addr:            "10.1.1.1",
		Username:        "configuser",
		Password:        "configpass",
		CredentialsFile: credFile,
	}

	client2 := New(poller2)
	creds2, err := client2.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds2.Username != "fileuser" {
		t.Errorf("Priority test: file should win, got creds.Username = %v, want fileuser", creds2.Username)
	}
	if creds2.Password != "filepass" {
		t.Errorf("Priority test: file should win, got creds.Password = %v, want filepass", creds2.Password)
	}

	// Test 3: Inline config is last priority (no script or file)
	poller3 := &config.Poller{
		Name:     "testcluster",
		Addr:     "10.1.1.1",
		Username: "configuser",
		Password: "configpass",
	}

	client3 := New(poller3)
	creds3, err := client3.getAuth(ctx)
	if err != nil {
		t.Fatalf("getAuth() error = %v", err)
	}

	if creds3.Username != "configuser" {
		t.Errorf("Priority test: inline should win, got creds.Username = %v, want configuser", creds3.Username)
	}
	if creds3.Password != "configpass" {
		t.Errorf("Priority test: inline should win, got creds.Password = %v, want configpass", creds3.Password)
	}
}

// hasBash checks if bash is available and skips the test if not
func hasBash(t *testing.T) bool {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available, skipping shell script test")
		return false
	}
	return true
}

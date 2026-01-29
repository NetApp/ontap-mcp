package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/netapp/ontap-mcp/config"
)

// credentialsCache holds cached credentials and their expiration info
type credentialsCache struct {
	username  string
	password  string
	authToken string
	cachedAt  time.Time
	mu        sync.RWMutex
}

// scriptResponse represents the expected YAML output from the credentials script
type scriptResponse struct {
	Password  string `yaml:"password"`
	Username  string `yaml:"username,omitempty"`
	AuthToken string `yaml:"authToken,omitempty"`
}

// executeCredentialsScript executes the credentials script and returns the parsed response
func executeCredentialsScript(ctx context.Context, poller *config.Poller) (*scriptResponse, error) {
	if poller.CredentialsScript.Path == "" {
		return nil, errors.New("credentials_script path is not configured")
	}

	// Parse timeout (default 10s)
	timeout := 10 * time.Second
	if poller.CredentialsScript.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(poller.CredentialsScript.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid credentials_script timeout '%s': %w", poller.CredentialsScript.Timeout, err)
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command arguments: script $addr [$username]
	args := []string{poller.Addr}
	if poller.Username != "" {
		args = append(args, poller.Username)
	}

	//nolint:gosec // G204: Script path and args are from trusted config file
	cmd := exec.CommandContext(execCtx, poller.CredentialsScript.Path, args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the script
	err := cmd.Run()
	if err != nil {
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("credentials_script timed out after %v", timeout)
		}
		return nil, fmt.Errorf("credentials_script execution failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse YAML response from stdout
	var response scriptResponse
	err = yaml.Unmarshal(stdout.Bytes(), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials_script YAML output: %w", err)
	}

	// Validate required field
	if response.Password == "" && response.AuthToken == "" {
		return nil, errors.New("credentials_script must return either 'password' or 'authToken' field")
	}

	return &response, nil
}

// parseSchedule converts schedule string to duration
// Special value "always" returns 0 duration
// Default is 24h if not specified or invalid
func parseSchedule(schedule string) time.Duration {
	if schedule == "always" || schedule == "" {
		return 0
	}

	duration, err := time.ParseDuration(schedule)
	if err != nil {
		// Default to 24h if invalid
		return 24 * time.Hour
	}

	return duration
}

// shouldRefreshCredentials determines if cached credentials should be refreshed
func (c *credentialsCache) shouldRefreshCredentials(schedule time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If schedule is 0 (always), always refresh
	if schedule == 0 {
		return true
	}

	// If cache is empty, refresh
	if c.cachedAt.IsZero() {
		return true
	}

	// Check if cache has expired
	return time.Since(c.cachedAt) >= schedule
}

func (c *credentialsCache) updateCache(username, password, authToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.username = username
	c.password = password
	c.authToken = authToken
	c.cachedAt = time.Now()
}

// getCredentials returns cached credentials
func (c *credentialsCache) getCredentials() credentials {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return credentials{
		Username:  c.username,
		Password:  c.password,
		AuthToken: c.authToken,
	}
}

// loadCredentialsFile loads credentials from a YAML file
func loadCredentialsFile(filePath string, pollerName string) (string, string, error) {
	if filePath == "" {
		return "", "", errors.New("credentials_file path is empty")
	}

	// Read the file
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read credentials_file %s: %w", filePath, err)
	}

	// Parse the YAML
	var cfg config.ONTAP
	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse credentials_file %s: %w", filePath, err)
	}

	// Look for the matching poller
	poller, ok := cfg.Pollers[pollerName]
	if !ok {
		return "", "", fmt.Errorf("poller '%s' not found in credentials_file %s", pollerName, filePath)
	}

	// Return the credentials
	return poller.Username, poller.Password, nil
}

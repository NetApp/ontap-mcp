package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestApplyDefaults_FillsMissingKeys(t *testing.T) {
	defaults := &Poller{
		Username:       "admin",
		Password:       "password",
		UseInsecureTLS: true,
	}
	poller := &Poller{
		Addr: "10.0.0.1",
	}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.Equal(t, poller.Username, "admin")
	assert.Equal(t, poller.Password, "password")
	assert.True(t, poller.UseInsecureTLS)
	assert.Equal(t, poller.Addr, "10.0.0.1")
}

func TestApplyDefaults_PollerOverridesDefault(t *testing.T) {
	defaults := &Poller{
		Username: "admin",
		Password: "password",
	}
	poller := &Poller{
		Username: "operator",
		Password: "secret",
	}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.Equal(t, poller.Username, "operator")
	assert.Equal(t, poller.Password, "secret")
}

func TestApplyDefaults_MergesNestedCredentialsScript(t *testing.T) {
	defaults := &Poller{
		CredentialsScript: CredentialsScript{
			Path:     "/default/get_pass",
			Schedule: "always",
			Timeout:  "5s",
		},
	}
	poller := &Poller{
		CredentialsScript: CredentialsScript{
			Path: "/poller/get_pass",
		},
	}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.Equal(t, poller.CredentialsScript.Path, "/poller/get_pass")
	assert.Equal(t, poller.CredentialsScript.Schedule, "always")
	assert.Equal(t, poller.CredentialsScript.Timeout, "5s")
}

func TestApplyDefaults_NilDefaultsIsNoOp(t *testing.T) {
	poller := &Poller{
		Addr:     "10.0.0.1",
		Username: "operator",
	}

	assert.Nil(t, poller.applyDefaults(nil))

	assert.Equal(t, poller.Addr, "10.0.0.1")
	assert.Equal(t, poller.Username, "operator")
	assert.Equal(t, poller.Password, "")
}

func TestReadConfig_AppliesDefaultsAcrossPollers(t *testing.T) {
	yamlContent := `
Defaults:
  username: admin
  password: password
  use_insecure_tls: true

Pollers:
  inherits-all:
    addr: 10.0.0.1
  overrides-user:
    addr: 10.0.0.2
    username: operator
    password: secret
`
	dir := t.TempDir()
	path := filepath.Join(dir, "ontap.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := ReadConfig(path)
	assert.Nil(t, err)

	inherits := cfg.Pollers["inherits-all"]
	assert.NotNil(t, inherits)
	assert.Equal(t, inherits.Username, "admin")
	assert.Equal(t, inherits.Password, "password")
	assert.True(t, inherits.UseInsecureTLS)
	assert.Equal(t, inherits.Name, "inherits-all")

	overrides := cfg.Pollers["overrides-user"]
	assert.NotNil(t, overrides)
	assert.Equal(t, overrides.Username, "operator")
	assert.Equal(t, overrides.Password, "secret")
	assert.True(t, overrides.UseInsecureTLS)
}

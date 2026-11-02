package config

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
)

func TestApplyDefaults_FillsMissingKeys(t *testing.T) {
	defaults := &Poller{
		Username:       "admin",
		Password:       "password",
		UseInsecureTLS: new(true),
	}
	poller := &Poller{
		Addr: "10.0.0.1",
	}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.Equal(t, poller.Username, "admin")
	assert.Equal(t, poller.Password, "password")
	assert.True(t, poller.InsecureTLS())
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
	assert.True(t, inherits.InsecureTLS())
	assert.Equal(t, inherits.Name, "inherits-all")

	overrides := cfg.Pollers["overrides-user"]
	assert.NotNil(t, overrides)
	assert.Equal(t, overrides.Username, "operator")
	assert.Equal(t, overrides.Password, "secret")
	assert.True(t, overrides.InsecureTLS())
}

func TestApplyDefaults_PollerOverridesTrueDefaultWithFalse(t *testing.T) {
	defaults := &Poller{
		UseInsecureTLS: new(true),
	}
	poller := &Poller{
		UseInsecureTLS: new(false),
	}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.NotNil(t, poller.UseInsecureTLS)
	assert.False(t, poller.InsecureTLS())
}

func TestApplyDefaults_PollerInheritsUnsetInsecureTLS(t *testing.T) {
	defaults := &Poller{
		UseInsecureTLS: new(true),
	}
	poller := &Poller{}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.NotNil(t, poller.UseInsecureTLS)
	assert.True(t, poller.InsecureTLS())
}

func TestApplyDefaults_PollerKeepsTrueOverFalseDefault(t *testing.T) {
	defaults := &Poller{
		UseInsecureTLS: new(false),
	}
	poller := &Poller{
		UseInsecureTLS: new(true),
	}

	assert.Nil(t, poller.applyDefaults(defaults))

	assert.True(t, poller.InsecureTLS())
}

func TestReadConfig_PollerOverridesInsecureTLSToFalse(t *testing.T) {
	yamlContent := `
Defaults:
  username: admin
  use_insecure_tls: true

Pollers:
  inherits-tls:
    addr: 10.0.0.1
  secure-poller:
    addr: 10.0.0.2
    use_insecure_tls: false
`
	dir := t.TempDir()
	path := filepath.Join(dir, "ontap.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := ReadConfig(path)
	assert.Nil(t, err)

	inherits := cfg.Pollers["inherits-tls"]
	assert.NotNil(t, inherits)
	assert.True(t, inherits.InsecureTLS())

	secure := cfg.Pollers["secure-poller"]
	assert.NotNil(t, secure)
	assert.False(t, secure.InsecureTLS())
}

func TestInsecureTLS_NilFieldIsFalse(t *testing.T) {
	poller := &Poller{}

	assert.False(t, poller.InsecureTLS())
}

func TestReadConfig_AlgAcceptsScalar(t *testing.T) {
	yamlContent := `
McpAuth:
  issuer: https://issuer.test
  audience: mcp-aud
  alg: RS256
`
	dir := t.TempDir()
	path := filepath.Join(dir, "ontap.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := ReadConfig(path)
	assert.Nil(t, err)
	assert.NotNil(t, cfg.McpAuth)
	assert.Equal(t, []string(cfg.McpAuth.Alg), []string{"RS256"})
}

func TestReadConfig_AlgAcceptsList(t *testing.T) {
	yamlContent := `
McpAuth:
  issuer: https://issuer.test
  audience: mcp-aud
  alg: [RS256, ES256]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "ontap.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := ReadConfig(path)
	assert.Nil(t, err)
	assert.NotNil(t, cfg.McpAuth)
	assert.Equal(t, []string(cfg.McpAuth.Alg), []string{"RS256", "ES256"})
}

func TestReadConfig_AlgOmittedIsEmpty(t *testing.T) {
	yamlContent := `
McpAuth:
  issuer: https://issuer.test
  audience: mcp-aud
`
	dir := t.TempDir()
	path := filepath.Join(dir, "ontap.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := ReadConfig(path)
	assert.Nil(t, err)
	assert.NotNil(t, cfg.McpAuth)
	assert.Equal(t, len(cfg.McpAuth.Alg), 0)
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ontap.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func skippedNames(cfg *ONTAP) []string {
	return slices.Sorted(maps.Keys(cfg.Skipped))
}

func TestReadConfig_SkipsNonONTAPCollectors(t *testing.T) {
	yamlContent := `
Pollers:
  unix:
    addr: localhost
    collectors: [Unix]
  sg-tmp:
    addr: 10.0.0.2
    collectors: [StorageGrid]
  sar:
    addr: 10.0.0.3
    collectors:
      - Rest
      - RestPerf
  eseries-admin:
    addr: 10.0.0.4
    collectors: [Eseries]
  templated:
    addr: 10.0.0.5
    collectors:
      - Rest:
          - limited.yaml
  plain:
    addr: 10.0.0.6
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	assert.Equal(t, cfg.PollersOrdered, []string{"sar", "templated", "plain"})
	assert.Equal(t, len(cfg.Pollers), 3)
	assert.Equal(t, skippedNames(cfg), []string{"eseries-admin", "sg-tmp", "unix"})
	for name, reason := range cfg.Skipped {
		assert.Equal(t, reason, SkipReasonNotONTAP)
		_, ok := cfg.Pollers[name]
		assert.False(t, ok)
	}

	templated := cfg.Pollers["templated"]
	assert.NotNil(t, templated)
	assert.Equal(t, []string(templated.Collectors), []string{"Rest"})
}

// Collector names are matched exactly, like Harvest: a wrong-case name such as
// "rest" is an invalid collector in Harvest, so it does not make a poller ONTAP.
// A list that mixes a non-ONTAP and an ONTAP collector is still ONTAP.
func TestReadConfig_CollectorsMatchExactlyAndMixed(t *testing.T) {
	yamlContent := `
Pollers:
  lower:
    addr: 10.0.0.1
    collectors: [rest]
  mixed:
    addr: 10.0.0.2
    collectors: [Unix, Rest]
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	assert.Equal(t, cfg.PollersOrdered, []string{"mixed"})
	assert.Equal(t, cfg.Skipped["lower"], SkipReasonNotONTAP)
	assert.Equal(t, len(cfg.Skipped), 1)
	_, ok := cfg.Pollers["lower"]
	assert.False(t, ok)
	assert.NotNil(t, cfg.Pollers["mixed"])
}

func TestReadConfig_InheritsNonONTAPCollectorsFromDefaults(t *testing.T) {
	yamlContent := `
Defaults:
  collectors: [StorageGrid]

Pollers:
  inherits:
    addr: 10.0.0.1
  own-list:
    addr: 10.0.0.2
    collectors: [Rest]
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	assert.Equal(t, cfg.PollersOrdered, []string{"own-list"})
	assert.Equal(t, skippedNames(cfg), []string{"inherits"})
	assert.Equal(t, []string(cfg.Pollers["own-list"].Collectors), []string{"Rest"})
}

func TestReadConfig_InheritsONTAPCollectorsFromDefaults(t *testing.T) {
	yamlContent := `
Defaults:
  collectors:
    - Zapi
    - ZapiPerf
    - Ems

Pollers:
  inherits:
    addr: 10.0.0.1
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	assert.Equal(t, cfg.PollersOrdered, []string{"inherits"})
	assert.Equal(t, len(cfg.Skipped), 0)
	assert.Equal(t, []string(cfg.Pollers["inherits"].Collectors), []string{"Zapi", "ZapiPerf", "Ems"})
}

// TestReadConfig_NABox uses the layout of a NABox-generated Harvest config.
// Passwords are placeholders.
func TestReadConfig_NABox(t *testing.T) {
	yamlContent := `
Defaults:
  debug: false
  collectors:
  - Zapi
  - ZapiPerf
  - Ems
  exporters:
  - nabox-victoriametrics
  enabled: false
Exporters:
  nabox-victoriametrics:
    exporter: Prometheus
    sort_labels: true
Pollers:
  10.113.13.61:
    debug: false
    port: 12005
    addr: 10.113.13.61
    collectors:
    - Eseries
    - EseriesPerf
    datacenter: es
    password: placeholder
    use_insecure_tls: true
    username: admin
    enabled: false
  A250-41-42-43:
    debug: false
    port: 12001
    addr: 10.195.15.41
    collectors:
    - Rest
    - RestPerf
    - Zapi
    - ZapiPerf
    - KeyPerf
    datacenter: test
    password: placeholder
    use_insecure_tls: true
    username: admin
    enabled: true
    api: Rest
  netapp_01:
    debug: false
    port: 12004
    addr: 10.113.13.9
    collectors:
    - Eseries
    - EseriesPerf
    datacenter: eseries
    password: placeholder
    use_insecure_tls: true
    username: admin
    enabled: true
  rtp-sa-asa01:
    debug: true
    port: 12003
    addr: 10.62.218.27
    collectors:
    - Rest
    - RestPerf
    - Zapi
    - ZapiPerf
    - Ems
    datacenter: asa
    password: placeholder
    use_insecure_tls: true
    username: harvest-team
    enabled: false
    api: Rest
  umeng-aff300-01-02:
    debug: true
    port: 12002
    addr: 10.193.48.154
    collectors:
    - Rest
    - RestPerf
    - Zapi
    - ZapiPerf
    - Ems
    - KeyPerf
    datacenter: sar
    password: placeholder
    use_insecure_tls: true
    username: admin
    enabled: true
    api: Rest
  umeng-aff300-05-06:
    debug: false
    port: 12006
    addr: 10.193.48.11
    collectors:
    - Zapi
    - ZapiPerf
    - Rest
    - KeyPerf
    datacenter: u2
    password: placeholder
    use_insecure_tls: true
    username: admin
    enabled: false
    api: Auto
  unix:
    debug: false
    port: 12000
    addr: localhost
    collectors:
    - Unix
    datacenter: local
    exporters:
    - nabox-victoriametrics
    enabled: true
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	// enabled: false is not handled yet, so rtp-sa-asa01 and
	// umeng-aff300-05-06 are kept.
	assert.Equal(t, cfg.PollersOrdered, []string{"A250-41-42-43", "rtp-sa-asa01", "umeng-aff300-01-02", "umeng-aff300-05-06"})
	assert.Equal(t, len(cfg.Pollers), 4)
	assert.Equal(t, skippedNames(cfg), []string{"10.113.13.61", "netapp_01", "unix"})
}

// TestReadConfig_SwitchesDoNotFilter pins the current scope: only collectors
// decide whether a poller is kept. disabled and enabled are not applied.
func TestReadConfig_SwitchesDoNotFilter(t *testing.T) {
	yamlContent := `
Pollers:
  disabled-poller:
    addr: 10.0.0.1
    collectors: [Rest]
    disabled: true
  enabled-false:
    addr: 10.0.0.2
    collectors: [Rest]
    enabled: false
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	assert.Equal(t, cfg.PollersOrdered, []string{"disabled-poller", "enabled-false"})
	assert.Equal(t, len(cfg.Skipped), 0)
}

func TestReadConfig_NoCollectorsKeepsAllPollers(t *testing.T) {
	yamlContent := `
Defaults:
  username: admin

Pollers:
  dc1:
    addr: 10.0.0.1
  dc2:
    addr: 10.0.0.2
`
	cfg, err := ReadConfig(writeConfig(t, yamlContent))
	assert.Nil(t, err)

	assert.Equal(t, cfg.PollersOrdered, []string{"dc1", "dc2"})
	assert.Equal(t, len(cfg.Pollers), 2)
	assert.Equal(t, len(cfg.Skipped), 0)
}

func TestReadConfig_InvalidCollectors(t *testing.T) {
	tests := []struct {
		name       string
		collectors string
	}{
		{name: "nested list", collectors: "\n      - [Rest]"},
		{name: "multi-key map", collectors: " [{Rest: [a.yaml], Zapi: [b.yaml]}]"},
		{name: "scalar", collectors: " Rest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := `
Pollers:
  bad:
    addr: 10.0.0.1
    collectors:` + tt.collectors + "\n"
			_, err := ReadConfig(writeConfig(t, yamlContent))
			assert.NotNil(t, err)
			if err != nil {
				assert.Contains(t, err.Error(), "collectors")
			}
		})
	}
}

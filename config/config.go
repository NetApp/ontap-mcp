package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/netapp/ontap-mcp/third_party/mergo"
)

const (
	AppName = "ontap-mcp"
)

func ReadConfig(path string) (*ONTAP, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ONTAP
	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Read the config again to determine poller order
	var orderedConfig OrderedConfig
	err = yaml.Unmarshal(contents, &orderedConfig)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling ordered config: %w", err)
	}
	cfg.PollersOrdered = orderedConfig.Pollers.namesInOrder

	// Set the Name field for each poller and apply Defaults for any
	// key that is missing from the poller's own configuration.
	for name, poller := range cfg.Pollers {
		poller.Name = name
		if err := poller.applyDefaults(cfg.Defaults); err != nil {
			return nil, fmt.Errorf("error applying defaults to poller %q: %w", name, err)
		}
	}

	return &cfg, nil
}

// applyDefaults fills any unset (zero-valued) field on the poller with the
// corresponding value from defaults. A value explicitly set on the poller
// always takes precedence over the default. Nested structs are merged
// field-by-field, so a poller can override one sub-field while inheriting
// the rest.
//
// Pointer fields such as UseInsecureTLS distinguish "unset" (nil) from an
// explicit value. mergo.WithoutDereference keeps a non-nil pointer on the
// poller intact, so a poller can override a true default back to false.
func (p *Poller) applyDefaults(defaults *Poller) error {
	if defaults == nil {
		return nil
	}

	return mergo.Merge(p, *defaults, mergo.WithoutDereference)
}

// InsecureTLS reports the effective use_insecure_tls value, treating an unset
// (nil) field as false.
func (p *Poller) InsecureTLS() bool {
	return p.UseInsecureTLS != nil && *p.UseInsecureTLS
}

type ONTAP struct {
	Pollers        map[string]*Poller `yaml:"Pollers,omitempty"`
	Defaults       *Poller            `yaml:"Defaults,omitempty"`
	McpAuth        *OAuth             `yaml:"McpAuth,omitempty"`
	PollersOrdered []string           `yaml:"-"` // poller names in same order as yaml config
}

type OAuth struct {
	Issuer   string      `yaml:"issuer"`
	Audience string      `yaml:"audience"`
	Alg      StringSlice `yaml:"alg,omitempty"`
	Scope    string      `yaml:"scope,omitempty"`
}

// StringSlice accepts either a single YAML scalar (alg: RS256) or a YAML
// sequence (alg: [RS256, ES256]) and always decodes into a slice of strings.
type StringSlice []string

// UnmarshalYAML implements the goccy/go-yaml BytesUnmarshaler interface so the
// field can be configured as a scalar or a list.
func (s *StringSlice) UnmarshalYAML(b []byte) error {
	var list []string
	if err := yaml.Unmarshal(b, &list); err == nil {
		*s = list
		return nil
	}
	var single string
	if err := yaml.Unmarshal(b, &single); err != nil {
		return fmt.Errorf("alg must be a string or a list of strings: %w", err)
	}
	*s = StringSlice{single}
	return nil
}

type Poller struct {
	Addr string `yaml:"addr,omitempty"`

	AuthStyle         string            `yaml:"auth_style,omitempty"`
	CaCertPath        string            `yaml:"ca_cert,omitempty"`
	CertificateScript CertificateScript `yaml:"certificate_script,omitempty"`
	ClientTimeout     string            `yaml:"client_timeout,omitempty"`
	CredentialsFile   string            `yaml:"credentials_file,omitempty"`
	CredentialsScript CredentialsScript `yaml:"credentials_script,omitempty"`
	Datacenter        string            `yaml:"datacenter,omitempty"`
	IsDisabled        bool              `yaml:"disabled,omitempty"`
	Password          string            `yaml:"password,omitempty"`
	Recorder          Recorder          `yaml:"recorder,omitempty"`
	SslCert           string            `yaml:"ssl_cert,omitempty"`
	SslKey            string            `yaml:"ssl_key,omitempty"`
	UseInsecureTLS    *bool             `yaml:"use_insecure_tls,omitempty"`
	Username          string            `yaml:"username,omitempty"`
	Name              string
}

type CredentialsScript struct {
	Path     string `yaml:"path,omitempty"`
	Schedule string `yaml:"schedule,omitempty"`
	Timeout  string `yaml:"timeout,omitempty"`
}

type CertificateScript struct {
	Path    string `yaml:"path,omitempty"`
	Timeout string `yaml:"timeout,omitempty"`
}

type Recorder struct {
	Path     string `yaml:"path,omitempty"`
	Mode     string `yaml:"mode,omitempty"`      // record or replay
	KeepLast string `yaml:"keep_last,omitempty"` // number of records to keep before overwriting
}

type OrderedConfig struct {
	Pollers Pollers `yaml:"Pollers,omitempty"`
}

type Pollers struct {
	namesInOrder []string
}

func (i *Pollers) UnmarshalYAML(n ast.Node) error {
	if n.Type() == ast.MappingType {
		for _, mn := range n.(*ast.MappingNode).Values {
			i.namesInOrder = append(i.namesInOrder, toString(mn.Key))
		}
	}

	return nil
}

func toString(n ast.Node) string {
	switch v := n.(type) {
	case *ast.StringNode:
		return v.Value
	case *ast.NullNode:
		return ""
	default:
		return n.String()
	}
}

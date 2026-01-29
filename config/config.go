package config

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"os"
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

	// Set the Name field for each poller
	for name, poller := range cfg.Pollers {
		poller.Name = name
	}

	return &cfg, nil
}

type ONTAP struct {
	Pollers        map[string]*Poller `yaml:"Pollers,omitempty"`
	Defaults       *Poller            `yaml:"Defaults,omitempty"`
	PollersOrdered []string           `yaml:"-"` // poller names in same order as yaml config
}

type Poller struct {
	Addr              string            `yaml:"addr,omitempty"`
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
	UseInsecureTLS    bool              `yaml:"use_insecure_tls,omitempty"`
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

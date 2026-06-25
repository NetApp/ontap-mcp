package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/server"
)

var logger = setupLogger()

type Globals struct {
	LogLevel   string `enum:"debug,info,warn,error" default:"info" env:"LOG_LEVEL" help:"Log level, one of: ${enum}"`
	ConfigPath string `name:"config" default:"ontap.yaml" env:"ONTAP_MCP_CONFIG" help:"ONTAP-MCP config path"`
}

type CLI struct {
	Globals
	Start    StartCmd    `cmd:"" help:"Start ONTAP MCP server"`
	Create   CreateCmd   `cmd:"" help:"Create ONTAP-MCP new server certificates"`
	Generate GenerateCmd `cmd:"" help:"Generate ONTAP MCP artifacts"`
}

type CreateCmd struct {
	DNSName   []string `default:"[]string{}" help:"localhost is always included. Comma-separated list or provide flag multiple times" env:"DNS_NAME"`
	Ipaddress []string `default:"[]string{}" help:"127.0.0.1 is always included. Comma-separated list or provide flag multiple times" env:"IP"`
	Days      int      `default:"365" help:"Number of days the certificate is valid" env:"DAYS"`
}

func (a *CreateCmd) Run(cli *CLI) error {
	GenerateAdminCerts(cli, "admin")
	return nil
}

type StartCmd struct {
	Host           string `default:"localhost" help:"Listening address"`
	Port           int    `default:"8080" help:"Listening port" env:"ONTAP_MCP_PORT"`
	InspectTraffic bool   `default:"false" help:"Inspect MCP HTTP traffic"`
	ReadOnly       bool   `default:"false" help:"Run MCP in read-only mode. This disables all tool calls that modify ONTAP state."`
	Stateless      bool   `default:"false" help:"Run in stateless mode (no mcp-session-id header validation). Required when deploying behind proxies or gateways that don't preserve session headers, e.g. on-premises data gateways."`
	JSONResponse   bool   `default:"false" help:"Respond with application/json instead of text/event-stream. Required when deploying behind proxies or gateways that do not relay SSE/chunked responses, e.g. on-premises data gateways."`
}

func (a *StartCmd) Run(cli *CLI) error {
	abs, err := filepath.Abs(cli.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to Abs config path=%s err=%w", cli.ConfigPath, err)
	}
	logger.Info("Reading config", slog.String("path", abs))
	cfg, err := config.ReadConfig(cli.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config path=%s err=%w", cli.ConfigPath, err)
	}

	opts := server.Options{
		Host:           cli.Start.Host,
		Port:           cli.Start.Port,
		InspectTraffic: cli.Start.InspectTraffic,
		ReadOnly:       cli.Start.ReadOnly,
		Stateless:      cli.Start.Stateless,
		JSONResponse:   cli.Start.JSONResponse,
	}

	app, err := server.NewApp(cfg, opts, logger)
	if err != nil {
		return err
	}
	app.StartServer()
	return nil
}

func Parse() {
	aCli := &CLI{}
	ctx := kong.Parse(aCli,
		kong.Name("ontap-mcp"),
		kong.Description("NetApp ONTAP MCP server"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	err := ctx.Run(aCli)
	ctx.FatalIfErrorf(err)
}

func setupLogger() *slog.Logger {
	level := slog.LevelInfo // default level

	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "DEBUG":
			level = slog.LevelDebug
		case "INFO":
			level = slog.LevelInfo
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		}
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				source.File = filepath.Base(source.File)
			}
			return a
		},
	})

	return slog.New(handler)
}

func GenerateAdminCerts(cli *CLI, flavor string) {
	certPath := fmt.Sprintf("cert/%s-cert.pem", flavor)
	if _, err := os.Stat(certPath); !os.IsNotExist(err) {
		log.Fatalf("%s already exists. not overwriting\n", certPath)
	}
	keyPath := fmt.Sprintf("cert/%s-key.pem", flavor)
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		log.Fatalf("%s already exists. not overwriting\n", keyPath)
	}

	pemCert, pemKey, err := generateSelfSignedCert(cli.Create.DNSName, cli.Create.Ipaddress, cli.Create.Days)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(certPath, pemCert, 0600); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s\n", certPath)

	if err := os.WriteFile(keyPath, pemKey, 0600); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s\n", keyPath)
}

// generateSelfSignedCert builds a self-signed ECDSA (P-256) certificate valid
// for the given number of days. "localhost" and 127.0.0.1 are always included
// as Subject Alternative Names; any additional DNS names and IP addresses are
// appended after trimming surrounding whitespace. Invalid IP addresses are
// skipped with a warning rather than producing a malformed SAN entry. It
// returns the PEM-encoded certificate and private key.
func generateSelfSignedCert(dnsNames, ipAddresses []string, days int) ([]byte, []byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Ontap-MCP"},
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Duration(days*24) * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	for _, ipaddress := range ipAddresses {
		ip := strings.TrimSpace(ipaddress)
		if ip == "" {
			continue
		}
		parsed := net.ParseIP(ip)
		if parsed == nil {
			log.Printf("skipping invalid IP address %q\n", ip)
			continue
		}
		template.IPAddresses = append(template.IPAddresses, parsed)
	}

	for _, n := range dnsNames {
		name := strings.TrimSpace(n)
		if name != "" {
			template.DNSNames = append(template.DNSNames, name)
		}
	}

	// Create self-signed certificate.
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if certPEM == nil {
		return nil, nil, errors.New("failed to encode certificate to PEM")
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if keyPEM == nil {
		return nil, nil, errors.New("failed to encode key to PEM")
	}

	return certPEM, keyPEM, nil
}

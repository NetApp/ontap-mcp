package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/server"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var logger = setupLogger()

type Globals struct {
	LogLevel   string `enum:"debug,info,warn,error" default:"info" env:"LOG_LEVEL" help:"Log level, one of: ${enum}"`
	ConfigPath string `name:"config" default:"ontap.yaml" env:"CONFIG" help:"ONTAP-MCP config path"`
}

type CLI struct {
	Globals
	Start StartCmd `cmd:"" help:"Start ONTAP MCP server"`
}

type StartCmd struct {
	Host           string `default:"localhost" help:"Listening address"`
	Port           int    `default:"8080" help:"Listening port" env:"ONTAP_MCP_PORT"`
	InspectTraffic bool   `default:"false" help:"Inspect MCP HTTP traffic"`
	ReadOnly       bool   `default:"false" help:"Run MCP in read-only mode. This disables all tool calls that modify ONTAP state."`
}

func (a *StartCmd) Run(cli *CLI) error {
	cfg, err := config.ReadConfig(cli.ConfigPath)
	if err != nil {
		return err
	}

	opts := server.Options{
		Host:           cli.Start.Host,
		Port:           cli.Start.Port,
		InspectTraffic: cli.Start.InspectTraffic,
		ReadOnly:       cli.Start.ReadOnly,
	}

	app := server.NewApp(cfg, opts, logger)
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

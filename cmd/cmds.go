package cmd

import (
	"fmt"
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
	ConfigPath string `name:"config" default:"ontap.yaml" env:"ONTAP_MCP_CONFIG" help:"ONTAP-MCP config path"`
}

type CLI struct {
	Globals
	Start    StartCmd    `cmd:"" help:"Start ONTAP MCP server"`
	Generate GenerateCmd `cmd:"" help:"Generate ONTAP MCP artifacts"`
}

type StartCmd struct {
	server.Config `embed:""`
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

	app := server.NewApp(cfg, a.Config, logger)
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

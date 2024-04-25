// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/goschtalt/goschtalt"
	_ "github.com/goschtalt/goschtalt/pkg/typical"
	_ "github.com/goschtalt/yaml-decoder"
	_ "github.com/goschtalt/yaml-encoder"
	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/touchstone/touchhttp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

const (
	applicationName = "argus"
	apiBase         = "api/v1"
	defaultKeyID    = "current"
)

var (
	commit  = "undefined"
	version = "undefined"
	date    = "undefined"
	builtBy = "undefined"
)

type CLI struct {
	Dev   bool     `optional:"" short:"d" help:"Run in development mode."`
	Show  bool     `optional:"" short:"s" help:"Show the configuration and exit."`
	Graph string   `optional:"" short:"g" help:"Output the dependency graph to the specified file."`
	Files []string `optional:"" short:"f" help:"Specific configuration files or directories."`
}

// Provides a named type so it's a bit easier to flow through & use in fx.
type cliArgs []string
type CandlelightConfigIn struct {
	fx.In
	C candlelight.Config `name:"tracing_initial_config" optional:"true"`
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}
	}()

	err := argus(os.Args[1:], true)

	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(-1)
}

func argus(arguments []string, run bool) error {
	var (
		gscfg *goschtalt.Config

		// Capture the dependency tree in case we need to debug something.
		g fx.DotGraph

		// Capture the command line arguments.
		cli *CLI
	)
	app := fx.New(
		fx.Supply(cliArgs(arguments)),
		fx.Populate(cli),
		fx.Populate(gscfg),
		fx.Populate(&g),

		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		metric.ProvideMetrics(),
		auth.Provide("auth.inbound"),
		touchhttp.Provide(),
		touchstone.Provide(),
		store.ProvideHandlers(),
		db.Provide(),
		fx.Provide(
			provideCLI,
			provideLogger,
			provideConfig,
			consts,
			goschtalt.UnmarshalFunc[sallust.Config]("logging"),
			goschtalt.UnmarshalFunc[db.Configs]("store"),
			goschtalt.UnmarshalFunc[touchstone.Config]("prometheus"),
			goschtalt.UnmarshalFunc[candlelight.Config]("tracing"),
			goschtalt.UnmarshalFunc[store.UserInputValidationConfig]("userInputValidation"),
			goschtalt.UnmarshalFunc[Auth]("authx"),
			// goschtalt.UnmarshalFunc[touchhttp.Config]("prometheus.handler"),

			fx.Annotated{
				Name:   "servers.health.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.health.http"),
			},
			fx.Annotated{
				Name:   "servers.metrics.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.metrics.http"),
			},
			fx.Annotated{
				Name:   "servers.primary.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.primary.http"),
			},
			candlelight.New,
		),

		arrangehttp.ProvideServer("servers.health"),
		arrangehttp.ProvideServer("servers.metrics"),
		arrangehttp.ProvideServer("servers.primary"),
	)

	if cli != nil && cli.Graph != "" {
		_ = os.WriteFile(cli.Graph, []byte(g), 0644)
	}

	if cli != nil && cli.Dev {
		defer func() {
			if gscfg != nil {
				fmt.Fprintln(os.Stderr, gscfg.Explain().String())
			}
		}()
	}

	if err := app.Err(); err != nil {
		return err
	}

	if run {
		app.Run()
	}

	return nil
}

// Provide the constants in the main package for other uber fx components to use.
type ConstOut struct {
	fx.Out
	APIBase      string `name:"api_base"`
	DefaultKeyID string `name:"default_key_id"`
}

func consts() ConstOut {
	return ConstOut{
		APIBase:      apiBase,
		DefaultKeyID: defaultKeyID,
	}
}

func provideCLI(args cliArgs) (*CLI, error) {
	return provideCLIWithOpts(args, false)
}

func provideCLIWithOpts(args cliArgs, testOpts bool) (*CLI, error) {
	var cli CLI

	// Create a no-op option to satisfy the kong.New() call.
	var opt kong.Option = kong.OptionFunc(
		func(*kong.Kong) error {
			return nil
		},
	)

	if testOpts {
		opt = kong.Writers(nil, nil)
	}

	parser, err := kong.New(&cli,
		kong.Name(applicationName),
		kong.Description("The cpe agent for Xmidt service.\n"+
			fmt.Sprintf("\tVersion:  %s\n", version)+
			fmt.Sprintf("\tDate:     %s\n", date)+
			fmt.Sprintf("\tCommit:   %s\n", commit)+
			fmt.Sprintf("\tBuilt By: %s\n", builtBy),
		),
		kong.UsageOnError(),
		opt,
	)
	if err != nil {
		return nil, err
	}

	if testOpts {
		parser.Exit = func(_ int) { panic("exit") }
	}

	_, err = parser.Parse(args)
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	return &cli, nil
}

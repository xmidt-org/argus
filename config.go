// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: LicenseRef-COMCAST

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/goschtalt/goschtalt"
	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db"
	"github.com/xmidt-org/argus/store/dynamodb"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/touchstone"
	"gopkg.in/dealancer/validate.v2"
)

// Config is the top level configuration for the caduceus service.  Everything
// is contained in this structure or it will intentially cause a failure.
type Config struct {
	Logging             sallust.Config
	Tracing             candlelight.Config
	Prometheus          touchstone.Config
	Servers             Servers
	Store               db.Configs
	JWTValidator        auth.JWTValidator
	UserInputValidation store.UserInputValidationConfig
	Auth                auth.Config
	Health              HealthConfig
}

type Servers struct {
	Health  HealthServer
	Metrics MetricsServer
	Primary PrimaryServer
}

type HealthServer struct {
	HTTP                  arrangehttp.ServerConfig
	DisableHTTPKeepAlives bool
	Path                  HealthPath `validate:"empty=false"`
}

type HealthPath string

type HealthConfig struct {
	DisableLogging bool
	Custom         CustomConfig
}

type CustomConfig struct {
	Server string
}
type MetricsServer struct {
	HTTP                  arrangehttp.ServerConfig
	DisableHTTPKeepAlives bool
	MetricsOptions        MetricsOption
	Path                  MetricsPath `validate:"empty=false"`
}

type MetricsPath string

type PrimaryServer struct {
	HTTP                  arrangehttp.ServerConfig
	DisableHTTPKeepAlives bool
}

// httpClientTimeout contains timeouts for an HTTP client and its requests.
type HttpClientTimeout struct {
	// ClientTimeout is HTTP Client Timeout.
	ClientTimeout time.Duration

	// NetDialerTimeout is the net dialer timeout
	NetDialerTimeout time.Duration
}

type MetricsOption struct {
	Namespace string
	Subsystem string
}

// Collect and process the configuration files and env vars and
// produce a configuration object.
func provideConfig(cli *CLI) (*goschtalt.Config, error) {
	gs, err := goschtalt.New(
		goschtalt.StdCfgLayout(applicationName, cli.Files...),
		goschtalt.ConfigIs("twoWords"),
		goschtalt.DefaultUnmarshalOptions(
			goschtalt.WithValidator(
				goschtalt.ValidatorFunc(validate.Validate),
			),
		),

		// Seed the program with the default, built-in configuration.
		// Mark this as a default so it is ordered correctly.
		goschtalt.AddValue("built-in", goschtalt.Root, defaultConfig,
			goschtalt.AsDefault()),
	)
	if err != nil {
		return nil, err
	}

	if cli.Show {
		// handleCLIShow handles the -s/--show option where the configuration is
		// shown, then the program is exited.
		//
		// Exit with success because if the configuration is broken it will be
		// very hard to debug where the problem originates.  This way you can
		// see the configuration and then run the service with the same
		// configuration to see the error.

		fmt.Fprintln(os.Stdout, gs.Explain().String())

		out, err := gs.Marshal()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintln(os.Stdout, "## Final Configuration\n---\n"+string(out))
		}

		os.Exit(0)
	}

	var tmp Config
	err = gs.Unmarshal(goschtalt.Root, &tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "There is a critical error in the configuration.")
		fmt.Fprintln(os.Stderr, "Run with -s/--show to see the configuration.")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Exit here to prevent a very difficult to debug error from occurring.
		os.Exit(-1)
	}

	return gs, nil
}

// -----------------------------------------------------------------------------
// Keep the default configuration at the bottom of the file so it is easy to
// see what the default configuration is.
// -----------------------------------------------------------------------------

// TODO: update default values to match what's expected of argus
var defaultConfig = Config{
	Servers: Servers{
		Health: HealthServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: ":80",
			},
			Path: HealthPath("/"),
		},
		Metrics: MetricsServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: "127.0.0.1:9361",
			},
			Path: MetricsPath("/metrics"),
		},
		Primary: PrimaryServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: ":443",
			},
		},
	},
	Prometheus: touchstone.Config{
		DefaultNamespace: "xmidt",
		DefaultSubsystem: "argus",
	},
	Tracing: candlelight.Config{
		ApplicationName: applicationName,
	},
	Store: db.Configs{
		Dynamo: &dynamodb.Config{
			Region:    "us-east-2",
			AccessKey: "accessKey",
			SecretKey: "secretKey",
		},
	},
}

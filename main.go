// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/bascule/basculechecks"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/touchstone/touchhttp"
	"go.uber.org/fx"
)

const (
	applicationName = "argus"
	apiBase         = "api/v1"
	defaultKeyID    = "current"
)

var (
	GitCommit = "undefined"
	Version   = "undefined"
	BuildTime = "undefined"
)

type CandlelightConfigIn struct {
	fx.In
	C candlelight.Config `name:"tracing_initial_config" optional:"true"`
}

func main() {
	v, logger, err := setup(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app := fx.New(
		arrange.LoggerFunc(logger.Sugar().Infof),
		arrange.ForViper(v),
		fx.Supply(logger),
		metric.ProvideMetrics(),
		auth.Provide("authx.inbound"),
		touchhttp.Provide(),
		touchstone.Provide(),
		store.ProvideHandlers(),
		db.Provide(),
		fx.Provide(
			consts,
			arrange.UnmarshalKey("userInputValidation", store.UserInputValidationConfig{}),
			arrange.UnmarshalKey("prometheus", touchstone.Config{}),
			arrange.UnmarshalKey("prometheus.handler", touchhttp.Config{}),
			fx.Annotated{
				Name:   "encoded_basic_auths",
				Target: arrange.UnmarshalKey("authx.inbound", basculehttp.EncodedBasicKeys{}),
			},
			arrange.UnmarshalKey("authx.inbound.capabilities", basculechecks.CapabilitiesValidatorConfig{}),
			fx.Annotated{
				Name:   "tracing_initial_config",
				Target: arrange.UnmarshalKey("tracing", candlelight.Config{}),
			},
			func(in CandlelightConfigIn) candlelight.Config {
				in.C.ApplicationName = applicationName
				return in.C
			},
			candlelight.New,
		),
		provideServers(),
	)

	switch err := app.Err(); {
	case errors.Is(err, pflag.ErrHelp):
		return
	case err == nil:
		app.Run()
	default:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
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

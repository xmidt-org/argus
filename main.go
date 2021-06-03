/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/sallust/sallustkit"
	"github.com/xmidt-org/themis/xmetrics/xmetricshttp"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"github.com/xmidt-org/webpa-common/basculemetrics"

	"github.com/InVisionApp/go-health"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/themis/config"
	"github.com/xmidt-org/themis/xhealth"
	"github.com/xmidt-org/themis/xhttp/xhttpserver"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	applicationName = "argus"

	DefaultKeyID = "current"
	apiBase      = "api/v1"
)

var (
	GitCommit = "undefined"
	Version   = "undefined"
	BuildTime = "undefined"
)

func setupViper(in config.ViperIn, v *viper.Viper) (err error) {
	if printVersion, _ := in.FlagSet.GetBool("version"); printVersion {
		printVersionInfo()
	}
	if file, _ := in.FlagSet.GetString("file"); len(file) > 0 {
		v.SetConfigFile(file)
		err = v.ReadInConfig()
	} else {
		v.SetConfigName(string(in.Name))
		v.AddConfigPath(fmt.Sprintf("/etc/%s", in.Name))
		v.AddConfigPath(fmt.Sprintf("$HOME/.%s", in.Name))
		v.AddConfigPath(".")
		err = v.ReadInConfig()
	}

	if err != nil {
		return
	}

	if debug, _ := in.FlagSet.GetBool("debug"); debug {
		v.Set("log.level", "DEBUG")
	}

	return nil
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
		fx.Supply(logger, v),
		provideMetrics(),
		metric.ProvideMetrics(),
		basculechecks.ProvideMetricsVec(),
		basculemetrics.ProvideMetricsVec(),
		auth.ProvidePrimaryServerChain(apiBase),
		store.ProvideHandlers(),
		fx.Provide(
			backwardsCompatibleLogger,
			backwardsCompatibleUnmarshaller,
			auth.ProfilesUnmarshaler{
				ConfigKey:        "authx.inbound.profiles",
				SupportedServers: []string{"primary"}}.Annotated(),
			db.Provide,
			xhealth.Unmarshal("health"),
			provideServerChainFactory,
			xmetricshttp.Unmarshal("prometheus", promhttp.HandlerOpts{}),
			xhttpserver.Unmarshal{Key: "servers.primary", Optional: true}.Annotated(),
			xhttpserver.Unmarshal{Key: "servers.metrics", Optional: true}.Annotated(),
			xhttpserver.Unmarshal{Key: "servers.health", Optional: true}.Annotated(),
			candlelight.New,
			func(u config.Unmarshaller) (candlelight.Config, error) {
				var config candlelight.Config
				err := u.UnmarshalKey("tracing", &config)
				if err != nil {
					return candlelight.Config{}, err
				}
				config.ApplicationName = applicationName
				return config, nil
			},
			fx.Annotated{
				Name: "primary_bascule_parse_url",
				Target: func() basculehttp.ParseURL {
					return basculehttp.CreateRemovePrefixURLFunc("/"+apiBase+"/", basculehttp.DefaultParseURLFunc)
				},
			},
		),

		fx.Invoke(
			xhealth.ApplyChecks(
				&health.Config{
					Name:     applicationName,
					Interval: 24 * time.Hour,
					Checker: xhealth.NopCheckable{
						Details: map[string]interface{}{
							"StartTime": time.Now().UTC().Format(time.RFC3339),
						},
					},
				},
			),
			BuildPrimaryRoutes,
			BuildMetricsRoutes,
			BuildHealthRoutes,
		),
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

func backwardsCompatibleLogger(l *zap.Logger) log.Logger {
	return sallustkit.Logger{
		Zap: l,
	}
}

func backwardsCompatibleUnmarshaller(v *viper.Viper) config.Unmarshaller {
	return config.ViperUnmarshaller{
		Viper: v,
	}
}

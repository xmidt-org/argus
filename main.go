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
	"fmt"
	"github.com/xmidt-org/bascule/basculehttp"
	"os"
	"runtime"
	"time"

	"github.com/justinas/alice"
	"github.com/xmidt-org/argus/auth"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/themis/xmetrics/xmetricshttp"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"github.com/xmidt-org/webpa-common/basculemetrics"

	"github.com/InVisionApp/go-health"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/themis/config"
	"github.com/xmidt-org/themis/xhealth"
	"github.com/xmidt-org/themis/xhttp/xhttpserver"
	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/themis/xlog/xloghttp"
	"go.uber.org/fx"
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

func setupFlagSet(fs *pflag.FlagSet) error {
	fs.StringP("file", "f", "", "the configuration file to use.  Overrides the search path.")
	fs.BoolP("debug", "d", false, "enables debug logging.  Overrides configuration.")
	fs.BoolP("version", "v", false, "print version and exit")

	return nil
}

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

func printVersionInfo() {
	fmt.Fprintf(os.Stdout, "%s:\n", applicationName)
	fmt.Fprintf(os.Stdout, "  version: \t%s\n", Version)
	fmt.Fprintf(os.Stdout, "  go version: \t%s\n", runtime.Version())
	fmt.Fprintf(os.Stdout, "  built time: \t%s\n", BuildTime)
	fmt.Fprintf(os.Stdout, "  git commit: \t%s\n", GitCommit)
	fmt.Fprintf(os.Stdout, "  os/arch: \t%s/%s\n", runtime.GOOS, runtime.GOARCH)
	os.Exit(0)
}

func main() {
	app := fx.New(
		xlog.Logger(),
		config.CommandLine{Name: applicationName}.Provide(setupFlagSet),
		provideMetrics(),
		metric.ProvideMetrics(),
		basculechecks.ProvideMetricsVec(),
		basculemetrics.ProvideMetricsVec(),

		auth.LogOptionsProvider{ServerName: "primary"}.Provide(),
		auth.ProvidePrimaryBasculeConstructor(),
		auth.ProvidePrimaryBasculeEnforcer(),
		auth.ProvidePrimaryTokenFactory(),

		fx.Provide(
			config.ProvideViper(setupViper),
			xlog.Unmarshal("log"),
			xloghttp.ProvideStandardBuilders,
			//auth.ProvidePrimaryChain,
			db.Provide,
			store.Provide,
			xhealth.Unmarshal("health"),
			provideServerChainFactory,
			xmetricshttp.Unmarshal("prometheus", promhttp.HandlerOpts{}),
			xhttpserver.Unmarshal{Key: "servers.primary", Optional: true}.Annotated(),
			xhttpserver.Unmarshal{Key: "servers.metrics", Optional: true}.Annotated(),
			xhttpserver.Unmarshal{Key: "servers.health", Optional: true}.Annotated(),

			auth.UnmarshalProfiles("bascule.inbound.profiles"),
			auth.ProfileProvider{ServerName: "primary"}.Annotated(),
			auth.BasculeMetricsListenerProvider{ServerName: "primary"}.Annotated(),

			fx.Annotated{
				Name: "primary_alice_listener",
				Target: func(in auth.PrimaryBasculeMetricListenerIn) alice.Constructor {
					return basculehttp.NewListenerDecorator(in.Listener)
				},
			},
			fx.Annotated{
				Name: "primary_auth_chain",
				Target: func(in auth.PrimaryChainIn) alice.Chain {
					return alice.New(in.SetLogger, in.Constructor, in.Enforcer, in.Listener)
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
			//getBasculeCapabilityMeasures,
			getProfiles, // ok
			////getPrimaryProfile,
			//getBasculeMetrics, // ok
			//getBasculeCapabilityMetrics, // ok
			//getBasculeCapabilityMeasures,
			//getPrimaryConstructorOptions,
			//getPrimaryBasculeMetricListenerIn,
			//getPrimaryLogger,
			//getPrimaryValidators,
			//getChains,
			BuildPrimaryRoutes,
			BuildMetricsRoutes,
			BuildHealthRoutes,
		),
	)

	switch err := app.Err(); err {
	case pflag.ErrHelp:
		return
	case nil:
		app.Run()
	default:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func getProfiles(profiles map[string]*auth.Profile) {
	for k, v := range profiles {
		fmt.Printf("k: %s, v: %v", k, v)
	}
}

func getPrimaryProfile(in auth.PrimaryBasculeProfileIn) {
	fmt.Printf("This is the primary profile: %v", in.Profile)
}

func getChains(in auth.PrimaryChainIn) {
	fmt.Printf("setLogger: %v", in)
}

type LogDebugging struct {
	fx.In
	SetLogger alice.Constructor `name:"primary_alice_set_logger"`
}

func getPrimaryValidators(in auth.PrimaryBearerValidatorsIn) {
	fmt.Println(in)
}

func getPrimaryLogger(in LogDebugging) {
	fmt.Print(in)
}

func getPrimaryConstructorOptions(in auth.PrimaryCOptionsIn) {
	fmt.Printf("This is the primary coptions %v", in)
}
func getPrimaryBasculeMetricListenerIn(in auth.PrimaryBasculeMetricListenerIn) {
	fmt.Printf("bascule metric listener in: %v", in)
}

func getBasculeMetrics(in auth.BasculeMetricsProviderIn) {
	fmt.Printf("this is bascule metrics provider: %v", in)
}

func getBasculeCapabilityMeasures(in auth.PrimaryCapabilityValidatorIn) {
	fmt.Printf("This is the bascule capability measures: %v", in.Measures)
}

func getBasculeCapabilityMetrics(in auth.BasculeCapabilityMetricsProviderIn) {
	fmt.Printf("this is basculeCapability metrics: %v", in)
}

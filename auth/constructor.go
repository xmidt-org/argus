package auth

import (
	"fmt"
	"reflect"

	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/webpa-common/basculechecks"

	"github.com/go-kit/kit/log/level"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/key"
	"github.com/xmidt-org/webpa-common/basculemetrics"
	"go.uber.org/fx"
)

type primaryCOptionsIn struct {
	LoggerIn
	Options []basculehttp.COption `group:"primary_bascule_constructor_options"`
}

type primaryBearerTokenFactoryIn struct {
	LoggerIn
	DefaultKeyID string         `name:"primary_bearer_default_kid"`
	Resolver     key.Resolver   `name:"primary_bearer_key_resolver"`
	Leeway       bascule.Leeway `name:"primary_bearer_leeway"`
}

type primaryBasculeMetricListenerIn struct {
	fx.In
	Listener *basculemetrics.MetricListener `name:"primary_bascule_metric_listener"`
}

func providePrimaryBasculeConstructorOptions(apiBase string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in primaryBasculeMetricListenerIn) basculehttp.COption {
				return basculehttp.WithCErrorResponseFunc(in.Listener.OnErrorResponse)
			},
		},
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in primaryBearerTokenFactoryIn) basculehttp.COption {
				if in.Resolver == nil {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "returning nil bearer token factory option as resolver was not defined", "server", "primary")
					return nil
				}
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building bearer token factory option", "server", "primary")
				return basculehttp.WithTokenFactory("Bearer", basculehttp.BearerTokenFactory{
					DefaultKeyId: in.DefaultKeyID,
					Resolver:     in.Resolver,
					Parser:       bascule.DefaultJWTParser,
					Leeway:       in.Leeway,
				})
			},
		},
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in primaryProfileIn) (basculehttp.COption, error) {
				if in.Profile == nil || len(in.Profile.Basic) < 1 {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "returning nil basic token factory option as config was not provided", "server", "primary")
					return nil, nil
				}
				basicTokenFactory, err := basculehttp.NewBasicTokenFactoryFromList(in.Profile.Basic)
				if err != nil {
					return nil, err
				}
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building basic token factory option", "server", "primary")
				return basculehttp.WithTokenFactory("Basic", basicTokenFactory), nil
			},
		},

		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func() basculehttp.COption {
				return basculehttp.WithParseURLFunc(basculehttp.CreateRemovePrefixURLFunc("/"+apiBase+"/", basculehttp.DefaultParseURLFunc))
			},
		},
	)
}

func providePrimaryBasculeConstructor(apiBase string) fx.Option {
	return fx.Options(
		providePrimaryBasculeConstructorOptions(apiBase),
		fx.Provide(
			fx.Annotated{
				Name: "primary_alice_constructor",
				Target: func(in primaryCOptionsIn) alice.Constructor {
					in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building alice constructor from bascule constructor options", "server", "primary")
					var filteredOptions []basculehttp.COption
					for _, option := range in.Options {
						if option == nil || reflect.ValueOf(option).IsNil() {
							continue
						}
						filteredOptions = append(filteredOptions, option)
					}
					return basculehttp.NewConstructor(filteredOptions...)
				},
			}),
	)
}

func providePrimaryTokenFactory() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name: "primary_bearer_default_kid",
			Target: func() string {
				return "current"
			},
		},
		fx.Annotated{
			Name: "primary_bearer_key_resolver",
			Target: func(in primaryProfileIn) (key.Resolver, error) {
				if anyNil(in.Profile, in.Profile.Bearer) {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "returning nil bearer key resolver as config wasn't provided", "server", "primary")
					return nil, nil
				}
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building bearer key resolver option", "server", "primary")
				return in.Profile.Bearer.Keys.NewResolver()
			},
		},
		fx.Annotated{
			Name: "primary_bearer_leeway",
			Target: func(in primaryProfileIn) bascule.Leeway {
				if anyNil(in.Profile, in.Profile.Bearer) {
					return bascule.Leeway{}
				}
				return in.Profile.Bearer.Leeway
			},
		},
	)
}

type basculeMetricsFactoryIn struct {
	LoggerIn
	NBFHistogram      *prometheus.HistogramVec `name:"auth_from_nbf_seconds"`
	ExpHistogram      *prometheus.HistogramVec `name:"auth_from_exp_seconds"`
	ValidationOutcome *prometheus.CounterVec   `name:"auth_validation"`
}

type basculeCapabilityMetricsFactoryIn struct {
	LoggerIn
	CapabilityCheckOutcome *prometheus.CounterVec `name:"auth_capability_check"`
}

type basculeMetricsListenerFactory struct {
	serverName string
}

func (b basculeMetricsListenerFactory) new(in basculeMetricsFactoryIn) (*basculemetrics.MetricListener, error) {
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building auth validation measures", "server", b.serverName)
	nbfHistogramVec, err := in.NBFHistogram.CurryWith(prometheus.Labels{"server": b.serverName})
	if err != nil {
		return nil, err
	}
	expHistogramVec, err := in.ExpHistogram.CurryWith(prometheus.Labels{"server": b.serverName})
	if err != nil {
		return nil, err
	}
	validationOutcomeCounterVec, err := in.ValidationOutcome.CurryWith(prometheus.Labels{"server": b.serverName})
	if err != nil {
		return nil, err
	}

	validationMeasures := &basculemetrics.AuthValidationMeasures{
		NBFHistogram:      gokitprometheus.NewHistogram(nbfHistogramVec.(*prometheus.HistogramVec)),
		ExpHistogram:      gokitprometheus.NewHistogram(expHistogramVec.(*prometheus.HistogramVec)),
		ValidationOutcome: gokitprometheus.NewCounter(validationOutcomeCounterVec),
	}
	return basculemetrics.NewMetricListener(validationMeasures), nil

}

func (b basculeMetricsListenerFactory) annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_bascule_metric_listener", b.serverName),
		Target: b.new,
	}
}

type basculeCapabilityMetricFactory struct {
	serverName string
}

func (b basculeCapabilityMetricFactory) new(in basculeCapabilityMetricsFactoryIn) (*basculechecks.AuthCapabilityCheckMeasures, error) {
	capabilityCheckOutcomeCounterVec, err := in.CapabilityCheckOutcome.CurryWith(prometheus.Labels{"server": b.serverName})
	if err != nil {
		return nil, err
	}
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building auth capability measures", "server", b.serverName)
	return &basculechecks.AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: gokitprometheus.NewCounter(capabilityCheckOutcomeCounterVec),
	}, nil
}

func (b basculeCapabilityMetricFactory) annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_capability_measures", b.serverName),
		Target: b.new,
	}
}

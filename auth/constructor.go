package auth

import (
	"fmt"

	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/webpa-common/basculechecks"

	"github.com/go-kit/kit/log"
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

type PrimaryCOptionsIn struct {
	fx.In
	Logger  log.Logger
	Options []basculehttp.COption `group:"primary_bascule_constructor_options"`
}

type PrimaryTokenFactoryIn struct {
	fx.In
	Logger       log.Logger
	DefaultKeyID string         `name:"primary_bearer_default_kid"`
	Resolver     key.Resolver   `name:"primary_bearer_key_resolver" optional:"true"`
	Leeway       bascule.Leeway `name:"primary_bearer_leeway"`
}

type PrimaryBasculeMetricListenerIn struct {
	fx.In
	Listener *basculemetrics.MetricListener `name:"primary_bascule_metric_listener"`
}

func ProvidePrimaryBasculeConstructor() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in PrimaryBasculeMetricListenerIn) basculehttp.COption {
				return basculehttp.WithCErrorResponseFunc(in.Listener.OnErrorResponse)
			},
		},
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in PrimaryTokenFactoryIn) basculehttp.COption {
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "providing token factory option", "server", "primary")
				if in.Resolver == nil {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "providing nil token factory option as resolver was not defined", "server", "primary")
					return nil
				}
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
				if in.Profile == nil {
					return nil, nil
				}
				basicTokenFactory, factoryErr := basculehttp.NewBasicTokenFactoryFromList(in.Profile.Basic)
				if factoryErr != nil {
					return nil, factoryErr
				}
				return basculehttp.WithTokenFactory("Basic", basicTokenFactory), nil
			},
		},
		fx.Annotated{
			Name: "primary_alice_constructor",
			Target: func(in PrimaryCOptionsIn) alice.Constructor {
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "providing alice constructor from bascule constructor options", "server", "primary")
				if anyNil(in.Options) {
					in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "providing nil alice constructor as some options were undefined", "server", "primary")
					return nil
				}
				return basculehttp.NewConstructor(in.Options...)
			},
		},
	)
}

func ProvidePrimaryTokenFactory() fx.Option {
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
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "providing bearer key resolver option", "server", "primary")
				if anyNil(in.Profile, in.Profile.Bearer) {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "No bearer key resolver provided", "server", "primary")
					return nil, nil
				}
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

type BasculeMetricsProviderIn struct {
	fx.In
	Logger            log.Logger
	NBFHistogram      *prometheus.HistogramVec `name:"auth_from_nbf_seconds"`
	ExpHistogram      *prometheus.HistogramVec `name:"auth_from_exp_seconds"`
	ValidationOutcome *prometheus.CounterVec   `name:"auth_validation"`
}

type BasculeCapabilityMetricsProviderIn struct {
	fx.In
	Logger                 log.Logger
	CapabilityCheckOutcome *prometheus.CounterVec `name:"auth_capability_check"`
}

type BasculeMetricsListenerProvider struct {
	ServerName string
}

func (b BasculeMetricsListenerProvider) Provide(in BasculeMetricsProviderIn) (*basculemetrics.MetricListener, error) {
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "providing auth validation measures", "server", b.ServerName)
	nbfHistogramVec, err := in.NBFHistogram.CurryWith(prometheus.Labels{"server": b.ServerName})
	if err != nil {
		return nil, err
	}
	expHistogramVec, err := in.ExpHistogram.CurryWith(prometheus.Labels{"server": b.ServerName})
	if err != nil {
		return nil, err
	}
	validationOutcomeCounterVec, err := in.ValidationOutcome.CurryWith(prometheus.Labels{"server": b.ServerName})
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

func (b BasculeMetricsListenerProvider) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_bascule_metric_listener", b.ServerName),
		Target: b.Provide,
	}
}

type BasculeCapabilityMetricProvider struct {
	ServerName string
}

func (b BasculeCapabilityMetricProvider) Provide(in BasculeCapabilityMetricsProviderIn) (*basculechecks.AuthCapabilityCheckMeasures, error) {
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "providing auth capability measures", "server", b.ServerName)
	capabilityCheckOutcomeCounterVec, err := in.CapabilityCheckOutcome.CurryWith(prometheus.Labels{"server": b.ServerName})
	if err != nil {
		return nil, err
	}
	return &basculechecks.AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: gokitprometheus.NewCounter(capabilityCheckOutcomeCounterVec),
	}, nil
}

func (b BasculeCapabilityMetricProvider) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_capability_measures", b.ServerName),
		Target: b.Provide,
	}
}

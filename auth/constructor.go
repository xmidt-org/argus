package auth

import (
	"fmt"

	"github.com/xmidt-org/webpa-common/basculechecks"

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
	Options []basculehttp.COption `group:"primary_bascule_constructor_options"`
}

type PrimaryTokenFactoryIn struct {
	fx.In
	DefaultKeyID string         `name:"primary_bearer_default_kid"`
	Resolver     key.Resolver   `name:"primary_bearer_key_resolver" optional:"true"`
	Leeway       bascule.Leeway `name:"primary_bearer_leeway" optional:"true"`
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
				if in.Resolver == nil {
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
			Name: "primary_alice_constructor",
			Target: func(in PrimaryCOptionsIn) alice.Constructor {
				if anyNil(in.Options) {
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
				if in.Profile == nil {
					return nil, nil
				}
				return in.Profile.Bearer.Keys.NewResolver()
			},
		},
		fx.Annotated{
			Name: "primary_bearer_leeway",
			Target: func(in primaryProfileIn) bascule.Leeway {
				if in.Profile == nil {
					return bascule.Leeway{}
				}
				return in.Profile.Bearer.Leeway
			},
		},
	)
}

type BasculeMetricsProviderIn struct {
	fx.In

	NBFHistogram      *prometheus.HistogramVec `name:"auth_from_nbf_seconds"`
	ExpHistogram      *prometheus.HistogramVec `name:"auth_from_exp_seconds"`
	ValidationOutcome *prometheus.CounterVec   `name:"auth_validation"`
}

type BasculeCapabilityMetricsProviderIn struct {
	fx.In

	CapabilityCheckOutcome *prometheus.CounterVec `name:"auth_capability_check"`
}

type BasculeMetricsListenerProvider struct {
	ServerName string
}

func (b BasculeMetricsListenerProvider) Provide(in BasculeMetricsProviderIn) (*basculemetrics.MetricListener, error) {
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

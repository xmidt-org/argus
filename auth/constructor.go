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

type PrimaryBasculeProfileIn struct {
	Profile *profile `name:"primary_profile"`
}

type PrimaryTokenFactoryIn struct {
	fx.Out
	DefaultKeyID string         `name:"primary_bearer_default_kid"`
	Resolver     key.Resolver   `name:"primary_bearer_key_resolver"`
	Leeway       bascule.Leeway `name:"primary_bearer_leeway"`
}

type PrimaryBasculeErrorResponseIn struct {
	BasculeMetricListener *basculemetrics.MetricListener `name:"primary_bascule_metric_listener"`
}

func providePrimaryBasculeConstructor() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in PrimaryBasculeErrorResponseIn) basculehttp.COption {
				return basculehttp.WithCErrorResponseFunc(in.BasculeMetricListener.OnErrorResponse)
			},
		},
		fx.Annotated{
			Group: "primary_bascule_constructor_options",
			Target: func(in PrimaryTokenFactoryIn) basculehttp.COption {
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
				return basculehttp.NewConstructor(in.Options...)
			},
		},
	)
}

func providePrimaryTokenFactory() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name: "primary_default_bearer_kid",
			Target: func() string {
				return "current"
			},
		},
		fx.Annotated{
			Name: "primary_bearer_key_resolver",
			Target: func(in PrimaryBasculeProfileIn) (key.Resolver, error) {
				return in.Profile.Bearer.Keys.NewResolver()
			},
		},
		fx.Annotated{
			Name: "primary_bearer_leeway",
			Target: func(in PrimaryBasculeProfileIn) bascule.Leeway {
				return in.Profile.Bearer.Leeway
			},
		},
	)
}

type basculeMetricsProviderIn struct {
	fx.In

	//TODO: We can update webpa-common/basculemetrics to provide these vectors
	NBFHistogram      prometheus.HistogramVec `name:"auth_from_nbf_seconds"`
	ExpHistogram      prometheus.HistogramVec `name:"auth_from_exp_seconds"`
	ValidationOutcome prometheus.CounterVec   `name:"auth_validation"`
}

type basculeCapabilityMetricsProviderIn struct {
	fx.In

	//TODO: We can update webpa-common/basculechecks to provide these vectors
	CapabilityCheckOutcome prometheus.CounterVec `name:"auth_capability_check"`
}

type basculeMetricsListenerProvider struct {
	ServerName string
}

func (b basculeMetricsListenerProvider) provide(in basculeMetricsProviderIn) (*basculemetrics.MetricListener, error) {
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

func (b basculeMetricsListenerProvider) annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_bascule_metric_listener", b.ServerName),
		Target: b.provide,
	}
}

type basculeCapabilityMetricProvider struct {
	ServerName string
}

func (b basculeCapabilityMetricProvider) provide(in basculeCapabilityMetricsProviderIn) (*basculechecks.AuthCapabilityCheckMeasures, error) {
	capabilityCheckOutcomeCounterVec, err := in.CapabilityCheckOutcome.CurryWith(prometheus.Labels{"server": b.ServerName})
	if err != nil {
		return nil, err
	}
	return &basculechecks.AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: gokitprometheus.NewCounter(capabilityCheckOutcomeCounterVec),
	}, nil
}

func (b basculeCapabilityMetricProvider) annotated() fx.Annotated {
	return fx.Annotated{
		Name:   b.ServerName,
		Target: b.provide,
	}
}

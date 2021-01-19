package auth

import (
	"fmt"

	"github.com/go-kit/kit/log/level"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"github.com/xmidt-org/webpa-common/basculemetrics"
	"go.uber.org/fx"
)

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

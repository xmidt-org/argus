package auth

import (
	"fmt"
	"regexp"

	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type capabilityValidatorConfig struct {
	Type            string
	Prefix          string
	AcceptAllMethod string
	EndpointBuckets []string
}

type primaryProfileIn struct {
	LoggerIn
	Profile *profile `name:"primary_profile"`
}

type primaryCapabilityValidatorIn struct {
	LoggerIn
	Profile  *profile                                   `name:"primary_profile"`
	Measures *basculechecks.AuthCapabilityCheckMeasures `name:"primary_bascule_capability_measures"`
}

func newPrimaryCapabilityValidator(in primaryCapabilityValidatorIn) (bascule.Validator, error) {
	if in.Profile == nil {
		in.Logger.Warn("undefined profile. CapabilityCheck disabled")
		return nil, nil
	}

	config := in.Profile.CapabilityCheck
	if config == nil {
		in.Logger.Warn("config not provided. CapabilityCheck disabled")
		return nil, nil
	}

	if config.Type != "enforce" && config.Type != "monitor" {
		in.Logger.Warn("unsupported capability check type. CapabilityCheck disabled", zap.String("type", config.Type))
		return nil, nil
	}

	c, err := basculechecks.NewEndpointRegexCheck(config.Prefix, config.AcceptAllMethod)
	if err != nil {
		return nil, fmt.Errorf("error initializing endpointRegexCheck: %w", err)
	}

	var endpoints []*regexp.Regexp
	for _, e := range config.EndpointBuckets {
		r, err := regexp.Compile(e)
		if err != nil {
			in.Logger.Warn("failed to compile regular expression", zap.String("regex", e))
			continue
		}
		endpoints = append(endpoints, r)
	}

	m := basculechecks.MetricValidator{
		C:         basculechecks.CapabilitiesValidator{Checker: c},
		Measures:  in.Measures,
		Endpoints: endpoints,
	}
	in.Logger.Info("building auth capability", zap.String("type", config.Type))
	return m.CreateValidator(config.Type == "enforce"), nil
}

func primaryCapabilityValidatorAnnotated() fx.Annotated {
	return fx.Annotated{
		Name:   "primary_bearer_validator_capability",
		Target: newPrimaryCapabilityValidator,
	}
}

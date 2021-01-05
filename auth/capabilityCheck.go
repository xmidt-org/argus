package auth

import (
	"fmt"
	"regexp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"go.uber.org/fx"
)

type capabilityValidatorConfig struct {
	Type            string
	Prefix          string
	AcceptAllMethod string
	EndpointBuckets []string
}

type PrimaryCapabilityValidatorIn struct {
	ProfileIn primaryProfileIn
	Measures  *basculechecks.AuthCapabilityCheckMeasures `name:"primary_capability_measures"`
	Logger    log.Logger
}

func ProvidePrimaryCapabilityValidator(in PrimaryCapabilityValidatorIn) (bascule.Validator, error) {
	profile := in.ProfileIn.Profile
	if profile == nil {
		in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "Undefined profile. CapabilityCheck disabled.")
		return nil, nil
	}

	config := profile.CapabilityCheck
	if config.Type != "enforce" && config.Type != "monitor" {
		in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "Unsupported capability check type. CapabilityCheck disabled.", "type", config.Type)
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
			in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "failed to compile regular expression", "regex", e, xlog.ErrorKey(), err.Error())
			continue
		}
		endpoints = append(endpoints, r)
	}

	m := basculechecks.MetricValidator{
		C:         basculechecks.CapabilitiesValidator{Checker: c},
		Measures:  in.Measures,
		Endpoints: endpoints,
	}

	return m.CreateValidator(config.Type == "enforce"), nil
}

func PrimaryCapabilityValidatorAnnotated() fx.Annotated {
	return fx.Annotated{
		Name:   "primary_bearer_validator_capability",
		Target: ProvidePrimaryCapabilityValidator,
	}
}

package auth

import (
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"go.uber.org/fx"
	"regexp"
)

type capabilityValidatorConfig struct {
	Type            string
	Prefix          string
	AcceptAllMethod string
	EndpointBuckets []string
}

type primaryCapabilityValidatorIn struct {
	Profile  *profile                                  `name:"primary_profile"`
	Measures basculechecks.AuthCapabilityCheckMeasures `name:"primary_capability_measures"`
	Logger   log.Logger
}

func providePrimaryCapabilityValidator(in primaryCapabilityValidatorIn) (bascule.Validator, error) {
	config := in.Profile.CapabilityCheck
	if config.Type != "enforce" && config.Type != "monitor" {
		return nil, errors.New("error providing primary capability validator as the type is not recognized")
	}

	c, err := basculechecks.NewEndpointRegexCheck(config.Prefix, config.AcceptAllMethod)
	if err != nil {
		return nil, fmt.Errorf("error initializing endpointRegexCheck: %w", err)
	}

	var endpoints []*regexp.Regexp
	for _, e := range config.EndpointBuckets {
		r, err := regexp.Compile(e)
		if err != nil {
			in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "failed to compile regular expression", "regex", e, xlog.ErrorKey(), err.Error())
			continue
		}
		endpoints = append(endpoints, r)
	}

	m := basculechecks.MetricValidator{
		C:         basculechecks.CapabilitiesValidator{Checker: c},
		Measures:  &in.Measures,
		Endpoints: endpoints,
	}

	return m.CreateValidator(config.Type == "enforce"), nil
}

func primaryCapabilityValidatorAnnotated() fx.Annotated {
	return fx.Annotated{
		Name:   "primary_bearer_validator_capabilities",
		Target: providePrimaryCapabilityValidator,
	}
}

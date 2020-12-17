package auth

import (
	"context"
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

type PrimaryCapabilityValidatorIn struct {
	fx.In
	Profile  *Profile                                   `name:"primary_profile"`
	Measures *basculechecks.AuthCapabilityCheckMeasures `name:"primary_capability_measures"`
	Logger   log.Logger
}

func dummy(ctx context.Context, token bascule.Token) error {
	return nil
}
func ProvidePrimaryCapabilityValidator(in PrimaryCapabilityValidatorIn) (bascule.Validator, error) {
	fmt.Printf("Creating capability validator. Profile: %v\n Measures: %v\n", in.Profile, in.Measures)

	config := in.Profile.CapabilityCheck
	if config.Type != "enforce" && config.Type != "monitor" {
		return bascule.ValidatorFunc(dummy), nil
	}

	c, err := basculechecks.NewEndpointRegexCheck(config.Prefix, config.AcceptAllMethod)
	if err != nil {
		return bascule.ValidatorFunc(dummy), fmt.Errorf("error initializing endpointRegexCheck: %w", err)
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

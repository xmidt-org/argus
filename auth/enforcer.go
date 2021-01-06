package auth

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/themis/xlog"
	"go.uber.org/fx"
)

// PrimaryBearerValidatorsIn provides the bascule checks to run against the jwt token.
type PrimaryBearerValidatorsIn struct {
	fx.In
	Logger     log.Logger
	Principal  bascule.Validator `name:"primary_bearer_validator_principal" optional:"true"`
	Type       bascule.Validator `name:"primary_bearer_validator_type" optional:"true"`
	Capability bascule.Validator `name:"primary_bearer_validator_capability" optional:"true"`
}

// PrimaryEOptionsIn is the uber.fx wired struct needed to group together the options
// for the bascule enforcer middleware, which runs checks against the jwt token.
type PrimaryEOptionsIn struct {
	fx.In
	Options []basculehttp.EOption `group:"primary_bascule_enforcer_options"`
	Logger  log.Logger
}

func ProvidePrimaryBasculeEnforcer() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name: "primary_bearer_validator_principal",
			Target: func() bascule.Validator {
				return bascule.CreateNonEmptyPrincipalCheck()
			},
		},

		fx.Annotated{
			Name: "primary_bearer_validator_type",
			Target: func() bascule.Validator {
				return bascule.CreateValidTypeCheck([]string{"jwt"})
			},
		},
		PrimaryCapabilityValidatorAnnotated(),
		BasculeCapabilityMetricProvider{ServerName: "primary"}.Annotated(),
		fx.Annotated{
			Group: "primary_bascule_enforcer_options",
			Target: func(in PrimaryBearerValidatorsIn) basculehttp.EOption {
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building bearer rules option", "server", "primary")
				validators := filterNilValidators([]bascule.Validator{in.Principal, in.Type, in.Capability})
				if len(validators) < 1 {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "providing nil bearer rules option", "server", "primary")
					return nil
				}
				return basculehttp.WithRules("Bearer", bascule.Validators(validators))
			},
		},
		fx.Annotated{
			Group: "primary_bascule_enforcer_options",
			Target: func() basculehttp.EOption {
				return basculehttp.WithRules("Basic", bascule.CreateAllowAllCheck())
			},
		},
		fx.Annotated{
			Group: "primary_bascule_enforcer_options",
			Target: func(in PrimaryBasculeMetricListenerIn) basculehttp.EOption {
				return basculehttp.WithEErrorResponseFunc(in.Listener.OnErrorResponse)
			},
		},

		fx.Annotated{
			Name: "primary_alice_enforcer",
			Target: func(in PrimaryEOptionsIn) alice.Constructor {
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building alice enforcer", "server", "primary")
				if anyNil(in.Options) {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "nil alice enforcer provided for primary server")
					return nil
				}
				return basculehttp.NewEnforcer(in.Options...)
			},
		},
	)
}

func filterNilValidators(vals []bascule.Validator) []bascule.Validator {
	var filteredVals []bascule.Validator
	for _, v := range vals {
		if v != nil {
			filteredVals = append(filteredVals, v)
		}
	}
	return filteredVals
}

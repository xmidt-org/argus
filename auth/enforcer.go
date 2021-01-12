package auth

import (
	"github.com/go-kit/kit/log/level"
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/themis/xlog"
	"go.uber.org/fx"
)

// primaryBearerValidatorsIn provides the bascule checks to run against the jwt token.
type primaryBearerValidatorsIn struct {
	LoggerIn
	Principal  bascule.Validator `name:"primary_bearer_validator_principal"`
	Type       bascule.Validator `name:"primary_bearer_validator_type"`
	Capability bascule.Validator `name:"primary_bearer_validator_capability"`
}

// primaryEOptionsIn is the uber.fx wired struct needed to group together the options
// for the bascule enforcer middleware, which runs checks against the jwt token.
type primaryEOptionsIn struct {
	LoggerIn
	Options []basculehttp.EOption `group:"primary_bascule_enforcer_options"`
}

func providePrimaryBasculeEnforcerOptions() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Group: "primary_bascule_enforcer_options",
			Target: func(in primaryBearerValidatorsIn) basculehttp.EOption {
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building bearer rules option", "server", "primary")
				var validators = []bascule.Validator{in.Principal, in.Type}
				if in.Capability != nil {
					validators = append(validators, in.Capability)
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
			Target: func(in primaryBasculeMetricListenerIn) basculehttp.EOption {
				return basculehttp.WithEErrorResponseFunc(in.Listener.OnErrorResponse)
			},
		},
	)
}

func providePrimaryBasculeEnforcer() fx.Option {
	return fx.Options(
		fx.Provide(
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
			primaryCapabilityValidatorAnnotated(),
			basculeCapabilityMetricFactory{serverName: "primary"}.annotated(),
			fx.Annotated{
				Name: "primary_alice_enforcer",
				Target: func(in primaryEOptionsIn) alice.Constructor {
					in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building alice enforcer", "server", "primary")
					return basculehttp.NewEnforcer(in.Options...)
				},
			},
		),
		providePrimaryBasculeEnforcerOptions(),
	)
}

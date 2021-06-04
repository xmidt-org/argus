package auth

import (
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule"
	bchecks "github.com/xmidt-org/bascule/basculechecks"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/webpa-common/basculechecks"
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
				in.Logger.Debug("building bearer rules option")
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
				return basculehttp.WithRules("Basic", bchecks.AllowAll())
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
					return bchecks.NonEmptyPrincipal()
				},
			},
			fx.Annotated{
				Name: "primary_bearer_validator_type",
				Target: func() bascule.Validator {
					return bchecks.ValidType([]string{"jwt"})
				},
			},
			primaryCapabilityValidatorAnnotated(),
			basculechecks.MeasuresFactory{ServerName: "primary"}.Annotated(),
			fx.Annotated{
				Name: "primary_alice_enforcer",
				Target: func(in primaryEOptionsIn) alice.Constructor {
					in.Logger.Debug("building alice enforcer")
					return basculehttp.NewEnforcer(in.Options...)
				},
			},
		),
		providePrimaryBasculeEnforcerOptions(),
	)
}

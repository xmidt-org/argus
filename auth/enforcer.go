package auth

import (
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"go.uber.org/fx"
)

// PrimaryBearerValidatorsIn provides the bascule checks to run against the jwt token.
type PrimaryBearerValidatorsIn struct {
	fx.In
	Principal  bascule.Validator `name:"primary_bearer_validator_principal"`
	Type       bascule.Validator `name:"primary_bearer_validator_type"`
	Capability bascule.Validator `name:"primary_bearer_validator_capability"`
}

// PrimaryEOptionsIn is the uber.fx wired struct needed to group together the options
// for the bascule enforcer middleware, which runs checks against the jwt token.
type PrimaryEOptionsIn struct {
	fx.In
	Options []basculehttp.EOption `group:"primary_bascule_enforcer_options"`
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
				rules := bascule.Validators{in.Principal, in.Type, in.Capability}
				return basculehttp.WithRules("Bearer", rules)
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
				return basculehttp.NewEnforcer(in.Options...)
			},
		},
	)
}

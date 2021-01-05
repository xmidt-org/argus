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
	Principal  bascule.Validator `name:"primary_bearer_validator_principal" optional:"true"`
	Type       bascule.Validator `name:"primary_bearer_validator_type" optional:"true"`
	Capability bascule.Validator `name:"primary_bearer_validator_capability" optional:"true"`
}

// PrimaryEOptionsIn is the uber.fx wired struct needed to group together the options
// for the bascule enforcer middleware, which runs checks against the jwt token.
type PrimaryEOptionsIn struct {
	fx.In
	Options []basculehttp.EOption `group:"primary_bascule_enforcer_options"`
	Logger log.Logger
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
				if anyNil(in.Principal, in.Type, in.Capability) {
					return nil
				}
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
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building primary alice enforcer", "in", in)
				//fmt.Printf("Primary Alice Enforcer options: %v", len(in.Options))
				if anyNil(in.Options) {
					return nil
				}
				return basculehttp.NewEnforcer(in.Options...)
			},
		},
	)
}

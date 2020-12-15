package auth

import (
	"github.com/justinas/alice"
	"go.uber.org/fx"
)

type PrimaryChainIn struct {
	fx.In
	SetLogger   alice.Constructor `name:"primary_bascule_set_logger"`
	Constructor alice.Constructor `name:"primary_bascule_constructor"`
	Enforcer    alice.Constructor `name:"primary_bascule_enforcer"`
	Listener    alice.Constructor `name:"primary_bascule_listener"`
}

// ProvideServerAuthChains builds the server auth chains.
func ProvidePrimaryServerAuthChain(profiles map[string]*profile) fx.Option {
	primaryProfile := profiles["primary"]
	if primaryProfile == nil {
		return fx.Options(
			fx.Provide(
				fx.Annotated{
					Name: "primary_auth_chain",
					Target: func() alice.Chain {
						return alice.New()
					},
				},
			),
		)
	}

	return fx.Options(
		fx.Provide(
			logOptionsProvider{ServerName: "primary"}.provide,
			providePrimaryBasculeConstructor,
			profileProvider{ServerName: "primary"}.annotated(),
			//TODO: providePrimaryBasculeEnforcer
			providePrimaryTokenFactory,
			basculeMetricsListenerProvider{ServerName: "primary"}.annotated(),
			unmarshalProfiles("bascule.inbound.profiles"),
			fx.Annotated{
				Name: "primary_auth_chain",
				Target: func(in PrimaryChainIn) alice.Chain {
					return alice.New(in.SetLogger, in.Constructor, in.Enforcer, in.Listener)
				},
			},
		),
	)
}

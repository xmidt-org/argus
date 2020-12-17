package auth

import (
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule/basculehttp"
	"go.uber.org/fx"
)

type PrimaryChainIn struct {
	fx.In
	SetLogger   alice.Constructor `name:"primary_alice_set_logger"`
	Constructor alice.Constructor `name:"primary_alice_constructor"`
	Enforcer    alice.Constructor `name:"primary_alice_enforcer"`
	Listener    alice.Constructor `name:"primary_alice_listener"`
}

// ProvidePrimaryServerAuthChain builds the server auth chains.
func ProvidePrimaryChain() fx.Option {
	//primaryProfile := profiles["primary"]
	//fmt.Printf("This is the primary profile: %v", primaryProfile)
	//if primaryProfile == nil {
	//	return fx.Options(
	//		fx.Provide(
	//			fx.Annotated{
	//				Name: "primary_auth_chain",
	//				Target: func() alice.Chain {
	//					return alice.New()
	//				},
	//			},
	//		),
	//	)
	//}
	//
	return fx.Options(
		LogOptionsProvider{ServerName: "primary"}.Provide(),
		ProvidePrimaryBasculeConstructor(),
		ProvidePrimaryBasculeEnforcer(),
		ProvidePrimaryTokenFactory(),
		fx.Provide(
			UnmarshalProfiles("bascule.inbound.profiles"),
			ProfileProvider{ServerName: "primary"}.Annotated(),
			BasculeMetricsListenerProvider{ServerName: "primary"}.Annotated(),
			BasculeCapabilityMetricProvider{ServerName: "primary"}.Annotated(),

			fx.Annotated{
				Name: "primary_alice_listener",
				Target: func(in PrimaryBasculeMetricListenerIn) alice.Constructor {
					return basculehttp.NewListenerDecorator(in.Listener)
				},
			},
			fx.Annotated{
				Name: "primary_auth_chain",
				Target: func(in PrimaryChainIn) alice.Chain {
					return alice.New(in.SetLogger, in.Constructor, in.Enforcer, in.Listener)
				},
			},
		))
}

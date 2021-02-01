package auth

import (
	"reflect"

	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/webpa-common/basculemetrics"
	"go.uber.org/fx"
)

type primaryChainIn struct {
	fx.In
	SetLogger   alice.Constructor `name:"primary_alice_set_logger"`
	Constructor alice.Constructor `name:"primary_alice_constructor"`
	Enforcer    alice.Constructor `name:"primary_alice_enforcer"`
	Listener    alice.Constructor `name:"primary_alice_listener"`
}

// ProvidePrimaryServerChain provides the auth alice.Chain for the primary server.
func ProvidePrimaryServerChain(apiBase string) fx.Option {
	return fx.Options(
		logOptionsProvider{serverName: "primary"}.provide(),
		providePrimaryBasculeConstructor(apiBase),
		providePrimaryBasculeEnforcer(),
		fx.Provide(
			profileFactory{serverName: "primary"}.annotated(),
			basculemetrics.MeasuresFactory{ServerName: "primary"}.Annotated(),
			fx.Annotated{
				Name: "primary_alice_listener",
				Target: func(in primaryBasculeMetricListenerIn) alice.Constructor {
					return basculehttp.NewListenerDecorator(in.Listener)
				},
			},
			fx.Annotated{
				Name: "primary_auth_chain",
				Target: func(in primaryChainIn) alice.Chain {
					return alice.New(in.SetLogger, in.Constructor, in.Enforcer, in.Listener)
				},
			},
		))
}

// anyNil returns true if any of the provided objects are nil, false otherwise.
func anyNil(objects ...interface{}) bool {
	for _, object := range objects {
		if object == nil || reflect.ValueOf(object).IsNil() {
			return true
		}
	}
	return false
}

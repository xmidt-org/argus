package auth

import (
	"reflect"

	"github.com/go-kit/kit/log"

	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule/basculehttp"
	"go.uber.org/fx"
)

type primaryChainIn struct {
	fx.In
	SetLogger   alice.Constructor `name:"primary_alice_set_logger" optional:"true"`
	Constructor alice.Constructor `name:"primary_alice_constructor" optional:"true"`
	Enforcer    alice.Constructor `name:"primary_alice_enforcer" optional:"true"`
	Listener    alice.Constructor `name:"primary_alice_listener" optional:"true"`
}

type primaryProfileIn struct {
	fx.In
	Logger  log.Logger
	Profile *profile `name:"primary_profile" optional:"true"`
}

// ProvidePrimaryServerChain provides the auth alice.Chain for the primary server.
func ProvidePrimaryServerChain(apiBase string) fx.Option {
	return fx.Options(
		LogOptionsProvider{ServerName: "primary"}.Provide(),
		ProvidePrimaryBasculeConstructor(apiBase),
		ProvidePrimaryBasculeEnforcer(),
		ProvidePrimaryTokenFactory(),
		fx.Provide(
			profileProvider{ServerName: "primary"}.Annotated(),
			BasculeMetricsListenerProvider{ServerName: "primary"}.Annotated(),

			fx.Annotated{
				Name: "primary_alice_listener",
				Target: func(in PrimaryBasculeMetricListenerIn) alice.Constructor {
					return basculehttp.NewListenerDecorator(in.Listener)
				},
			},
			fx.Annotated{
				Name: "primary_auth_chain",
				Target: func(in primaryChainIn) alice.Chain {
					if anyNil(in.SetLogger, in.Constructor, in.Enforcer, in.Listener) {
						return alice.Chain{}
					}
					return alice.New(in.SetLogger, in.Constructor, in.Enforcer, in.Listener)
				},
			},
		))
}

// anyNil returns true if any of the provided objects are nil or if any of the objects
// are lists that contain any nil elements.
func anyNil(objects ...interface{}) bool {
	for _, object := range objects {
		if object == nil {
			return true
		}
		val := reflect.ValueOf(object)
		if val.IsNil() {
			return true
		}

		switch val.Type().Kind() {
		case reflect.Slice:
			slice := reflect.ValueOf(object)
			for i := 0; i < slice.Len(); i++ {
				if slice.Index(i).IsNil() {
					return true
				}
			}
		}
	}
	return false
}

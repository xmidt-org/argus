package auth

import (
	"reflect"

	"github.com/xmidt-org/themis/xlog"

	"github.com/go-kit/kit/log/level"
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/key"
	"github.com/xmidt-org/webpa-common/basculemetrics"
	"go.uber.org/fx"
)

const primaryBasculeCOptionsName = "primary_bascule_constructor_options"

type primaryCOptionsIn struct {
	LoggerIn
	Options []basculehttp.COption `group:"primary_bascule_constructor_options"`
}

type primaryBearerTokenFactoryIn struct {
	LoggerIn
	DefaultKeyID string         `name:"primary_bearer_default_kid"`
	Resolver     key.Resolver   `name:"primary_bearer_key_resolver"`
	Leeway       bascule.Leeway `name:"primary_bearer_leeway"`
	AccessLevel  AccessLevel    `name:"primary_bearer_access_level"`
}

type primaryBasculeMetricListenerIn struct {
	fx.In
	Listener *basculemetrics.MetricListener `name:"primary_bascule_metric_listener"`
}

type primaryBasculeOnHTTPErrorResponseIn struct {
	fx.In
	OnErrorHTTPResponse basculehttp.OnErrorHTTPResponse `name:"primary_bascule_on_error_http_response" optional:"true"`
}

type primaryBasculeParseURLFuncIn struct {
	fx.In
	ParseURL basculehttp.ParseURL `name:"primary_bascule_parse_url"`
}

func providePrimaryBasculeConstructorOptions(apiBase string) fx.Option {
	return fx.Provide(
		fx.Annotated{
			Group: primaryBasculeCOptionsName,
			Target: func(in primaryBasculeMetricListenerIn) basculehttp.COption {
				return basculehttp.WithCErrorResponseFunc(in.Listener.OnErrorResponse)
			},
		},
		fx.Annotated{
			Group: primaryBasculeCOptionsName,
			Target: func(in primaryBearerTokenFactoryIn) basculehttp.COption {
				if in.Resolver == nil {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "returning nil bearer token factory option as resolver was not defined", "server", "primary")
					return nil
				}
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building bearer token factory option", "server", "primary")
				return basculehttp.WithTokenFactory("Bearer", accessLevelBearerTokenFactory{
					DefaultKeyID: in.DefaultKeyID,
					Resolver:     in.Resolver,
					Parser:       bascule.DefaultJWTParser,
					Leeway:       in.Leeway,
					AccessLevel:  in.AccessLevel,
				})
			},
		},
		fx.Annotated{
			Group: primaryBasculeCOptionsName,
			Target: func(in primaryProfileIn) (basculehttp.COption, error) {
				if in.Profile == nil || len(in.Profile.Basic) < 1 {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "returning nil basic token factory option as config was not provided", "server", "primary")
					return nil, nil
				}
				basicTokenFactory, err := basculehttp.NewBasicTokenFactoryFromList(in.Profile.Basic)
				if err != nil {
					return nil, err
				}
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building basic token factory option", "server", "primary")
				return basculehttp.WithTokenFactory("Basic", basicTokenFactory), nil
			},
		},
		fx.Annotated{
			Group: primaryBasculeCOptionsName,
			Target: func(in primaryBasculeParseURLFuncIn) basculehttp.COption {
				return basculehttp.WithParseURLFunc(in.ParseURL)
			},
		},
		fx.Annotated{
			Group: primaryBasculeCOptionsName,
			Target: func(in primaryBasculeOnHTTPErrorResponseIn) basculehttp.COption {
				if in.OnErrorHTTPResponse == nil {
					return nil
				}
				return basculehttp.WithCErrorHTTPResponseFunc(in.OnErrorHTTPResponse)
			},
		},
	)
}

func providePrimaryBasculeConstructor(apiBase string) fx.Option {
	return fx.Options(
		providePrimaryBasculeConstructorOptions(apiBase),
		providePrimaryTokenFactoryInput(),
		fx.Provide(
			fx.Annotated{
				Name: "primary_alice_constructor",
				Target: func(in primaryCOptionsIn) alice.Constructor {
					in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building alice constructor from bascule constructor options", "server", "primary")
					var filteredOptions []basculehttp.COption
					for _, option := range in.Options {
						if option == nil || reflect.ValueOf(option).IsNil() {
							continue
						}
						filteredOptions = append(filteredOptions, option)
					}
					return basculehttp.NewConstructor(filteredOptions...)
				},
			}),
	)
}

func providePrimaryTokenFactoryInput() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name: "primary_bearer_default_kid",
			Target: func() string {
				return "current"
			},
		},
		fx.Annotated{
			Name: "primary_bearer_key_resolver",
			Target: func(in primaryProfileIn) (key.Resolver, error) {
				if anyNil(in.Profile, in.Profile.Bearer) {
					in.Logger.Log(level.Key(), level.WarnValue(), xlog.MessageKey(), "returning nil bearer key resolver as config wasn't provided", "server", "primary")
					return nil, nil
				}
				in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building bearer key resolver option", "server", "primary")
				return in.Profile.Bearer.Keys.NewResolver()
			},
		},
		fx.Annotated{
			Name: "primary_bearer_leeway",
			Target: func(in primaryProfileIn) bascule.Leeway {
				if anyNil(in.Profile, in.Profile.Bearer) {
					return bascule.Leeway{}
				}
				return in.Profile.Bearer.Leeway
			},
		},
		fx.Annotated{
			Name: "primary_bearer_access_level",
			Target: func(in primaryProfileIn) AccessLevel {
				if anyNil(in.Profile, in.Profile.AccessLevel) {
					return defaultAccessLevel()
				}
				return newContainsAttributeAccessLevel(in.Profile.AccessLevel)
			},
		},
	)
}

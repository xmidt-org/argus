/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"emperror.dev/emperror"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/justinas/alice"
	"github.com/spf13/viper"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/key"
	"github.com/xmidt-org/themis/xlog"
	"github.com/xmidt-org/themis/xmetrics"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"github.com/xmidt-org/webpa-common/basculemetrics"
	"go.uber.org/fx"
)

func SetLogger(logger log.Logger) func(delegate http.Handler) http.Handler {
	return func(delegate http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				logHeader := r.Header.Clone()
				if str := logHeader.Get("Authorization"); str != "" {
					logHeader.Del("Authorization")
					logHeader.Set("Authorization-Type", strings.Split(str, " ")[0])
				}
				ctx := r.WithContext(xlog.With(r.Context(),
					log.With(logger, "requestHeaders", logHeader, "requestURL", r.URL.EscapedPath(), "method", r.Method)))
				delegate.ServeHTTP(w, ctx)
			})
	}
}

func GetLogger(ctx context.Context) bascule.Logger {
	logger := log.With(xlog.GetDefault(ctx, nil), xlog.TimestampKey(), log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	return logger
}

// JWTValidator provides a convenient way to define jwt validator through config files
type JWTValidator struct {
	// JWTKeys is used to create the key.Resolver for JWT verification keys
	Keys key.ResolverFactory `json:"keys"`

	// Leeway is used to set the amount of time buffer should be given to JWT
	// time values, such as nbf
	Leeway bascule.Leeway
}

type authAcquirerConfig struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type CapabilityConfig struct {
	Type            string
	Prefix          string
	AcceptAllMethod string
	EndpointBuckets []string
}

type AuthChainOut struct {
	fx.Out
	Primary alice.Chain `name:"servers.primary.auth"`
	Metrics alice.Chain `name:"servers.metrics.auth"`
	Health  alice.Chain `name:"servers.health.auth"`
}

type AuthChainIn struct {
	fx.In

	Viper              *viper.Viper
	Logger             log.Logger
	Registry           xmetrics.Registry
	ValidationMeasures basculemetrics.AuthValidationMeasures
	CheckMeasures      basculechecks.AuthCapabilityCheckMeasures
}

type authSetupHelper struct {
	in AuthChainIn
}

func (a *authSetupHelper) createBasicAuthFactory(encodedBasicAuthKeys []string, profile string) (basculehttp.BasicTokenFactory, error) {
	a.in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "Attempting to create basic auth factory", "encodedBasicAuthKeys", encodedBasicAuthKeys)
	basicAuthFactory, err := basculehttp.NewBasicTokenFactoryFromList(encodedBasicAuthKeys)
	if err != nil {
		return nil, err
	}
	return basicAuthFactory, nil
}

func (a *authSetupHelper) createBearerAuthFactory(p Profile) (basculehttp.TokenFactory, error) {
	resolver, err := p.Bearer.Keys.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	return &AccessLevelBearerTokenFactory{
		DefaultKeyID:      DefaultKeyID,
		Resolver:          resolver,
		Parser:            bascule.DefaultJWTParser,
		Leeway:            p.Bearer.Leeway,
		AccessLevelConfig: p.AccessLevel,
	}, nil
}

func (a *authSetupHelper) createCapabilityCheckValidator(profileName string) (bascule.Validator, error) {
	var capabilityCheck CapabilityConfig
	c, err := basculechecks.NewEndpointRegexCheck(capabilityCheck.Prefix, capabilityCheck.AcceptAllMethod)
	if err != nil {
		return nil, emperror.With(err, "failed to create capability check")
	}
	var endpoints []*regexp.Regexp
	for _, e := range capabilityCheck.EndpointBuckets {
		r, err := regexp.Compile(e)
		if err != nil {
			a.in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "failed to compile regular expression", "regex", e, "profile", profileName, xlog.ErrorKey(), err.Error())
			continue
		}
		endpoints = append(endpoints, r)
	}
	m := basculechecks.MetricValidator{
		C:         basculechecks.CapabilitiesValidator{Checker: c},
		Measures:  &a.in.CheckMeasures,
		Endpoints: endpoints,
	}
	return m.CreateValidator(capabilityCheck.Type == "enforce"), nil
}

type Profile struct {
	Name        string
	Basic       []string
	Bearer      JWTValidator
	AccessLevel AccessLevelConfig
}

type superUserProfile struct {
	Profile
	SuperUser superUserAccessLevelConfig
}

// authenticationHandler configures the authorization requirements for requests to reach the main handler
func ProvideAuthChain(in AuthChainIn) (AuthChainOut, error) {
	// TODO: alternatively, 'profiles' could be called 'target_servers' in the config file indicating a certain auth chain
	// should be applied to handlers under a given server
	supportedProfiles := map[string]bool{"primary": true, "metrics": true, "health": true}
	out := AuthChainOut{}

	var profiles []superUserProfile
	err := in.Viper.UnmarshalKey("bascule.inbound.profiles", &profiles)
	if err != nil {
		return out, err
	}

	authHelper := authSetupHelper{in: in}

	for _, profile := range profiles {
		supported := supportedProfiles[profile.Name]
		if !supported {
			return out, fmt.Errorf("profile '%s' is not supported", profile.Name)
		}

		//TODO: We'll need to update bascule so it's aware of the notion of profiles (maybe as an additional label?)
		listener := basculemetrics.NewMetricListener(&in.ValidationMeasures)

		options := []basculehttp.COption{
			basculehttp.WithCLogger(GetLogger),
			basculehttp.WithCErrorResponseFunc(listener.OnErrorResponse),
			basculehttp.WithParseURLFunc(basculehttp.CreateRemovePrefixURLFunc("/"+apiBase+"/", basculehttp.DefaultParseURLFunc)),
		}

		if len(profile.Basic) > 0 {
			basicAuthFactory, err := authHelper.createBasicAuthFactory(profile.Basic, profile.Name)
			if err != nil {
				in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "Failed to create basic auth factory", "profile", profile.Name, "err", err)
				return out, err
			}

			options = append(options, basculehttp.WithTokenFactory("basic", basicAuthFactory))
			in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "Enabling basic auth", "profile", profile.Name)
		}

		if profile.Bearer.Keys.URI != "" {
			if profile.SuperUser.SuperUserCapability != "" {
				profile.AccessLevel.Resolver = superUserAccessLevelResolver(profile.SuperUser)
			}

			bearerAuthFactory, err := authHelper.createBearerAuthFactory(profile.Profile)
			if err != nil {
				in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "Failed to create bearer auth factory", "profile", profile.Name, "err", err)
				return out, err
			}

			options = append(options, basculehttp.WithTokenFactory("bearer", bearerAuthFactory))
			in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "Enabling bearer auth", "profile", profile.Name, "superUserEnabled", profile.SuperUser.SuperUserCapability != "")
		}

		authConstructor := basculehttp.NewConstructor(options...)

		tokenValidators := bascule.Validators{
			bascule.CreateNonEmptyPrincipalCheck(),
			bascule.CreateNonEmptyTypeCheck(),
			bascule.CreateValidTypeCheck([]string{"jwt"}),
		}

		capabilityCheckValidator, err := authHelper.createCapabilityCheckValidator(profile.Name)
		if err != nil {
			in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "Failed to create capability check validator", "profile", profile.Name, "err", err)
			return out, err
		}

		tokenValidators = append(tokenValidators, capabilityCheckValidator)

		authEnforcer := basculehttp.NewEnforcer(
			basculehttp.WithELogger(GetLogger),
			basculehttp.WithRules("Basic", bascule.Validators{
				bascule.CreateAllowAllCheck(),
			}),
			basculehttp.WithRules("Bearer", tokenValidators),
			basculehttp.WithEErrorResponseFunc(listener.OnErrorResponse),
		)

		constructors := []alice.Constructor{SetLogger(in.Logger), authConstructor, authEnforcer, basculehttp.NewListenerDecorator(listener)}
		chain := alice.New(constructors...)
		switch profile.Name {
		case "primary":
			out.Primary = chain
		case "metrics":
			out.Metrics = chain
		case "health":
			out.Health = chain
		}
	}
	return out, nil
}

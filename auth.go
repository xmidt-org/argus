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
	"bytes"
	"context"
	"encoding/base64"
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
	Chain *alice.Chain `name:"auth_chain"`
}

type AuthChainIn struct {
	fx.In

	Viper              *viper.Viper
	Logger             log.Logger
	Registry           xmetrics.Registry
	ValidationMeasures basculemetrics.AuthValidationMeasures
	CheckMeasures      basculechecks.AuthCapabilityCheckMeasures
}

// authenticationHandler configures the authorization requirements for requests to reach the main handler
func ProvideAuthChain(in AuthChainIn) (AuthChainOut, error) {

	listener := basculemetrics.NewMetricListener(&in.ValidationMeasures)

	basicAllowed := make(map[string]string)
	basicAuth := in.Viper.GetStringSlice("authHeader")
	for _, a := range basicAuth {
		decoded, err := base64.StdEncoding.DecodeString(a)
		if err != nil {
			in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "failed to decode auth header", "authHeader", a, xlog.ErrorKey(), err.Error())
		}

		i := bytes.IndexByte(decoded, ':')
		in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "decoded string", "string", decoded, "i", i)
		if i > 0 {
			basicAllowed[string(decoded[:i])] = string(decoded[i+1:])
		}
	}
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "Created list of allowed basic auths", "allowed", basicAllowed, "config", basicAuth)

	options := []basculehttp.COption{
		basculehttp.WithCLogger(GetLogger),
		basculehttp.WithCErrorResponseFunc(listener.OnErrorResponse),
		basculehttp.WithParseURLFunc(basculehttp.CreateRemovePrefixURLFunc("/"+apiBase+"/", basculehttp.DefaultParseURLFunc)),
	}
	if len(basicAllowed) > 0 {
		options = append(options, basculehttp.WithTokenFactory("Basic", basculehttp.BasicTokenFactory(basicAllowed)))
	}
	var jwtVal JWTValidator

	in.Viper.UnmarshalKey("jwtValidator", &jwtVal)
	if jwtVal.Keys.URI != "" {
		resolver, err := jwtVal.Keys.NewResolver()
		if err != nil {
			return AuthChainOut{Chain: &alice.Chain{}}, emperror.With(err, "failed to create resolver")
		}

		options = append(options, basculehttp.WithTokenFactory("Bearer", basculehttp.BearerTokenFactory{
			DefaultKeyId: DefaultKeyID,
			Resolver:     resolver,
			Parser:       bascule.DefaultJWTParser,
			Leeway:       jwtVal.Leeway,
		}))
	}

	authConstructor := basculehttp.NewConstructor(options...)

	bearerRules := bascule.Validators{
		bascule.CreateNonEmptyPrincipalCheck(),
		bascule.CreateNonEmptyTypeCheck(),
		bascule.CreateValidTypeCheck([]string{"jwt"}),
	}

	// only add capability check if the configuration is set
	var capabilityCheck CapabilityConfig
	in.Viper.UnmarshalKey("capabilityCheck", &capabilityCheck)
	if capabilityCheck.Type == "enforce" || capabilityCheck.Type == "monitor" {
		var endpoints []*regexp.Regexp
		for _, e := range capabilityCheck.EndpointBuckets {
			r, err := regexp.Compile(e)
			if err != nil {
				in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "failed to compile regular expression", "regex", e, xlog.ErrorKey(), err.Error())
				continue
			}
			endpoints = append(endpoints, r)
		}
		checker, err := basculechecks.NewCapabilityChecker(&in.CheckMeasures, capabilityCheck.Prefix, capabilityCheck.AcceptAllMethod, endpoints)
		if err != nil {
			return AuthChainOut{Chain: nil}, emperror.With(err, "failed to create capability check")
		}
		bearerRules = append(bearerRules, checker.CreateBasculeCheck(capabilityCheck.Type == "enforce"))
	}

	authEnforcer := basculehttp.NewEnforcer(
		basculehttp.WithELogger(GetLogger),
		basculehttp.WithRules("Basic", bascule.Validators{
			bascule.CreateAllowAllCheck(),
		}),
		basculehttp.WithRules("Bearer", bearerRules),
		basculehttp.WithEErrorResponseFunc(listener.OnErrorResponse),
	)

	constructors := []alice.Constructor{SetLogger(in.Logger), authConstructor, authEnforcer, basculehttp.NewListenerDecorator(listener)}

	chain := alice.New(constructors...)
	return AuthChainOut{Chain: &chain}, nil
}

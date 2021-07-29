/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
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

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"emperror.dev/emperror"
	"github.com/golang-jwt/jwt"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/key"
	"go.uber.org/fx"
)

const jwtPrincipalKey = "sub"

// accessLevelBearerTokenFactory extends basculehttp.BearerTokenFactory by letting
// the user of the factory inject an access level attribute to the jwt token.
// Application code should handle case in which the value is not injected (i.e. basic auth tokens).
type accessLevelBearerTokenFactory struct {
	fx.In
	DefaultKeyID string            `name:"default_key_id"`
	Resolver     key.Resolver      `name:"key_resolver"`
	Parser       bascule.JWTParser `optional:"true"`
	Leeway       bascule.Leeway    `name:"jwt_leeway" optional:"true"`
	AccessLevel  AccessLevel
}

// ParseAndValidate expects the given value to be a JWT with a kid header.  The
// kid should be resolvable by the Resolver and the JWT should be Parseable and
// pass any basic validation checks done by the Parser.  If everything goes
// well, a Token of type "jwt" is returned.
func (a accessLevelBearerTokenFactory) ParseAndValidate(ctx context.Context, _ *http.Request, _ bascule.Authorization, value string) (bascule.Token, error) {
	if len(value) == 0 {
		return nil, errors.New("empty value")
	}

	leewayclaims := bascule.ClaimsWithLeeway{
		MapClaims: make(jwt.MapClaims),
		Leeway:    a.Leeway,
	}

	jwsToken, err := a.Parser.ParseJWT(value, &leewayclaims, defaultKeyfunc(ctx, a.DefaultKeyID, a.Resolver))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to parse JWS")
	}
	if !jwsToken.Valid {
		return nil, basculehttp.ErrInvalidToken
	}

	claims, ok := jwsToken.Claims.(*bascule.ClaimsWithLeeway)

	if !ok {
		return nil, emperror.Wrap(basculehttp.ErrUnexpectedClaims, "failed to parse JWS")
	}

	claimsMap, err := claims.GetMap()
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to get map of claims", "claims struct", claims)
	}

	jwtClaims := bascule.NewAttributes(claimsMap)

	principalVal, ok := jwtClaims.Get(jwtPrincipalKey)
	if !ok {
		return nil, emperror.WrapWith(basculehttp.ErrInvalidPrincipal, "principal value not found", "principal key", jwtPrincipalKey, "jwtClaims", claimsMap)
	}
	principal, ok := principalVal.(string)
	if !ok {
		return nil, emperror.WrapWith(basculehttp.ErrInvalidPrincipal, "principal value not a string", "principal", principalVal)
	}

	if a.AccessLevel.Resolve != nil {
		claimsMap[a.AccessLevel.AttributeKey] = a.AccessLevel.Resolve(jwtClaims)
		jwtClaims = bascule.NewAttributes(claimsMap)
	}

	return bascule.NewToken("jwt", principal, jwtClaims), nil
}

func defaultKeyfunc(ctx context.Context, defaultKeyID string, keyResolver key.Resolver) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		keyID, ok := token.Header["kid"].(string)
		if !ok {
			keyID = defaultKeyID
		}

		pair, err := keyResolver.ResolveKey(ctx, keyID)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to resolve key")
		}
		return pair.Public(), nil
	}
}

func provideBearerTokenFactory(configKey string) fx.Option {
	return fx.Options(
		key.ProvideResolver(fmt.Sprintf("%s.bearer.key", configKey), true),
		provideAccessLevel(fmt.Sprintf("%s.accessLevel", configKey)),
		fx.Provide(
			fx.Annotated{
				Name: "jwt_leeway",
				Target: arrange.UnmarshalKey(fmt.Sprintf("%s.bearer.leeway", configKey),
					bascule.Leeway{}),
			},
			fx.Annotated{
				Group: "bascule_constructor_options",
				Target: func(f accessLevelBearerTokenFactory) (basculehttp.COption, error) {
					if f.Parser == nil {
						f.Parser = bascule.DefaultJWTParser
					}
					if f.Resolver == nil {
						return nil, nil
					}
					return basculehttp.WithTokenFactory(basculehttp.BearerAuthorization, f), nil
				},
			},
		),
	)
}

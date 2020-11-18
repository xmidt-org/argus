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
	"errors"
	"net/http"

	"emperror.dev/emperror"
	"github.com/dgrijalva/jwt-go"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/key"
)

const (
	jwtPrincipalKey = "sub"
)

// AccessLevelBearerTokenFactory extends basculehttp.BearerTokenFactory by injecting an access level attribute to the jwt token. How the level is generated is
// left to the user.
type AccessLevelBearerTokenFactory struct {
	DefaultKeyID      string
	Resolver          key.Resolver
	Parser            bascule.JWTParser
	Leeway            bascule.Leeway
	AccessLevelConfig AccessLevelConfig
}

// ParseAndValidate expects the given value to be a JWT with a kid header.  The
// kid should be resolvable by the Resolver and the JWT should be Parseable and
// pass any basic validation checks done by the Parser.  If everything goes
// well, a Token of type "jwt" is returned.
func (a AccessLevelBearerTokenFactory) ParseAndValidate(ctx context.Context, _ *http.Request, _ bascule.Authorization, value string) (bascule.Token, error) {
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
		return nil, basculehttp.ErrorInvalidToken
	}

	claims, ok := jwsToken.Claims.(*bascule.ClaimsWithLeeway)

	if !ok {
		return nil, emperror.Wrap(basculehttp.ErrorUnexpectedClaims, "failed to parse JWS")
	}

	claimsMap, err := claims.GetMap()
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to get map of claims", "claims struct", claims)
	}

	jwtClaims := bascule.NewAttributes(claimsMap)

	principalVal, ok := jwtClaims.Get(jwtPrincipalKey)
	if !ok {
		return nil, emperror.WrapWith(basculehttp.ErrorInvalidPrincipal, "principal value not found", "principal key", jwtPrincipalKey, "jwtClaims", claimsMap)
	}
	principal, ok := principalVal.(string)
	if !ok {
		return nil, emperror.WrapWith(basculehttp.ErrorInvalidPrincipal, "principal value not a string", "principal", principalVal)
	}
	jwtClaims = a.injectAccessLevelAttribute(claimsMap, jwtClaims)

	return bascule.NewToken("jwt", principal, jwtClaims), nil
}

func (a AccessLevelBearerTokenFactory) injectAccessLevelAttribute(claimsMap map[string]interface{}, attributes bascule.Attributes) bascule.Attributes {
	accessLevelResolver := a.AccessLevelConfig.GetResolver()
	claimsMap[a.AccessLevelConfig.GetAttributeKey()] = accessLevelResolver(attributes)
	return bascule.NewAttributes(claimsMap)
}

// TODO: maybe we should have bascule export something like this
var defaultKeyfunc = func(ctx context.Context, defaultKeyID string, keyResolver key.Resolver) jwt.Keyfunc {
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

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/xmidt-org/clortho"
	"github.com/xmidt-org/clortho/clorthofx"
	"go.uber.org/fx"
)

const jwtPrincipalKey = "sub"

// accessLevelBearerTokenFactory extends basculehttp.BearerTokenFactory by letting
// the user of the factory inject an access level attribute to the jwt token.
// Application code should handle case in which the value is not injected (i.e. basic auth tokens).
type accessLevelBearerTokenFactory struct {
	fx.In
	DefaultKeyID string `name:"default_key_id"`
	Resolver     clortho.Resolver
	Parser       bascule.JWTParser `optional:"true"`
	Leeway       bascule.Leeway    `name:"jwt_leeway" optional:"true"`
	AccessLevel  AccessLevel
}

// JWTValidator provides a convenient way to define jwt validator through config files
type JWTValidator struct {
	// Config is used to create the clortho Resolver & Refresher for JWT verification keys
	Config clortho.Config `json:"config"`
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

func defaultKeyfunc(ctx context.Context, defaultKeyID string, resolver clortho.Resolver) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		keyID, ok := token.Header["kid"].(string)
		if !ok {
			keyID = defaultKeyID
		}

		pair, err := resolver.Resolve(ctx, keyID)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to resolve key")
		}
		return pair.Public(), nil
	}
}

func provideBearerTokenFactory(configKey string) fx.Option {

	return fx.Options(

		clorthofx.Provide(),
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
					return basculehttp.WithTokenFactory(basculehttp.BearerAuthorization, f), nil
				},
			},
			arrange.UnmarshalKey("jwtValidator", JWTValidator{}),
			func(jv JWTValidator) clortho.Config {
				return jv.Config
			},
		),
	)
}

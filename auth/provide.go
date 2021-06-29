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
	"fmt"

	"github.com/xmidt-org/bascule/basculechecks"
	"github.com/xmidt-org/bascule/basculehttp"
	"go.uber.org/fx"
)

type APIBaseIn struct {
	fx.In
	Val string `name:"api_base"`
}

// Provide provides the auth alice.Chain for the primary server.
func Provide(configKey string) fx.Option {
	return fx.Options(
		basculehttp.ProvideMetrics(),
		basculechecks.ProvideMetrics(),
		fx.Provide(
			func(in APIBaseIn) basculehttp.ParseURL {
				return basculehttp.CreateRemovePrefixURLFunc(in.Val, nil)
			},
		),
		basculehttp.ProvideBasicAuth(configKey),
		provideBearerTokenFactory(configKey),
		basculechecks.ProvideRegexCapabilitiesValidator(fmt.Sprintf("%v.capabilities", configKey)),
		basculehttp.ProvideBearerValidator(),
		basculehttp.ProvideServerChain(),
	)
}

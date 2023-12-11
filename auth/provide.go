// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
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
				return basculehttp.CreateRemovePrefixURLFunc("/"+in.Val, nil)
			},
		),
		basculehttp.ProvideBasicAuth(configKey),
		provideBearerTokenFactory(configKey),
		basculechecks.ProvideRegexCapabilitiesValidator(),
		basculehttp.ProvideBearerValidator(),
		basculehttp.ProvideServerChain(),
	)
}

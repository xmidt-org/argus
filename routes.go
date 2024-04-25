// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/touchstone/touchhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.uber.org/fx"
)

type RoutesIn struct {
	fx.In
	PrimaryMetrics touchhttp.ServerInstrumenter `name:"servers.primary.metrics"`
	HealthMetrics  touchhttp.ServerInstrumenter `name:"servers.health.metrics"`
	// Routes           Routes
	// GraphQLHandler   *handler.Server
	// EventAuth        func(http.Handler) http.Handler `name:"wrp-listener.auth"`
	// EventHandler     *event.Handler
}

type PrimaryRouterIn struct {
	fx.In
	Router    *mux.Router `name:"server_primary"`
	APIBase   string      `name:"api_base"`
	AuthChain alice.Chain `name:"auth_chain"`
	// Tracing will be used to set up tracing instrumentation code.
	Tracing  candlelight.Tracing
	Handlers PrimaryHandlersIn
}

type PrimaryHandlersIn struct {
	fx.In
	Set    store.Handler `name:"set_handler"`
	Delete store.Handler `name:"delete_handler"`
	Get    store.Handler `name:"get_handler"`
	GetAll store.Handler `name:"get_all_handler"`
}

// type MetricRouterIn struct {
// 	fx.In
// 	Router  *mux.Router `name:"server_metrics"`
// 	Handler touchhttp.Handler
// }

// type PrimaryMMIn struct {
// 	fx.In
// 	Primary alice.Chain `name:"middleware_primary_metrics"`
// }

// type HealthMMIn struct {
// 	fx.In
// 	Health alice.Chain `name:"middleware_health_metrics"`
// }

type MetricMiddlewareOut struct {
	fx.Out
	Primary alice.Chain `name:"middleware_primary_metrics"`
	Health  alice.Chain `name:"middleware_health_metrics"`
}

// func provideServers() fx.Option {
// 	return fx.Options(
// 		arrangehttp.Server{
// 			Name: "server_primary",
// 			Key:  "servers.primary",
// 			Inject: arrange.Inject{
// 				PrimaryMMIn{},
// 			},
// 		}.Provide(),
// 		fx.Provide(
// 			metricMiddleware,
// 		),
// 		arrangehttp.Server{
// 			Name: "server_health",
// 			Key:  "servers.health",
// 			Inject: arrange.Inject{
// 				HealthMMIn{},
// 			},
// 			Invoke: arrange.Invoke{
// 				func(r *mux.Router) {
// 					r.Handle("/health", httpaux.ConstantHandler{
// 						StatusCode: http.StatusOK,
// 					}).Methods("GET")
// 				},
// 			},
// 		}.Provide(),
// 		arrangehttp.Server{
// 			Name: "server_metrics",
// 			Key:  "servers.metrics",
// 		}.Provide(),

// 		fx.Invoke(
// 			handlePrimaryEndpoint,
// 			handleMetricEndpoint,
// 		),
// 	)
// }

func handlePrimaryEndpoint(in PrimaryRouterIn) {
	options := []otelmux.Option{
		otelmux.WithTracerProvider(in.Tracing.TracerProvider()),
		otelmux.WithPropagators(in.Tracing.Propagator()),
	}
	in.Router.Use(
		in.AuthChain.Then,
		otelmux.Middleware("server_primary", options...),
		candlelight.EchoFirstTraceNodeInfo(in.Tracing.Propagator(), false),
	)

	bucketPath := fmt.Sprintf("/%s/store/{bucket}", in.APIBase)
	itemPath := fmt.Sprintf("%s/{id}", bucketPath)
	in.Router.Handle(itemPath, in.Handlers.Set).Methods(http.MethodPut)
	in.Router.Handle(itemPath, in.Handlers.Get).Methods(http.MethodGet)
	in.Router.Handle(bucketPath, in.Handlers.GetAll).Methods(http.MethodGet)
	in.Router.Handle(itemPath, in.Handlers.Delete).Methods(http.MethodDelete)
}

// The name should be 'primary' or 'alternate'.
func provideCoreEndpoints() fx.Option {
	return fx.Provide(
		fx.Annotated{
			Name: "servers.primary.metrics",
			Target: touchhttp.ServerBundle{}.NewInstrumenter(
				touchhttp.ServerLabel, "primary",
			),
		},
		fx.Annotated{
			Name: "servers.health.metrics",
			Target: touchhttp.ServerBundle{}.NewInstrumenter(
				touchhttp.ServerLabel, "health",
			),
		},
		func(in RoutesIn) MetricMiddlewareOut {
			return MetricMiddlewareOut{
				Primary: alice.New(in.PrimaryMetrics.Then),
				Health:  alice.New(in.HealthMetrics.Then),
			}
		},
	)
}

// func handleMetricEndpoint(in MetricRouterIn) {
// 	in.Router.Handle("/metrics", in.Handler).Methods("GET")
// }

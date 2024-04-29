// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/justinas/alice"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/recovery"
	"github.com/xmidt-org/touchstone/touchhttp"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.uber.org/fx"
)

type RoutesIn struct {
	fx.In
	PrimaryMetrics touchhttp.ServerInstrumenter `name:"servers.primary.metrics"`
	HealthMetrics  touchhttp.ServerInstrumenter `name:"servers.health.metrics"`
	PrimaryRouter  PrimaryRouterIn
	Tracing        candlelight.Tracing
}

type RoutesOut struct {
	fx.Out
	Primary arrangehttp.Option[http.Server] `group:"servers.primary.options"`
}

type PrimaryRouterIn struct {
	fx.In
	APIBase   string      `name:"api_base"`
	// AuthChain alice.Chain `name:"auth_chain"`
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

type MetricMiddlewareOut struct {
	fx.Out
	Primary alice.Chain `name:"middleware_primary_metrics"`
	Health  alice.Chain `name:"middleware_health_metrics"`
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
		func(in RoutesIn) RoutesOut {
			return RoutesOut{
				Primary: provideCoreOption("primary", in),
			}
		},
	)
}

func provideCoreOption(server string, in RoutesIn) arrangehttp.Option[http.Server] {
	return arrangehttp.AsOption[http.Server](
		func(s *http.Server) {

			mux := chi.NewMux()
			mux.Use(recovery.Middleware(recovery.WithStatusCode(555)))

			options := []otelmux.Option{
				otelmux.WithTracerProvider(in.Tracing.TracerProvider()),
				otelmux.WithPropagators(in.Tracing.Propagator()),
			}

			// TODO: should probably customize things a bit
			mux.Use(otelmux.Middleware("server_primary", options...),
				candlelight.EchoFirstTraceNodeInfo(in.Tracing.Propagator(), false))

			if server == "primary" {
				bucketPath := fmt.Sprintf("/%s/store/{bucket}", apiBase)
				itemPath := fmt.Sprintf("%s/{id}", bucketPath)
				mux.Method("PUT", itemPath, in.PrimaryRouter.Handlers.Set)
				mux.Method("GET", itemPath, in.PrimaryRouter.Handlers.Get)
				mux.Method("GET", bucketPath, in.PrimaryRouter.Handlers.GetAll)
				mux.Method("DELETE", itemPath, in.PrimaryRouter.Handlers.Delete)

				s.Handler = in.PrimaryMetrics.Then(mux)
			}
		},
	)

}

func provideHealthCheck() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(metrics touchhttp.ServerInstrumenter, path HealthPath) arrangehttp.Option[http.Server] {
				return arrangehttp.AsOption[http.Server](
					func(s *http.Server) {
						mux := chi.NewMux()
						mux.Method("GET", string(path), httpaux.ConstantHandler{
							StatusCode: http.StatusOK,
						})
						s.Handler = metrics.Then(mux)
					},
				)
			},
			fx.ParamTags(`name:"servers.health.metrics"`),
			fx.ResultTags(`group:"servers.health.options"`),
		),
	)
}

func provideMetricEndpoint() fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(metrics touchhttp.Handler, path MetricsPath) arrangehttp.Option[http.Server] {
				return arrangehttp.AsOption[http.Server](
					func(s *http.Server) {
						mux := chi.NewMux()
						mux.Method("GET", string(path), metrics)
						s.Handler = mux
					},
				)
			},
			fx.ResultTags(`group:"servers.metrics.options"`),
		),
	)
}

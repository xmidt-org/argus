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
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/themis/xhealth"
	"github.com/xmidt-org/themis/xhttp/xhttpserver"
	"github.com/xmidt-org/themis/xmetrics"
	"github.com/xmidt-org/themis/xmetrics/xmetricshttp"
	"go.uber.org/fx"
)

type ServerChainIn struct {
	fx.In

	RequestCount     *prometheus.CounterVec   `name:"server_request_count"`
	RequestDuration  *prometheus.HistogramVec `name:"server_request_duration_ms"`
	RequestsInFlight *prometheus.GaugeVec     `name:"server_requests_in_flight"`
}

func provideServerChainFactory(in ServerChainIn) xhttpserver.ChainFactory {
	return xhttpserver.ChainFactoryFunc(func(name string, o xhttpserver.Options) (alice.Chain, error) {
		var (
			curryLabel = prometheus.Labels{
				ServerLabel: name,
			}

			serverLabellers = xmetricshttp.NewServerLabellers(
				xmetricshttp.CodeLabeller{},
				xmetricshttp.MethodLabeller{},
			)
		)

		requestCount, err := in.RequestCount.CurryWith(curryLabel)
		if err != nil {
			return alice.Chain{}, err
		}

		requestDuration, err := in.RequestDuration.CurryWith(curryLabel)
		if err != nil {
			return alice.Chain{}, err
		}

		requestsInFlight, err := in.RequestsInFlight.CurryWith(curryLabel)
		if err != nil {
			return alice.Chain{}, err
		}

		return alice.New(
			xmetricshttp.HandlerCounter{
				Metric:   xmetrics.LabelledCounterVec{CounterVec: requestCount},
				Labeller: serverLabellers,
			}.Then,
			xmetricshttp.HandlerDuration{
				Metric:   xmetrics.LabelledObserverVec{ObserverVec: requestDuration},
				Labeller: serverLabellers,
			}.Then,
			xmetricshttp.HandlerInFlight{
				Metric: xmetrics.LabelledGaugeVec{GaugeVec: requestsInFlight},
			}.Then,
		), nil
	})
}

type PrimaryRouter struct {
	fx.In
	Router    *mux.Router  `name:"servers.primary"`
	AuthChain *alice.Chain `name:"servers.primary.auth"`
}

type PrimaryRoutes struct {
	fx.In
	Set    store.Handler `name:"setHandler"`
	Delete store.Handler `name:"deleteHandler"`
	Get    store.Handler `name:"getHandler"`
	GetAll store.Handler `name:"getAllHandler"`
}

func BuildPrimaryRoutes(router PrimaryRouter, routes PrimaryRoutes) {
	router.Router.Use(router.AuthChain.Then)
	itemPathFormat := "/%s/store/{bucket}/{id}"

	if routes.Set != nil {
		router.Router.Handle(fmt.Sprintf(itemPathFormat, apiBase), routes.Set).Methods(http.MethodPut)
	}
	if routes.Get != nil {
		router.Router.Handle(fmt.Sprintf(itemPathFormat, apiBase), routes.Get).Methods(http.MethodGet)
	}
	if routes.GetAll != nil {
		router.Router.Handle(fmt.Sprintf("/%s/store/{bucket}", apiBase), routes.GetAll).Methods(http.MethodGet)
	}
	if routes.Delete != nil {
		router.Router.Handle(fmt.Sprintf(itemPathFormat, apiBase), routes.Delete).Methods(http.MethodDelete)
	}
}

type MetricsRoutesIn struct {
	fx.In
	Router  *mux.Router `name:"servers.metrics"`
	Handler xmetricshttp.Handler
}

func BuildMetricsRoutes(in MetricsRoutesIn) {
	if in.Router != nil && in.Handler != nil {
		in.Router.Handle("/metrics", in.Handler).Methods("GET")
	}
}

type HealthRoutesIn struct {
	fx.In
	Router  *mux.Router `name:"servers.health"`
	Handler xhealth.Handler
}

func BuildHealthRoutes(in HealthRoutesIn) {
	if in.Router != nil && in.Handler != nil {
		in.Router.Handle("/health", in.Handler).Methods("GET")
	}
}

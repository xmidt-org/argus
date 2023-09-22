// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
)

// Metric names.
const (
	// Common across data backends.
	QueryDurationSecondsHistogram = "db_query_duration_seconds"
	QueriesCounter                = "db_queries_total"

	// DynamoDB-specific metrics.
	DynamodbConsumedCapacityCounter = "dynamodb_consumed_capacity_total"
	DynamodbGetAllGauge             = "dynamodb_get_all_results"
)

// Metric label keys.
const (
	QueryOutcomeLabelKey     = "outcome"
	QueryTypeLabelKey        = "type"
	DynamoCapacityOpLabelKey = "op"
)

// Metric label values for DAO operation types.
const (
	GetQueryType    = "get"
	GetAllQueryType = "getall"
	DeleteQueryType = "delete"
	PushQueryType   = "push"
	PingQueryType   = "ping"
)

// Metric label values for Query Outcomes.
const (
	FailQueryOutcome    = "fail"
	SuccessQueryOutcome = "success"
)

// Metric label values for DynamoDB Consumed capacity type
const (
	DynamoCapacityReadOp  = "read"
	DynamoCapacityWriteOp = "write"
)

// ProvideMetrics returns the Metrics relevant to this package
func ProvideMetrics() fx.Option {
	return fx.Options(
		touchstone.CounterVec(
			prometheus.CounterOpts{
				Name: QueriesCounter,
				Help: "The total number of DB queries Argus has performed.",
			},
			QueryOutcomeLabelKey,
			QueryTypeLabelKey,
		),

		touchstone.HistogramVec(
			prometheus.HistogramOpts{
				Name:    QueryDurationSecondsHistogram,
				Help:    "A histogram of latencies for queries.",
				Buckets: []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
			},
			QueryTypeLabelKey,
		),

		touchstone.CounterVec(
			prometheus.CounterOpts{
				Name: DynamodbConsumedCapacityCounter,
				Help: "Capacity units consumed by the DynamoDB operation.",
			},
			QueryTypeLabelKey,
			DynamoCapacityOpLabelKey,
		),

		touchstone.Gauge(
			prometheus.GaugeOpts{
				Name: DynamodbGetAllGauge,
				Help: "Amount of records returned for a GetAll dynamodb request.",
			},
		),
	)
}

type Measures struct {
	fx.In
	Queries                  *prometheus.CounterVec `name:"db_queries_total"`
	QueryDurationSeconds     prometheus.ObserverVec `name:"db_query_duration_seconds"`
	DynamodbConsumedCapacity *prometheus.CounterVec `name:"dynamodb_consumed_capacity_total"`
	DynamodbGetAllGauge      prometheus.Gauge       `name:"dynamodb_get_all_results"`
}

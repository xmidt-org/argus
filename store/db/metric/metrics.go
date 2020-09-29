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

package metric

import (
	"github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/themis/xmetrics"
	"go.uber.org/fx"
)

// Metric names.
const (
	// Common across data backends.
	QueryDurationSecondsHistogram = "db_query_duration_seconds"
	QueriesCounter                = "db_queries_total"

	// DynamoDB-specific metrics.
	DynamodbConsumedCapacityCounter = "dynamodb_consumed_capacity_total"
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
	return fx.Provide(
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: QueriesCounter,
				Help: "The total number of DB queries Argus has performed.",
			},
			QueryOutcomeLabelKey,
			QueryTypeLabelKey,
		),

		xmetrics.ProvideHistogram(
			prometheus.HistogramOpts{
				Name:    QueryDurationSecondsHistogram,
				Help:    "A histogram of latencies for queries.",
				Buckets: []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
			},
			QueryTypeLabelKey,
		),

		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: DynamodbConsumedCapacityCounter,
				Help: "Capacity units consumed by the DynamoDB operation.",
			},
			QueryTypeLabelKey,
			DynamoCapacityOpLabelKey,
		),
	)
}

type Measures struct {
	fx.In
	Queries                  metrics.Counter   `name:"db_queries_total"`
	QueryDurationSeconds     metrics.Histogram `name:"db_query_duration_seconds"`
	DynamodbConsumedCapacity metrics.Counter   `name:"dynamodb_consumed_capacity_total"`
}

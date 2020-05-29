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
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/themis/xmetrics"
	"go.uber.org/fx"
)

// Generic Metrics
const (
	PoolInUseConnectionsGauge = "pool_in_use_connections"
	SQLDurationSeconds        = "sql_duration_seconds"
	SQLQuerySuccessCounter    = "sql_query_success_count"
	SQLQueryFailureCounter    = "sql_query_failure_count"
	SQLInsertedRecordsCounter = "sql_inserted_rows_count"
	SQLReadRecordsCounter     = "sql_read_rows_count"
	SQLDeletedRecordsCounter  = "sql_deleted_rows_count"
)

// DynamoDB metrics
const (
	CapacityUnitConsumedCounter  = "capacity_unit_consumed"
	ReadCapacityConsumedCounter  = "read_capacity_unit_consumed"
	WriteCapacityConsumedCounter = "write_capacity_unit_consumed"
)

// Metrics returns the Metrics relevant to this package
func ProvideMetrics() fx.Option {
	return fx.Provide(
		xmetrics.ProvideGauge(
			prometheus.GaugeOpts{
				Name: PoolInUseConnectionsGauge,
				Help: "The number of connections currently in use",
			},
		),
		xmetrics.ProvideHistogram(
			prometheus.HistogramOpts{
				Name:    SQLDurationSeconds,
				Help:    "A histogram of latencies for requests.",
				Buckets: []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
			},
			store.TypeLabel,
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SQLQuerySuccessCounter,
				Help: "The total number of successful SQL queries",
			},
			store.TypeLabel,
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SQLQueryFailureCounter,
				Help: "The total number of failed SQL queries",
			},
			store.TypeLabel,
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SQLInsertedRecordsCounter,
				Help: "The total number of rows inserted",
			},
		),

		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SQLReadRecordsCounter,
				Help: "The total number of rows read",
			},
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SQLDeletedRecordsCounter,
				Help: "The total number of rows deleted",
			},
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: CapacityUnitConsumedCounter,
				Help: "The number of capacity units consumed by the operation.",
			},
			store.TypeLabel,
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: ReadCapacityConsumedCounter,
				Help: "The number of read capacity units consumed by the operation.",
			},
			store.TypeLabel,
		),
		xmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: WriteCapacityConsumedCounter,
				Help: "The number of write capacity units consumed by the operation.",
			},
			store.TypeLabel,
		),
	)
}

type Measures struct {
	fx.In
	PoolInUseConnections metrics.Gauge     `name:"pool_in_use_connections"`
	SQLDuration          metrics.Histogram `name:"sql_duration_seconds"`
	SQLQuerySuccessCount metrics.Counter   `name:"sql_query_success_count"`
	SQLQueryFailureCount metrics.Counter   `name:"sql_query_failure_count"`
	SQLInsertedRecords   metrics.Counter   `name:"sql_inserted_rows_count"`
	SQLReadRecords       metrics.Counter   `name:"sql_read_rows_count"`
	SQLDeletedRecords    metrics.Counter   `name:"sql_deleted_rows_count"`

	// DynamoDB Metrics
	CapacityUnitConsumedCount      metrics.Counter `name:"capacity_unit_consumed"`
	ReadCapacityUnitConsumedCount  metrics.Counter `name:"read_capacity_unit_consumed"`
	WriteCapacityUnitConsumedCount metrics.Counter `name:"write_capacity_unit_consumed"`
}

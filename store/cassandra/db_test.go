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

package cassandra

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/storetest"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
	"testing"
)

func TestCassandra(t *testing.T) {
	// TODO: Test metrics
	// require := require.New(t)
	mockDB := &mockDB{}
	mockDB.On("Push", mock.Anything, mock.Anything).Return(nil)
	mockDB.On("Get", mock.Anything).Return(storetest.GenericTestKeyPair.Item, nil).Once()
	mockDB.On("Get", mock.Anything).Return(store.Item{}, nil).Once()
	mockDB.On("Delete", mock.Anything, mock.Anything).Return(storetest.GenericTestKeyPair.Item, nil)
	mockDB.On("GetAll", mock.Anything).Return(map[string]store.Item{"earth": storetest.GenericTestKeyPair.Item}, nil).Once()
	mockDB.On("GetAll", mock.Anything).Return(map[string]store.Item{}, nil).Once()

	mockDB.On("Ping").Return(nil)

	p := xmetricstest.NewProvider(nil, func() []xmetrics.Metric {
		return []xmetrics.Metric{
			{
				Name: PoolInUseConnectionsGauge,
				Type: "gauge",
				Help: " The number of connections currently in use",
			},
			{
				Name:       SQLDurationSeconds,
				Type:       "histogram",
				Help:       "A histogram of latencies for requests.",
				Buckets:    []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name:       SQLQuerySuccessCounter,
				Type:       "counter",
				Help:       "The total number of successful SQL queries",
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name:       SQLQueryFailureCounter,
				Type:       "counter",
				Help:       "The total number of failed SQL queries",
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name: SQLInsertedRecordsCounter,
				Type: "counter",
				Help: "The total number of rows inserted",
			},
			{
				Name: SQLReadRecordsCounter,
				Type: "counter",
				Help: "The total number of rows read",
			},
			{
				Name: SQLDeletedRecordsCounter,
				Type: "counter",
				Help: "The total number of rows deleted",
			},
		}
	})
	// register, err := xmetrics.New(xmetrics.Options{
	// 	DefaultNamespace:        "testing",
	// 	DefaultSubsystem:        "neat",
	// 	Pedantic:                false,
	// 	DisableGoCollector:      false,
	// 	DisableProcessCollector: false,
	// 	ConstLabels:             nil,
	// })
	// require.NoError(err)
	// poolInUseConnections, err := register.NewGauge(
	// 	prometheus.GaugeOpts{
	// 		Name: PoolInUseConnectionsGauge,
	// 		Help: "The number of connections currently in use",
	// 	}, []string{},
	// )
	// require.NoError(err)
	// sqlDurationSeconds, err := register.NewHistogram(
	// 	prometheus.HistogramOpts{
	// 		Name:    SQLDurationSeconds,
	// 		Help:    "A histogram of latencies for requests.",
	// 		Buckets: []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
	// 	},
	// 	[]string{store.TypeLabel},
	// )
	// require.NoError(err)
	// sqlQuerySuccessCounter, err := register.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: SQLQuerySuccessCounter,
	// 		Help: "The total number of successful SQL queries",
	// 	},
	// 	[]string{store.TypeLabel},
	// )
	// require.NoError(err)
	// sqlQueryFailureCounter, err := register.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: SQLQueryFailureCounter,
	// 		Help: "The total number of successful SQL queries",
	// 	},
	// 	[]string{store.TypeLabel},
	// )
	// require.NoError(err)
	// sqlInsertedRecordsCounter, err := register.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: SQLInsertedRecordsCounter,
	// 		Help: "The total number of successful SQL queries",
	// 	},
	// 	[]string{},
	// )
	// require.NoError(err)
	// sqlReadRecordsCounter, err := register.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: SQLReadRecordsCounter,
	// 		Help: "The total number of successful SQL queries",
	// 	},
	// 	[]string{},
	// )
	// require.NoError(err)
	// sqlDeletedRecordsCounter, err := register.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: SQLDeletedRecordsCounter,
	// 		Help: "The total number of successful SQL queries",
	// 	},
	// 	[]string{},
	// )
	// require.NoError(err)

	s := &CassandraClient{
		client: mockDB,
		config: CassandraConfig{},
		logger: logging.NewTestLogger(nil, t),
		measures: Measures{
			PoolInUseConnections: p.NewGauge(PoolInUseConnectionsGauge),
			SQLDuration:          p.NewHistogram(SQLDurationSeconds, 11),
			SQLQuerySuccessCount: p.NewCounter(SQLQuerySuccessCounter),
			SQLQueryFailureCount: p.NewCounter(SQLQueryFailureCounter),
			SQLInsertedRecords:   p.NewCounter(SQLInsertedRecordsCounter),
			SQLReadRecords:       p.NewCounter(SQLReadRecordsCounter),
			SQLDeletedRecords:    p.NewCounter(SQLDeletedRecordsCounter),
		},
	}
	p.Assert(t, SQLQuerySuccessCounter)(xmetricstest.Value(0.0))
	p.Assert(t, SQLQueryFailureCounter)(xmetricstest.Value(0.0))

	storetest.StoreTest(s, 0, t)
	p.Assert(t, SQLQuerySuccessCounter, store.TypeLabel, store.ReadType)(xmetricstest.Value(3.0))
	p.Assert(t, SQLQuerySuccessCounter, store.TypeLabel, store.InsertType)(xmetricstest.Value(1.0))
	p.Assert(t, SQLQuerySuccessCounter, store.TypeLabel, store.DeleteType)(xmetricstest.Value(1.0))
	p.Assert(t, SQLInsertedRecordsCounter)(xmetricstest.Value(0.0))
	p.Assert(t, SQLReadRecordsCounter)(xmetricstest.Value(0.0))
	p.Assert(t, SQLDeletedRecordsCounter)(xmetricstest.Value(0.0))

}

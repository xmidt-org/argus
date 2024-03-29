// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"testing"
)

func TestCassandra(t *testing.T) {
	// TODO: Test metrics
	// require := require.New(t)
	// mockDB := &test.MockDB{}
	// mockDB.On("Push", mock.Anything, mock.Anything).Return(nil)
	// mockDB.On("Get", mock.Anything).Return(test.GenericTestKeyPair.OwnableItem, nil).Once()
	// mockDB.On("Get", mock.Anything).Return(store.OwnableItem{}, nil).Once()
	// mockDB.On("Delete", mock.Anything, mock.Anything).Return(test.GenericTestKeyPair.OwnableItem, nil)
	// mockDB.On("GetAll", mock.Anything).Return(map[string]store.OwnableItem{"earth": test.GenericTestKeyPair.OwnableItem}, nil).Once()
	// mockDB.On("GetAll", mock.Anything).Return(map[string]store.OwnableItem{}, nil).Once()

	// mockDB.On("Ping").Return(nil)

	// p := xmetricstest.NewProvider(nil, func() []xmetrics.Metric {
	// 	return []xmetrics.Metric{
	// 		{
	// 			Name: metric.PoolInUseConnectionsGauge,
	// 			Type: "gauge",
	// 			Help: " The number of connections currently in use",
	// 		},
	// 		{
	// 			Name:       metric.SQLDurationSeconds,
	// 			Type:       "histogram",
	// 			Help:       "A histogram of latencies for requests.",
	// 			Buckets:    []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
	// 			LabelNames: []string{store.TypeLabel},
	// 		},
	// 		{
	// 			Name:       metric.SQLQuerySuccessCounter,
	// 			Type:       "counter",
	// 			Help:       "The total number of successful SQL queries",
	// 			LabelNames: []string{store.TypeLabel},
	// 		},
	// 		{
	// 			Name:       metric.SQLQueryFailureCounter,
	// 			Type:       "counter",
	// 			Help:       "The total number of failed SQL queries",
	// 			LabelNames: []string{store.TypeLabel},
	// 		},
	// 		{
	// 			Name: metric.SQLInsertedRecordsCounter,
	// 			Type: "counter",
	// 			Help: "The total number of rows inserted",
	// 		},
	// 		{
	// 			Name: metric.SQLReadRecordsCounter,
	// 			Type: "counter",
	// 			Help: "The total number of rows read",
	// 		},
	// 		{
	// 			Name: metric.SQLDeletedRecordsCounter,
	// 			Type: "counter",
	// 			Help: "The total number of rows deleted",
	// 		},
	// 	}
	// })

	// s := &CassandraClient{
	// 	client: mockDB,
	// 	config: CassandraConfig{},
	// 	logger: sallust.Default(),
	// 	measures: metric.Measures{
	// 		PoolInUseConnections: p.NewGauge(metric.PoolInUseConnectionsGauge),
	// 		SQLDuration:          p.NewHistogram(metric.SQLDurationSeconds, 11),
	// 		SQLQuerySuccessCount: p.NewCounter(metric.SQLQuerySuccessCounter),
	// 		SQLQueryFailureCount: p.NewCounter(metric.SQLQueryFailureCounter),
	// 		SQLInsertedRecords:   p.NewCounter(metric.SQLInsertedRecordsCounter),
	// 		SQLReadRecords:       p.NewCounter(metric.SQLReadRecordsCounter),
	// 		SQLDeletedRecords:    p.NewCounter(metric.SQLDeletedRecordsCounter),
	// 	},
	// }
	// p.Assert(t, metric.SQLQuerySuccessCounter)(xmetricstest.Value(0.0))
	// p.Assert(t, metric.SQLQueryFailureCounter)(xmetricstest.Value(0.0))

	// test.StoreTest(s, 0, t)
	// p.Assert(t, metric.SQLQuerySuccessCounter, store.TypeLabel, store.ReadType)(xmetricstest.Value(3.0))
	// p.Assert(t, metric.SQLQuerySuccessCounter, store.TypeLabel, store.InsertType)(xmetricstest.Value(1.0))
	// p.Assert(t, metric.SQLQuerySuccessCounter, store.TypeLabel, store.DeleteType)(xmetricstest.Value(1.0))
	// p.Assert(t, metric.SQLInsertedRecordsCounter)(xmetricstest.Value(0.0))
	// p.Assert(t, metric.SQLReadRecordsCounter)(xmetricstest.Value(0.0))
	// p.Assert(t, metric.SQLDeletedRecordsCounter)(xmetricstest.Value(0.0))

}

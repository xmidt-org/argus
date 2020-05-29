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

package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/argus/store/test"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
	"testing"
)

func TestDynamo(t *testing.T) {
	// require := require.New(t)
	mockDB := &MockDB{}
	mockDB.On("Push", mock.Anything, mock.Anything).Return(&dynamodb.ConsumedCapacity{CapacityUnits: aws.Float64(0.5), TableName: aws.String("fignoc")}, nil)
	mockDB.On("Get", mock.Anything).Return(test.GenericTestKeyPair.OwnableItem, &dynamodb.ConsumedCapacity{CapacityUnits: aws.Float64(1.0), TableName: aws.String("fignoc")}, nil).Once()
	mockDB.On("Get", mock.Anything).Return(store.OwnableItem{}, &dynamodb.ConsumedCapacity{CapacityUnits: aws.Float64(1.0), TableName: aws.String("fignoc")}, nil).Once()
	mockDB.On("Delete", mock.Anything, mock.Anything).Return(test.GenericTestKeyPair.OwnableItem, &dynamodb.ConsumedCapacity{CapacityUnits: aws.Float64(1.0), TableName: aws.String("fignoc")}, nil)
	mockDB.On("GetAll", mock.Anything).Return(map[string]store.OwnableItem{"earth": test.GenericTestKeyPair.OwnableItem}, &dynamodb.ConsumedCapacity{CapacityUnits: aws.Float64(1.0), TableName: aws.String("fignoc")}, nil).Once()
	mockDB.On("GetAll", mock.Anything).Return(map[string]store.OwnableItem{}, &dynamodb.ConsumedCapacity{CapacityUnits: aws.Float64(0.0), TableName: aws.String("fignoc")}, nil).Once()

	mockDB.On("Ping").Return(nil)

	p := xmetricstest.NewProvider(nil, func() []xmetrics.Metric {
		return []xmetrics.Metric{
			{
				Name: metric.PoolInUseConnectionsGauge,
				Type: "gauge",
				Help: " The number of connections currently in use",
			},
			{
				Name:       metric.SQLDurationSeconds,
				Type:       "histogram",
				Help:       "A histogram of latencies for requests.",
				Buckets:    []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name:       metric.SQLQuerySuccessCounter,
				Type:       "counter",
				Help:       "The total number of successful SQL queries",
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name:       metric.SQLQueryFailureCounter,
				Type:       "counter",
				Help:       "The total number of failed SQL queries",
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name: metric.SQLInsertedRecordsCounter,
				Type: "counter",
				Help: "The total number of rows inserted",
			},
			{
				Name: metric.SQLReadRecordsCounter,
				Type: "counter",
				Help: "The total number of rows read",
			},
			{
				Name: metric.SQLDeletedRecordsCounter,
				Type: "counter",
				Help: "The total number of rows deleted",
			},
			{
				Name:       metric.CapacityUnitConsumedCounter,
				Type:       "counter",
				Help:       "The number of capacity units consumed by the operation.",
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name:       metric.ReadCapacityConsumedCounter,
				Type:       "counter",
				Help:       "The number of read capacity units consumed by the operation.",
				LabelNames: []string{store.TypeLabel},
			},
			{
				Name:       metric.WriteCapacityConsumedCounter,
				Type:       "counter",
				Help:       "The number of write capacity units consumed by the operation.",
				LabelNames: []string{store.TypeLabel},
			},
		}
	})

	s := &DynamoClient{
		client: mockDB,
		config: Config{},
		logger: logging.NewTestLogger(nil, t),
		measures: metric.Measures{
			PoolInUseConnections:           p.NewGauge(metric.PoolInUseConnectionsGauge),
			SQLDuration:                    p.NewHistogram(metric.SQLDurationSeconds, 11),
			SQLQuerySuccessCount:           p.NewCounter(metric.SQLQuerySuccessCounter),
			SQLQueryFailureCount:           p.NewCounter(metric.SQLQueryFailureCounter),
			SQLInsertedRecords:             p.NewCounter(metric.SQLInsertedRecordsCounter),
			SQLReadRecords:                 p.NewCounter(metric.SQLReadRecordsCounter),
			SQLDeletedRecords:              p.NewCounter(metric.SQLDeletedRecordsCounter),
			CapacityUnitConsumedCount:      p.NewCounter(metric.CapacityUnitConsumedCounter),
			ReadCapacityUnitConsumedCount:  p.NewCounter(metric.ReadCapacityConsumedCounter),
			WriteCapacityUnitConsumedCount: p.NewCounter(metric.WriteCapacityConsumedCounter),
		},
	}
	p.Assert(t, metric.SQLQuerySuccessCounter)(xmetricstest.Value(0.0))
	p.Assert(t, metric.SQLQueryFailureCounter)(xmetricstest.Value(0.0))

	test.StoreTest(s, 0, t)
	p.Assert(t, metric.SQLQuerySuccessCounter, store.TypeLabel, store.ReadType)(xmetricstest.Value(3.0))
	p.Assert(t, metric.SQLQuerySuccessCounter, store.TypeLabel, store.InsertType)(xmetricstest.Value(1.0))
	p.Assert(t, metric.SQLQuerySuccessCounter, store.TypeLabel, store.DeleteType)(xmetricstest.Value(1.0))
	p.Assert(t, metric.SQLInsertedRecordsCounter)(xmetricstest.Value(0.0))
	p.Assert(t, metric.SQLReadRecordsCounter)(xmetricstest.Value(0.0))
	p.Assert(t, metric.SQLDeletedRecordsCounter)(xmetricstest.Value(0.0))
	p.Assert(t, metric.CapacityUnitConsumedCounter, store.TypeLabel, store.ReadType)(xmetricstest.Value(2.0))
	p.Assert(t, metric.CapacityUnitConsumedCounter, store.TypeLabel, store.InsertType)(xmetricstest.Value(0.5))
	p.Assert(t, metric.CapacityUnitConsumedCounter, store.TypeLabel, store.DeleteType)(xmetricstest.Value(1.0))
}

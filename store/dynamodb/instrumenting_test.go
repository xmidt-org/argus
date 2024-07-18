// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package dynamodb

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/touchstone/touchtest"
)

func setupUpdateCalls(u *mockMeasuresUpdater, consumedCapacity *dynamodb.ConsumedCapacity, err error, now time.Time) {
	pushMeasureUpdateRequest := &measureUpdateRequest{
		err:              err,
		consumedCapacity: consumedCapacity,
		queryType:        metric.PushQueryType,
		start:            now,
	}

	getMeasureUpdateRequest := &measureUpdateRequest{
		err:              err,
		consumedCapacity: consumedCapacity,
		queryType:        metric.GetQueryType,
		start:            now,
	}

	getAllMeasureUpdateRequest := &measureUpdateRequest{
		consumedCapacity: consumedCapacity,
		queryType:        metric.GetAllQueryType,
		start:            now,
	}

	deleteMeasureUpdateRequest := &measureUpdateRequest{
		consumedCapacity: consumedCapacity,
		queryType:        metric.DeleteQueryType,
		start:            now,
	}

	u.On("Update", deleteMeasureUpdateRequest).Once()
	u.On("Update", getMeasureUpdateRequest).Once()
	u.On("Update", pushMeasureUpdateRequest).Once()
	u.On("Update", getAllMeasureUpdateRequest).Once()
}

func TestInstrumentingService(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	u := new(mockMeasuresUpdater)
	now := time.Now()
	fixedNow := func() time.Time {
		return now
	}
	key := model.Key{}
	item := store.OwnableItem{}
	items := map[string]store.OwnableItem{}
	consumedCapacity := &dynamodb.ConsumedCapacity{}
	err := errors.New("err")

	svc := newInstrumentingService(u, m, fixedNow)

	m.On("Push", key, item).Return(consumedCapacity, err).Once()
	m.On("Get", key).Return(item, consumedCapacity, err).Once()
	m.On("Delete", key).Return(item, consumedCapacity, nil).Once()
	m.On("GetAll", "bucket").Return(items, consumedCapacity, nil).Once()

	setupUpdateCalls(u, consumedCapacity, err, now)

	cc, e := svc.Push(key, item)
	assert.Equal(consumedCapacity, cc)
	assert.Equal(err, e)

	i, cc, e := svc.Get(key)
	assert.Equal(item, i)
	assert.Equal(consumedCapacity, cc)
	assert.Equal(err, e)

	i, cc, e = svc.Delete(key)
	assert.Equal(item, i)
	assert.Equal(consumedCapacity, cc)
	assert.Nil(e)

	is, cc, e := svc.GetAll("bucket")
	assert.Equal(items, is)
	assert.Equal(consumedCapacity, cc)
	assert.Nil(e)

	m.AssertExpectations(t)
	u.AssertExpectations(t)
}

func TestMeasuresUpdate(t *testing.T) {
	var (
		capacityUnits float64 = 5
		errDummy              = errors.New("bummer")
	)
	tcs := []struct {
		Name            string
		QueryType       string
		IncludeCapacity bool
		Err             error
		ExpectedOutcome string
		ExpectedOpType  string
	}{
		{
			Name:            "Consumed Capacity Missing",
			ExpectedOutcome: metric.SuccessQueryOutcome,
		},

		{
			Name:            "Successful Get Query",
			IncludeCapacity: true,
			ExpectedOutcome: metric.SuccessQueryOutcome,
			ExpectedOpType:  metric.DynamoCapacityReadOp,
			QueryType:       metric.GetQueryType,
		},

		{
			Name:            "Successful GetAll Query",
			IncludeCapacity: true,
			ExpectedOutcome: metric.SuccessQueryOutcome,
			QueryType:       metric.GetAllQueryType,
			ExpectedOpType:  metric.DynamoCapacityReadOp,
		},

		{
			Name:            "Successful Delete Query",
			IncludeCapacity: true,
			ExpectedOutcome: metric.SuccessQueryOutcome,
			QueryType:       metric.DeleteQueryType,
			ExpectedOpType:  metric.DynamoCapacityWriteOp,
		},

		{
			Name:            "Failed Push Query",
			IncludeCapacity: true,
			Err:             errDummy,
			ExpectedOutcome: metric.FailQueryOutcome,
			QueryType:       metric.PushQueryType,
			ExpectedOpType:  metric.DynamoCapacityWriteOp,
		},

		{
			Name:            "Get Query. Item not found",
			IncludeCapacity: true,
			ExpectedOutcome: metric.SuccessQueryOutcome,
			Err:             store.ErrItemNotFound,
			QueryType:       metric.GetQueryType,
			ExpectedOpType:  metric.DynamoCapacityReadOp,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			assert := assert.New(t)
			testAssert := touchtest.New(t)
			expectedRegistry := prometheus.NewPedanticRegistry()
			expectedMeasures := &metric.Measures{
				Queries: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "testQueriesCounter",
						Help: "testQueriesCounter",
					},
					[]string{
						metric.QueryOutcomeLabelKey,
						metric.QueryTypeLabelKey,
					},
				),
				DynamodbConsumedCapacity: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "testDynamoCounter",
						Help: "testDynamoCounter",
					},
					[]string{
						metric.DynamoCapacityOpLabelKey,
						metric.QueryTypeLabelKey,
					},
				),
			}
			expectedRegistry.MustRegister(expectedMeasures.DynamodbConsumedCapacity,
				expectedMeasures.Queries)
			expectedMeasures.Queries.With(prometheus.Labels{
				metric.QueryOutcomeLabelKey: tc.ExpectedOutcome,
				metric.QueryTypeLabelKey:    tc.QueryType,
			}).Inc()
			if tc.IncludeCapacity {
				expectedMeasures.DynamodbConsumedCapacity.With(prometheus.Labels{
					metric.DynamoCapacityOpLabelKey: tc.ExpectedOpType,
					metric.QueryTypeLabelKey:        tc.QueryType,
				}).Add(capacityUnits)
			}
			actualRegistry := prometheus.NewPedanticRegistry()
			m := &metric.Measures{
				Queries: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "testQueriesCounter",
						Help: "testQueriesCounter",
					},
					[]string{
						metric.QueryOutcomeLabelKey,
						metric.QueryTypeLabelKey,
					},
				),
				QueryDurationSeconds: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "testDurationHistogram",
						Help:    "testDurationHistogram",
						Buckets: []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
					},
					[]string{
						metric.QueryTypeLabelKey,
					},
				),
				DynamodbConsumedCapacity: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "testDynamoCounter",
						Help: "testDynamoCounter",
					},
					[]string{
						metric.DynamoCapacityOpLabelKey,
						metric.QueryTypeLabelKey,
					},
				),
			}
			actualRegistry.MustRegister(m.Queries, m.QueryDurationSeconds, m.DynamodbConsumedCapacity)
			updater := &dynamoMeasuresUpdater{
				measures: m,
			}
			r := &measureUpdateRequest{
				queryType: tc.QueryType,
				start:     time.Now(),
				err:       tc.Err,
			}

			if tc.IncludeCapacity {
				r.consumedCapacity = &dynamodb.ConsumedCapacity{
					CapacityUnits: aws.Float64(capacityUnits),
				}
			}

			updater.Update(r)

			testAssert.Expect(expectedRegistry)
			assert.True(testAssert.GatherAndCompare(actualRegistry,
				"testQueriesCounter", "testDynamoCounter"))

			//TODO: explore the values observed in a histogram for the QueryDurationSeconds
		})
	}
}

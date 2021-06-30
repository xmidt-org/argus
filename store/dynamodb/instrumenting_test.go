package dynamodb

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
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
		Name                  string
		QueryType             string
		IncludeCapacity       bool
		Err                   error
		ExpectedSuccessCount  float64
		ExpectedFailCount     float64
		ExpectedReadCapacity  float64
		ExpectedWriteCapacity float64
	}{
		{
			Name:                 "Consumed Capacity Missing",
			ExpectedSuccessCount: 1,
		},

		{
			Name:                 "Successful Get Query",
			IncludeCapacity:      true,
			ExpectedSuccessCount: 1,
			QueryType:            metric.GetQueryType,
			ExpectedReadCapacity: capacityUnits,
		},

		{
			Name:                 "Successful GetAll Query",
			IncludeCapacity:      true,
			ExpectedSuccessCount: 1,
			QueryType:            metric.GetAllQueryType,
			ExpectedReadCapacity: capacityUnits,
		},

		{
			Name:                  "Successful Delete Query",
			IncludeCapacity:       true,
			ExpectedSuccessCount:  1,
			QueryType:             metric.DeleteQueryType,
			ExpectedWriteCapacity: capacityUnits,
		},

		{
			Name:                  "Failed Push Query",
			IncludeCapacity:       true,
			Err:                   errDummy,
			ExpectedFailCount:     1,
			QueryType:             metric.PushQueryType,
			ExpectedWriteCapacity: capacityUnits,
		},

		{
			Name:                 "Get Query. Item not found",
			IncludeCapacity:      true,
			ExpectedSuccessCount: 1,
			Err:                  store.ErrItemNotFound,
			QueryType:            metric.GetQueryType,
			ExpectedReadCapacity: capacityUnits,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {

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

			// TODO: check metric values again.
			// p.Assert(t, "queries", metric.QueryOutcomeLabelKey, metric.SuccessQueryOutcome, metric.QueryTypeLabelKey, tc.QueryType)(xmetricstest.Value(tc.ExpectedSuccessCount))
			// p.Assert(t, "queries", metric.QueryOutcomeLabelKey, metric.FailQueryOutcome, metric.QueryTypeLabelKey, tc.QueryType)(xmetricstest.Value(tc.ExpectedFailCount))

			// p.Assert(t, "dynamo", metric.QueryTypeLabelKey, tc.QueryType, metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityReadOp)(xmetricstest.Value(tc.ExpectedReadCapacity))
			// p.Assert(t, "dynamo", metric.QueryTypeLabelKey, tc.QueryType, metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityWriteOp)(xmetricstest.Value(tc.ExpectedWriteCapacity))

			//TODO: due to limitations in xmetricstest, we can't explore the values observed in a histogram for the QueryDurationSeconds
		})
	}
}

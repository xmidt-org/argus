package dynamodb

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"

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
		readCapacityUnits  float64 = 3
		writeCapacityUnits float64 = 5
	)
	testCases := []struct {
		Name                  string
		IncludeReadCapacity   bool
		IncludeWriteCapacity  bool
		IncludeError          bool
		ExpectedSuccessCount  float64
		ExpectedFailCount     float64
		ExpectedReadCapacity  float64
		ExpectedWriteCapacity float64
	}{
		{
			Name:                  "Successful Query",
			IncludeReadCapacity:   true,
			IncludeWriteCapacity:  true,
			ExpectedSuccessCount:  1,
			ExpectedReadCapacity:  readCapacityUnits,
			ExpectedWriteCapacity: writeCapacityUnits,
		},

		{
			Name:                  "Failed Query",
			IncludeReadCapacity:   true,
			IncludeWriteCapacity:  true,
			IncludeError:          true,
			ExpectedFailCount:     1,
			ExpectedReadCapacity:  readCapacityUnits,
			ExpectedWriteCapacity: writeCapacityUnits,
		},

		{
			Name:                 "Consumed Capacity Missing",
			ExpectedSuccessCount: 1,
		},
		{
			Name:                  "Read Consumed Capacity Missing",
			IncludeWriteCapacity:  true,
			ExpectedSuccessCount:  1,
			ExpectedWriteCapacity: writeCapacityUnits,
		},

		{
			Name:                 "Write Consumed Capacity Missing",
			IncludeReadCapacity:  true,
			ExpectedSuccessCount: 1,
			ExpectedReadCapacity: readCapacityUnits,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			p := xmetricstest.NewProvider(&xmetrics.Options{})
			updater := &dynamoMeasuresUpdater{
				measures: &metric.Measures{
					Queries:                  p.NewCounter("queries"),
					QueryDurationSeconds:     p.NewHistogram("duration", 5),
					DynamodbConsumedCapacity: p.NewCounter("dynamo"),
				},
			}
			r := &measureUpdateRequest{
				queryType: "insert",
				start:     time.Now(),
			}

			if testCase.IncludeError {
				r.err = errors.New("bummer")
			}

			if testCase.IncludeReadCapacity || testCase.IncludeWriteCapacity {
				r.consumedCapacity = &dynamodb.ConsumedCapacity{}
				if testCase.IncludeReadCapacity {
					r.consumedCapacity.ReadCapacityUnits = aws.Float64(readCapacityUnits)
				}
				if testCase.IncludeWriteCapacity {
					r.consumedCapacity.WriteCapacityUnits = aws.Float64(writeCapacityUnits)
				}
			}

			updater.Update(r)
			p.Assert(t, "queries", metric.QueryOutcomeLabelKey, metric.SuccessQueryOutcome, metric.QueryTypeLabelKey, "insert")(xmetricstest.Value(testCase.ExpectedSuccessCount))
			p.Assert(t, "queries", metric.QueryOutcomeLabelKey, metric.FailQueryOutcome, metric.QueryTypeLabelKey, "insert")(xmetricstest.Value(testCase.ExpectedFailCount))

			p.Assert(t, "dynamo", metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityReadOp)(xmetricstest.Value(testCase.ExpectedReadCapacity))
			p.Assert(t, "dynamo", metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityWriteOp)(xmetricstest.Value(testCase.ExpectedWriteCapacity))

			//TODO: due to limitations in xmetricstest, we can't explore the values observed in a histogram for the QueryDurationSeconds
		})
	}
}

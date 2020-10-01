package dynamodb

import (
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
)

type instrumentingService struct {
	service
	measures measuresUpdater
	now      func() time.Time
}

type measuresUpdater interface {
	Update(*measureUpdateRequest)
}

type measureUpdateRequest struct {
	err              error
	consumedCapacity *dynamodb.ConsumedCapacity
	queryType        string
	start            time.Time
}

func (s *instrumentingService) Push(key model.Key, item store.OwnableItem) (consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.measures.Update(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.PushQueryType,
			start:            start,
		})
	}(s.now())

	consumedCapacity, err = s.service.Push(key, item)
	return
}

func (s *instrumentingService) Get(key model.Key) (item store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.measures.Update(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.GetQueryType,
			start:            start,
		})
	}(s.now())

	item, consumedCapacity, err = s.service.Get(key)
	return
}

func (s *instrumentingService) Delete(key model.Key) (item store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.measures.Update(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.DeleteQueryType,
			start:            start,
		})
	}(s.now())

	item, consumedCapacity, err = s.service.Delete(key)
	return
}

func (s *instrumentingService) GetAll(bucket string) (items map[string]store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.measures.Update(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.GetAllQueryType,
			start:            start,
		})
	}(s.now())

	items, consumedCapacity, err = s.service.GetAll(bucket)
	return
}

type dynamoMeasuresUpdater struct {
	measures *metric.Measures
}

func (m *dynamoMeasuresUpdater) Update(request *measureUpdateRequest) {
	queryDurationSeconds := time.Since(request.start).Seconds()
	m.measures.QueryDurationSeconds.With(metric.QueryTypeLabelKey, request.queryType).Observe(queryDurationSeconds)

	m.updateDynamoCapacityMeasures(request.consumedCapacity, request.queryType)
	m.updateQueryMeasures(request.err, request.queryType)
}

func (m *dynamoMeasuresUpdater) updateDynamoCapacityMeasures(consumedCapacity *dynamodb.ConsumedCapacity, queryType string) {
	if consumedCapacity == nil {
		return
	}

	if consumedCapacity.ReadCapacityUnits != nil {
		m.measures.DynamodbConsumedCapacity.With(metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityReadOp).Add(*consumedCapacity.ReadCapacityUnits)
	}

	if consumedCapacity.WriteCapacityUnits != nil {
		m.measures.DynamodbConsumedCapacity.With(metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityWriteOp).Add(*consumedCapacity.WriteCapacityUnits)
	}
}

func (m *dynamoMeasuresUpdater) updateQueryMeasures(err error, queryType string) {
	if err != nil {
		m.measures.Queries.With(metric.QueryOutcomeLabelKey, metric.FailQueryOutcome, metric.QueryTypeLabelKey, queryType).Add(1)
	} else {
		m.measures.Queries.With(metric.QueryOutcomeLabelKey, metric.SuccessQueryOutcome, metric.QueryTypeLabelKey, queryType).Add(1.0)
	}
}

func newInstrumentingService(updater measuresUpdater, s service, now func() time.Time) service {
	return &instrumentingService{
		measures: updater,
		service:  s,
		now:      now,
	}
}

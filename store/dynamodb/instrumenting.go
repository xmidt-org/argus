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
	measures metric.Measures
}
type measureUpdateRequest struct {
	err              error
	consumedCapacity *dynamodb.ConsumedCapacity
	queryType        string
	start            time.Time
}

func (s *instrumentingService) Push(key model.Key, item store.OwnableItem) (consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.updateMeasures(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.PushQueryType,
			start:            start,
		})
	}(time.Now())

	consumedCapacity, err = s.service.Push(key, item)
	return
}

func (s *instrumentingService) Get(key model.Key) (item store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.updateMeasures(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.GetQueryType,
			start:            start,
		})
	}(time.Now())

	item, consumedCapacity, err = s.service.Get(key)
	return
}

func (s *instrumentingService) Delete(key model.Key) (item store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.updateMeasures(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.DeleteQueryType,
			start:            start,
		})
	}(time.Now())

	item, consumedCapacity, err = s.service.Delete(key)
	return
}

func (s *instrumentingService) GetAll(bucket string) (items map[string]store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func(start time.Time) {
		s.updateMeasures(&measureUpdateRequest{
			err:              err,
			consumedCapacity: consumedCapacity,
			queryType:        metric.GetAllQueryType,
			start:            start,
		})
	}(time.Now())

	items, consumedCapacity, err = s.service.GetAll(bucket)
	return
}

func (s *instrumentingService) updateMeasures(request *measureUpdateRequest) {
	queryDurationSeconds := time.Since(request.start).Seconds()
	s.measures.QueryDurationSeconds.With(metric.QueryTypeLabelKey, request.queryType).Observe(queryDurationSeconds)

	s.updateDynamoCapacityMeasures(request.consumedCapacity, request.queryType)
	s.updateQueryMeasures(request.err, request.queryType)
}

func (s *instrumentingService) updateDynamoCapacityMeasures(consumedCapacity *dynamodb.ConsumedCapacity, queryType string) {
	if consumedCapacity == nil {
		return
	}

	if consumedCapacity.ReadCapacityUnits != nil {
		s.measures.DynamodbConsumedCapacity.With(metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityReadOp).Add(1)
	}

	if consumedCapacity.WriteCapacityUnits != nil {
		s.measures.DynamodbConsumedCapacity.With(metric.DynamoCapacityOpLabelKey, metric.DynamoCapacityWriteOp).Add(1)
	}
}

func (s *instrumentingService) updateQueryMeasures(err error, queryType string) {
	if err != nil {
		s.measures.Queries.With(metric.QueryOutcomeLabelKey, metric.FailQueryOutcome, metric.QueryTypeLabelKey, queryType).Add(1)
	} else {
		s.measures.Queries.With(metric.QueryOutcomeLabelKey, metric.SuccessQueryOutcome, metric.QueryTypeLabelKey, queryType).Add(1)
	}
}

func newInstrumentingService(measures metric.Measures, s service) service {
	return &instrumentingService{measures: measures, service: s}
}

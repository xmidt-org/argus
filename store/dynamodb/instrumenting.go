// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package dynamodb

import (
	"errors"
	"time"

	awsv2dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/prometheus/client_golang/prometheus"
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
	consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
	queryType        string
	start            time.Time
}

func (s *instrumentingService) Push(key model.Key, item store.OwnableItem) (*awsv2dynamodbTypes.ConsumedCapacity, error) {
	start := s.now()
	consumedCapacity, err := s.service.Push(key, item)

	s.measures.Update(&measureUpdateRequest{
		err:              err,
		consumedCapacity: consumedCapacity,
		queryType:        metric.PushQueryType,
		start:            start,
	})

	return consumedCapacity, err
}

func (s *instrumentingService) Get(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	start := s.now()
	item, consumedCapacity, err := s.service.Get(key)

	s.measures.Update(&measureUpdateRequest{
		err:              err,
		consumedCapacity: consumedCapacity,
		queryType:        metric.GetQueryType,
		start:            start,
	})

	return item, consumedCapacity, err
}

func (s *instrumentingService) Delete(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	start := s.now()
	item, consumedCapacity, err := s.service.Delete(key)

	s.measures.Update(&measureUpdateRequest{
		err:              err,
		consumedCapacity: consumedCapacity,
		queryType:        metric.DeleteQueryType,
		start:            start,
	})

	return item, consumedCapacity, err
}

func (s *instrumentingService) GetAll(bucket string) (map[string]store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	start := s.now()
	items, consumedCapacity, err := s.service.GetAll(bucket)

	s.measures.Update(&measureUpdateRequest{
		err:              err,
		consumedCapacity: consumedCapacity,
		queryType:        metric.GetAllQueryType,
		start:            start,
	})

	return items, consumedCapacity, err
}

type dynamoMeasuresUpdater struct {
	measures *metric.Measures
}

func (m *dynamoMeasuresUpdater) Update(request *measureUpdateRequest) {
	queryDurationSeconds := time.Since(request.start).Seconds()
	m.measures.QueryDurationSeconds.With(prometheus.Labels{
		metric.QueryTypeLabelKey: request.queryType,
	}).Observe(queryDurationSeconds)

	m.updateDynamoCapacityMeasures(request.consumedCapacity, request.queryType)
	m.updateQueryMeasures(request.err, request.queryType)
}

// For some reason, the go-aws sdk does not return the consumed read and write units separately.
// Here https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/ProvisionedThroughput.html we found
// that provisioned tables consume read units for GetItem and Query operations while write units for Putitem and DeleteItem.
func (m *dynamoMeasuresUpdater) updateDynamoCapacityMeasures(consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, queryType string) {
	if consumedCapacity == nil || consumedCapacity.CapacityUnits == nil {
		return
	}

	capacityOp := metric.DynamoCapacityReadOp
	if queryType == metric.PushQueryType || queryType == metric.DeleteQueryType {
		capacityOp = metric.DynamoCapacityWriteOp
	}

	m.measures.DynamodbConsumedCapacity.With(prometheus.Labels{
		metric.QueryTypeLabelKey:        queryType,
		metric.DynamoCapacityOpLabelKey: capacityOp,
	}).Add(*consumedCapacity.CapacityUnits)
}

func (m *dynamoMeasuresUpdater) updateQueryMeasures(err error, queryType string) {
	if err != nil && !errors.Is(err, store.ErrItemNotFound) {
		m.measures.Queries.With(prometheus.Labels{
			metric.QueryOutcomeLabelKey: metric.FailQueryOutcome,
			metric.QueryTypeLabelKey:    queryType,
		}).Add(1)
	} else {
		m.measures.Queries.With(prometheus.Labels{
			metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
			metric.QueryTypeLabelKey:    queryType,
		}).Add(1.0)
	}
}

func newInstrumentingService(updater measuresUpdater, s service, now func() time.Time) service {
	return &instrumentingService{
		measures: updater,
		service:  s,
		now:      now,
	}
}

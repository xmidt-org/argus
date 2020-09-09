package dynamodb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
)

type instrumentingService struct {
	service
	measures metric.Measures
}

func (s *instrumentingService) updateCapacity(consumedCapacity *dynamodb.ConsumedCapacity, action string) {
	if consumedCapacity == nil {
		return
	}

	if consumedCapacity.CapacityUnits != nil {
		s.measures.CapacityUnitConsumedCount.With(store.TypeLabel, action).Add(*consumedCapacity.CapacityUnits)
	}

	if consumedCapacity.ReadCapacityUnits != nil {
		s.measures.ReadCapacityUnitConsumedCount.With(store.TypeLabel, action).Add(*consumedCapacity.ReadCapacityUnits)
	}

	if consumedCapacity.WriteCapacityUnits != nil {
		s.measures.WriteCapacityUnitConsumedCount.With(store.TypeLabel, action).Add(*consumedCapacity.WriteCapacityUnits)
	}
}

func (s *instrumentingService) updateQueryResult(err error, action string) {
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, action).Add(1.0)
	} else {
		s.measures.SQLQuerySuccessCount.With(store.TypeLabel, action).Add(1.0)
	}
}

func (s *instrumentingService) Push(key model.Key, item store.OwnableItem) (consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func() {
		s.updateCapacity(consumedCapacity, store.InsertType)
		s.updateQueryResult(err, store.InsertType)
	}()
	consumedCapacity, err = s.service.Push(key, item)
	return
}

func (s *instrumentingService) Get(key model.Key) (item store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func() {
		s.updateCapacity(consumedCapacity, store.ReadType)
		s.updateQueryResult(err, store.ReadType)
	}()
	item, consumedCapacity, err = s.service.Get(key)
	return
}

func (s *instrumentingService) Delete(key model.Key) (item store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func() {
		s.updateCapacity(consumedCapacity, store.DeleteType)
		s.updateQueryResult(err, store.DeleteType)
	}()
	item, consumedCapacity, err = s.service.Delete(key)
	return
}

func (s *instrumentingService) GetAll(bucket string) (items map[string]store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func() {
		s.updateCapacity(consumedCapacity, store.ReadType)
		s.updateQueryResult(err, store.ReadType)
	}()
	items, consumedCapacity, err = s.service.GetAll(bucket)
	return
}

func newInstrumentingService(measures metric.Measures, s service) service {
	return &instrumentingService{measures: measures, service: s}
}

package dynamodb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/store"
)

type loggingService struct {
	service
	debugLogger log.Logger
}

func newLoggingService(logger log.Logger, s service) service {
	return &loggingService{service: s, debugLogger: log.WithPrefix(logger, level.Key(), level.DebugValue())}
}

func (s *loggingService) GetAll(bucket string) (items map[string]store.OwnableItem, consumedCapacity *dynamodb.ConsumedCapacity, err error) {
	defer func() {
		s.debugLogger.Log("itemsSize", len(items), "err", err, "bucket", bucket)
	}()
	items, consumedCapacity, err = s.service.GetAll(bucket)
	return
}

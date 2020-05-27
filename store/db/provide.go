package db

import (
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/cassandra"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/argus/store/dynamodb"
	"github.com/xmidt-org/argus/store/inmem"
	"github.com/xmidt-org/themis/config"
	"go.uber.org/fx"
)

const DynamoDB = "dynamo"

func Provide(unmarshaller config.Unmarshaller, measures metric.Measures, lc fx.Lifecycle, logger log.Logger) (store.S, error) {
	if unmarshaller.IsSet(dynamodb.DynamoDB) {
		return dynamodb.ProvideDynamodDB(unmarshaller, measures, logger)
	}
	if unmarshaller.IsSet(cassandra.Yugabyte) {
		return cassandra.ProvideCassandra(unmarshaller, measures, lc, logger)
	}
	return inmem.ProvideInMem(), nil
}

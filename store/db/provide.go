// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/cassandra"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/argus/store/dynamodb"
	"github.com/xmidt-org/argus/store/inmem"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const DynamoDB = "dynamo"

type Configs struct {
	Dynamo   *dynamodb.Config
	Yugabyte *cassandra.Config
}

type SetupIn struct {
	fx.In
	Configs  Configs
	Measures metric.Measures
	LC       fx.Lifecycle
	Logger   *zap.Logger
}

func Provide() fx.Option {
	return fx.Options(
		fx.Provide(
			SetupStore,
		),
	)
}

func SetupStore(in SetupIn) (store.S, error) {
	if in.Configs.Dynamo != nil {
		in.Logger.Info("using dynamodb store implementation")
		return dynamodb.NewDynamoDB(*in.Configs.Dynamo, in.Measures)
	}
	if in.Configs.Yugabyte != nil {
		in.Logger.Info("using yugabyte store implementation")
		return cassandra.NewCassandra(*in.Configs.Yugabyte, in.Measures, in.LC,
			in.Logger)
	}
	in.Logger.Info("using in memory store implementation")
	return inmem.NewInMem(), nil
}

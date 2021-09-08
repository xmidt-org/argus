/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package db

import (
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/cassandra"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/argus/store/dynamodb"
	"github.com/xmidt-org/argus/store/inmem"
	"github.com/xmidt-org/arrange"
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
			arrange.UnmarshalKey("store", Configs{}),
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

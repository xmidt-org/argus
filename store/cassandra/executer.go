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

package cassandra

import (
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/log"
	"github.com/gocql/gocql"
	"github.com/hailocab/go-hostpool"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/webpa-common/logging"
)

type dbStore interface {
	store.S
	Close()
	Ping() error
}

var (
	noDataResponse = errors.New("no data from query")
	serverClosed   = errors.New("server is closed")
)

type cassandraExecutor struct {
	session *gocql.Session
	logger  log.Logger
}

func connect(clusterConfig *gocql.ClusterConfig, logger log.Logger) (dbStore, error) {
	clusterConfig.PoolConfig.HostSelectionPolicy = gocql.HostPoolHostPolicy(hostpool.New(nil))
	session, err := clusterConfig.CreateSession()
	if err != nil {
		return nil, err
	}

	return &cassandraExecutor{session: session, logger: logger}, nil
}

func (s *cassandraExecutor) Push(key model.Key, item store.OwnableItem) error {
	data, err := json.Marshal(&item)
	if err != nil {
		return err
	}

	return s.session.Query("INSERT INTO gifnoc (bucket, id, data) VALUES (?,?,?) USING TTL ?", key.Bucket, key.ID, data, item.TTL).Exec()
}

func (s *cassandraExecutor) Get(key model.Key) (store.OwnableItem, error) {
	var (
		data []byte
		ttl  int64
	)
	iter := s.session.Query("SELECT data, ttl(data) from gifnoc WHERE bucket = ? AND id = ?", key.Bucket, key.ID).Iter()
	defer func() {
		err := iter.Close()
		if err != nil {
			logging.Error(s.logger).Log(logging.MessageKey(), "failed to close iter ", "bucket", key.Bucket, "id", key.ID)
		}
	}()
	for iter.Scan(&data, &ttl) {
		item := store.OwnableItem{}
		err := json.Unmarshal(data, &item)
		item.TTL = ttl
		return item, err
	}
	return store.OwnableItem{}, noDataResponse
}

func (s *cassandraExecutor) Delete(key model.Key) (store.OwnableItem, error) {
	item, err := s.Get(key)
	if err != nil {
		return item, err
	}
	err = s.session.Query("DELETE from gifnoc WHERE bucket = ? AND id = ?", key.Bucket, key.ID).Exec()
	return item, err
}

func (s *cassandraExecutor) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	result := map[string]store.OwnableItem{}
	var (
		key  string
		data []byte
		ttl  int64
	)
	iter := s.session.Query("SELECT id, data, ttl(data) from gifnoc WHERE bucket = ?", bucket).Iter()
	for iter.Scan(&key, &data, &ttl) {
		item := store.OwnableItem{}
		err := json.Unmarshal(data, &item)
		if err != nil {
			logging.Error(s.logger).Log(logging.MessageKey(), "failed to unmarshal data", "bucket", bucket, "id", key)
			data = []byte{}
			key = ""
			continue
		}
		item.TTL = ttl
		result[key] = item
	}
	err := iter.Close()
	return result, err
}

func (s *cassandraExecutor) Close() {
	s.session.Close()
}

func (s *cassandraExecutor) Ping() error {
	if s.session.Closed() {
		return serverClosed
	}
	return nil
}

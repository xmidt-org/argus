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

func (s *cassandraExecutor) Push(key store.Key, item store.Item) error {
	data, err := json.Marshal(&item)
	if err != nil {
		return err
	}

	return s.session.Query("INSERT INTO config (bucket, id, data) VALUES (?,?,?)", key.Bucket, key.ID, data).Exec()
}

func (s *cassandraExecutor) Get(key store.Key) (store.Item, error) {
	var data []byte
	iter := s.session.Query("SELECT data from config WHERE bucket = ? AND id = ?", key.Bucket, key.ID).Iter()
	defer func() {
		err := iter.Close()
		if err != nil {
			logging.Error(s.logger).Log(logging.MessageKey(), "failed to close iter ", "bucket", key.Bucket, "id", key.ID)
		}
	}()
	for iter.Scan(&data) {
		item := store.Item{}
		err := json.Unmarshal(data, &item)
		return item, err
	}
	return store.Item{}, noDataResponse
}

func (s *cassandraExecutor) Delete(key store.Key) (store.Item, error) {
	var data []byte
	iter := s.session.Query("DELETE from config WHERE bucket = ? AND id = ?", key.Bucket, key.ID).Iter()
	defer func() {
		err := iter.Close()
		if err != nil {
			logging.Error(s.logger).Log(logging.MessageKey(), "failed to close iter ", "bucket", key.Bucket, "id", key.ID)
		}
	}()
	for iter.Scan(&data) {
		item := store.Item{}
		err := json.Unmarshal(data, &item)
		return item, err
	}
	return store.Item{}, noDataResponse
}

func (s *cassandraExecutor) GetAll(bucket string) (map[string]store.Item, error) {
	result := map[string]store.Item{}
	var (
		key  string
		data []byte
	)
	iter := s.session.Query("SELECT id, data from config WHERE bucket = ?", bucket).Iter()
	for iter.Scan(&key, &data) {
		item := store.Item{}
		err := json.Unmarshal(data, &item)
		if err != nil {
			logging.Error(s.logger).Log(logging.MessageKey(), "failed to unmarshal data", "bucket", bucket, "id", key)
			data = []byte{}
			key = ""
			continue
		}
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

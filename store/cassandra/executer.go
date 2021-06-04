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
	"fmt"

	"github.com/gocql/gocql"
	"github.com/hailocab/go-hostpool"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

type dbStore interface {
	store.S
	Close()
	Ping() error
}

var serverClosed = errors.New("server is closed")

type cassandraExecutor struct {
	session *gocql.Session
}

func connect(clusterConfig *gocql.ClusterConfig) (dbStore, error) {
	clusterConfig.PoolConfig.HostSelectionPolicy = gocql.HostPoolHostPolicy(hostpool.New(nil))
	session, err := clusterConfig.CreateSession()
	if err != nil {
		return nil, err
	}

	return &cassandraExecutor{session: session}, nil
}

func (s *cassandraExecutor) Push(key model.Key, item store.OwnableItem) error {
	data, err := json.Marshal(&item)
	if err != nil {
		return store.ItemOperationError{Err: fmt.Errorf("%w: %v", store.ErrJSONEncode, err), Key: key, Operation: "push"}
	}
	err = s.session.Query("INSERT INTO gifnoc (bucket, id, data) VALUES (?,?,?) USING TTL ?", key.Bucket, key.ID, data, item.TTL).Exec()
	if err != nil {
		return store.ItemOperationError{Err: fmt.Errorf("%w: %v", store.ErrQueryExecution, err), Key: key, Operation: "push"}
	}
	return nil
}

func (s *cassandraExecutor) Get(key model.Key) (store.OwnableItem, error) {
	var (
		data []byte
		ttl  int64
	)
	iter := s.session.Query("SELECT data, ttl(data) from gifnoc WHERE bucket = ? AND id = ?", key.Bucket, key.ID).Iter()
	ok := iter.Scan(&data, &ttl)
	err := iter.Close()
	if !ok {
		if err != nil {
			return store.OwnableItem{}, store.ItemOperationError{Err: fmt.Errorf("%w: %v", store.ErrQueryExecution, err), Key: key, Operation: "get"}
		}
		return store.OwnableItem{}, store.ItemOperationError{Err: fmt.Errorf("%w: %v", store.ErrItemNotFound, err), Key: key, Operation: "get"}
	}
	item := store.OwnableItem{}
	err = json.Unmarshal(data, &item)
	if err != nil {
		return store.OwnableItem{}, store.ItemOperationError{Err: fmt.Errorf("%w: %v", store.ErrJSONDecode, err), Key: key, Operation: "get"}
	}
	item.TTL = &ttl
	return item, nil
}

func (s *cassandraExecutor) Delete(key model.Key) (store.OwnableItem, error) {
	item, err := s.Get(key)
	if err != nil {
		return item, store.ItemOperationError{Err: err, Key: key, Operation: "delete"}
	}
	err = s.session.Query("DELETE from gifnoc WHERE bucket = ? AND id = ?", key.Bucket, key.ID).Exec()
	if err != nil {
		return store.OwnableItem{}, store.ItemOperationError{Err: fmt.Errorf("%w: %v", store.ErrQueryExecution, err), Key: key, Operation: "delete"}
	}
	return item, nil
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
			iter.Close()
			return result, store.GetAllItemsOperationErr{Err: store.ErrJSONDecode, Bucket: bucket}
		}
		item.TTL = &ttl
		result[key] = item
	}
	err := iter.Close()
	if err != nil {
		return result, store.GetAllItemsOperationErr{Err: store.ErrQueryExecution, Bucket: bucket}
	}
	return result, nil
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

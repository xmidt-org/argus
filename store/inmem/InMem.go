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

package inmem

import (
	"sync"

	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

type InMem struct {
	data map[string]map[string]store.OwnableItem
	lock sync.RWMutex
}

func NewInMem() store.S {
	return &InMem{
		data: map[string]map[string]store.OwnableItem{},
	}
}

func (i *InMem) Push(key model.Key, item store.OwnableItem) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.data[key.Bucket] == nil {
		i.data[key.Bucket] = map[string]store.OwnableItem{}
	}
	i.data[key.Bucket][key.ID] = item
	return nil
}

func (i *InMem) Get(key model.Key) (store.OwnableItem, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	bucket, ok := i.data[key.Bucket]
	if !ok {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "get"})
	}
	item, ok := bucket[key.ID]
	if !ok {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "get"})
	}
	return item, nil
}

func (i *InMem) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	i.lock.RLock()
	defer i.lock.RUnlock()
	items := i.data[bucket]
	if items == nil {
		items = map[string]store.OwnableItem{}
	}
	return items, nil
}

func (i *InMem) Delete(key model.Key) (store.OwnableItem, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	bucket := i.data[key.Bucket]
	if bucket == nil {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "delete"})
	}
	item, ok := bucket[key.ID]
	if !ok {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "delete"})
	}
	delete(bucket, key.ID)
	return item, nil
}

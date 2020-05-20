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
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"sync"
)

type InMem struct {
	data map[string]map[string]store.InternalItem
	lock sync.RWMutex
}

func ProvideInMem() store.S {
	return &InMem{
		data: map[string]map[string]store.InternalItem{},
	}
}

func (i *InMem) Push(key model.Key, item store.InternalItem) error {
	i.lock.Lock()
	if _, ok := i.data[key.Bucket]; !ok {
		i.data[key.Bucket] = map[string]store.InternalItem{
			key.ID: item,
		}
	} else {
		i.data[key.Bucket][key.ID] = item
	}
	i.lock.Unlock()
	return nil
}

func (i *InMem) Get(key model.Key) (store.InternalItem, error) {
	var (
		item store.InternalItem
		err  error
	)
	i.lock.RLock()
	if _, ok := i.data[key.Bucket]; !ok {
		err = store.KeyNotFoundError{Key: key}
	} else {
		if value, ok := i.data[key.Bucket][key.ID]; !ok {
			err = store.KeyNotFoundError{Key: key}
		} else {
			item = value
		}
	}
	i.lock.RUnlock()
	return item, err
}

func (i *InMem) GetAll(bucket string) (map[string]store.InternalItem, error) {
	var (
		items map[string]store.InternalItem
		err   error
	)

	i.lock.RLock()
	if item, ok := i.data[bucket]; ok {
		items = item
	} else {
		err = store.KeyNotFoundError{Key: model.Key{
			Bucket: bucket,
			ID:     "",
		}}
	}
	i.lock.RUnlock()
	return items, err
}

func (i *InMem) Delete(key model.Key) (store.InternalItem, error) {
	var (
		item store.InternalItem
		err  error
	)
	i.lock.Lock()
	if _, ok := i.data[key.Bucket]; !ok {
		err = store.KeyNotFoundError{Key: key}
	} else {
		if value, ok := i.data[key.Bucket][key.ID]; !ok {
			err = store.KeyNotFoundError{Key: key}
		} else {
			item = value
			delete(i.data[key.Bucket], key.ID)
		}
	}
	i.lock.Unlock()
	return item, err
}

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
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/httpaux"
)

// internal errors to be wrapped with more information for logging purposes
var (
	errItemNotFound   = errors.New("Item at resource path not found")
	errBucketNotFound = errors.New("Bucket path not found")
)

// sanitized HTTP errors
var (
	errHTTPItemNotFound   = httpaux.Error{Err: errors.New("Item not found"), Code: http.StatusNotFound}
	errHTTPBucketNotFound = httpaux.Error{Err: errors.New("Bucket not found"), Code: http.StatusNotFound}
	errHTTPOpFailed       = httpaux.Error{Err: errors.New("InMem operation failed"), Code: http.StatusInternalServerError}
)

type InMem struct {
	data map[string]map[string]store.OwnableItem
	lock sync.RWMutex
}

func ProvideInMem() store.S {
	return &InMem{
		data: map[string]map[string]store.OwnableItem{},
	}
}

func (i *InMem) Push(key model.Key, item store.OwnableItem) error {
	i.lock.Lock()
	if _, ok := i.data[key.Bucket]; !ok {
		i.data[key.Bucket] = map[string]store.OwnableItem{
			key.ID: item,
		}
	} else {
		i.data[key.Bucket][key.ID] = item
	}
	i.lock.Unlock()
	return nil
}

func (i *InMem) Get(key model.Key) (store.OwnableItem, error) {
	var (
		item store.OwnableItem
		err  error
	)
	i.lock.RLock()
	defer i.lock.RUnlock()
	if _, ok := i.data[key.Bucket]; !ok {
		err = fmt.Errorf("%w: %v", errBucketNotFound, key.Bucket)
	} else {
		if value, ok := i.data[key.Bucket][key.ID]; !ok {
			err = fmt.Errorf("%w: %v/%v", errItemNotFound, key.Bucket, key.ID)
		} else {
			item = value
		}
	}
	return item, sanitizeError(err)
}

func (i *InMem) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	var (
		items map[string]store.OwnableItem
		err   error
	)

	i.lock.RLock()
	if item, ok := i.data[bucket]; ok {
		items = item
	} else {
		err = fmt.Errorf("%w: %v", errBucketNotFound, bucket)
	}
	i.lock.RUnlock()
	return items, sanitizeError(err)
}

func (i *InMem) Delete(key model.Key) (store.OwnableItem, error) {
	var (
		item store.OwnableItem
		err  error
	)
	i.lock.Lock()
	if _, ok := i.data[key.Bucket]; !ok {
		err = fmt.Errorf("%w: %v", errBucketNotFound, key.Bucket)
	} else {
		if value, ok := i.data[key.Bucket][key.ID]; !ok {
			err = fmt.Errorf("%w: %v/%v", errItemNotFound, key.Bucket, key.ID)
		} else {
			item = value
			delete(i.data[key.Bucket], key.ID)
		}
	}
	i.lock.Unlock()
	return item, sanitizeError(err)
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	var errHTTP = errHTTPOpFailed

	switch {
	case errors.Is(err, errItemNotFound):
		errHTTP = errHTTPItemNotFound
	case errors.Is(err, errBucketNotFound):
		errHTTP = errHTTPBucketNotFound
	}
	return store.SanitizedError{Err: err, ErrHTTP: errHTTP}
}

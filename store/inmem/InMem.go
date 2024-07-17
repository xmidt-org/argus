// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package inmem

import (
	"sync"
	"time"

	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/argus/store"
)

type expireableItem struct {
	store.OwnableItem
	expiration *time.Time
}

type InMem struct {
	data map[string]map[string]expireableItem
	lock sync.Mutex
	now  func() time.Time
}

func NewInMem() store.S {
	return &InMem{
		data: map[string]map[string]expireableItem{},
		now:  time.Now,
	}
}

func (i *InMem) Push(key model.Key, item store.OwnableItem) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.data[key.Bucket] == nil {
		i.data[key.Bucket] = map[string]expireableItem{}
	}
	storingItem := expireableItem{OwnableItem: item}
	if item.TTL != nil {
		ttlSeconds := time.Duration(*item.TTL)
		expiration := i.now().Add(time.Second * ttlSeconds)
		storingItem.expiration = &expiration
	}
	i.data[key.Bucket][key.ID] = storingItem
	return nil
}

// hasExpired returns true if the given item has expired and false otherwise.
// For an unexpired item with an expiration date, the current TTL is updated.
// Note: expired items are automatically removed from the internal map.
func (i *InMem) hasExpired(item *expireableItem, bucket map[string]expireableItem, bucketName, ID string) bool {
	if item.expiration == nil {
		return false
	}
	secondsBeforeExpiry := int64(item.expiration.Sub(i.now()).Seconds())
	if secondsBeforeExpiry <= 0 {
		i.deleteItem(bucketName, ID, bucket)
		return true
	}
	item.TTL = &secondsBeforeExpiry
	return false
}

func (i *InMem) Get(key model.Key) (store.OwnableItem, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	bucket, ok := i.data[key.Bucket]
	if !ok {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "get"})
	}
	item, ok := bucket[key.ID]
	if !ok {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "get"})
	}

	if i.hasExpired(&item, bucket, key.Bucket, key.ID) {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "get"})
	}

	return item.OwnableItem, nil
}

func (i *InMem) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	items := i.data[bucket]
	result := make(map[string]store.OwnableItem)
	for idx := range items {
		item := items[idx]
		if !i.hasExpired(&item, items, bucket, item.ID) {
			result[item.ID] = item.OwnableItem
		}
	}
	return result, nil
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

	if i.hasExpired(&item, bucket, key.Bucket, key.ID) {
		return store.OwnableItem{}, store.SanitizeError(store.ItemOperationError{Err: store.ErrItemNotFound, Key: key, Operation: "delete"})
	}
	i.deleteItem(key.Bucket, key.ID, bucket)
	return item.OwnableItem, nil
}

func (i *InMem) deleteItem(bucketName string, itemID string, bucket map[string]expireableItem) {
	delete(bucket, itemID)
	if len(bucket) == 0 {
		delete(i.data, bucketName)
	}
}

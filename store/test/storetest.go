// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

var testInt int64 = 3
var GenericTestKeyPair = store.KeyItemPairRequest{
	Key: model.Key{
		Bucket: "world",
		ID:     "earth",
	},
	OwnableItem: store.OwnableItem{
		Item: model.Item{
			Data: map[string]interface{}{
				"year":  float64(1967),
				"words": []interface{}{"What", "a", "Wonderful", "World"},
			},
			TTL: &testInt,
		},
		Owner: "Louis Armstrong",
	},
}

// StoreTest validates that a given store implementation works.
func StoreTest(t *testing.T, s store.S, storeTiming time.Duration) {
	t.Helper()
	assert := assert.New(t)

	t.Log("Basic Test")
	err := s.Push(GenericTestKeyPair.Key, GenericTestKeyPair.OwnableItem)
	require.NoError(t, err)
	retVal, err := s.Get(GenericTestKeyPair.Key)
	require.NoError(t, err)
	assert.Equal(GenericTestKeyPair.OwnableItem, retVal)

	items, err := s.GetAll("world")
	require.NoError(t, err)
	assert.Equal(map[string]store.OwnableItem{"earth": GenericTestKeyPair.OwnableItem}, items)

	retVal, err = s.Delete(GenericTestKeyPair.Key)
	require.NoError(t, err)
	assert.Equal(GenericTestKeyPair.OwnableItem, retVal)

	items, err = s.GetAll("world")
	require.NoError(t, err)
	assert.Equal(map[string]store.OwnableItem{}, items)

	if storeTiming > 0 {
		t.Log("staring duration tests")
		err := s.Push(GenericTestKeyPair.Key, GenericTestKeyPair.OwnableItem)
		require.NoError(t, err)
		retVal, err := s.Get(GenericTestKeyPair.Key)
		require.NoError(t, err)
		assert.Equal(GenericTestKeyPair.OwnableItem, retVal)
		time.Sleep(storeTiming + time.Second)
		retVal, err = s.Get(GenericTestKeyPair.Key)
		assert.Equal(store.OwnableItem{}, retVal)
		assert.Equal(store.KeyNotFoundError{Key: GenericTestKeyPair.Key}, err)
	}
}

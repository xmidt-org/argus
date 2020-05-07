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

package storetest

import (
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/store"
	"testing"
	"time"
)

var GenericTestKeyPair = store.KeyItemPair{
	Key: store.Key{
		Bucket: "world",
		ID:     "earth",
	},
	Item: store.Item{
		Attributes: nil,
		Value: map[string]interface{}{
			"year":  1967,
			"words": []string{"What", "a", "Wonderful", "World"},
		},
	},
}

func StoreTest(s store.S, storeTiming time.Duration, t *testing.T) {
	assert := assert.New(t)

	t.Log("Basic Test")
	err := s.Push(GenericTestKeyPair.Key, GenericTestKeyPair.Item)
	assert.NoError(err)
	retVal, err := s.Get(GenericTestKeyPair.Key)
	assert.NoError(err)
	assert.Equal(GenericTestKeyPair.Item, retVal)

	items, err := s.GetAll("world")
	assert.NoError(err)
	assert.Equal(map[string]store.Item{"earth": GenericTestKeyPair.Item}, items)

	retVal, err = s.Delete(GenericTestKeyPair.Key)
	assert.NoError(err)
	assert.Equal(GenericTestKeyPair.Item, retVal)

	items, err = s.GetAll("world")
	assert.NoError(err)
	assert.Equal(map[string]store.Item{}, items)

	if storeTiming > 0 {
		t.Log("staring duration tests")
		err := s.Push(GenericTestKeyPair.Key, GenericTestKeyPair.Item)
		assert.NoError(err)
		retVal, err := s.Get(GenericTestKeyPair.Key)
		assert.NoError(err)
		assert.Equal(GenericTestKeyPair.Item, retVal)
		time.Sleep(storeTiming + time.Second)
		retVal, err = s.Get(GenericTestKeyPair.Key)
		assert.Equal(store.Item{}, retVal)
		assert.Equal(store.KeyNotFoundError{Key: GenericTestKeyPair.Key}, err)
	}
}

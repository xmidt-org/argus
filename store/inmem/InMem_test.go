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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/test"
)

func TestInMemStore(t *testing.T) {
	s := ProvideInMem()
	test.StoreTest(s, 0, t)
}

type InMemTestSuite struct {
	suite.Suite
	BucketName         string
	ItemID             string
	Item               store.OwnableItem
	DataBucketNotFound map[string]map[string]store.OwnableItem
	DataBucketFound    map[string]map[string]store.OwnableItem
	DataItemNotFound   map[string]map[string]store.OwnableItem
	DataItemFound      map[string]map[string]store.OwnableItem
	DataItemsFound     map[string]map[string]store.OwnableItem
}

func (s *InMemTestSuite) SetupTest() {
	s.DataBucketNotFound = map[string]map[string]store.OwnableItem{}
	s.DataBucketFound = map[string]map[string]store.OwnableItem{
		s.BucketName: {},
	}
	s.DataItemNotFound = map[string]map[string]store.OwnableItem{
		s.BucketName: {
			"other-item-id": store.OwnableItem{},
		},
	}
	s.DataItemFound = map[string]map[string]store.OwnableItem{
		s.BucketName: {
			s.ItemID: s.Item,
		},
	}
}

func (s *InMemTestSuite) SetupSuite() {
	s.BucketName = "test-bucket-name"
	s.ItemID = "test-item-id"
	s.Item = store.OwnableItem{
		Owner: "test-owner",
		Item: model.Item{
			ID: s.ItemID,
			Data: map[string]interface{}{
				"k": "v",
			},
		},
	}
}

func (s *InMemTestSuite) TestPush() {
	var expectedData = map[string]map[string]store.OwnableItem{
		s.BucketName: {
			s.ItemID: s.Item,
		},
	}

	tcs := []struct {
		Description  string
		Data         map[string]map[string]store.OwnableItem
		ExpectedData map[string]map[string]store.OwnableItem
	}{
		{
			Description: "BucketNotFound",
			Data:        s.DataBucketNotFound,
		},
		{
			Description: "BucketFound",
			Data:        s.DataBucketFound,
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			storage := InMem{
				data: tc.Data,
			}
			err := storage.Push(model.Key{Bucket: s.BucketName, ID: s.ItemID}, s.Item)
			assert.Nil(err)
			assert.EqualValues(expectedData, storage.data)
		})
	}
}

func TestInMem(t *testing.T) {
	suite.Run(t, new(InMemTestSuite))
}

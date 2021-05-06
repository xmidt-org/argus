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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

type InMemTestSuite struct {
	suite.Suite
	BucketName         string
	ItemOneKey         model.Key
	ItemOneID          string
	ItemOne            store.OwnableItem
	ItemTwoID          string
	ItemTwo            store.OwnableItem
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
			s.ItemOneID: s.ItemOne,
		},
	}
	s.DataItemsFound = map[string]map[string]store.OwnableItem{
		s.BucketName: {
			s.ItemOneID: s.ItemOne,
			s.ItemTwoID: s.ItemTwo,
		},
	}
}

func (s *InMemTestSuite) SetupSuite() {
	s.BucketName = "test-bucket-name"
	s.ItemOneID = "test-item-id-1"
	s.ItemOne = store.OwnableItem{
		Owner: "test-owner-1",
		Item: model.Item{
			ID: s.ItemOneID,
			Data: map[string]interface{}{
				"k1": "v1",
			},
		},
	}
	s.ItemOneKey = model.Key{ID: s.ItemOneID, Bucket: s.BucketName}
	s.ItemTwoID = "test-item-id-2"
	s.ItemTwo = store.OwnableItem{
		Owner: "test-owner-2",
		Item: model.Item{
			ID: "test-item-id-2",
			Data: map[string]interface{}{
				"k": "v",
			},
		},
	}
}

func (s *InMemTestSuite) TestPush() {
	var expectedData = map[string]map[string]store.OwnableItem{
		s.BucketName: {
			s.ItemOneID: s.ItemOne,
		},
	}

	tcs := []struct {
		Description  string
		Data         map[string]map[string]store.OwnableItem
		ExpectedData map[string]map[string]store.OwnableItem
	}{
		{
			Description: "Create bucket",
			Data:        s.DataBucketNotFound,
		},
		{
			Description: "Push into existing bucket",
			Data:        s.DataBucketFound,
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			storage := InMem{
				data: tc.Data,
			}
			err := storage.Push(s.ItemOneKey, s.ItemOne)
			assert.Nil(err)
			assert.EqualValues(expectedData, storage.data)
		})
	}
}

func (s *InMemTestSuite) TestGet() {
	tcs := []struct {
		Description   string
		Data          map[string]map[string]store.OwnableItem
		ExpectedError error
	}{
		{
			Description:   "Bucket missing",
			Data:          s.DataBucketNotFound,
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description:   "Item missing",
			Data:          s.DataItemNotFound,
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description: "Item found",
			Data:        s.DataItemFound,
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			storage := InMem{data: tc.Data}
			actualItem, err := storage.Get(s.ItemOneKey)
			if tc.ExpectedError != nil {
				var sErr store.SanitizedError
				require.True(errors.As(err, &sErr), "Expected '%v' to be a store.SanitizedError", err)
				var iErr store.ItemOperationError
				require.True(errors.As(err, &iErr), "Expected '%v' to be a store.ItemOperationError", err)
				assert.Equal("get", iErr.Operation)
				assert.Equal(s.ItemOneKey, iErr.Key)
				assert.True(errors.Is(err, tc.ExpectedError), "Expected to find match for '%v' in error chain of '%v'", tc.ExpectedError, err)
			} else {
				assert.Nil(err)
				assert.Equal(s.ItemOne, actualItem)
			}
		})
	}
}

func (s *InMemTestSuite) TestGetAll() {
	tcs := []struct {
		Description   string
		Data          map[string]map[string]store.OwnableItem
		ExpectedItems map[string]store.OwnableItem
	}{
		{
			Description:   "Bucket missing",
			Data:          s.DataBucketNotFound,
			ExpectedItems: map[string]store.OwnableItem{},
		},
		{
			Description: "Items",
			Data:        s.DataItemsFound,
			ExpectedItems: map[string]store.OwnableItem{
				s.ItemOneID: s.ItemOne,
				s.ItemTwoID: s.ItemTwo,
			},
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			storage := InMem{data: tc.Data}
			items, err := storage.GetAll(s.BucketName)
			assert.Nil(err)
			assert.Equal(tc.ExpectedItems, items)
		})
	}
}

func (s *InMemTestSuite) TestDelete() {
	tcs := []struct {
		Description   string
		Data          map[string]map[string]store.OwnableItem
		ExpectedData  map[string]map[string]store.OwnableItem
		ExpectedError error
	}{
		{
			Description:   "Bucket missing",
			Data:          s.DataBucketNotFound,
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description:   "Item missing",
			Data:          s.DataItemNotFound,
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description: "Item found",
			Data:        s.DataItemFound,
			ExpectedData: map[string]map[string]store.OwnableItem{
				s.BucketName: {},
			},
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			storage := InMem{data: tc.Data}
			actualItem, err := storage.Delete(s.ItemOneKey)
			if tc.ExpectedError != nil {
				var sErr store.SanitizedError
				require.True(errors.As(err, &sErr), "Expected '%v' to be a store.SanitizedError", err)
				var iErr store.ItemOperationError
				require.True(errors.As(err, &iErr), "Expected '%v' to be a store.ItemOperationError", err)
				assert.Equal("delete", iErr.Operation)
				assert.Equal(s.ItemOneKey, iErr.Key)
				assert.True(errors.Is(err, tc.ExpectedError), "Expected to find match for '%v' in error chain of '%v'", tc.ExpectedError, err)
			} else {
				assert.Nil(err)
				assert.Equal(s.ItemOne, actualItem)
				assert.Equal(tc.ExpectedData, storage.data)
			}
		})
	}

}

func TestInMem(t *testing.T) {
	suite.Run(t, new(InMemTestSuite))
}

type InMemTestParallelSuite struct {
	suite.Suite
	Storage    *InMem
	BucketName string
	ItemOneKey model.Key
	ItemOneID  string
	ItemOne    store.OwnableItem
}

func (s *InMemTestParallelSuite) SetupSuite() {
	s.Storage = &InMem{
		data: map[string]map[string]store.OwnableItem{},
	}
	s.BucketName = "test-bucket-name"
	s.ItemOneID = "test-item-id-1"
	s.ItemOne = store.OwnableItem{
		Owner: "test-owner-1",
		Item: model.Item{
			ID: s.ItemOneID,
			Data: map[string]interface{}{
				"k1": "v1",
			},
		},
	}
	s.ItemOneKey = model.Key{ID: s.ItemOneID, Bucket: s.BucketName}
}

func (s *InMemTestParallelSuite) TestGet() {
	s.T().Parallel()
	s.Storage.Get(s.ItemOneKey)
}

func (s *InMemTestParallelSuite) TestPush() {
	s.T().Parallel()
	s.Storage.Push(s.ItemOneKey, s.ItemOne)
}

func (s *InMemTestParallelSuite) TestDelete() {
	s.T().Parallel()
	s.Storage.Delete(s.ItemOneKey)
}

func (s *InMemTestParallelSuite) TestGetAll() {
	s.T().Parallel()
	s.Storage.GetAll(s.BucketName)
}

func TestInMemParallel(t *testing.T) {
	suite.Run(t, new(InMemTestParallelSuite))
}

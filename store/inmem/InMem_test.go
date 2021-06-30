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
	"testing"
	"time"

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
	ItemOne            expireableItem
	ItemTwoID          string
	ItemTwo            expireableItem
	ItemThreeKey       model.Key
	ItemThreeID        string
	ItemThree          expireableItem
	DataBucketNotFound map[string]map[string]expireableItem
	DataBucketFound    map[string]map[string]expireableItem
	DataItemNotFound   map[string]map[string]expireableItem
	DataItemExpired    map[string]map[string]expireableItem
	DataItemFound      map[string]map[string]expireableItem
	DataItemsMixed     map[string]map[string]expireableItem
	Now                time.Time
	NowFunc            func() time.Time
}

func (s *InMemTestSuite) SetupTest() {
	s.DataBucketNotFound = map[string]map[string]expireableItem{}
	s.DataBucketFound = map[string]map[string]expireableItem{
		s.BucketName: {},
	}
	s.DataItemNotFound = map[string]map[string]expireableItem{
		s.BucketName: {
			"other-item-id": expireableItem{},
		},
	}
	s.DataItemExpired = map[string]map[string]expireableItem{
		s.BucketName: {
			s.ItemThreeID: s.ItemThree,
		},
	}
	s.DataItemFound = map[string]map[string]expireableItem{
		s.BucketName: {
			s.ItemOneID: s.ItemOne,
		},
	}
	s.DataItemsMixed = map[string]map[string]expireableItem{
		s.BucketName: {
			s.ItemOneID:   s.ItemOne,   // never expires
			s.ItemTwoID:   s.ItemTwo,   // expires in the future
			s.ItemThreeID: s.ItemThree, // already expired
		},
	}
}

func (s *InMemTestSuite) SetupSuite() {
	s.Now = time.Now()
	s.NowFunc = func() time.Time {
		return s.Now
	}
	s.BucketName = "test-bucket-name"
	s.ItemOneID = "test-item-id-1"
	s.ItemOne = expireableItem{
		OwnableItem: store.OwnableItem{
			Owner: "test-owner-1",
			Item: model.Item{
				ID: s.ItemOneID,
				Data: map[string]interface{}{
					"k1": "v1",
				},
			},
		},
	}
	s.ItemOneKey = model.Key{ID: s.ItemOneID, Bucket: s.BucketName}
	s.ItemTwoID = "test-item-id-2"
	s.ItemTwo = expireableItem{
		OwnableItem: store.OwnableItem{
			Owner: "test-owner-2",
			Item: model.Item{
				ID: s.ItemTwoID,
				Data: map[string]interface{}{
					"k": "v",
				},
			},
		},
		expiration: s.getItemTwoExpiration(),
	}
	s.ItemThreeID = "test-item-id-3"
	s.ItemThreeKey = model.Key{ID: s.ItemThreeID, Bucket: s.BucketName}
	s.ItemThree = expireableItem{
		OwnableItem: store.OwnableItem{
			Owner: "test-owner-3",
			Item: model.Item{
				ID: s.ItemThreeID,
				Data: map[string]interface{}{
					"cool": "story",
				},
			},
		},
		expiration: s.getItemThreeExpiration(),
	}
}

func (s *InMemTestSuite) getItemTwoExpiration() *time.Time {
	inAnHour := s.Now.Add(time.Hour)
	return &inAnHour
}

func (s *InMemTestSuite) getItemThreeExpiration() *time.Time {
	aMinAgo := s.Now.Add(-time.Minute)
	return &aMinAgo
}

func (s *InMemTestSuite) TestPush() {
	var expectedData = map[string]map[string]expireableItem{
		s.BucketName: {
			s.ItemOneID: s.ItemOne,
		},
	}

	tcs := []struct {
		Description  string
		Data         map[string]map[string]expireableItem
		ExpectedData map[string]map[string]expireableItem
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
			err := storage.Push(s.ItemOneKey, s.ItemOne.OwnableItem)
			assert.Nil(err)
			assert.EqualValues(expectedData, storage.data)
		})
	}
}

func (s *InMemTestSuite) TestGet() {
	tcs := []struct {
		Description        string
		OriginalState      map[string]map[string]expireableItem
		ExpectedFinalState map[string]map[string]expireableItem
		ExpectedError      error
		ItemKey            model.Key
		ExpectedItem       store.OwnableItem
	}{
		{
			Description:        "Bucket missing",
			OriginalState:      s.DataBucketNotFound,
			ExpectedFinalState: map[string]map[string]expireableItem{},
			ItemKey:            s.ItemOneKey,
			ExpectedError:      store.ErrItemNotFound,
		},
		{
			Description:   "Item missing",
			OriginalState: s.DataItemNotFound,
			ExpectedFinalState: map[string]map[string]expireableItem{
				s.BucketName: {
					"other-item-id": expireableItem{},
				},
			},
			ItemKey:       s.ItemOneKey,
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description:        "Item expired",
			OriginalState:      s.DataItemExpired,
			ExpectedFinalState: map[string]map[string]expireableItem{},
			ItemKey:            s.ItemThreeKey,
			ExpectedError:      store.ErrItemNotFound,
		},
		{
			Description:   "Item found",
			OriginalState: s.DataItemFound,
			ExpectedFinalState: map[string]map[string]expireableItem{
				s.BucketName: {
					s.ItemOneID: s.ItemOne,
				},
			},
			ItemKey:      s.ItemOneKey,
			ExpectedItem: s.ItemOne.OwnableItem,
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			storage := InMem{data: tc.OriginalState, now: s.NowFunc}
			actualItem, err := storage.Get(tc.ItemKey)
			if tc.ExpectedError != nil {
				var sErr store.SanitizedError
				require.True(errors.As(err, &sErr), "Expected '%v' to be a store.SanitizedError", err)
				var iErr store.ItemOperationError
				require.True(errors.As(err, &iErr), "Expected '%v' to be a store.ItemOperationError", err)
				assert.Equal("get", iErr.Operation)
				assert.Equal(tc.ItemKey, iErr.Key)
				assert.True(errors.Is(err, tc.ExpectedError), "Expected to find match for '%v' in error chain of '%v'", tc.ExpectedError, err)
			} else {
				assert.Nil(err)
				assert.Equal(tc.ExpectedItem, actualItem)
			}
		})
	}
}

func (s *InMemTestSuite) TestGetAll() {
	tcs := []struct {
		Description        string
		OriginalState      map[string]map[string]expireableItem
		ExpectedFinalState map[string]map[string]expireableItem
		ExpectedItems      map[string]store.OwnableItem
	}{
		{
			Description:        "Bucket missing",
			OriginalState:      s.DataBucketNotFound,
			ExpectedFinalState: map[string]map[string]expireableItem{},
			ExpectedItems:      map[string]store.OwnableItem{},
		},
		{
			Description:   "Mixed Items",
			OriginalState: s.DataItemsMixed,
			ExpectedFinalState: map[string]map[string]expireableItem{
				s.BucketName: {
					s.ItemOneID: s.ItemOne,
					s.ItemTwoID: s.ItemTwo,
				},
			},
			ExpectedItems: map[string]store.OwnableItem{
				s.ItemOneID: s.ItemOne.OwnableItem,
				s.ItemTwoID: s.ItemTwo.OwnableItem,
			},
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			storage := InMem{data: tc.OriginalState, now: s.NowFunc}
			items, err := storage.GetAll(s.BucketName)
			assert.Nil(err)
			assert.Equal(tc.ExpectedItems, items)
			assert.Equal(tc.ExpectedFinalState, storage.data)
		})
	}
}

func (s *InMemTestSuite) TestDelete() {
	tcs := []struct {
		Description        string
		OriginalState      map[string]map[string]expireableItem
		ExpectedFinalState map[string]map[string]expireableItem
		ItemKey            model.Key
		ExpectedItem       store.OwnableItem
		ExpectedError      error
	}{
		{
			Description:        "Bucket missing",
			OriginalState:      s.DataBucketNotFound,
			ExpectedFinalState: map[string]map[string]expireableItem{},
			ItemKey:            s.ItemOneKey,
			ExpectedError:      store.ErrItemNotFound,
		},
		{
			Description:   "Item missing",
			OriginalState: s.DataItemNotFound,
			ExpectedFinalState: map[string]map[string]expireableItem{
				s.BucketName: {
					"other-item-id": expireableItem{},
				},
			},
			ItemKey:       s.ItemOneKey,
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description:        "Item expired",
			OriginalState:      s.DataItemExpired,
			ExpectedFinalState: map[string]map[string]expireableItem{},
			ItemKey:            s.ItemThreeKey,
			ExpectedError:      store.ErrItemNotFound,
		},
		{
			Description:        "Item found and deleted",
			OriginalState:      s.DataItemFound,
			ExpectedFinalState: map[string]map[string]expireableItem{},
			ItemKey:            s.ItemOneKey,
			ExpectedItem:       s.ItemOne.OwnableItem,
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			storage := InMem{data: tc.OriginalState, now: s.NowFunc}
			actualItem, err := storage.Delete(tc.ItemKey)
			if tc.ExpectedError != nil {
				var sErr store.SanitizedError
				require.True(errors.As(err, &sErr), "Expected '%v' to be a store.SanitizedError", err)
				var iErr store.ItemOperationError
				require.True(errors.As(err, &iErr), "Expected '%v' to be a store.ItemOperationError", err)
				assert.Equal("delete", iErr.Operation)
				assert.Equal(tc.ItemKey, iErr.Key)
				assert.True(errors.Is(err, tc.ExpectedError), "Expected to find match for '%v' in error chain of '%v'", tc.ExpectedError, err)
			} else {
				assert.Nil(err)
				assert.Equal(tc.ExpectedItem, actualItem)
				assert.Equal(tc.ExpectedFinalState, storage.data)
			}
		})
	}
}

func TestInMem(t *testing.T) {
	suite.Run(t, new(InMemTestSuite))
}

func TestInMemConcurrent(t *testing.T) {
	Storage := &InMem{
		data: map[string]map[string]expireableItem{},
		now:  time.Now,
	}
	BucketName := "test-bucket-name"
	ItemOneID := "test-item-id-1"
	ItemOne := store.OwnableItem{
		Owner: "test-owner-1",
		Item: model.Item{
			ID: ItemOneID,
			Data: map[string]interface{}{
				"k1": "v1",
			},
		},
	}
	ItemOneKey := model.Key{ID: ItemOneID, Bucket: BucketName}
	for i := 0; i < 30; i++ {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			t.Parallel()
			Storage.Push(ItemOneKey, ItemOne)
			Storage.Delete(ItemOneKey)
			Storage.GetAll(BucketName)
			Storage.Get(ItemOneKey)
		})
	}
}

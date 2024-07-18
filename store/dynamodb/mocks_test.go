// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package dynamodb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/argus/store"
)

type mockService struct {
	mock.Mock
}

func (s *mockService) Push(key model.Key, item store.OwnableItem) (*dynamodb.ConsumedCapacity, error) {
	args := s.Called(key, item)
	return args.Get(0).(*dynamodb.ConsumedCapacity), args.Error(1)
}

func (s *mockService) Get(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	args := s.Called(key)
	return args.Get(0).(store.OwnableItem), args.Get(1).(*dynamodb.ConsumedCapacity), args.Error(2)
}

func (s *mockService) Delete(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	args := s.Called(key)
	return args.Get(0).(store.OwnableItem), args.Get(1).(*dynamodb.ConsumedCapacity), args.Error(2)
}

func (s *mockService) GetAll(bucket string) (map[string]store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	args := s.Called(bucket)
	return args.Get(0).(map[string]store.OwnableItem), args.Get(1).(*dynamodb.ConsumedCapacity), args.Error(2)
}

type mockClient struct {
	mock.Mock
}

func (c *mockClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (c *mockClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (c *mockClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*dynamodb.DeleteItemOutput), args.Error(1)
}

func (c *mockClient) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

type mockMeasuresUpdater struct {
	mock.Mock
}

func (m *mockMeasuresUpdater) Update(r *measureUpdateRequest) {
	m.Called(r)
}

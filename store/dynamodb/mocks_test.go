// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package dynamodb

import (
	"context"

	awsv2dynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsv2dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

type mockService struct {
	mock.Mock
}

func (s *mockService) Push(key model.Key, item store.OwnableItem) (*awsv2dynamodbTypes.ConsumedCapacity, error) {
	args := s.Called(key, item)
	return args.Get(0).(*awsv2dynamodbTypes.ConsumedCapacity), args.Error(1)
}

func (s *mockService) Get(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	args := s.Called(key)
	return args.Get(0).(store.OwnableItem), args.Get(1).(*awsv2dynamodbTypes.ConsumedCapacity), args.Error(2)
}

func (s *mockService) Delete(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	args := s.Called(key)
	return args.Get(0).(store.OwnableItem), args.Get(1).(*awsv2dynamodbTypes.ConsumedCapacity), args.Error(2)
}

func (s *mockService) GetAll(bucket string) (map[string]store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	args := s.Called(bucket)
	return args.Get(0).(map[string]store.OwnableItem), args.Get(1).(*awsv2dynamodbTypes.ConsumedCapacity), args.Error(2)
}

type mockMeasuresUpdater struct {
	mock.Mock
}

func (m *mockMeasuresUpdater) Update(r *measureUpdateRequest) {
	m.Called(r)
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) PutItem(ctx context.Context, params *awsv2dynamodb.PutItemInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.PutItemOutput, error) {
	args := m.Called(mock.Anything, mock.Anything, mock.Anything)
	var out *awsv2dynamodb.PutItemOutput
	if v := args.Get(0); v != nil {
		out = v.(*awsv2dynamodb.PutItemOutput)
	}
	return out, args.Error(1)
}

func (m *mockClient) GetItem(ctx context.Context, params *awsv2dynamodb.GetItemInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.GetItemOutput, error) {
	// DEBUG: Print when GetItem is called and with what key
	if params != nil && params.Key != nil {
		bucket := ""
		id := ""
		if v, ok := params.Key["bucket"].(*awsv2dynamodbTypes.AttributeValueMemberS); ok {
			bucket = v.Value
		}
		if v, ok := params.Key["id"].(*awsv2dynamodbTypes.AttributeValueMemberS); ok {
			id = v.Value
		}
		println("DEBUG: mockClient.GetItem called with bucket=", bucket, ", id=", id)
	}
	args := m.Called(mock.Anything, mock.Anything, mock.Anything)
	var out *awsv2dynamodb.GetItemOutput
	if v := args.Get(0); v != nil {
		out = v.(*awsv2dynamodb.GetItemOutput)
	}
	return out, args.Error(1)
}

func (m *mockClient) DeleteItem(ctx context.Context, params *awsv2dynamodb.DeleteItemInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.DeleteItemOutput, error) {
	args := m.Called(mock.Anything, mock.Anything, mock.Anything)
	var out *awsv2dynamodb.DeleteItemOutput
	if v := args.Get(0); v != nil {
		out = v.(*awsv2dynamodb.DeleteItemOutput)
	}
	return out, args.Error(1)
}

func (m *mockClient) Query(ctx context.Context, params *awsv2dynamodb.QueryInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.QueryOutput, error) {
	args := m.Called(mock.Anything, mock.Anything, mock.Anything)
	var out *awsv2dynamodb.QueryOutput
	if v := args.Get(0); v != nil {
		out = v.(*awsv2dynamodb.QueryOutput)
	}
	return out, args.Error(1)
}

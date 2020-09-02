package dynamodb

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

// TODO: PutItem
// Returned Error Types:
//   * ConditionalCheckFailedException
//   A condition specified in the operation could not be evaluated.
//
//   * ProvisionedThroughputExceededException
//   Your request rate is too high. The AWS SDKs for DynamoDB automatically retry
//   requests that receive this exception. Your request is eventually successful,
//   unless your retry queue is too large to finish. Reduce the frequency of requests
//   and use exponential backoff. For more information, go to Error Retries and
//   Exponential Backoff (https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html#Programming.Errors.RetryAndBackoff)
//   in the Amazon DynamoDB Developer Guide.
//
//   * ResourceNotFoundException
//   The operation tried to access a nonexistent table or index. The resource
//   might not be specified correctly, or its status might not be ACTIVE.
//
//   * ItemCollectionSizeLimitExceededException
//   An item collection is too large. This exception is only returned for tables
//   that have one or more local secondary indexes.
//
//   * TransactionConflictException
//   Operation was rejected because there is an ongoing transaction for the item.
//
//   * RequestLimitExceeded
//   Throughput exceeds the current throughput limit for your account. Please
//   contact AWS Support at AWS Support (https://aws.amazon.com/support) to request
//   a limit increase.
//
//   * InternalServerError
//   An error occurred on the server side.
//
func TestPush(t *testing.T) {
	testCases := []struct {
		name         string
		dynamoOutput *dynamodb.PutItemOutput
		dynamoErr    error
		expectedErr  error
	}{
		{
			name:         "Throughput exceeded",
			dynamoOutput: new(dynamodb.PutItemOutput),
			dynamoErr:    new(dynamodb.ProvisionedThroughputExceededException),
			expectedErr:  store.InternalError{Reason: dynamodb.ErrCodeProvisionedThroughputExceededException, Retryable: true},
		},
		{
			name:         "Resource not found",
			dynamoOutput: new(dynamodb.PutItemOutput),
			dynamoErr:    new(dynamodb.ResourceNotFoundException),
			expectedErr:  store.InternalError{Reason: dynamodb.ErrCodeResourceNotFoundException, Retryable: false},
		},
		{
			name:         "Request Limit exceeded",
			dynamoOutput: new(dynamodb.PutItemOutput),
			dynamoErr:    new(dynamodb.RequestLimitExceeded),
			expectedErr:  store.InternalError{Reason: dynamodb.ErrCodeRequestLimitExceeded, Retryable: false},
		},
		{
			name:         "Internal server error",
			dynamoOutput: new(dynamodb.PutItemOutput),
			dynamoErr:    new(dynamodb.InternalServerError),
			expectedErr:  store.InternalError{Reason: dynamodb.ErrCodeInternalServerError, Retryable: true},
		},
		{
			name:         "Non AWS Error",
			dynamoOutput: new(dynamodb.PutItemOutput),
			dynamoErr:    errors.New("non AWS internal error"),
			expectedErr:  store.InternalError{Reason: "non AWS internal error", Retryable: false},
		},
		{
			name: "Success",
			dynamoOutput: &dynamodb.PutItemOutput{
				ConsumedCapacity: &dynamodb.ConsumedCapacity{
					CapacityUnits: aws.Float64(64),
				},
			},
		},
	}

	item := store.OwnableItem{
		Item: model.Item{
			Identifier: "id01",
			Data:       map[string]interface{}{"dataKey": "dataValue"},
			TTL:        int64(time.Second * 300),
		},
		Owner: "xmidt",
	}
	key := model.Key{
		Bucket: "bucket01",
		ID:     "id01",
	}
	expirableItem := element{
		OwnableItem: item,
		Expires:     time.Now().Unix() + item.TTL,
		Key:         key,
	}
	encodedItem, err := dynamodbattribute.MarshalMap(expirableItem)
	input := &dynamodb.PutItemInput{
		Item:                   encodedItem,
		TableName:              aws.String("testTable"),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}
	require := require.New(t)
	require.Nil(err)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			assert.NotNil(m)
			m.On("PutItem", input).Return(testCase.dynamoOutput, testCase.dynamoErr)
			service := &executor{
				c:         m,
				tableName: "testTable",
			}
			resp, err := service.Push(key, item)
			assert.Equal(testCase.expectedErr, err)
			assert.Equal(testCase.dynamoOutput.ConsumedCapacity, resp)
			m.AssertExpectations(t)
		})
	}
}

//TODO: Delete
// Returned Error Types:
//   * ConditionalCheckFailedException
//   A condition specified in the operation could not be evaluated.
//
//   * ProvisionedThroughputExceededException
//   Your request rate is too high. The AWS SDKs for DynamoDB automatically retry
//   requests that receive this exception. Your request is eventually successful,
//   unless your retry queue is too large to finish. Reduce the frequency of requests
//   and use exponential backoff. For more information, go to Error Retries and
//   Exponential Backoff (https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html#Programming.Errors.RetryAndBackoff)
//   in the Amazon DynamoDB Developer Guide.
//
//   * ResourceNotFoundException
//   The operation tried to access a nonexistent table or index. The resource
//   might not be specified correctly, or its status might not be ACTIVE.
//
//   * ItemCollectionSizeLimitExceededException
//   An item collection is too large. This exception is only returned for tables
//   that have one or more local secondary indexes.
//
//   * TransactionConflictException
//   Operation was rejected because there is an ongoing transaction for the item.
//
//   * RequestLimitExceeded
//   Throughput exceeds the current throughput limit for your account. Please
//   contact AWS Support at AWS Support (https://aws.amazon.com/support) to request
//   a limit increase.
//
//   * InternalServerError
//   An error occurred on the server side.

//TODO: GetItem
// Returned Error Types:
//   * ProvisionedThroughputExceededException
//   Your request rate is too high. The AWS SDKs for DynamoDB automatically retry
//   requests that receive this exception. Your request is eventually successful,
//   unless your retry queue is too large to finish. Reduce the frequency of requests
//   and use exponential backoff. For more information, go to Error Retries and
//   Exponential Backoff (https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html#Programming.Errors.RetryAndBackoff)
//   in the Amazon DynamoDB Developer Guide.
//
//   * ResourceNotFoundException
//   The operation tried to access a nonexistent table or index. The resource
//   might not be specified correctly, or its status might not be ACTIVE.
//
//   * RequestLimitExceeded
//   Throughput exceeds the current throughput limit for your account. Please
//   contact AWS Support at AWS Support (https://aws.amazon.com/support) to request
//   a limit increase.
//
//   * InternalServerError
//   An error occurred on the server side.

//TODO: Query
// Returned Error Types:
//   * ProvisionedThroughputExceededException
//   Your request rate is too high. The AWS SDKs for DynamoDB automatically retry
//   requests that receive this exception. Your request is eventually successful,
//   unless your retry queue is too large to finish. Reduce the frequency of requests
//   and use exponential backoff. For more information, go to Error Retries and
//   Exponential Backoff (https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html#Programming.Errors.RetryAndBackoff)
//   in the Amazon DynamoDB Developer Guide.
//
//   * ResourceNotFoundException
//   The operation tried to access a nonexistent table or index. The resource
//   might not be specified correctly, or its status might not be ACTIVE.
//
//   * RequestLimitExceeded
//   Throughput exceeds the current throughput limit for your account. Please
//   contact AWS Support at AWS Support (https://aws.amazon.com/support) to request
//   a limit increase.
//
//   * InternalServerError
//   An error occurred on the server side

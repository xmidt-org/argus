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
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

const (
	testTableName  = "table01"
	testBucketName = "bucket01"
	testIDName     = "ID01"
)

var (
	putItemInput    *dynamodb.PutItemInput
	getItemInput    *dynamodb.GetItemInput
	queryInput      *dynamodb.QueryInput
	deleteItemInput *dynamodb.DeleteItemInput
)

var (
	item = store.OwnableItem{
		Item: model.Item{
			Identifier: testIDName,
			Data:       map[string]interface{}{"dataKey": "dataValue"},
			TTL:        int64(time.Second * 300),
		},
		Owner: "xmidt",
	}
	key = model.Key{
		Bucket: testBucketName,
		ID:     testIDName,
	}
)

type errorCase struct {
	name        string
	dynamoErr   error
	expectedErr error
}

type operationType struct {
	name          string
	mockedMethod  string
	mockedArgs    []interface{}
	mockedReturns []interface{}
}

type ClientErrorTestSuite struct {
	suite.Suite
	operationTypes   []operationType
	clientErrorCases []errorCase
}

func (s *ClientErrorTestSuite) SetupTest() {
	s.setupOperations()
	s.setupErrorCases()
}
func (s *ClientErrorTestSuite) TestClientErrors() {
	for _, operationType := range s.operationTypes {
		s.T().Run(operationType.name, func(t *testing.T) {
			for _, clientErrorCase := range s.clientErrorCases {
				s.T().Run(clientErrorCase.name, func(t *testing.T) {
					assert := assert.New(t)
					require := require.New(t)
					m := new(mockClient)
					require.NotNil(m)
					m.On(operationType.mockedMethod, operationType.mockedArgs...).Return(append(operationType.mockedReturns, clientErrorCase.dynamoErr)...)
					service := &executor{
						c:         m,
						tableName: testTableName,
					}

					switch operationType.name {
					case "Push":
						consumedCapacity, err := service.Push(key, item)
						assert.Equal(clientErrorCase.expectedErr, err)
						assert.Nil(consumedCapacity)
					case "Get":
						ownableItem, consumedCapacity, err := service.Get(key)
						assert.Equal(clientErrorCase.expectedErr, err)
						assert.Nil(consumedCapacity)
						assert.Equal(store.OwnableItem{}, ownableItem)
					case "GetAll":
						ownableItems, consumedCapacity, err := service.GetAll(testBucketName)
						assert.Equal(clientErrorCase.expectedErr, err)
						assert.Nil(consumedCapacity)
						assert.Equal(map[string]store.OwnableItem{}, ownableItems)
					case "Delete":
						ownableItem, consumedCapacity, err := service.Delete(key)
						assert.Equal(clientErrorCase.expectedErr, err)
						assert.Nil(consumedCapacity)
						assert.Equal(store.OwnableItem{}, ownableItem)
					}
					m.AssertExpectations(t)
				})
			}
		})
	}
}

func testClientErrors(t *testing.T) {
	suite.Run(t, new(ClientErrorTestSuite))
}

func TestAll(t *testing.T) {
	initGlobalInputs()
	t.Run("ClientErrors", testClientErrors)
	t.Run("Push", testPush)
	t.Run("GetItem", func(t *testing.T) {
		t.Run("Success", testGetItem)
		t.Run("NotFound", testGetItemNotFound)
	})
	t.Run("Delete", func(t *testing.T) {
		t.Run("Success", testDelete)
		t.Run("NotFound", testDeleteNotFound)
	})
	t.Run("GetAll", func(t *testing.T) {
		t.Run("Success", testGetAll)
	})
}

func testGetAll(t *testing.T) {
	assert := assert.New(t)
	m := new(mockClient)
	expectedConsumedCapacity := &dynamodb.ConsumedCapacity{
		CapacityUnits: aws.Float64(67),
	}
	queryOutput := &dynamodb.QueryOutput{
		ConsumedCapacity: expectedConsumedCapacity,
		Items: []map[string]*dynamodb.AttributeValue{
			{
				bucketAttributeKey: {
					S: aws.String(testBucketName),
				},
				idAttributeKey: {
					S: aws.String("id01"),
				},
			},
			{
				bucketAttributeKey: {
					S: aws.String(testBucketName),
				},
				idAttributeKey: {
					S: aws.String("id02"),
				},
			},
		},
	}

	m.On("Query", queryInput).Return(queryOutput, error(nil))
	service := &executor{
		tableName: testTableName,
		c:         m,
	}
	ownableItems, actualConsumedCapacity, err := service.GetAll(testBucketName)
	assert.Nil(err)
	assert.Len(ownableItems, 2)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func testDelete(t *testing.T) {
	assert := assert.New(t)
	m := new(mockClient)
	expectedConsumedCapacity := &dynamodb.ConsumedCapacity{
		CapacityUnits: aws.Float64(67),
	}
	deleteItemOutput := &dynamodb.DeleteItemOutput{
		ConsumedCapacity: expectedConsumedCapacity,
		Attributes: map[string]*dynamodb.AttributeValue{
			bucketAttributeKey: {
				S: aws.String(testBucketName),
			},
			idAttributeKey: {
				S: aws.String(testIDName),
			},
			"data": {
				M: map[string]*dynamodb.AttributeValue{
					"key": {
						S: aws.String("stringVal"),
					},
				},
			},
			"owner": {
				S: aws.String("xmidt"),
			},

			"identifier": {
				S: aws.String("id01"),
			},
		},
	}
	expectedData := map[string]interface{}{
		"key": "stringVal",
	}
	m.On("DeleteItem", deleteItemInput).Return(deleteItemOutput, error(nil))
	service := &executor{
		tableName: testTableName,
		c:         m,
	}
	ownableItem, actualConsumedCapacity, err := service.Delete(key)
	assert.Nil(err)
	assert.Equal("xmidt", ownableItem.Owner)
	assert.Equal("id01", ownableItem.Identifier)
	assert.Equal(expectedData, ownableItem.Data)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func testDeleteNotFound(t *testing.T) {
	assert := assert.New(t)
	m := new(mockClient)
	expectedConsumedCapacity := &dynamodb.ConsumedCapacity{
		CapacityUnits: aws.Float64(67),
	}
	deleteItemOutput := &dynamodb.DeleteItemOutput{
		ConsumedCapacity: expectedConsumedCapacity,
	}
	m.On("DeleteItem", deleteItemInput).Return(deleteItemOutput, error(nil))
	service := &executor{
		tableName: testTableName,
		c:         m,
	}
	ownableItem, actualConsumedCapacity, err := service.Delete(key)
	assert.NotNil(ownableItem)
	assert.Equal(store.KeyNotFoundError{Key: key}, err)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func testPush(t *testing.T) {
	assert := assert.New(t)
	m := new(mockClient)
	expectedConsumedCapacity := &dynamodb.ConsumedCapacity{
		CapacityUnits: aws.Float64(67),
	}
	putItemOutput := &dynamodb.PutItemOutput{
		ConsumedCapacity: expectedConsumedCapacity,
	}
	m.On("PutItem", putItemInput).Return(putItemOutput, error(nil))
	service := &executor{
		tableName: testTableName,
		c:         m,
	}
	actualConsumedCapacity, err := service.Push(key, item)
	assert.Nil(err)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func testGetItemNotFound(t *testing.T) {
	assert := assert.New(t)
	m := new(mockClient)
	expectedConsumedCapacity := &dynamodb.ConsumedCapacity{
		CapacityUnits: aws.Float64(67),
	}
	getItemOutput := &dynamodb.GetItemOutput{
		ConsumedCapacity: expectedConsumedCapacity,
	}
	m.On("GetItem", getItemInput).Return(getItemOutput, error(nil))
	service := &executor{
		tableName: testTableName,
		c:         m,
	}
	ownableItem, actualConsumedCapacity, err := service.Get(key)
	assert.NotNil(ownableItem)
	assert.Equal(store.KeyNotFoundError{Key: key}, err)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func testGetItem(t *testing.T) {
	assert := assert.New(t)
	m := new(mockClient)
	expectedConsumedCapacity := &dynamodb.ConsumedCapacity{
		CapacityUnits: aws.Float64(67),
	}
	getItemOutput := &dynamodb.GetItemOutput{
		ConsumedCapacity: expectedConsumedCapacity,
		Item: map[string]*dynamodb.AttributeValue{
			bucketAttributeKey: {
				S: aws.String(testBucketName),
			},
			idAttributeKey: {
				S: aws.String(testIDName),
			},
			"data": {
				M: map[string]*dynamodb.AttributeValue{
					"key": {
						S: aws.String("stringVal"),
					},
				},
			},
			"owner": {
				S: aws.String("xmidt"),
			},

			"identifier": {
				S: aws.String("id01"),
			},
		},
	}
	expectedData := map[string]interface{}{
		"key": "stringVal",
	}
	m.On("GetItem", getItemInput).Return(getItemOutput, error(nil))
	service := &executor{
		tableName: testTableName,
		c:         m,
	}
	ownableItem, actualConsumedCapacity, err := service.Get(key)
	assert.Nil(err)
	assert.Equal("xmidt", ownableItem.Owner)
	assert.Equal("id01", ownableItem.Identifier)
	assert.Equal(expectedData, ownableItem.Data)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func initGlobalInputs() {
	getItemInput = &dynamodb.GetItemInput{
		TableName: aws.String(testTableName),
		Key: map[string]*dynamodb.AttributeValue{
			bucketAttributeKey: {
				S: aws.String(key.Bucket),
			},
			idAttributeKey: {
				S: aws.String(key.ID),
			},
		},
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}

	deleteItemInput = &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			bucketAttributeKey: {
				S: aws.String(key.Bucket),
			},
			idAttributeKey: {
				S: aws.String(key.ID),
			},
		},
		ReturnValues:           aws.String(dynamodb.ReturnValueAllOld),
		TableName:              aws.String(testTableName),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}

	expirableItem := element{
		OwnableItem: item,
		Expires:     time.Now().Unix() + item.TTL,
		Key:         key,
	}
	encodedItem, err := dynamodbattribute.MarshalMap(expirableItem)
	if err != nil {
		panic(err)
	}

	putItemInput = &dynamodb.PutItemInput{
		Item:                   encodedItem,
		TableName:              aws.String(testTableName),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}

	queryInput = &dynamodb.QueryInput{
		TableName:              aws.String(testTableName),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
		KeyConditions: map[string]*dynamodb.Condition{
			"bucket": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(testBucketName),
					},
				},
			},
		},
	}
}

func (s *ClientErrorTestSuite) setupOperations() {
	s.operationTypes = []operationType{
		{
			name:          "Push",
			mockedMethod:  "PutItem",
			mockedArgs:    []interface{}{putItemInput},
			mockedReturns: []interface{}{new(dynamodb.PutItemOutput)},
		},

		{
			name:          "Get",
			mockedMethod:  "GetItem",
			mockedArgs:    []interface{}{getItemInput},
			mockedReturns: []interface{}{new(dynamodb.GetItemOutput)},
		},
		{
			name:          "GetAll",
			mockedMethod:  "Query",
			mockedArgs:    []interface{}{queryInput},
			mockedReturns: []interface{}{new(dynamodb.QueryOutput)},
		},
		{
			name:          "Delete",
			mockedMethod:  "DeleteItem",
			mockedArgs:    []interface{}{deleteItemInput},
			mockedReturns: []interface{}{new(dynamodb.DeleteItemOutput)},
		},
	}
}

func (s *ClientErrorTestSuite) setupErrorCases() {
	s.clientErrorCases = []errorCase{
		{
			name:        "Throughput exceeded",
			dynamoErr:   new(dynamodb.ProvisionedThroughputExceededException),
			expectedErr: store.InternalError{Reason: dynamodb.ErrCodeProvisionedThroughputExceededException, Retryable: true},
		},
		{
			name:        "Resource not found",
			dynamoErr:   new(dynamodb.ResourceNotFoundException),
			expectedErr: store.InternalError{Reason: dynamodb.ErrCodeResourceNotFoundException, Retryable: false},
		},
		{
			name:        "Request Limit exceeded",
			dynamoErr:   new(dynamodb.RequestLimitExceeded),
			expectedErr: store.InternalError{Reason: dynamodb.ErrCodeRequestLimitExceeded, Retryable: false},
		},
		{
			name:        "Internal server error",
			dynamoErr:   new(dynamodb.InternalServerError),
			expectedErr: store.InternalError{Reason: dynamodb.ErrCodeInternalServerError, Retryable: true},
		},
		{
			name:        "Non AWS Error",
			dynamoErr:   errors.New("non AWS internal error"),
			expectedErr: store.InternalError{Reason: "non AWS internal error", Retryable: false},
		},
	}
}

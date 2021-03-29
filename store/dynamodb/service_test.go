package dynamodb

import (
	"errors"
	"strconv"
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
	testID         = "NaYFGE961cS_3dpzJcoP3QTL4kBYcw9ua3Q6Hy5E4nI"
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
			ID:   testID,
			Data: map[string]interface{}{"dataKey": "dataValue"},
			TTL:  aws.Int64(int64((time.Second * 300).Seconds())),
		},
		Owner: "xmidt",
	}
	key = model.Key{
		Bucket: testBucketName,
		ID:     testID,
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

					var (
						consumedCapacity *dynamodb.ConsumedCapacity
						err              error
						ownableItems     = map[string]store.OwnableItem{}
						ownableItem      = store.OwnableItem{}
					)

					switch operationType.name {
					case "Push":
						consumedCapacity, err = service.Push(key, item)
					case "Get":
						ownableItem, consumedCapacity, err = service.Get(key)
					case "GetAll":
						ownableItems, consumedCapacity, err = service.GetAll(testBucketName)
					case "Delete":
						ownableItem, consumedCapacity, err = service.Delete(key)
					}
					assert.Nil(consumedCapacity)
					assert.Equal(store.OwnableItem{}, ownableItem)
					assert.Equal(map[string]store.OwnableItem{}, ownableItems)
					m.AssertExpectations(t)

					assert.Equal(clientErrorCase.expectedErr, err)
				})
			}
		})
	}
}

// TODO: We might not need this anymore.
func testClientErrors(t *testing.T) {
	initGlobalInputs()
	suite.Run(t, new(ClientErrorTestSuite))
}

func genTestPutItemInput(key model.Key, item store.OwnableItem) *dynamodb.PutItemInput {
	storingItem := storableItem{
		OwnableItem: item,
		Key:         key,
	}

	if item.TTL != nil {
		unixExpSeconds := time.Now().Unix() + *item.TTL
		storingItem.Expires = &unixExpSeconds
	}

	av, err := dynamodbattribute.MarshalMap(storingItem)
	if err != nil {
		panic("must be able to marshal")
	}
	return &dynamodb.PutItemInput{
		Item:                   av,
		TableName:              aws.String("testTable"),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}
}

func TestNewPushItem(t *testing.T) {
	var (
		dbErr    = errors.New("dynamodb error")
		capacity = &dynamodb.ConsumedCapacity{}
	)
	tcs := []struct {
		Description              string
		Key                      model.Key
		Item                     store.OwnableItem
		PutItemFails             bool
		ExpectedConsumedCapacity *dynamodb.ConsumedCapacity
		ExpectedError            error
	}{
		{
			Description: "PutItem Fails",
			Key:         model.Key{Bucket: "testBucket", ID: "id001"},
			Item: store.OwnableItem{
				Owner: "testOwner",
				Item: model.Item{
					ID:  "id001",
					TTL: aws.Int64(5),
				},
			},
			PutItemFails:             true,
			ExpectedConsumedCapacity: nil,
			ExpectedError:            dbErr,
		},
		{
			Description: "Success. No TTL",
			Key:         model.Key{Bucket: "testBucket", ID: "id001"},
			Item: store.OwnableItem{
				Owner: "testOwner",
				Item: model.Item{
					ID: "id001",
				},
			},
			PutItemFails:             false,
			ExpectedConsumedCapacity: capacity,
			ExpectedError:            nil,
		},
		{
			Description: "Success with TTL",
			Key:         model.Key{Bucket: "testBucket", ID: "id001"},
			Item: store.OwnableItem{
				Owner: "testOwner",
				Item: model.Item{
					ID: "id001",
				},
			},
			PutItemFails:             false,
			ExpectedConsumedCapacity: capacity,
			ExpectedError:            nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			sv := executor{
				c:         m,
				tableName: "testTable",
			}
			var (
				putItemOutput = &dynamodb.PutItemOutput{
					ConsumedCapacity: capacity,
				}
				putItemErr error
			)
			if tc.PutItemFails {
				putItemOutput, putItemErr = nil, dbErr
			}

			m.On("PutItem", genTestPutItemInput(tc.Key, tc.Item)).Return(putItemOutput, putItemErr)
			cc, err := sv.Push(tc.Key, tc.Item)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedError, err)
			m.AssertExpectations(t)
		})

	}
	//  Cases
	// (2) Include TTL, dynamo client fails
	// (2) Include TTL, dynamo client succeeds
	// (1) No TTL, Success

}

func TestPushItem(t *testing.T) {
	initGlobalInputs()

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

func TestGetAllItems(t *testing.T) {
	initGlobalInputs()
	assert := assert.New(t)
	m := new(mockClient)
	now := time.Now().Unix()
	secondsInHour := int64(time.Hour.Seconds())
	pastExpiration := strconv.Itoa(int(now - secondsInHour))
	futureExpiration := strconv.Itoa(int(now + secondsInHour))

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
					S: aws.String("-mTqHoLhIG-CirKgKRfH6SrMuY47lYgaG0rVK5FLZuM"),
				},
				expirationAttributeKey: {
					N: aws.String(pastExpiration),
				},
			},
			{
				bucketAttributeKey: {
					S: aws.String(testBucketName),
				},
				idAttributeKey: {
					S: aws.String("1wzI3cbHlIHD9TUi9LgOz1Vt1cZIOloD4PvlB5uFT4E"),
				},
				expirationAttributeKey: {
					N: aws.String(futureExpiration),
				},
			},

			{
				bucketAttributeKey: {
					S: aws.String(testBucketName),
				},
				idAttributeKey: {
					S: aws.String("dbtIlYXQsAoAmexD6zGV8ZfVImEjsFGHcMJdhCZ-1L4"),
				},
			},

			{
				bucketAttributeKey: {
					S: aws.String(testBucketName),
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

	for _, item := range ownableItems {
		assert.NotEmpty(item.ID)
		if item.TTL != nil {
			assert.NotZero(*item.TTL)
		}
	}
}

func TestDeleteItem(t *testing.T) {
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
				S: aws.String(testID),
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
	assert.Equal(expectedData, ownableItem.Data)
	assert.Equal(expectedConsumedCapacity, actualConsumedCapacity)
}

func TestGetItem(t *testing.T) {
	initGlobalInputs()
	now := time.Now().Unix()
	secondsInHour := int64(time.Hour.Seconds())
	pastExpiration := strconv.Itoa(int(now - secondsInHour))
	futureExpiration := strconv.Itoa(int(now + secondsInHour))

	testCases := []struct {
		Name                string
		GetItemOutput       *dynamodb.GetItemOutput
		GetItemOutputErr    error
		ItemExpires         bool
		ExpectedResponse    store.OwnableItem
		ExpectedResponseErr error
	}{
		{
			Name: "Item does not expire",
			GetItemOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					bucketAttributeKey: {
						S: aws.String(testBucketName),
					},
					idAttributeKey: {
						S: aws.String(testID),
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
				},
			},
			ExpectedResponse: store.OwnableItem{
				Owner: "xmidt",
				Item: model.Item{
					ID: testID,
					Data: map[string]interface{}{
						"key": "stringVal",
					},
				},
			},
		},

		{
			Name:        "Expired item",
			ItemExpires: true,
			GetItemOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"expires": {
						N: aws.String(pastExpiration),
					},
					bucketAttributeKey: {
						S: aws.String(testBucketName),
					},
					idAttributeKey: {
						S: aws.String(testID),
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
				},
			},
			ExpectedResponseErr: store.KeyNotFoundError{Key: model.Key{
				ID:     testID,
				Bucket: testBucketName,
			}},
		},

		{
			Name:        "Item not yet expired",
			ItemExpires: true,
			GetItemOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"expires": {
						N: aws.String(futureExpiration),
					},
					bucketAttributeKey: {
						S: aws.String(testBucketName),
					},
					idAttributeKey: {
						S: aws.String(testID),
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
				},
			},
			ExpectedResponse: store.OwnableItem{
				Owner: "xmidt",
				Item: model.Item{
					ID: testID,
					Data: map[string]interface{}{
						"key": "stringVal",
					},
				},
			},
		},

		{
			Name: "Item not found",
			GetItemOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			ExpectedResponseErr: store.KeyNotFoundError{Key: key},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			m.On("GetItem", getItemInput).Return(testCase.GetItemOutput, error(nil))
			service := &executor{
				tableName: testTableName,
				c:         m,
			}
			ownableItem, actualConsumedCapacity, err := service.Get(key)
			if testCase.ExpectedResponseErr == nil {
				assert.Nil(err)
				assert.Equal(testCase.GetItemOutput.ConsumedCapacity, actualConsumedCapacity)
				assert.Equal(testCase.ExpectedResponse.Owner, ownableItem.Owner)
				assert.Equal(testCase.ExpectedResponse.Data, ownableItem.Data)
				assert.Equal(testCase.ExpectedResponse.ID, ownableItem.ID)

				if testCase.ItemExpires {
					assert.NotZero(*ownableItem.TTL)
				}
			} else {
				assert.Equal(testCase.ExpectedResponseErr, err)
			}
		})
	}
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

	expirableItem := storableItem{
		OwnableItem: item,
		Expires:     aws.Int64(time.Now().Unix() + *item.TTL),
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
	var (
		errResourceNotFound = new(dynamodb.ResourceNotFoundException)
		errValidation       = &dynamodb.TransactionCanceledException{Message_: aws.String("ValidationException: Nesting Levels have exceeded supported limits")}
	)

	s.clientErrorCases = []errorCase{
		{
			name:        "Resource not found",
			dynamoErr:   errResourceNotFound,
			expectedErr: store.SanitizedError{ErrHTTP: errDefaultDynamoDBFailure, Err: errResourceNotFound},
		},
		{
			name:        "ValidationException",
			dynamoErr:   errValidation,
			expectedErr: store.SanitizedError{ErrHTTP: errBadRequest, Err: errValidation},
		},
	}
}

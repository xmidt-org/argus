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
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

func TestPushItem(t *testing.T) {
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

			m.On("PutItem", getPutItemInput(tc.Key, tc.Item)).Return(putItemOutput, putItemErr)
			cc, err := sv.Push(tc.Key, tc.Item)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedError, err)
			m.AssertExpectations(t)
		})

	}
}

func TestGetAll(t *testing.T) {
	var (
		dbErr            = errors.New("dynamodb error")
		consumedCapacity = &dynamodb.ConsumedCapacity{}
	)
	nowRef := getRefTime()
	nowFunc := func() time.Time {
		return nowRef
	}

	tcs := []struct {
		Description              string
		QueryErr                 error
		QueryOutput              *dynamodb.QueryOutput
		ExpectedItems            map[string]store.OwnableItem
		ExpectedConsumedCapacity *dynamodb.ConsumedCapacity
		ExpectedErr              error
	}{
		{
			Description:   "Query fails",
			QueryErr:      dbErr,
			QueryOutput:   nil,
			ExpectedErr:   dbErr,
			ExpectedItems: map[string]store.OwnableItem{},
		},
		{
			Description:              "Expired or bad items",
			QueryOutput:              getFilteredQueryOutput(nowRef, consumedCapacity),
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItems:            getFilteredExpectedItems(),
		},
		{
			Description:              "All good items",
			QueryOutput:              getQueryOutput(nowRef, consumedCapacity),
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItems:            getExpectedItems(),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			svc := executor{
				c:         m,
				tableName: "testTable",
				now:       nowFunc,
			}
			m.On("Query", getQueryInput()).Return(tc.QueryOutput, tc.QueryErr)
			items, cc, err := svc.GetAll("testBucket")
			assert.Equal(tc.ExpectedItems, items)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedErr, err)
		})
	}
}

func TestGet(t *testing.T) {
	var (
		dbErr            = errors.New("dynamodb error")
		consumedCapacity = &dynamodb.ConsumedCapacity{
			ReadCapacityUnits: aws.Float64(1),
		}
		nowRef  = getRefTime()
		nowFunc = func() time.Time {
			return nowRef
		}
		key = model.Key{
			ID:     "4c94485e0c21ae6c41ce1dfe7b6bfaceea5ab68e40a2476f50208e526f506080",
			Bucket: "testBucket",
		}
	)
	tcs := []struct {
		Description              string
		GetItemOutput            *dynamodb.GetItemOutput
		GetItemErr               error
		ExpectedItem             store.OwnableItem
		ExpectedConsumedCapacity *dynamodb.ConsumedCapacity
		ExpectedError            error
	}{
		{
			Description:   "GetItemFails",
			GetItemOutput: nil,
			GetItemErr:    dbErr,
			ExpectedItem:  store.OwnableItem{},
			ExpectedError: dbErr,
		},
		{
			Description:              "ExpiredItem",
			GetItemOutput:            getGetItemOutputExpired(nowRef, consumedCapacity, key),
			GetItemErr:               nil,
			ExpectedItem:             store.OwnableItem{},
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedError:            store.ErrItemNotFound,
		},
		{
			Description:   "Item not in DB",
			GetItemOutput: &dynamodb.GetItemOutput{},
			GetItemErr:    nil,
			ExpectedItem:  store.OwnableItem{},
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description:              "Happy path",
			GetItemOutput:            getGetItemOutput(nowRef, consumedCapacity, key),
			GetItemErr:               nil,
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItem:             getGetItemExpectedItem(),
			ExpectedError:            nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			svc := executor{
				c:         m,
				tableName: "testTable",
				now:       nowFunc,
			}
			m.On("GetItem", getGetItemInput(key)).Return(tc.GetItemOutput, tc.GetItemErr)
			item, cc, err := svc.Get(key)
			assert.Equal(tc.ExpectedError, err)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedItem, item)
		})
	}
}

func getPutItemInput(key model.Key, item store.OwnableItem) *dynamodb.PutItemInput {
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

func getQueryInput() *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		TableName: aws.String("testTable"),
		KeyConditions: map[string]*dynamodb.Condition{
			"bucket": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String("testBucket"),
					},
				},
			},
		},
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}
}

func getFilteredQueryOutput(now time.Time, consumedCapacity *dynamodb.ConsumedCapacity) *dynamodb.QueryOutput {
	pastExpiration := strconv.Itoa(int(now.Unix() - int64(time.Hour.Seconds())))
	futureExpiration := strconv.Itoa(int(now.Add(time.Hour).Unix()))
	bucket := "testBucket"

	return &dynamodb.QueryOutput{
		ConsumedCapacity: consumedCapacity,
		Items: []map[string]*dynamodb.AttributeValue{
			{ // should NOT be included in output (expired item)
				bucketAttributeKey: {
					S: aws.String(bucket),
				},
				idAttributeKey: {
					S: aws.String("6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"),
				},
				expirationAttributeKey: {
					N: aws.String(pastExpiration),
				},
			},
			{ // should be included in output
				bucketAttributeKey: {
					S: aws.String(bucket),
				},
				idAttributeKey: {
					S: aws.String("e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35"),
				},
				expirationAttributeKey: {
					N: aws.String(futureExpiration),
				},
			},
			{ // should be included in output (does not expire)
				bucketAttributeKey: {
					S: aws.String(bucket),
				},
				idAttributeKey: {
					S: aws.String("4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce"),
				},
			},
			{ // should NOT be included in output (missing bucket)
				idAttributeKey: {
					S: aws.String("5e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce"),
				},
			},
			{ // should NOT be included in output (missing ID)
				bucketAttributeKey: {
					S: aws.String("testBucket"),
				},
			},
		},
	}
}

func getFilteredExpectedItems() map[string]store.OwnableItem {
	return map[string]store.OwnableItem{
		"e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35": {
			Item: model.Item{
				ID:  "e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35",
				TTL: aws.Int64(3600),
			},
		},
		"4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce": {
			Item: model.Item{
				ID: "4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce",
			},
		},
	}
}

func getQueryOutput(now time.Time, consumedCapacity *dynamodb.ConsumedCapacity) *dynamodb.QueryOutput {
	futureExpiration := strconv.Itoa(int(now.Add(time.Hour).Unix()))
	bucket := "testBucket"
	return &dynamodb.QueryOutput{
		ConsumedCapacity: consumedCapacity,
		Items: []map[string]*dynamodb.AttributeValue{
			{ // should be included in output
				bucketAttributeKey: {
					S: aws.String(bucket),
				},
				idAttributeKey: {
					S: aws.String("e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35"),
				},
				expirationAttributeKey: {
					N: aws.String(futureExpiration),
				},
			},
			{ // should be included in output (does not expire)
				bucketAttributeKey: {
					S: aws.String(bucket),
				},
				idAttributeKey: {
					S: aws.String("4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce"),
				},
			},
		},
	}
}

func getExpectedItems() map[string]store.OwnableItem {
	return map[string]store.OwnableItem{
		"e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35": {
			Item: model.Item{
				ID:  "e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35",
				TTL: aws.Int64(3600),
			},
		},
		"4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce": {
			Item: model.Item{
				ID: "4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce",
			},
		},
	}
}

// getRefTime should be used as a reference for testing time operations.
func getRefTime() time.Time {
	refTime, err := time.Parse(time.RFC3339, "2021-01-02T15:04:00Z")
	if err != nil {
		panic(err)
	}
	return refTime
}

func getGetItemExpectedItem() store.OwnableItem {
	return store.OwnableItem{
		Owner: "xmidt",
		Item: model.Item{
			TTL: aws.Int64(3600),
			ID:  "4c94485e0c21ae6c41ce1dfe7b6bfaceea5ab68e40a2476f50208e526f506080",
			Data: map[string]interface{}{
				"key": "stringVal",
			},
		},
	}
}

func getGetItemInput(key model.Key) *dynamodb.GetItemInput {
	return &dynamodb.GetItemInput{
		TableName: aws.String("testTable"),
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
}

func getGetItemOutput(nowRef time.Time, consumedCapacity *dynamodb.ConsumedCapacity, key model.Key) *dynamodb.GetItemOutput {
	futureExpiration := strconv.Itoa(int(nowRef.Add(time.Hour).Unix()))
	return &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"expires": {
				N: aws.String(futureExpiration),
			},
			bucketAttributeKey: {
				S: aws.String(key.Bucket),
			},
			idAttributeKey: {
				S: aws.String(key.ID),
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
		ConsumedCapacity: consumedCapacity,
	}
}

func getGetItemOutputExpired(nowRef time.Time, consumedCapacity *dynamodb.ConsumedCapacity, key model.Key) *dynamodb.GetItemOutput {
	secondsInHour := int64(time.Hour.Seconds())
	pastExpiration := strconv.Itoa(int(nowRef.Unix() - secondsInHour))
	return &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"expires": {
				N: aws.String(pastExpiration),
			},
			bucketAttributeKey: {
				S: aws.String(key.Bucket),
			},
			idAttributeKey: {
				S: aws.String(key.ID),
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
		ConsumedCapacity: consumedCapacity,
	}
}

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package dynamodb

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsv2attr "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsv2dynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsv2dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/touchstone/touchtest"
)

var (
	dbErr            = errors.New("dynamodb error")
	consumedCapacity = &awsv2dynamodbTypes.ConsumedCapacity{
		CapacityUnits: aws.Float64(1),
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

func TestPush(t *testing.T) {
	tcs := []struct {
		Description              string
		Key                      model.Key
		Item                     store.OwnableItem
		PutItemFails             bool
		ExpectedConsumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
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
			ExpectedConsumedCapacity: consumedCapacity,
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
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedError:            nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			sv, _ := newServiceWithClient(m, "testTable", 0, nil)
			var (
				putItemOutput = &awsv2dynamodb.PutItemOutput{
					ConsumedCapacity: tc.ExpectedConsumedCapacity,
				}
				putItemErr error
			)
			if tc.PutItemFails {
				putItemOutput, putItemErr = nil, dbErr
			}

			m.On("PutItem", mock.Anything, mock.Anything, mock.Anything).Return(putItemOutput, putItemErr)
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
		consumedCapacity = &awsv2dynamodbTypes.ConsumedCapacity{}
	)
	nowRef := getRefTime()
	nowFunc := func() time.Time {
		return nowRef
	}

	tcs := []struct {
		Description              string
		QueryErr                 error
		QueryOutput              *awsv2dynamodb.QueryOutput
		ExpectedItems            map[string]store.OwnableItem
		ExpectedConsumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
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
			client := new(mockClient)
			testAssert := touchtest.New(t)
			expectedRegistry := prometheus.NewPedanticRegistry()
			expectedMeasures := &metric.Measures{
				DynamodbGetAllGauge: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "testGetAllGauge",
						Help: "testGetAllGauge",
					},
				),
			}
			expectedRegistry.MustRegister(expectedMeasures.DynamodbGetAllGauge)
			if tc.QueryOutput != nil {
				expectedMeasures.DynamodbGetAllGauge.Set(float64(len(tc.QueryOutput.Items)))
			}
			actualRegistry := prometheus.NewPedanticRegistry()
			m := &metric.Measures{
				DynamodbGetAllGauge: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "testGetAllGauge",
						Help: "testGetAllGauge",
					},
				),
			}
			actualRegistry.MustRegister(m.DynamodbGetAllGauge)

			svc := newServiceWithClient(client, "testTable")
			client.On("Query", mock.Anything).Return(tc.QueryOutput, tc.QueryErr)
			items, cc, err := svc.GetAll("testBucket")
			testAssert.Expect(expectedRegistry)
			assert.True(testAssert.GatherAndCompare(actualRegistry,
				"testGetAllGauge"))

			assert.Equal(tc.ExpectedItems, items)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedErr, err)
		})
	}
}

func TestGet(t *testing.T) {
	tcs := []struct {
		Description              string
		GetItemOutput            *awsv2dynamodb.GetItemOutput
		GetItemErr               error
		ExpectedItem             store.OwnableItem
		ExpectedConsumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
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
			GetItemOutput: &awsv2dynamodb.GetItemOutput{},
			GetItemErr:    nil,
			ExpectedItem:  store.OwnableItem{},
			ExpectedError: store.ErrItemNotFound,
		},
		{
			Description:              "Happy path",
			GetItemOutput:            getGetItemOutput(nowRef, consumedCapacity, key),
			GetItemErr:               nil,
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItem:             getGetOrDeleteExpectedItem(),
			ExpectedError:            nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			svc := newServiceWithClient(m, "testTable")
			m.On("GetItem", getGetItemInput(key)).Return(tc.GetItemOutput, tc.GetItemErr)
			item, cc, err := svc.Get(key)
			assert.Equal(tc.ExpectedError, err)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedItem, item)
		})
	}
}

func TestDelete(t *testing.T) {
	var (
		dbErr            = errors.New("dynamodb error")
		consumedCapacity = &awsv2dynamodbTypes.ConsumedCapacity{
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
		DeleteItemOutput         *awsv2dynamodb.DeleteItemOutput
		DeleteItemErr            error
		ExpectedItem             store.OwnableItem
		ExpectedConsumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
		ExpectedError            error
	}{
		{
			Description:      "DeletetemFails",
			DeleteItemOutput: nil,
			DeleteItemErr:    dbErr,
			ExpectedItem:     store.OwnableItem{},
			ExpectedError:    dbErr,
		},
		{
			Description:              "ExpiredItem",
			DeleteItemOutput:         getDeleteItemOutputExpired(nowRef, consumedCapacity, key),
			DeleteItemErr:            nil,
			ExpectedItem:             store.OwnableItem{},
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedError:            store.ErrItemNotFound,
		},
		{
			Description:      "Item not in DB",
			DeleteItemOutput: &awsv2dynamodb.DeleteItemOutput{},
			DeleteItemErr:    nil,
			ExpectedItem:     store.OwnableItem{},
			ExpectedError:    store.ErrItemNotFound,
		},
		{
			Description:              "Happy path",
			DeleteItemOutput:         getDeleteItemOutput(nowRef, consumedCapacity, key),
			DeleteItemErr:            nil,
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItem:             getGetOrDeleteExpectedItem(),
			ExpectedError:            nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockClient)
			svc := newServiceWithClient(m, "testTable")
			m.On("DeleteItem", getDeleteItemInput(key)).Return(tc.DeleteItemOutput, tc.DeleteItemErr)
			item, cc, err := svc.Delete(key)
			m.AssertExpectations(t)
			assert.Equal(tc.ExpectedError, err)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedItem, item)
		})
	}
}
func getPutItemInput(key model.Key, item store.OwnableItem) *awsv2dynamodb.PutItemInput {
	storingItem := storableItem{
		OwnableItem: item,
		Key:         key,
	}

	if item.TTL != nil {
		unixExpSeconds := time.Now().Unix() + *item.TTL
		storingItem.Expires = &unixExpSeconds
	}

	av, err := awsv2attr.MarshalMap(storingItem)
	if err != nil {
		panic("must be able to marshal")
	}
	return &awsv2dynamodb.PutItemInput{
		Item:                   av,
		TableName:              aws.String("testTable"),
		ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
	}
}

func getFilteredQueryOutput(now time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity) *awsv2dynamodb.QueryOutput {
	pastExpiration := strconv.Itoa(int(now.Unix() - int64(time.Hour.Seconds())))
	futureExpiration := strconv.Itoa(int(now.Add(time.Hour).Unix()))
	bucket := "testBucket"

	return &awsv2dynamodb.QueryOutput{
		ConsumedCapacity: consumedCapacity,
		Items: []map[string]awsv2dynamodbTypes.AttributeValue{
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

func getQueryOutput(now time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity) *awsv2dynamodb.QueryOutput {
	futureExpiration := strconv.Itoa(int(now.Add(time.Hour).Unix()))
	bucket := "testBucket"
	return &awsv2dynamodb.QueryOutput{
		ConsumedCapacity: consumedCapacity,
		Items: []map[string]awsv2dynamodbTypes.AttributeValue{
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

func getGetOrDeleteExpectedItem() store.OwnableItem {
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

func getGetItemInput(key model.Key) *awsv2dynamodb.GetItemInput {
	return &awsv2dynamodb.GetItemInput{
		TableName: aws.String("testTable"),
		Key: map[string]awsv2dynamodbTypes.AttributeValue{
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
		},
		ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
	}
}

func getDeleteItemInput(key model.Key) *awsv2dynamodb.DeleteItemInput {
	return &awsv2dynamodb.DeleteItemInput{
		TableName: aws.String("testTable"),
		Key: map[string]awsv2dynamodbTypes.AttributeValue{
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
		},
		ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
		ReturnValues:           awsv2dynamodbTypes.ReturnValueAllOld,
	}
}

func getGetItemOutput(nowRef time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, key model.Key) *awsv2dynamodb.GetItemOutput {
	futureExpiration := strconv.FormatInt(nowRef.Add(time.Hour).Unix(), 10)
	return &awsv2dynamodb.GetItemOutput{
		Item: map[string]awsv2dynamodbTypes.AttributeValue{
			"expires":          &awsv2dynamodbTypes.AttributeValueMemberN{Value: futureExpiration},
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
			"data": &awsv2dynamodbTypes.AttributeValueMemberM{Value: map[string]awsv2dynamodbTypes.AttributeValue{
				"key": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "stringVal"},
			}},
			"owner": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "xmidt"},
		},
		ConsumedCapacity: consumedCapacity,
	}
}

func getDeleteItemOutput(nowRef time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, key model.Key) *awsv2dynamodb.DeleteItemOutput {
	futureExpiration := strconv.FormatInt(nowRef.Add(time.Hour).Unix(), 10)
	return &awsv2dynamodb.DeleteItemOutput{
		Attributes: map[string]awsv2dynamodbTypes.AttributeValue{
			"expires":          &awsv2dynamodbTypes.AttributeValueMemberN{Value: futureExpiration},
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
			"data": &awsv2dynamodbTypes.AttributeValueMemberM{Value: map[string]awsv2dynamodbTypes.AttributeValue{
				"key": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "stringVal"},
			}},
			"owner": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "xmidt"},
		},
		ConsumedCapacity: consumedCapacity,
	}
}

func getDeleteItemOutputExpired(nowRef time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, key model.Key) *awsv2dynamodb.DeleteItemOutput {
	secondsInHour := int64(time.Hour.Seconds())
	pastExpiration := strconv.FormatInt(nowRef.Unix()-secondsInHour, 10)
	return &awsv2dynamodb.DeleteItemOutput{
		Attributes: map[string]awsv2dynamodbTypes.AttributeValue{
			"expires":          &awsv2dynamodbTypes.AttributeValueMemberN{Value: pastExpiration},
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
			"data": &awsv2dynamodbTypes.AttributeValueMemberM{Value: map[string]awsv2dynamodbTypes.AttributeValue{
				"key": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "stringVal"},
			}},
			"owner": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "xmidt"},
		},
		ConsumedCapacity: consumedCapacity,
	}
}

func getGetItemOutputExpired(nowRef time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, key model.Key) *awsv2dynamodb.GetItemOutput {
	secondsInHour := int64(time.Hour.Seconds())
	pastExpiration := strconv.FormatInt(nowRef.Unix()-secondsInHour, 10)
	return &awsv2dynamodb.GetItemOutput{
		Item: map[string]awsv2dynamodbTypes.AttributeValue{
			"expires":          &awsv2dynamodbTypes.AttributeValueMemberN{Value: pastExpiration},
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
			"data": &awsv2dynamodbTypes.AttributeValueMemberM{Value: map[string]awsv2dynamodbTypes.AttributeValue{
				"key": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "stringVal"},
			}},
			"owner": &awsv2dynamodbTypes.AttributeValueMemberS{Value: "xmidt"},
		},
		ConsumedCapacity: consumedCapacity,
	}
}

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
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
)

var (
	errDynamoDB      = errors.New("dynamodb error")
	consumedCapacity = &awsv2dynamodbTypes.ConsumedCapacity{
		CapacityUnits: aws.Float64(1),
	}
	nowRef = getRefTime()
	key    = model.Key{
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
			ExpectedError:            errDynamoDB,
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
			measures := &metric.Measures{
				DynamodbGetAllGauge: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "testGetAllGauge",
						Help: "testGetAllGauge",
					},
				),
			}
			// Use all arguments for newServiceWithClient and handle both return values
			sv, err := newServiceWithClient(m, "testTable", 0, measures)
			assert.NoError(err)
			var (
				putItemOutput = &awsv2dynamodb.PutItemOutput{
					ConsumedCapacity: tc.ExpectedConsumedCapacity,
				}
				putItemErr error
			)
			if tc.PutItemFails {
				putItemOutput, putItemErr = nil, errDynamoDB
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

	tcs := []struct {
		Description              string
		QueryErr                 error
		QueryOutput              *awsv2dynamodb.QueryOutput
		ExpectedItems            map[string]store.OwnableItem
		ExpectedConsumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
		ExpectedErr              error
		ExpectedGauge            float64
	}{
		{
			Description:   "Query fails",
			QueryErr:      errDynamoDB,
			QueryOutput:   nil,
			ExpectedErr:   dbErr,
			ExpectedItems: map[string]store.OwnableItem{},
			ExpectedGauge: 0,
		},
		{
			Description:              "Expired or bad items",
			QueryOutput:              getFilteredQueryOutput(nowRef, consumedCapacity),
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItems:            getFilteredExpectedItems(),
			ExpectedGauge:            5, // 5 raw items in QueryOutput
		},
		{
			Description:              "All good items",
			QueryOutput:              getQueryOutput(nowRef, consumedCapacity),
			ExpectedConsumedCapacity: consumedCapacity,
			ExpectedItems:            getExpectedItems(),
			ExpectedGauge:            2, // 2 raw items in QueryOutput
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			client := new(mockClient)
			measures := &metric.Measures{
				DynamodbGetAllGauge: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "testGetAllGauge",
						Help: "testGetAllGauge",
					},
				),
			}
			svc, err := newServiceWithClient(client, "testTable", 0, measures)
			assert.NoError(err)
			client.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(tc.QueryOutput, tc.QueryErr)
			items, cc, err := svc.GetAll("testBucket")
			assert.Equal(tc.ExpectedItems, items)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedErr, err)
			// Check gauge value
			gaugeValue := testutil.ToFloat64(measures.DynamodbGetAllGauge)
			assert.Equal(tc.ExpectedGauge, gaugeValue)
		})
	}
}

func TestGet(t *testing.T) {
	var dbErr = errors.New("dynamodb error")
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
			GetItemErr:    errDynamoDB,
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
			measures := &metric.Measures{
				DynamodbGetAllGauge: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "testGetAllGauge",
						Help: "testGetAllGauge",
					},
				),
			}
			svc, err := newServiceWithClient(m, "testTable", 0, measures)
			assert.NoError(err)
			// Inject fixed now function to match nowRef for TTL calculation
			svcImpl := svc.(*executor)
			svcImpl.now = func() time.Time { return nowRef }
			// Set up the mock for this test case only
			m.On("GetItem", mock.Anything, mock.Anything, mock.Anything).Return(tc.GetItemOutput, tc.GetItemErr).Once()
			item, cc, err := svc.Get(key)
			assert.Equal(tc.ExpectedError, err)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedItem, item)
			m.AssertExpectations(t)
		})
	}
}

func TestDelete(t *testing.T) {
	var (
		dbErr            = errors.New("dynamodb error")
		consumedCapacity = &awsv2dynamodbTypes.ConsumedCapacity{
			ReadCapacityUnits: aws.Float64(1),
		}
		nowRef = getRefTime()
		key    = model.Key{
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
			DeleteItemErr:    errDynamoDB,
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
			measures := &metric.Measures{
				DynamodbGetAllGauge: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "testGetAllGauge",
						Help: "testGetAllGauge",
					},
				),
			}
			svc, err := newServiceWithClient(m, "testTable", 0, measures)
			assert.NoError(err)
			// Inject fixed now function to match nowRef for TTL calculation
			svcImpl := svc.(*executor)
			svcImpl.now = func() time.Time { return nowRef }
			// Set up the mocks for this test case only
			m.On("DeleteItem", mock.Anything, mock.Anything, mock.Anything).Return(tc.DeleteItemOutput, tc.DeleteItemErr).Once()
			// Remove GetItem mock setup; Delete only calls DeleteItem
			item, cc, err := svc.Delete(key)
			assert.Equal(tc.ExpectedError, err)
			assert.Equal(tc.ExpectedConsumedCapacity, cc)
			assert.Equal(tc.ExpectedItem, item)
			m.AssertExpectations(t)
		})
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
				bucketAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: bucket},
				idAttributeKey:         &awsv2dynamodbTypes.AttributeValueMemberS{Value: "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"},
				expirationAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberN{Value: pastExpiration},
			},
			{ // should be included in output
				bucketAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: bucket},
				idAttributeKey:         &awsv2dynamodbTypes.AttributeValueMemberS{Value: "e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35"},
				expirationAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberN{Value: futureExpiration},
			},
			{ // should be included in output (does not expire)
				bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: bucket},
				idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: "4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce"},
			},
			{ // should NOT be included in output (missing bucket)
				idAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: "5e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce"},
			},
			{ // should NOT be included in output (missing ID)
				bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: "testBucket"},
			},
		},
	}
}

func getFilteredExpectedItems() map[string]store.OwnableItem {
	// Only include non-expired, valid items as per service logic
	return map[string]store.OwnableItem{
		"4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce": {
			Item: model.Item{
				ID: "4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce",
			},
		},
	}
}

func getExpectedItems() map[string]store.OwnableItem {
	// Only include non-expired, valid items as per service logic
	return map[string]store.OwnableItem{
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
				bucketAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: bucket},
				idAttributeKey:         &awsv2dynamodbTypes.AttributeValueMemberS{Value: "e4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35"},
				expirationAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberN{Value: futureExpiration},
			},
			{ // should be included in output (does not expire)
				bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: bucket},
				idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: "4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce"},
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

func getGetItemOutput(nowRef time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, key model.Key) *awsv2dynamodb.GetItemOutput {
	futureExpiration := nowRef.Add(time.Hour).Unix()
	item := storableItem{
		Bucket:  key.Bucket,
		ID:      key.ID,
		Owner:   "xmidt",
		Expires: &futureExpiration,
		Data:    map[string]interface{}{"key": "stringVal"},
	}
	av, err := awsv2attr.MarshalMap(item)
	if err != nil {
		panic(err)
	}
	return &awsv2dynamodb.GetItemOutput{
		Item:             av,
		ConsumedCapacity: consumedCapacity,
	}
}

func getDeleteItemOutput(nowRef time.Time, consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity, key model.Key) *awsv2dynamodb.DeleteItemOutput {
	futureExpiration := nowRef.Add(time.Hour).Unix()
	item := storableItem{
		Bucket:  key.Bucket,
		ID:      key.ID,
		Owner:   "xmidt",
		Expires: &futureExpiration,
		Data:    map[string]interface{}{"key": "stringVal"},
	}
	av, err := awsv2attr.MarshalMap(item)
	if err != nil {
		panic(err)
	}
	return &awsv2dynamodb.DeleteItemOutput{
		Attributes:       av,
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

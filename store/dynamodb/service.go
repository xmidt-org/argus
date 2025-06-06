// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package dynamodb

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsv2attr "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsv2dynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsv2dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
)

var (
	errNilMeasures = errors.New("measures cannot be nil")
)

// DynamoDBAPI defines the subset of the DynamoDB client used by executor, for mocking/testing.
type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *awsv2dynamodb.PutItemInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *awsv2dynamodb.GetItemInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.GetItemOutput, error)
	DeleteItem(ctx context.Context, params *awsv2dynamodb.DeleteItemInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.DeleteItemOutput, error)
	Query(ctx context.Context, params *awsv2dynamodb.QueryInput, optFns ...func(*awsv2dynamodb.Options)) (*awsv2dynamodb.QueryOutput, error)
}

// service defines the dynamodb specific DAO interface. It helps keeping middleware
// such as logging and instrumentation orthogonal to business logic.
type service interface {
	Push(key model.Key, item store.OwnableItem) (*awsv2dynamodbTypes.ConsumedCapacity, error)
	Get(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error)
	Delete(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error)
	GetAll(bucket string) (map[string]store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error)
}

// executor satisfies the service interface so dao can then adapt the outputs to match
// the Argus' abstract DAO.
type executor struct {
	c DynamoDBAPI

	// tableName is the name of the dynamodb table
	tableName string

	// getAllLimit is the maximum number of records to return for a GetAll
	getAllLimit int32

	now func() time.Time

	measures *metric.Measures
}

type storableItem struct {
	Bucket  string                 `json:"bucket" dynamodbav:"bucket"`
	ID      string                 `json:"id" dynamodbav:"id"`
	Owner   string                 `json:"owner" dynamodbav:"owner"`
	Expires *int64                 `json:"expires,omitempty" dynamodbav:"expires"`
	Data    map[string]interface{} `json:"data" dynamodbav:"data"`
	TTL     *int64                 `json:"ttl,omitempty" dynamodbav:"ttl"`
}

// Dynamo DB attribute keys
const (
	bucketAttributeKey     = "bucket"
	idAttributeKey         = "id"
	expirationAttributeKey = "expires"
)

func (d *executor) Push(key model.Key, item store.OwnableItem) (*awsv2dynamodbTypes.ConsumedCapacity, error) {
	storingItem := storableItem{
		Bucket: key.Bucket,
		ID:     key.ID,
		Owner:  item.Owner,
		Data:   item.Data,
		TTL:    item.TTL,
	}
	if item.TTL != nil {
		unixExpSeconds := time.Now().Unix() + *item.TTL
		storingItem.Expires = &unixExpSeconds
	}
	av, err := awsv2attr.MarshalMap(storingItem)
	if err != nil {
		return nil, err
	}
	input := &awsv2dynamodb.PutItemInput{
		Item:                   av,
		TableName:              &d.tableName,
		ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
	}
	result, err := d.c.PutItem(context.Background(), input)
	var consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
	if result != nil {
		consumedCapacity = result.ConsumedCapacity
	}
	if err != nil {
		return consumedCapacity, err
	}
	return consumedCapacity, nil
}

func (d *executor) executeGetOrDelete(key model.Key, delete bool) (*awsv2dynamodbTypes.ConsumedCapacity, map[string]awsv2dynamodbTypes.AttributeValue, error) {
	if delete {
		deleteInput := &awsv2dynamodb.DeleteItemInput{
			TableName: &d.tableName,
			Key: map[string]awsv2dynamodbTypes.AttributeValue{
				bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
				idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
			},
			ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
			ReturnValues:           awsv2dynamodbTypes.ReturnValueAllOld,
		}
		deleteOutput, err := d.c.DeleteItem(context.Background(), deleteInput)
		if err != nil {
			return nil, nil, err
		}
		return deleteOutput.ConsumedCapacity, deleteOutput.Attributes, nil
	}
	getInput := &awsv2dynamodb.GetItemInput{
		TableName: &d.tableName,
		Key: map[string]awsv2dynamodbTypes.AttributeValue{
			bucketAttributeKey: &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.Bucket},
			idAttributeKey:     &awsv2dynamodbTypes.AttributeValueMemberS{Value: key.ID},
		},
		ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
	}
	getOutput, err := d.c.GetItem(context.Background(), getInput)
	if err != nil {
		return nil, nil, err
	}
	return getOutput.ConsumedCapacity, getOutput.Item, nil
}

func (d *executor) getOrDelete(key model.Key, delete bool) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	consumedCapacity, attributes, err := d.executeGetOrDelete(key, delete)
	if err != nil {
		return store.OwnableItem{}, consumedCapacity, err
	}
	if len(attributes) == 0 {
		return store.OwnableItem{}, consumedCapacity, store.ErrItemNotFound
	}
	item := new(storableItem)
	err = awsv2attr.UnmarshalMap(attributes, item)
	if err != nil {
		return store.OwnableItem{}, consumedCapacity, err
	}
	if itemNotFound(item) {
		return store.OwnableItem{}, consumedCapacity, store.ErrItemNotFound
	}
	if item.Expires != nil {
		expiryTime := time.Unix(*item.Expires, 0)
		remainingTTLSeconds := int64(expiryTime.Sub(d.now()).Seconds())
		if remainingTTLSeconds < 1 {
			return store.OwnableItem{}, consumedCapacity, store.ErrItemNotFound
		}
		item.TTL = &remainingTTLSeconds
	}
	return store.OwnableItem{
		Owner: item.Owner,
		Item: model.Item{
			ID:   item.ID,
			Data: item.Data,
			TTL:  item.TTL,
		},
	}, consumedCapacity, nil
}

func (d *executor) Get(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	return d.getOrDelete(key, false)
}

func (d *executor) Delete(key model.Key) (store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	return d.getOrDelete(key, true)
}

func (d *executor) GetAll(bucket string) (map[string]store.OwnableItem, *awsv2dynamodbTypes.ConsumedCapacity, error) {
	result := map[string]store.OwnableItem{}
	now := strconv.FormatInt(d.now().Unix(), 10)
	input := &awsv2dynamodb.QueryInput{
		TableName: &d.tableName,
		IndexName: aws.String("Expires-index"),
		KeyConditions: map[string]awsv2dynamodbTypes.Condition{
			"bucket": {
				ComparisonOperator: awsv2dynamodbTypes.ComparisonOperatorEq,
				AttributeValueList: []awsv2dynamodbTypes.AttributeValue{
					&awsv2dynamodbTypes.AttributeValueMemberS{Value: bucket},
				},
			},
			"expires": {
				ComparisonOperator: awsv2dynamodbTypes.ComparisonOperatorGt,
				AttributeValueList: []awsv2dynamodbTypes.AttributeValue{
					&awsv2dynamodbTypes.AttributeValueMemberN{Value: now},
				},
			},
		},
		ReturnConsumedCapacity: awsv2dynamodbTypes.ReturnConsumedCapacityTotal,
	}
	if d.getAllLimit > 0 {
		input.Limit = &d.getAllLimit
	}
	queryResult, err := d.c.Query(context.Background(), input)
	var consumedCapacity *awsv2dynamodbTypes.ConsumedCapacity
	if queryResult != nil {
		consumedCapacity = queryResult.ConsumedCapacity
	}
	if err != nil {
		return map[string]store.OwnableItem{}, consumedCapacity, err
	}
	d.measures.DynamodbGetAllGauge.Set(float64(len(queryResult.Items)))
	for _, i := range queryResult.Items {
		item := new(storableItem)
		err = awsv2attr.UnmarshalMap(i, item)
		if err != nil {
			continue
		}
		if itemNotFound(item) {
			continue
		}
		if item.Expires != nil {
			expiryTime := time.Unix(*item.Expires, 0)
			remainingTTLSeconds := int64(expiryTime.Sub(d.now()).Seconds())
			if remainingTTLSeconds < 1 {
				continue
			}
			item.TTL = &remainingTTLSeconds
		}
		result[item.ID] = store.OwnableItem{
			Owner: item.Owner,
			Item: model.Item{
				ID:   item.ID,
				Data: item.Data,
				TTL:  item.TTL,
			},
		}
	}
	return result, consumedCapacity, nil
}

func itemNotFound(item *storableItem) bool {
	return item.Bucket == "" || item.ID == ""
}

func newServiceWithClient(client DynamoDBAPI, tableName string, getAllLimit int32, measures *metric.Measures) (service, error) {
	if measures == nil {
		return nil, errNilMeasures
	}
	return &executor{
		c:           client,
		tableName:   tableName,
		getAllLimit: getAllLimit,
		now:         time.Now,
		measures:    measures,
	}, nil
}

func newService(awsCfg aws.Config, config Config, getAllLimit int32, measures *metric.Measures) (service, error) {
	if measures == nil {
		return nil, errNilMeasures
	}

	client := awsv2dynamodb.NewFromConfig(awsCfg, func(o *awsv2dynamodb.Options) {
		if config.Endpoint != "" {
			o.BaseEndpoint = &config.Endpoint
		}
	})

	return newServiceWithClient(client, config.Table, getAllLimit, measures)
}

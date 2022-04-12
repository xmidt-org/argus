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

package dynamodb

import (
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
)

var (
	errNilMeasures = errors.New("measures cannot be nil")
)

// client captures the methods of interest from the dynamoDB API. This
// should help mock API calls as well.
type client interface {
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	DeleteItem(*dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
	Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
}

// service defines the dynamodb specific DAO interface. It helps keeping middleware
// such as logging and instrumentation orthogonal to business logic.
type service interface {
	Push(key model.Key, item store.OwnableItem) (*dynamodb.ConsumedCapacity, error)
	Get(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error)
	Delete(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error)
	GetAll(bucket string) (map[string]store.OwnableItem, *dynamodb.ConsumedCapacity, error)
}

// executor satisfies the service interface so dao can then adapt the outputs to match
// the Argus' abstract DAO.
type executor struct {
	// c is the dynamodb client
	c client

	// tableName is the name of the dynamodb table
	tableName string

	// getAllLimit is the maximum number of records to return for a GetAll
	getAllLimit int64

	now func() time.Time

	measures *metric.Measures
}

type storableItem struct {
	store.OwnableItem
	Expires *int64 `json:"expires,omitempty"`
	model.Key
}

// Dynamo DB attribute keys
const (
	bucketAttributeKey     = "bucket"
	idAttributeKey         = "id"
	expirationAttributeKey = "expires"
)

func (d *executor) Push(key model.Key, item store.OwnableItem) (*dynamodb.ConsumedCapacity, error) {
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
		return nil, err
	}
	input := &dynamodb.PutItemInput{
		Item:                   av,
		TableName:              aws.String(d.tableName),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}

	result, err := d.c.PutItem(input)
	var consumedCapacity *dynamodb.ConsumedCapacity
	if result != nil {
		consumedCapacity = result.ConsumedCapacity
	}

	if err != nil {
		return consumedCapacity, err
	}
	return consumedCapacity, nil
}
func (d *executor) executeGetOrDelete(key model.Key, delete bool) (*dynamodb.ConsumedCapacity, map[string]*dynamodb.AttributeValue, error) {
	if delete {
		deleteInput := &dynamodb.DeleteItemInput{
			TableName: aws.String(d.tableName),
			Key: map[string]*dynamodb.AttributeValue{
				bucketAttributeKey: {
					S: aws.String(key.Bucket),
				},
				idAttributeKey: {
					S: aws.String(key.ID),
				},
			},
			ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
			ReturnValues:           aws.String(dynamodb.ReturnValueAllOld),
		}
		deleteOutput, err := d.c.DeleteItem(deleteInput)
		if err != nil {
			return nil, nil, err
		}
		return deleteOutput.ConsumedCapacity, deleteOutput.Attributes, nil
	}
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
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
	getOutput, err := d.c.GetItem(getInput)
	if err != nil {
		return nil, nil, err
	}
	return getOutput.ConsumedCapacity, getOutput.Item, nil
}
func (d *executor) getOrDelete(key model.Key, delete bool) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	consumedCapacity, attributes, err := d.executeGetOrDelete(key, delete)
	if err != nil {
		return store.OwnableItem{}, consumedCapacity, err
	}
	item := new(storableItem)
	err = dynamodbattribute.UnmarshalMap(attributes, item)
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

	item.OwnableItem.ID = key.ID

	return item.OwnableItem, consumedCapacity, err

}

func (d *executor) Get(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	return d.getOrDelete(key, false)
}

func (d *executor) Delete(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	return d.getOrDelete(key, true)
}

//TODO: For data >= 1MB, we'll need to handle pagination
func (d *executor) GetAll(bucket string) (map[string]store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	result := map[string]store.OwnableItem{}
	now := strconv.Itoa(int(d.now().Unix()))
	input := &dynamodb.QueryInput{
		TableName: aws.String(d.tableName),
		IndexName: aws.String("Expires-index"),
		KeyConditions: map[string]*dynamodb.Condition{
			"bucket": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: &bucket,
					},
				},
			},
			"Expires-index": {
				ComparisonOperator: aws.String("GT"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						N: &now,
					},
				},
			},
		},
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	}
	if d.getAllLimit > 0 {
		input.Limit = &d.getAllLimit
	}
	queryResult, err := d.c.Query(input)

	var consumedCapacity *dynamodb.ConsumedCapacity
	if queryResult != nil {
		consumedCapacity = queryResult.ConsumedCapacity
	}
	if err != nil {
		return map[string]store.OwnableItem{}, consumedCapacity, err
	}
	d.measures.DynamodbGetAllGauge.Set(float64(len(queryResult.Items)))

	for _, i := range queryResult.Items {
		item := new(storableItem)
		err = dynamodbattribute.UnmarshalMap(i, item)
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
		item.OwnableItem.ID = item.Key.ID

		result[item.Key.ID] = item.OwnableItem
	}
	return result, consumedCapacity, nil
}

func itemNotFound(item *storableItem) bool {
	return item.Key.Bucket == "" || item.Key.ID == ""
}

func newService(config aws.Config, awsProfile string, tableName string, getAllLimit int64, measures *metric.Measures) (service, error) {
	if measures == nil {
		return nil, errNilMeasures
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            config,
		Profile:           awsProfile,
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return nil, err
	}

	return &executor{
		c:           dynamodb.New(sess),
		tableName:   tableName,
		getAllLimit: getAllLimit,
		now:         time.Now,
		measures:    measures,
	}, nil
}

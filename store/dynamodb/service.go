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
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/httpaux"
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

var (
	errDefaultDynamoDBFailure = httpaux.Error{
		Err:  errors.New("dynamodb operation failed"),
		Code: http.StatusInternalServerError,
	}
	errBadRequest = httpaux.Error{
		Err:  errors.New("bad request to dynamodb"),
		Code: http.StatusBadRequest,
	}
)

func handleClientError(err error) error {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		if awsErr.Code() == dynamodb.ErrCodeTransactionCanceledException {
			if strings.Contains(awsErr.Message(), "ValidationException") {
				return store.SanitizedError{Err: err, ErrHTTP: errBadRequest}
			}
		}
	}
	return store.SanitizedError{Err: err, ErrHTTP: errDefaultDynamoDBFailure}
}

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
		return consumedCapacity, handleClientError(err)
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
		return store.OwnableItem{}, consumedCapacity, handleClientError(err)
	}
	item := new(storableItem)
	err = dynamodbattribute.UnmarshalMap(attributes, item)
	if err != nil {
		return store.OwnableItem{}, consumedCapacity, err
	}

	if itemNotFound(item) {
		return item.OwnableItem, consumedCapacity, store.KeyNotFoundError{Key: key}
	}

	if item.Expires != nil {
		remainingTTLSeconds := int64(time.Unix(*item.Expires, 0).Sub(time.Now()).Seconds())
		if remainingTTLSeconds < 1 {
			return item.OwnableItem, consumedCapacity, store.KeyNotFoundError{Key: key}
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
	queryResult, err := d.c.Query(&dynamodb.QueryInput{
		TableName: aws.String(d.tableName),
		KeyConditions: map[string]*dynamodb.Condition{
			"bucket": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: &bucket,
					},
				},
			},
		},
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	})

	var consumedCapacity *dynamodb.ConsumedCapacity
	if queryResult != nil {
		consumedCapacity = queryResult.ConsumedCapacity
	}
	if err != nil {
		return map[string]store.OwnableItem{}, consumedCapacity, handleClientError(err)
	}

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
			remainingTTLSeconds := int64(time.Until(time.Unix(*item.Expires, 0)).Seconds())
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

func newService(config aws.Config, awsProfile string, tableName string, logger log.Logger) (service, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            config,
		Profile:           awsProfile,
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return nil, err
	}

	return &executor{
		c:         dynamodb.New(sess),
		tableName: tableName,
	}, nil
}

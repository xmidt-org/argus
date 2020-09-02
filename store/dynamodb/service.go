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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
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
	c         client
	tableName string
}

type element struct {
	store.OwnableItem
	Expires int64 `json:"expires"`
	model.Key
}

var retryableAWSCodes = map[string]bool{
	dynamodb.ErrCodeProvisionedThroughputExceededException: true,
	dynamodb.ErrCodeInternalServerError:                    true,
}

func handleClientError(err error) error {
	awsErr, ok := err.(awserr.Error)
	if !ok {
		return store.InternalError{Reason: err.Error(), Retryable: false}
	}
	msg := awsErr.Code()
	retryable := retryableAWSCodes[msg]

	return store.InternalError{Reason: msg, Retryable: retryable}
}

func (d *executor) Push(key model.Key, item store.OwnableItem) (*dynamodb.ConsumedCapacity, error) {
	expirableItem := element{
		OwnableItem: item,
		Expires:     time.Now().Unix() + item.TTL,
		Key:         key,
	}
	av, err := dynamodbattribute.MarshalMap(expirableItem)
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

func (d *executor) Get(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	result, err := d.c.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"bucket": {
				S: aws.String(key.Bucket),
			},
			"id": {
				S: aws.String(key.ID),
			},
		},
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	})

	var consumedCapacity *dynamodb.ConsumedCapacity
	if result != nil {
		consumedCapacity = result.ConsumedCapacity
	}
	if err != nil {
		return store.OwnableItem{}, consumedCapacity, handleClientError(err)
	}
	var expirableItem element
	err = dynamodbattribute.UnmarshalMap(result.Item, &expirableItem)
	expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())
	if expirableItem.Key.Bucket == "" || expirableItem.Key.ID == "" {
		return expirableItem.OwnableItem, consumedCapacity, store.KeyNotFoundError{Key: key}
	}
	return expirableItem.OwnableItem, consumedCapacity, err
}

func (d *executor) Delete(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	result, err := d.c.DeleteItem(&dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"bucket": {
				S: aws.String(key.Bucket),
			},
			"id": {
				S: aws.String(key.ID),
			},
		},
		ReturnValues:           aws.String("ALL_OLD"),
		TableName:              aws.String(d.tableName),
		ReturnConsumedCapacity: aws.String(dynamodb.ReturnConsumedCapacityTotal),
	})
	var consumedCapacity *dynamodb.ConsumedCapacity
	if result != nil {
		consumedCapacity = result.ConsumedCapacity
	}
	if err != nil {
		return store.OwnableItem{}, consumedCapacity, handleClientError(err)
	}

	var expirableItem element
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &expirableItem)
	expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())
	if expirableItem.Key.Bucket == "" || expirableItem.Key.ID == "" {
		return expirableItem.OwnableItem, result.ConsumedCapacity, store.KeyNotFoundError{Key: key}
	}
	return expirableItem.OwnableItem, result.ConsumedCapacity, err
}

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
		var expirableItem element
		err = dynamodbattribute.UnmarshalMap(i, &expirableItem)
		if err != nil {
			//logging.Error(d.logger).Log(logging.MessageKey(), "failed to unmarshal item", logging.ErrorKey(), err)
			continue
		}
		expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())

		result[expirableItem.Key.ID] = expirableItem.OwnableItem
	}
	return result, queryResult.ConsumedCapacity, nil
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

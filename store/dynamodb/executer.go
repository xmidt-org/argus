package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/webpa-common/logging"
	"time"
)

type dynamoDBExecutor struct {
	client    *dynamodb.DynamoDB
	logger    log.Logger
	tableName string
}

type dynamoElement struct {
	store.OwnableItem
	Expires int64 `json:"expires"`
	model.Key
}

type storeConsumable interface {
	Push(key model.Key, item store.OwnableItem) (*dynamodb.ConsumedCapacity, error)
	Get(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error)
	Delete(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error)
	GetAll(bucket string) (map[string]store.OwnableItem, *dynamodb.ConsumedCapacity, error)
}

func (d *dynamoDBExecutor) Push(key model.Key, item store.OwnableItem) (*dynamodb.ConsumedCapacity, error) {
	expirableItem := dynamoElement{
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

	result, err := d.client.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				return result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: true}
			case dynamodb.ErrCodeRequestLimitExceeded:
				return result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			case dynamodb.ErrCodeInternalServerError:
				return result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: true}
			default:
				return result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			}
		}
		return result.ConsumedCapacity, store.InternalError{Reason: err, Retryable: false}
	}
	return result.ConsumedCapacity, nil
}

func (d *dynamoDBExecutor) Get(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	result, err := d.client.GetItem(&dynamodb.GetItemInput{
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
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: true}
			case dynamodb.ErrCodeResourceNotFoundException:
				return store.OwnableItem{}, result.ConsumedCapacity, store.KeyNotFoundError{Key: key}
			case dynamodb.ErrCodeRequestLimitExceeded:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			case dynamodb.ErrCodeInternalServerError:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			default:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			}
		}
		return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: err, Retryable: false}
	}
	expirableItem := dynamoElement{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &expirableItem)
	expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())
	if expirableItem.Key.Bucket == "" || expirableItem.Key.ID == "" {
		return expirableItem.OwnableItem, result.ConsumedCapacity, store.KeyNotFoundError{Key: key}
	}
	return expirableItem.OwnableItem, result.ConsumedCapacity, err
}

func (d *dynamoDBExecutor) Delete(key model.Key) (store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	result, err := d.client.DeleteItem(&dynamodb.DeleteItemInput{

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
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: true}
			case dynamodb.ErrCodeResourceNotFoundException:
				return store.OwnableItem{}, result.ConsumedCapacity, store.KeyNotFoundError{Key: key}
			case dynamodb.ErrCodeRequestLimitExceeded:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			case dynamodb.ErrCodeInternalServerError:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			default:
				return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			}
		}
		return store.OwnableItem{}, result.ConsumedCapacity, store.InternalError{Reason: err, Retryable: false}
	}
	expirableItem := dynamoElement{}
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &expirableItem)
	expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())
	if expirableItem.Key.Bucket == "" || expirableItem.Key.ID == "" {
		return expirableItem.OwnableItem, result.ConsumedCapacity, store.KeyNotFoundError{Key: key}
	}
	return expirableItem.OwnableItem, result.ConsumedCapacity, err
}

func (d *dynamoDBExecutor) GetAll(bucket string) (map[string]store.OwnableItem, *dynamodb.ConsumedCapacity, error) {
	result := map[string]store.OwnableItem{}

	queryResult, err := d.client.Query(&dynamodb.QueryInput{
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
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				return map[string]store.OwnableItem{}, queryResult.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: true}
			case dynamodb.ErrCodeResourceNotFoundException:
				return map[string]store.OwnableItem{}, queryResult.ConsumedCapacity, store.KeyNotFoundError{Key: model.Key{Bucket: bucket}}
			case dynamodb.ErrCodeRequestLimitExceeded:
				return map[string]store.OwnableItem{}, queryResult.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			case dynamodb.ErrCodeInternalServerError:
				return map[string]store.OwnableItem{}, queryResult.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			default:
				return map[string]store.OwnableItem{}, queryResult.ConsumedCapacity, store.InternalError{Reason: aerr.Error(), Retryable: false}
			}
		}
		return map[string]store.OwnableItem{}, queryResult.ConsumedCapacity, store.InternalError{Reason: err, Retryable: false}
	}
	for _, i := range queryResult.Items {
		expirableItem := dynamoElement{}
		err = dynamodbattribute.UnmarshalMap(i, &expirableItem)
		if err != nil {
			logging.Error(d.logger).Log(logging.MessageKey(), "failed to unmarshal item", logging.ErrorKey(), err)
			continue
		}
		expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())

		result[expirableItem.Key.ID] = expirableItem.OwnableItem
	}
	return result, queryResult.ConsumedCapacity, nil
}

func createDynamoDBexecutor(config aws.Config, awsProfile string, tableName string, logger log.Logger) (storeConsumable, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            config,
		Profile:           awsProfile,
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return nil, err
	}

	return &dynamoDBExecutor{
		client:    dynamodb.New(sess),
		logger:    logger,
		tableName: tableName,
	}, nil
}

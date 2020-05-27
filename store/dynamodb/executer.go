package dynamodb

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/webpa-common/logging"
	"time"
)

var noDataResponse = errors.New("no data from query")

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

func (d *dynamoDBExecutor) Push(key model.Key, item store.OwnableItem) error {
	expirableItem := dynamoElement{
		OwnableItem: item,
		Expires:     time.Now().Unix() + item.TTL,
		Key:         key,
	}
	av, err := dynamodbattribute.MarshalMap(expirableItem)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(d.tableName),
	}

	_, err = d.client.PutItem(input)
	return err
}

func (d *dynamoDBExecutor) Get(key model.Key) (store.OwnableItem, error) {
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
	})
	if err != nil {
		return store.OwnableItem{}, err
	}
	expirableItem := dynamoElement{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &expirableItem)
	expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())
	if expirableItem.Key.Bucket == "" || expirableItem.Key.ID == "" {
		return expirableItem.OwnableItem, noDataResponse
	}
	return expirableItem.OwnableItem, err
}

func (d *dynamoDBExecutor) Delete(key model.Key) (store.OwnableItem, error) {
	result, err := d.client.DeleteItem(&dynamodb.DeleteItemInput{

		Key: map[string]*dynamodb.AttributeValue{
			"bucket": {
				S: aws.String(key.Bucket),
			},
			"id": {
				S: aws.String(key.ID),
			},
		},
		ReturnValues: aws.String("ALL_OLD"),
		TableName:    aws.String(d.tableName),
	})
	if err != nil {
		return store.OwnableItem{}, err
	}
	expirableItem := dynamoElement{}
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &expirableItem)
	expirableItem.OwnableItem.TTL = int64(time.Unix(expirableItem.Expires, 0).Sub(time.Now()).Seconds())
	if expirableItem.Key.Bucket == "" || expirableItem.Key.ID == "" {
		return expirableItem.OwnableItem, noDataResponse
	}
	return expirableItem.OwnableItem, err
}

func (d *dynamoDBExecutor) GetAll(bucket string) (map[string]store.OwnableItem, error) {
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
	})
	if err != nil {
		return result, err
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
	return result, nil
}

func createDynamoDBexecutor(config aws.Config, awsProfile string, tableName string, logger log.Logger) (store.S, error) {
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

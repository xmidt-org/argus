package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/themis/config"
	"github.com/xmidt-org/webpa-common/logging"
)

const (
	DynamoDB = "dynamo"

	defaultTable        = "gifnoc"
	defaultMaxRetries   = 3
	defaultWaitTimeMult = 1
)

type Config struct {
	Table      string
	Endpoint   string
	Region     string
	MaxRetries int
	AccessKey  string
	SecretKey  string
}

type DynamoClient struct {
	client   store.S
	config   Config
	logger   log.Logger
	measures metric.Measures
}

func ProvideDynamodDB(unmarshaller config.Unmarshaller, measures metric.Measures, logger log.Logger) (store.S, error) {
	var config Config
	err := unmarshaller.UnmarshalKey(DynamoDB, &config)
	if err != nil {
		return nil, err
	}
	validateConfig(config)
	awsConfig := *aws.NewConfig().
		WithEndpoint(config.Endpoint).
		WithUseDualStack(true).
		WithMaxRetries(config.MaxRetries).
		WithCredentialsChainVerboseErrors(true).
		WithRegion(config.Region).
		WithCredentials(credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     config.AccessKey,
			SecretAccessKey: config.SecretKey,
		}))

	executer, err := createDynamoDBexecutor(awsConfig, "", config.Table, func(consumedCapacity dynamodb.ConsumedCapacity, action string) {
		logging.Debug(logger).Log(logging.MessageKey(), "Updating consumed capacity", "consumed", consumedCapacity)
		if consumedCapacity.ReadCapacityUnits != nil {
			measures.ConsumedReadCapacityCount.With(store.TypeLabel, action).Add(*consumedCapacity.ReadCapacityUnits)
		}
		if consumedCapacity.WriteCapacityUnits != nil {
			measures.ConsumedWriteCapacityCount.With(store.TypeLabel, action).Add(*consumedCapacity.WriteCapacityUnits)
		}
	}, logger)
	if err != nil {
		return nil, err
	}
	return &DynamoClient{
		client:   executer,
		config:   config,
		measures: measures,
		logger:   logger,
	}, nil
}

func (s *DynamoClient) Push(key model.Key, item store.OwnableItem) error {
	err := s.client.Push(key, item)
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.InsertType).Add(1.0)
		return err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.InsertType).Add(1.0)
	return nil
}

func (s *DynamoClient) Get(key model.Key) (store.OwnableItem, error) {
	item, err := s.client.Get(key)
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.ReadType).Add(1.0)
		return item, err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.ReadType).Add(1.0)
	return item, nil
}

func (s *DynamoClient) Delete(key model.Key) (store.OwnableItem, error) {
	item, err := s.client.Delete(key)
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.DeleteType).Add(1.0)
		return item, err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.DeleteType).Add(1.0)
	return item, err
}

func (s *DynamoClient) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	item, err := s.client.GetAll(bucket)
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.ReadType).Add(1.0)
		return item, err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.ReadType).Add(1.0)
	return item, err
}

func validateConfig(config Config) {

	if config.Table == "" {
		config.Table = defaultTable
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultMaxRetries
	}
}

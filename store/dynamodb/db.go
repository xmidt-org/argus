package dynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"
	"github.com/xmidt-org/themis/config"
	"github.com/xmidt-org/webpa-common/logging"
)

const (
	DynamoDB = "dynamo"

	defaultTable      = "gifnoc"
	defaultMaxRetries = 3
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
	client   storeConsumable
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

	executer, err := createDynamoDBexecutor(awsConfig, "", config.Table, logger)
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
	consumedCapacity, err := s.client.Push(key, item)
	if consumedCapacity != nil {
		logging.Debug(s.logger).Log(logging.MessageKey(), "Updating consumed capacity", "consumed", consumedCapacity)
		if consumedCapacity.CapacityUnits != nil {
			s.measures.CapacityUnitConsumedCount.With(store.TypeLabel, store.InsertType).Add(*consumedCapacity.CapacityUnits)
		}
		if consumedCapacity.ReadCapacityUnits != nil {
			s.measures.ReadCapacityUnitConsumedCount.With(store.TypeLabel, store.InsertType).Add(*consumedCapacity.ReadCapacityUnits)
		}
		if consumedCapacity.WriteCapacityUnits != nil {
			s.measures.WriteCapacityUnitConsumedCount.With(store.TypeLabel, store.InsertType).Add(*consumedCapacity.WriteCapacityUnits)
		}

	}
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.InsertType).Add(1.0)
		return err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.InsertType).Add(1.0)
	return nil
}

func (s *DynamoClient) Get(key model.Key) (store.OwnableItem, error) {
	item, consumedCapacity, err := s.client.Get(key)
	if consumedCapacity != nil {
		logging.Debug(s.logger).Log(logging.MessageKey(), "Updating consumed capacity", "consumed", consumedCapacity)
		if consumedCapacity.CapacityUnits != nil {
			s.measures.CapacityUnitConsumedCount.With(store.TypeLabel, store.ReadType).Add(*consumedCapacity.CapacityUnits)
		}
		if consumedCapacity.ReadCapacityUnits != nil {
			s.measures.ReadCapacityUnitConsumedCount.With(store.TypeLabel, store.ReadType).Add(*consumedCapacity.ReadCapacityUnits)
		}
		if consumedCapacity.WriteCapacityUnits != nil {
			s.measures.WriteCapacityUnitConsumedCount.With(store.TypeLabel, store.ReadType).Add(*consumedCapacity.WriteCapacityUnits)
		}
	}
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.ReadType).Add(1.0)
		return item, err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.ReadType).Add(1.0)
	return item, nil
}

func (s *DynamoClient) Delete(key model.Key) (store.OwnableItem, error) {
	item, consumedCapacity, err := s.client.Delete(key)
	if consumedCapacity != nil {
		logging.Debug(s.logger).Log(logging.MessageKey(), "Updating consumed capacity", "consumed", consumedCapacity)
		if consumedCapacity.CapacityUnits != nil {
			s.measures.CapacityUnitConsumedCount.With(store.TypeLabel, store.DeleteType).Add(*consumedCapacity.CapacityUnits)
		}
		if consumedCapacity.ReadCapacityUnits != nil {
			s.measures.ReadCapacityUnitConsumedCount.With(store.TypeLabel, store.DeleteType).Add(*consumedCapacity.ReadCapacityUnits)
		}
		if consumedCapacity.WriteCapacityUnits != nil {
			s.measures.WriteCapacityUnitConsumedCount.With(store.TypeLabel, store.DeleteType).Add(*consumedCapacity.WriteCapacityUnits)
		}
	}
	if err != nil {
		s.measures.SQLQueryFailureCount.With(store.TypeLabel, store.DeleteType).Add(1.0)
		return item, err
	}
	s.measures.SQLQuerySuccessCount.With(store.TypeLabel, store.DeleteType).Add(1.0)
	return item, err
}

func (s *DynamoClient) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	item, consumedCapacity, err := s.client.GetAll(bucket)
	if consumedCapacity != nil {
		logging.Debug(s.logger).Log(logging.MessageKey(), "Updating consumed capacity", "consumed", consumedCapacity)
		if consumedCapacity.CapacityUnits != nil {
			s.measures.CapacityUnitConsumedCount.With(store.TypeLabel, store.ReadType).Add(*consumedCapacity.CapacityUnits)
		}
		if consumedCapacity.ReadCapacityUnits != nil {
			s.measures.ReadCapacityUnitConsumedCount.With(store.TypeLabel, store.ReadType).Add(*consumedCapacity.ReadCapacityUnits)
		}
		if consumedCapacity.WriteCapacityUnits != nil {
			s.measures.WriteCapacityUnitConsumedCount.With(store.TypeLabel, store.ReadType).Add(*consumedCapacity.WriteCapacityUnits)
		}
	}
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

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

package cassandra

import (
	"context"
	"errors"
	"time"

	"emperror.dev/emperror"
	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/argus/store/db/metric"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	Yugabyte = "yugabyte"

	defaultOpTimeout             = time.Duration(10) * time.Second
	defaultDatabase              = "argus"
	defaultNumRetries            = 0
	defaultWaitTimeMult          = 1
	defaultMaxNumberConnsPerHost = 2
)

type Config struct {
	// Hosts to  connect to. Must have at least one
	Hosts []string

	// Database aka Keyspace for cassandra
	Database string

	// OpTimeout
	OpTimeout time.Duration

	// SSLRootCert used for enabling tls to the cluster. SSLKey, and SSLCert must also be set.
	SSLRootCert string
	// SSLKey used for enabling tls to the cluster. SSLRootCert, and SSLCert must also be set.
	SSLKey string
	// SSLCert used for enabling tls to the cluster. SSLRootCert, and SSLRootCert must also be set.
	SSLCert string
	// If you want to verify the hostname and server cert (like a wildcard for cass cluster) then you should turn this on
	// This option is basically the inverse of InSecureSkipVerify
	// See InSecureSkipVerify in http://golang.org/pkg/crypto/tls/ for more info
	EnableHostVerification bool

	// Username to authenticate into the cluster. Password must also be provided.
	Username string
	// Password to authenticate into the cluster. Username must also be provided.
	Password string

	// NumRetries for connecting to the db
	NumRetries int

	// WaitTimeMult the amount of time to wait before retrying to connect to the db
	WaitTimeMult time.Duration

	// MaxConnsPerHost max number of connections per host
	MaxConnsPerHost int
}

type Client struct {
	client   dbStore
	config   Config
	logger   *zap.Logger
	measures metric.Measures
}

func NewCassandra(config Config, metricsIn metric.Measures, lc fx.Lifecycle, logger *zap.Logger) (store.S, error) {
	client, err := CreateCassandraClient(config, metricsIn)
	ticker := doEvery(time.Second*5, func(_ time.Time) {
		err := client.Ping()
		if err != nil {
			logger.Error("ping failed", zap.Error(err))
		}
	})
	if err != nil {
		return client, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(context context.Context) error {
			ticker.Stop()
			client.Close()
			return nil
		},
	})
	return client, nil
}

func doEvery(d time.Duration, f func(time.Time)) *time.Ticker {
	ticker := time.NewTicker(d)
	go func() {
		for x := range ticker.C {
			f(x)
		}
	}()
	return ticker
}

func CreateCassandraClient(config Config, measures metric.Measures) (*Client, error) {
	if len(config.Hosts) == 0 {
		return nil, errors.New("number of hosts must be > 0")
	}

	validateConfig(&config)

	clusterConfig := gocql.NewCluster(config.Hosts...)
	clusterConfig.Consistency = gocql.LocalQuorum
	clusterConfig.Keyspace = config.Database
	clusterConfig.Timeout = config.OpTimeout
	// let retry package handle it
	clusterConfig.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 1}
	// setup ssl
	if config.SSLRootCert != "" && config.SSLCert != "" && config.SSLKey != "" {
		clusterConfig.SslOpts = &gocql.SslOptions{
			CertPath:               config.SSLCert,
			KeyPath:                config.SSLKey,
			CaPath:                 config.SSLRootCert,
			EnableHostVerification: config.EnableHostVerification,
		}
	}
	// setup authentication
	if config.Username != "" && config.Password != "" {
		clusterConfig.Authenticator = gocql.PasswordAuthenticator{
			Username: config.Username,
			Password: config.Password,
		}
	}

	session, err := connect(clusterConfig)

	// retry if it fails
	waitTime := 1 * time.Second
	for attempt := 0; attempt < config.NumRetries && err != nil; attempt++ {
		time.Sleep(waitTime)
		session, err = connect(clusterConfig)
		waitTime = waitTime * config.WaitTimeMult
	}
	if err != nil {
		return nil, emperror.WrapWith(err, "Connecting to database failed", "hosts", config.Hosts)
	}

	return &Client{
		client:   session,
		config:   config,
		measures: measures,
	}, nil
}

func (s *Client) Push(key model.Key, item store.OwnableItem) error {
	err := s.client.Push(key, item)
	if err != nil {
		s.measures.Queries.With(prometheus.Labels{
			metric.QueryTypeLabelKey:    metric.PushQueryType,
			metric.QueryOutcomeLabelKey: metric.FailQueryOutcome,
		}).Add(1)
		return err
	}
	s.measures.Queries.With(prometheus.Labels{
		metric.QueryTypeLabelKey:    metric.PushQueryType,
		metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
	}).Add(1)
	return nil
}

func (s *Client) Get(key model.Key) (store.OwnableItem, error) {
	item, err := s.client.Get(key)
	if err != nil {
		if errors.Is(err, store.ErrItemNotFound) {
			s.measures.Queries.With(prometheus.Labels{
				metric.QueryTypeLabelKey:    metric.GetQueryType,
				metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
			}).Add(1)
		} else {
			s.measures.Queries.With(prometheus.Labels{
				metric.QueryTypeLabelKey:    metric.GetQueryType,
				metric.QueryOutcomeLabelKey: metric.FailQueryOutcome,
			}).Add(1)
		}
		return item, store.SanitizeError(err)
	}
	s.measures.Queries.With(prometheus.Labels{
		metric.QueryTypeLabelKey:    metric.GetQueryType,
		metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
	}).Add(1)
	return item, nil
}

func (s *Client) Delete(key model.Key) (store.OwnableItem, error) {
	item, err := s.client.Delete(key)
	if err != nil {
		if errors.Is(err, store.ErrItemNotFound) {
			s.measures.Queries.With(prometheus.Labels{
				metric.QueryTypeLabelKey:    metric.DeleteQueryType,
				metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
			}).Add(1)
		} else {
			s.measures.Queries.With(prometheus.Labels{
				metric.QueryTypeLabelKey:    metric.DeleteQueryType,
				metric.QueryOutcomeLabelKey: metric.FailQueryOutcome,
			}).Add(1)
		}
		return item, store.SanitizeError(err)
	}
	s.measures.Queries.With(prometheus.Labels{
		metric.QueryTypeLabelKey:    metric.DeleteQueryType,
		metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
	}).Add(1)
	return item, err
}

func (s *Client) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	item, err := s.client.GetAll(bucket)
	if err != nil {
		s.measures.Queries.With(prometheus.Labels{
			metric.QueryTypeLabelKey:    metric.GetAllQueryType,
			metric.QueryOutcomeLabelKey: metric.FailQueryOutcome,
		}).Add(1)
		return item, store.SanitizeError(err)
	}
	s.measures.Queries.With(prometheus.Labels{
		metric.QueryTypeLabelKey:    metric.GetAllQueryType,
		metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
	}).Add(1)
	return item, err
}

func (s *Client) Close() {
	s.client.Close()
}

// Ping is for pinging the database to verify that the connection is still good.
func (s *Client) Ping() error {
	err := s.client.Ping()
	if err != nil {
		s.measures.Queries.With(prometheus.Labels{
			metric.QueryTypeLabelKey:    metric.PingQueryType,
			metric.QueryOutcomeLabelKey: metric.FailQueryOutcome,
		}).Add(1)
		return emperror.WrapWith(err, "Pinging connection failed")
	}
	s.measures.Queries.With(prometheus.Labels{
		metric.QueryTypeLabelKey:    metric.PingQueryType,
		metric.QueryOutcomeLabelKey: metric.SuccessQueryOutcome,
	}).Add(1)
	return nil
}

func validateConfig(config *Config) {
	zeroDuration := time.Duration(0) * time.Second

	if config.OpTimeout == zeroDuration {
		config.OpTimeout = defaultOpTimeout
	}

	if config.Database == "" {
		config.Database = defaultDatabase
	}
	if config.NumRetries < 0 {
		config.NumRetries = defaultNumRetries
	}
	if config.WaitTimeMult < 1 {
		config.WaitTimeMult = defaultWaitTimeMult
	}
	if config.MaxConnsPerHost <= 0 {
		config.MaxConnsPerHost = defaultMaxNumberConnsPerHost
	}
}

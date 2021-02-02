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

package chrysom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/themis/xlog"
)

// PushResult is a simple type to indicate the result type for the
// PushItem operation.
type PushResult string

// Types of pushItem successful results.
const (
	CreatedPushResult PushResult = "created"
	UpdatedPushResult PushResult = "ok"
)

type ClientConfig struct {
	HTTPClient      *http.Client
	Bucket          string
	PullInterval    time.Duration
	Address         string
	Auth            Auth
	MetricsProvider provider.Provider
	Logger          log.Logger
	Listener        Listener
}

type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type Client struct {
	client             *http.Client
	ticker             *time.Ticker
	auth               acquire.Acquirer
	metrics            *measures
	listener           Listener
	bucketName         string
	remoteStoreAddress string
	logger             log.Logger
}

func initMetrics(p provider.Provider) *measures {
	return &measures{
		pollCount: p.NewCounter(PollCounter),
	}
}

func CreateClient(config ClientConfig) (*Client, error) {
	err := validateConfig(&config)
	if err != nil {
		return nil, err
	}
	auth, err := determineTokenAcquirer(config)
	if err != nil {
		return nil, err
	}
	clientStore := &Client{
		client:             config.HTTPClient,
		ticker:             time.NewTicker(config.PullInterval),
		auth:               auth,
		metrics:            initMetrics(config.MetricsProvider),
		logger:             config.Logger,
		listener:           config.Listener,
		remoteStoreAddress: config.Address,
		bucketName:         config.Bucket,
	}

	if config.PullInterval > 0 {
		clientStore.ticker = time.NewTicker(config.PullInterval)
	}

	return clientStore, nil
}

func validateConfig(config *ClientConfig) error {
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	if config.Address == "" {
		return errors.New("address can't be empty")
	}
	if config.Bucket == "" {
		config.Bucket = "testing"
	}
	if config.MetricsProvider == nil {
		return errors.New("a metrics provider is required")
	}

	if config.PullInterval == 0 {
		config.PullInterval = time.Second * 5
	}

	if config.Logger == nil {
		config.Logger = log.NewNopLogger()
	}
	return nil
}

func determineTokenAcquirer(config ClientConfig) (acquire.Acquirer, error) {
	defaultAcquirer := &acquire.DefaultAcquirer{}
	if config.Auth.JWT.AuthURL != "" && config.Auth.JWT.Buffer != 0 && config.Auth.JWT.Timeout != 0 {
		return acquire.NewRemoteBearerTokenAcquirer(config.Auth.JWT)
	}

	if config.Auth.Basic != "" {
		return acquire.NewFixedAuthAcquirer(config.Auth.Basic)
	}

	return defaultAcquirer, nil
}

func (c *Client) GetItems(owner string) ([]model.Item, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/store/%s", c.remoteStoreAddress, c.bucketName), nil)
	if err != nil {
		return nil, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return nil, err
	}

	request.Header.Set(store.ItemOwnerHeaderKey, owner)

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with non-200 response for GetItems request", "code", response.StatusCode)
		return nil, errors.New("failed to get items, non 200 statuscode")
	}

	items := []model.Item{}
	err = json.NewDecoder(response.Body).Decode(&items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (c *Client) Push(item model.Item, owner string) (PushResult, error) {
	if item.ID == "" {
		return "", errors.New("id can't be empty")
	}

	if item.TTL != nil && *item.TTL < 1 {
		return "", errors.New("when provided, TTL must be > 0")
	}

	data, err := json.Marshal(&item)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/store/%s/%s", c.remoteStoreAddress, c.bucketName, item.ID), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return "", err
	}
	request.Header.Add(store.ItemOwnerHeaderKey, owner)

	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}

	switch response.StatusCode {
	case http.StatusCreated:
		return CreatedPushResult, nil
	case http.StatusOK:
		return UpdatedPushResult, nil
	}
	level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with a non-successful status code for a Push request", "code", response.StatusCode)
	return "", errors.New("Failed to set item as DB responded with non-success statuscode")
}

func (c *Client) Remove(id string, owner string) (model.Item, error) {
	if id == "" {
		return model.Item{}, errors.New("id can't be empty")
	}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/store/%s/%s", c.remoteStoreAddress, c.bucketName, id), nil)
	if err != nil {
		return model.Item{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return model.Item{}, err
	}

	request.Header.Add(store.ItemOwnerHeaderKey, owner)

	response, err := c.client.Do(request)
	if err != nil {
		return model.Item{}, err
	}
	if response.StatusCode != 200 {
		return model.Item{}, errors.New("failed to delete item, non 200 statuscode")
	}
	responsePayload, _ := ioutil.ReadAll(response.Body)
	item := model.Item{}
	err = json.Unmarshal(responsePayload, &item)
	if err != nil {
		return model.Item{}, err
	}
	return item, nil
}

func (c *Client) Start(ctx context.Context) error {
	if c.ticker == nil {
		return errors.New("interval ticker is nil")
	}

	if c.listener == nil {
		level.Info(c.logger).Log(xlog.MessageKey(), "No listener was setup to receive updates.")
		return nil
	}

	go func() {
		for range c.ticker.C {
			outcome := SuccessOutcome
			items, err := c.GetItems("")
			if err == nil {
				c.listener.Update(items)
			} else {
				outcome = FailureOutcome
				level.Error(c.logger).Log(xlog.MessageKey(), "Failed to get items for listeners", xlog.ErrorKey(), err)
			}
			c.metrics.pollCount.With(OutcomeLabel, outcome).Add(1)
		}
	}()

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.ticker != nil {
		c.ticker.Stop()
	}
	return nil
}

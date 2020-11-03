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
	"github.com/xmidt-org/bascule/acquire"
)

// SetResult is a simple type to indicate the result type for the
// SetItem operation.
type SetResult string

// Types of set item successful results.
const (
	CreatedSetResult SetResult = "created"
	UpdatedSetResult SetResult = "ok"
)

type ClientConfig struct {
	HTTPClient      *http.Client
	Bucket          string
	PullInterval    time.Duration
	Address         string
	Auth            Auth
	DefaultTTL      int64
	MetricsProvider provider.Provider
	Logger          log.Logger
	Listener        Listener
	AdminToken      string
}

type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type loggerGroup struct {
	Info  log.Logger
	Error log.Logger
	Debug log.Logger
}

type Client struct {
	client              *http.Client
	ticker              *time.Ticker
	auth                acquire.Acquirer
	metrics             *measures
	listener            Listener
	bucketName          string
	remoteStoreAddress  string
	defaultStoreItemTTL int64
	loggers             loggerGroup
	adminToken          string
}

func initLoggers(logger log.Logger) loggerGroup {
	return loggerGroup{
		Info:  level.Info(logger),
		Error: level.Error(logger),
		Debug: level.Debug(logger),
	}
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
		client:              config.HTTPClient,
		ticker:              time.NewTicker(config.PullInterval),
		auth:                auth,
		metrics:             initMetrics(config.MetricsProvider),
		loggers:             initLoggers(config.Logger),
		listener:            config.Listener,
		remoteStoreAddress:  config.Address,
		defaultStoreItemTTL: config.DefaultTTL,
		bucketName:          config.Bucket,
		adminToken:          config.AdminToken,
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
	if config.DefaultTTL < 1 {
		config.DefaultTTL = 300
	}
	if config.MetricsProvider == nil {
		return errors.New("a metrics provider is required")
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

func (c *Client) GetItems(owner string, adminMode bool) ([]model.Item, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/store/%s", c.remoteStoreAddress, c.bucketName), nil)
	if err != nil {
		return nil, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return nil, err
	}
	if owner != "" {
		request.Header.Set("X-Midt-Owner", owner)
	}

	if adminMode {
		if c.adminToken == "" {
			return nil, errors.New("adminToken needed to run as admin")
		}
		request.Header.Set("X-Midt-Admin-Token", c.adminToken)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == 404 {
		return []model.Item{}, nil
	}
	if response.StatusCode != 200 {
		c.loggers.Error.Log("msg", "DB responded with non-200 response for request to get items", "code", response.StatusCode)
		return nil, errors.New("failed to get items, non 200 statuscode")
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	body := map[string]model.Item{}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return nil, err
	}

	responseData := make([]model.Item, len(body))
	index := 0
	for _, value := range body {
		responseData[index] = value
		index++
	}
	return responseData, nil
}

func (c *Client) Set(item model.Item, owner string, adminMode bool) (SetResult, error) {
	if item.Identifier == "" {
		return "", errors.New("identifier can't be empty")
	}

	if item.UUID == "" {
		return "", errors.New("uuid can't be empty")
	}

	if item.TTL != nil && *item.TTL < 1 {
		return "", errors.New("TTL must be > 0")
	}

	data, err := json.Marshal(&item)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/store/%s/%s", c.remoteStoreAddress, c.bucketName, item.UUID), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return "", err
	}
	request.Header.Add("X-Midt-Owner", owner)

	if adminMode {
		if c.adminToken == "" {
			return "", errors.New("adminToken needed to run as admin")
		}
		request.Header.Set("X-Midt-Admin-Token", c.adminToken)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}

	switch response.StatusCode {
	case http.StatusCreated:
		return CreatedSetResult, nil
	case http.StatusOK:
		return UpdatedSetResult, nil
	}

	c.loggers.Error.Log("msg", "DB responded with non-successful response for request to update an item", "code", response.StatusCode)
	return "", errors.New("Failed to set item as DB responded with non-success statuscode")
}

func (c *Client) Remove(uuid string, owner string, adminMode bool) (model.Item, error) {
	if uuid == "" {
		return model.Item{}, errors.New("uuid can't be empty")
	}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/store/%s/%s", c.remoteStoreAddress, c.bucketName, uuid), nil)
	if err != nil {
		return model.Item{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return model.Item{}, err
	}

	request.Header.Add("X-Midt-Owner", owner)

	if adminMode {
		if c.adminToken == "" {
			return "", errors.New("adminToken needed to run as admin")
		}
		request.Header.Set("X-Midt-Admin-Token", c.adminToken)
	}

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
		c.loggers.Info.Log("msg", "No listener setup for updates")
		return nil
	}

	go func() {
		for range c.ticker.C {
			outcome := SuccessOutcome
			items, err := c.GetItems("", true)
			if err == nil {
				c.listener.Update(items)
			} else {
				outcome = FailureOutcomme
				c.loggers.Error.Log("msg", "failed to get items", level.ErrorValue(), err)
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

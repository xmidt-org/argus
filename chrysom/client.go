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
	"io"
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

const storeAPIPath = "/api/v1/store"

// errors that can be returned by this package
var (
	ErrAddressEmpty             = errors.New("argus address is required")
	ErrBucketEmpty              = errors.New("bucket name is required")
	ErrItemIDEmpty              = errors.New("item ID is required")
	ErrUndefinedMetricsProvider = errors.New("a metrics provider is required")
	ErrUndefinedIntervalTicker  = errors.New("interval ticket is nil. Can't listen for updates")
	ErrGetItemsFailure          = errors.New("failed to get items. Non-200 statuscode was received")
	ErrDeleteItemFailure        = errors.New("failed delete item. Non-200 statuscode was received")
	ErrPushItemFailure          = errors.New("failed push item. Non-success statuscode was received")

	ErrUndefinedInput      = errors.New("input for operation was nil")
	ErrNewRequestFailure   = errors.New("failed creating an HTTP request")
	ErrAuthAcquirerFailure = errors.New("failed acquiring auth token")
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
	storeBaseURL       string
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

func NewClient(config *ClientConfig) (*Client, error) {
	err := validateConfig(config)
	if err != nil {
		return nil, err
	}
	tokenAcquirer, err := buildTokenAcquirer(&config.Auth)
	if err != nil {
		return nil, err
	}
	clientStore := &Client{
		client:             config.HTTPClient,
		ticker:             time.NewTicker(config.PullInterval),
		auth:               tokenAcquirer,
		metrics:            initMetrics(config.MetricsProvider),
		logger:             config.Logger,
		listener:           config.Listener,
		remoteStoreAddress: config.Address,
		bucketName:         config.Bucket,
		storeBaseURL:       config.Address + storeAPIPath,
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
		return ErrAddressEmpty
	}
	if config.Bucket == "" {
		config.Bucket = "testing"
	}
	if config.MetricsProvider == nil {
		return ErrUndefinedMetricsProvider
	}

	if config.PullInterval == 0 {
		config.PullInterval = time.Second * 5
	}

	if config.Logger == nil {
		config.Logger = log.NewNopLogger()
	}
	return nil
}
func shouldUseJWTAcquirer(options acquire.RemoteBearerTokenAcquirerOptions) bool {
	return len(options.AuthURL) > 0 && options.Buffer != 0 && options.Timeout != 0
}

func buildTokenAcquirer(auth *Auth) (acquire.Acquirer, error) {
	if shouldUseJWTAcquirer(auth.JWT) {
		return acquire.NewRemoteBearerTokenAcquirer(auth.JWT)
	} else if len(auth.Basic) > 0 {
		return acquire.NewFixedAuthAcquirer(auth.Basic)
	}
	return &acquire.DefaultAcquirer{}, nil
}

func validateGetItemsInput(input *GetItemsInput) error {
	if input == nil {
		return ErrUndefinedInput
	}

	if len(input.Bucket) < 1 {
		return ErrBucketEmpty
	}
	return nil
}

func (c *Client) makeRequest(owner, method, URL string, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNewRequestFailure, err)
	}
	err = acquire.AddAuth(r, c.auth)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthAcquirerFailure, err)
	}

	if len(owner) > 0 {
		r.Header.Set(store.ItemOwnerHeaderKey, owner)
	}

	return r, nil
}

type doResponse struct {
	body []byte
	code int
}

func (c *Client) do(r *http.Request) (*doResponse, error) {
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dResp = doResponse{
		code: resp.StatusCode,
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &dResp, err
	}
	dResp.body = bodyBytes
	return &dResp, nil
}

// GetItems fetches all items in a bucket that belong to a given owner.
func (c *Client) GetItems(input *GetItemsInput) (*GetItemsOutput, error) {
	err := validateGetItemsInput(input)
	if err != nil {
		return nil, err
	}

	URL := fmt.Sprintf("%s/%s", c.storeBaseURL, input.Bucket)
	request, err := c.makeRequest(input.Owner, http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	response, err := c.do(request)
	if err != nil {
		return nil, err
	}

	if response.code != http.StatusOK {
		level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with non-200 response for GetItems request", "code", response.code)
		return nil, ErrGetItemsFailure
	}

	var output GetItemsOutput

	err = json.Unmarshal(response.body, &output.Items)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func (c *Client) Push(item model.Item, owner string) (PushResult, error) {
	if item.ID == "" {
		return "", ErrItemIDEmpty
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

	defer response.Body.Close()

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
		return model.Item{}, ErrItemIDEmpty
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
		return model.Item{}, ErrDeleteItemFailure
	}
	defer response.Body.Close()
	responsePayload, _ := ioutil.ReadAll(response.Body)
	item := model.Item{}
	err = json.Unmarshal(responsePayload, &item)
	if err != nil {
		return model.Item{}, err
	}
	return item, nil
}

func (c *Client) Start(ctx context.Context, input *GetItemsInput) error {
	if c.ticker == nil {
		return ErrUndefinedIntervalTicker
	}

	err := validateGetItemsInput(input)
	if err != nil {
		return err
	}

	if c.listener == nil {
		level.Warn(c.logger).Log(xlog.MessageKey(), "No listener was setup to receive updates.")
		return nil
	}

	go func() {
		for range c.ticker.C {
			outcome := SuccessOutcome
			output, err := c.GetItems(input)
			if err == nil {
				c.listener.Update(output.Items)
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

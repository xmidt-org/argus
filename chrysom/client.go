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
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/provider"
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
	ErrItemIDMismatch           = errors.New("item ID must match that in payload")
	ErrItemDataEmpty            = errors.New("data field in item is required")
	ErrUndefinedMetricsProvider = errors.New("a metrics provider is required")
	ErrUndefinedIntervalTicker  = errors.New("interval ticker is nil. Can't listen for updates")
	ErrGetItemsFailure          = errors.New("failed to get items. Non-200 statuscode was received")
	ErrRemoveItemFailure        = errors.New("failed to delete item. Non-200 statuscode was received")
	ErrPushItemFailure          = errors.New("failed to push item. Non-success statuscode was received")

	ErrUndefinedInput      = errors.New("input for operation was nil")
	ErrNewRequestFailure   = errors.New("failed creating an HTTP request")
	ErrAuthAcquirerFailure = errors.New("failed acquiring auth token")
	ErrDoRequestFailure    = errors.New("http client failed while sending request")
	ErrReadingBodyFailure  = errors.New("failed while reading http response body")
	ErrJSONUnmarshal       = errors.New("failed unmarshaling JSON response payload")
	ErrJSONMarshal         = errors.New("failed marshaling item as JSON payload")
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
	Body []byte
	Code int
}

func (c *Client) do(r *http.Request) (*doResponse, error) {
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDoRequestFailure, err)
	}
	defer resp.Body.Close()

	var dResp = doResponse{
		Code: resp.StatusCode,
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &dResp, fmt.Errorf("%w: %v", ErrReadingBodyFailure, err)
	}
	dResp.Body = bodyBytes
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

	if response.Code != http.StatusOK {
		level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with non-200 response for GetItems request", "code", response.Code)
		return nil, fmt.Errorf("statusCode %v: %w", response.Code, ErrGetItemsFailure)
	}

	var output GetItemsOutput

	err = json.Unmarshal(response.Body, &output.Items)
	if err != nil {
		return nil, fmt.Errorf("GetItems: %w: %v", ErrJSONUnmarshal, err)
	}

	return &output, nil
}

func validatePushItemInput(input *PushItemInput) error {
	if input == nil {
		return ErrUndefinedInput
	}

	if len(input.Bucket) < 1 {
		return ErrBucketEmpty
	}

	if len(input.ID) < 1 || len(input.Item.ID) < 1 {
		return ErrItemIDEmpty
	}

	if strings.ToLower(input.ID) != strings.ToLower(input.Item.ID) {
		return ErrItemIDMismatch
	}

	// TODO: we can also validate the ID format here
	// we'll need to create an exporter validator in argus though

	if len(input.Item.Data) < 1 {
		return ErrItemDataEmpty
	}

	return nil
}

// PushItem creates a new item if one doesn't already exist at
// the resource path '{BUCKET}/{ID}'. If an item exists and the ownership matches,
// the item is simply updated.
func (c *Client) PushItem(input *PushItemInput) (*PushItemOutput, error) {
	err := validatePushItemInput(input)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(input.Item)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJSONMarshal, err)
	}

	URL := fmt.Sprintf("%s/%s/%s", c.storeBaseURL, input.Bucket, input.ID)
	request, err := c.makeRequest(input.Owner, http.MethodPut, URL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	response, err := c.do(request)
	if err != nil {
		return nil, err
	}

	if response.Code == http.StatusCreated {
		return &PushItemOutput{Result: CreatedPushResult}, nil
	}

	if response.Code == http.StatusOK {
		return &PushItemOutput{Result: UpdatedPushResult}, nil
	}

	level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with a non-successful status code for a PushItem request", "code", response.Code)

	return nil, fmt.Errorf("statusCode %v: %w", response.Code, ErrPushItemFailure)
}

func validateRemoveItemInput(input *RemoveItemInput) error {
	if input == nil {
		return ErrUndefinedInput
	}

	if len(input.Bucket) < 1 {
		return ErrBucketEmpty
	}

	if len(input.ID) < 1 {
		return ErrItemIDEmpty
	}
	return nil
}

// RemoveItem removes the item if it exists and returns the data associated to it.
func (c *Client) RemoveItem(input *RemoveItemInput) (*RemoveItemOutput, error) {
	err := validateRemoveItemInput(input)
	if err != nil {
		return nil, err
	}

	URL := fmt.Sprintf("%s/%s/%s", c.storeBaseURL, input.Bucket, input.ID)
	request, err := c.makeRequest(input.Owner, http.MethodDelete, URL, nil)
	if err != nil {
		return nil, err
	}

	response, err := c.do(request)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusOK {
		return nil, ErrRemoveItemFailure
	}

	var output RemoveItemOutput
	err = json.Unmarshal(response.Body, &output.Item)
	if err != nil {
		return nil, fmt.Errorf("RemoveItem: %w: %v", ErrJSONUnmarshal, err)
	}
	return &output, nil
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

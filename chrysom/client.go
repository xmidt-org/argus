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
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/themis/xlog"
)

const storeAPIPath = "/api/v1/store"

// Errors that can be returned by this package. Since some of these errors are returned wrapped, it
// is safest to use errors.Is() to check for them.
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
	// HTTPClient refers to the client that will be used to send
	// HTTP requests.
	// (Optional) http.DefaultClient is used if left empty.
	HTTPClient *http.Client

	// Address is the Argus URL (i.e. https://example-argus.io:8090)
	Address string

	// Auth provides the mechanism to add auth headers to outgoing
	// requests
	// (Optional) If not provided, no auth headers are added.
	Auth Auth

	// MetricsProvider allows measures updated by the client to be collected.
	MetricsProvider provider.Provider

	Logger log.Logger

	// Listener is the component that consumes the latest list of owned items in a
	// bucket.
	Listener Listener

	// PullInterval is how often listeners should get updates.
	PullInterval time.Duration

	// Bucket to be used in listener requests.
	Bucket string

	// Owner to be used in listener requests.
	// (Optional) If left empty, items without an owner will be watched.
	Owner string
}

type response struct {
	Body []byte
	Code int
}

type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type Client struct {
	client       *http.Client
	auth         acquire.Acquirer
	storeBaseURL string
	logger       log.Logger
	observer     *listenerConfig
}

type listenerConfig struct {
	listener  Listener
	ticker    *time.Ticker
	pollCount metrics.Counter
	bucket    string
	owner     string
}

func NewClient(config ClientConfig) (*Client, error) {
	err := validateConfig(&config)
	if err != nil {
		return nil, err
	}
	tokenAcquirer, err := buildTokenAcquirer(&config.Auth)
	if err != nil {
		return nil, err
	}

	clientStore := &Client{
		client:       config.HTTPClient,
		auth:         tokenAcquirer,
		logger:       config.Logger,
		observer:     createObserver(config.Logger, config),
		storeBaseURL: config.Address + storeAPIPath,
	}

	return clientStore, nil
}

func createObserver(logger log.Logger, config ClientConfig) *listenerConfig {
	if config.Listener == nil {
		return nil
	}
	return &listenerConfig{
		listener:  config.Listener,
		ticker:    time.NewTicker(config.PullInterval),
		pollCount: config.MetricsProvider.NewCounter(PollCounter),
		bucket:    config.Bucket,
		owner:     config.Owner,
	}
}

func validateConfig(config *ClientConfig) error {
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	if config.Address == "" {
		return ErrAddressEmpty
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

func isEmpty(options acquire.RemoteBearerTokenAcquirerOptions) bool {
	return len(options.AuthURL) < 1 || options.Buffer == 0 || options.Timeout == 0
}

func buildTokenAcquirer(auth *Auth) (acquire.Acquirer, error) {
	if !isEmpty(auth.JWT) {
		return acquire.NewRemoteBearerTokenAcquirer(auth.JWT)
	} else if len(auth.Basic) > 0 {
		return acquire.NewFixedAuthAcquirer(auth.Basic)
	}
	return &acquire.DefaultAcquirer{}, nil
}

func (c Client) sendRequest(owner, method, url string, body io.Reader) (response, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return response{}, fmt.Errorf("%w: %s", ErrNewRequestFailure, err.Error())
	}
	err = acquire.AddAuth(r, c.auth)
	if err != nil {
		return response{}, fmt.Errorf("%w: %s", ErrAuthAcquirerFailure, err.Error())
	}
	if len(owner) > 0 {
		r.Header.Set(store.ItemOwnerHeaderKey, owner)
	}
	resp, err := c.client.Do(r)
	if err != nil {
		return response{}, fmt.Errorf("%w: %s", ErrDoRequestFailure, err.Error())
	}
	defer resp.Body.Close()

	var sqResp = response{
		Code: resp.StatusCode,
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sqResp, fmt.Errorf("%w: %s", ErrReadingBodyFailure, err.Error())
	}
	sqResp.Body = bodyBytes
	return sqResp, nil
}

type Items []model.Item

// GetItems fetches all items in a bucket that belong to a given owner.
func (c *Client) GetItems(Bucket, Owner string) (Items, error) {
	if len(Bucket) < 1 {
		return nil, ErrBucketEmpty
	}

	URL := fmt.Sprintf("%s/%s", c.storeBaseURL, Bucket)
	response, err := c.sendRequest(Owner, http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusOK {
		level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with non-200 response for GetItems request", "code", response.Code)
		return nil, fmt.Errorf("statusCode %v: %w", response.Code, ErrGetItemsFailure)
	}

	var items Items

	err = json.Unmarshal(response.Body, &items)
	if err != nil {
		return nil, fmt.Errorf("GetItems: %w: %s", ErrJSONUnmarshal, err.Error())
	}

	return items, nil
}

// PushItem creates a new item if one doesn't already exist at
// the resource path '{BUCKET}/{ID}'. If an item exists and the ownership matches,
// the item is simply updated.
func (c *Client) PushItem(id, bucket, owner string, item model.Item) (PushResult, error) {
	err := validatePushItemInput(bucket, owner, id, item)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrJSONMarshal, err.Error())
	}

	URL := fmt.Sprintf("%s/%s/%s", c.storeBaseURL, bucket, id)
	response, err := c.sendRequest(owner, http.MethodPut, URL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if response.Code == http.StatusCreated {
		return CreatedPushResult, nil
	}

	if response.Code == http.StatusOK {
		return UpdatedPushResult, nil
	}

	level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with a non-successful status code for a PushItem request", "code", response.Code)

	return "", fmt.Errorf("statusCode %v: %w", response.Code, ErrPushItemFailure)
}

// RemoveItem removes the item if it exists and returns the data associated to it.
func (c *Client) RemoveItem(id, bucket, owner string) (model.Item, error) {
	err := validateRemoveItemInput(bucket, id)
	if err != nil {
		return model.Item{}, err
	}

	url := fmt.Sprintf("%s/%s/%s", c.storeBaseURL, bucket, id)
	resp, err := c.sendRequest(owner, http.MethodDelete, url, nil)
	if err != nil {
		return model.Item{}, err
	}

	if resp.Code != http.StatusOK {
		return model.Item{}, fmt.Errorf("statusCode %v: %w", resp.Code, ErrRemoveItemFailure)
	}

	var item model.Item
	err = json.Unmarshal(resp.Body, &item)
	if err != nil {
		return item, fmt.Errorf("RemoveItem: %w: %s", ErrJSONUnmarshal, err.Error())
	}
	return item, nil
}

func (c *Client) Start(ctx context.Context, input *GetItemsInput) error {
	if c.observer == nil {
		level.Warn(c.logger).Log(xlog.MessageKey(), "No listener was setup to receive updates.")
		return nil
	}

	if c.observer.ticker == nil {
		return ErrUndefinedIntervalTicker
	}

	go func() {
		observer := c.observer
		for range observer.ticker.C {
			outcome := SuccessOutcome
			items, err := c.GetItems(observer.bucket, observer.owner)
			if err == nil {
				observer.listener.Update(items)
			} else {
				outcome = FailureOutcome
				level.Error(c.logger).Log(xlog.MessageKey(), "Failed to get items for listeners", xlog.ErrorKey(), err)
			}
			observer.pollCount.With(OutcomeLabel, outcome).Add(1)
		}
	}()

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.observer != nil && c.observer.ticker != nil {
		c.observer.ticker.Stop()
	}
	return nil
}

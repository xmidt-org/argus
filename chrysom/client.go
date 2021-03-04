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
	"sync/atomic"
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

const (
	storeAPIPath     = "/api/v1/store"
	errWrappedFmt    = "%w: %s"
	errStatusCodeFmt = "statusCode %v: %w"
)

// Errors that can be returned by this package. Since some of these errors are returned wrapped, it
// is safest to use errors.Is() to check for them.
// Some internal errors might be unwrapped from output errors but unless these errors become exported,
// they are not part of the library API and may change in future versions.
var (
	ErrAddressEmpty            = errors.New("argus address is required")
	ErrBucketEmpty             = errors.New("bucket name is required")
	ErrItemIDEmpty             = errors.New("item ID is required")
	ErrItemIDMismatch          = errors.New("item ID must match that in payload")
	ErrItemDataEmpty           = errors.New("data field in item is required")
	ErrUndefinedIntervalTicker = errors.New("interval ticker is nil. Can't listen for updates")
	ErrAuthAcquirerFailure     = errors.New("failed acquiring auth token")

	ErrFailedAuthentication = errors.New("failed to authentication with argus")
	ErrBadRequest           = errors.New("argus rejected the request as invalid")

	ErrListenerNotStopped = errors.New("listener is either running or starting")
	ErrListenerNotRunning = errors.New("listener is either stopped or stopping")
)

var (
	errNonSuccessResponse = errors.New("argus responded with a non-success status code")
	errNewRequestFailure  = errors.New("failed creating an HTTP request")
	errDoRequestFailure   = errors.New("http client failed while sending request")
	errReadingBodyFailure = errors.New("failed while reading http response body")
	errJSONUnmarshal      = errors.New("failed unmarshaling JSON response payload")
	errJSONMarshal        = errors.New("failed marshaling item as JSON payload")
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
	// Address is the Argus URL (i.e. https://example-argus.io:8090)
	Address string

	// Bucket partition to be used by this client.
	Bucket string

	// HTTPClient refers to the client that will be used to send requests.
	// (Optional) Defaults to http.DefaultClient.
	HTTPClient *http.Client

	// Auth provides the mechanism to add auth headers to outgoing requests.
	// (Optional) If not provided, no auth headers are added.
	Auth Auth

	// MetricsProvider helps initialize metrics collectors.
	// (Optional). By default a discard provider will be used.
	MetricsProvider provider.Provider

	// Logger to be used by the client.
	// (Optional). By default a no op logger will be used.
	Logger log.Logger

	// Listener provides a mechanism to fetch a copy of all items within a bucket on
	// an interval.
	// (Optional). If not provided, listening won't be enabled for this client.
	Listener Listener

	// PullInterval is how often listeners should get updates.
	// (Optional). Defaults to 5 seconds.
	PullInterval time.Duration
}

type response struct {
	Body             []byte
	ArgusErrorHeader string
	Code             int
}

type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type Items []model.Item

type Client struct {
	client       *http.Client
	auth         acquire.Acquirer
	storeBaseURL string
	logger       log.Logger
	bucket       string
	observer     *listenerConfig
}

// listening states
const (
	stopped int32 = iota
	running
	transitioning
)

type listenerConfig struct {
	listener     Listener
	ticker       *time.Ticker
	pullInterval time.Duration
	pollCount    metrics.Counter
	shutdown     chan struct{}
	state        int32
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
		observer:     newObserver(config.Logger, config),
		bucket:       config.Bucket,
		storeBaseURL: config.Address + storeAPIPath,
	}

	return clientStore, nil
}

// translateNonSuccessStatusCode returns as specific error
// for known Argus status codes.
func translateNonSuccessStatusCode(code int) error {
	switch code {
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrFailedAuthentication
	default:
		return errNonSuccessResponse
	}
}

func newObserver(logger log.Logger, config ClientConfig) *listenerConfig {
	if config.Listener == nil {
		return nil
	}
	return &listenerConfig{
		listener:     config.Listener,
		ticker:       time.NewTicker(config.PullInterval),
		pullInterval: config.PullInterval,
		pollCount:    config.MetricsProvider.NewCounter(PollCounter),
		shutdown:     make(chan struct{}),
	}
}

func validateConfig(config *ClientConfig) error {
	if config.Address == "" {
		return ErrAddressEmpty
	}

	if config.Bucket == "" {
		return ErrBucketEmpty
	}

	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	if config.MetricsProvider == nil {
		config.MetricsProvider = provider.NewDiscardProvider()
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

func (c *Client) sendRequest(owner, method, url string, body io.Reader) (response, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return response{}, fmt.Errorf(errWrappedFmt, errNewRequestFailure, err.Error())
	}
	err = acquire.AddAuth(r, c.auth)
	if err != nil {
		return response{}, fmt.Errorf(errWrappedFmt, ErrAuthAcquirerFailure, err.Error())
	}
	if len(owner) > 0 {
		r.Header.Set(store.ItemOwnerHeaderKey, owner)
	}
	resp, err := c.client.Do(r)
	if err != nil {
		return response{}, fmt.Errorf(errWrappedFmt, errDoRequestFailure, err.Error())
	}
	defer resp.Body.Close()

	var sqResp = response{
		Code:             resp.StatusCode,
		ArgusErrorHeader: resp.Header.Get(store.XmidtErrorHeaderKey),
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sqResp, fmt.Errorf(errWrappedFmt, errReadingBodyFailure, err.Error())
	}
	sqResp.Body = bodyBytes
	return sqResp, nil
}

// GetItems fetches all items that belong to a given owner.
func (c *Client) GetItems(owner string) (Items, error) {
	response, err := c.sendRequest(owner, http.MethodGet, fmt.Sprintf("%s/%s", c.storeBaseURL, c.bucket), nil)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusOK {
		level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with non-200 response for GetItems request",
			"code", response.Code, "ErrorHeader", response.ArgusErrorHeader)
		return nil, fmt.Errorf(errStatusCodeFmt, response.Code, translateNonSuccessStatusCode(response.Code))
	}

	var items Items

	err = json.Unmarshal(response.Body, &items)
	if err != nil {
		return nil, fmt.Errorf("GetItems: %w: %s", errJSONUnmarshal, err.Error())
	}

	return items, nil
}

// PushItem creates a new item if one doesn't already exist at
// the resource path '{BUCKET}/{ID}'. If an item exists and the ownership matches,
// the item is simply updated.
func (c *Client) PushItem(id, owner string, item model.Item) (PushResult, error) {
	err := validatePushItemInput(owner, id, item)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf(errWrappedFmt, errJSONMarshal, err.Error())
	}

	response, err := c.sendRequest(owner, http.MethodPut, fmt.Sprintf("%s/%s/%s", c.storeBaseURL, c.bucket, id), bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if response.Code == http.StatusCreated {
		return CreatedPushResult, nil
	}

	if response.Code == http.StatusOK {
		return UpdatedPushResult, nil
	}

	level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with a non-successful status code for a PushItem request",
		"code", response.Code, "ErrorHeader", response.ArgusErrorHeader)

	return "", fmt.Errorf(errStatusCodeFmt, response.Code, translateNonSuccessStatusCode(response.Code))
}

// RemoveItem removes the item if it exists and returns the data associated to it.
func (c *Client) RemoveItem(id, owner string) (model.Item, error) {
	if len(id) < 1 {
		return model.Item{}, ErrItemIDEmpty
	}

	resp, err := c.sendRequest(owner, http.MethodDelete, fmt.Sprintf("%s/%s/%s", c.storeBaseURL, c.bucket, id), nil)
	if err != nil {
		return model.Item{}, err
	}

	if resp.Code != http.StatusOK {
		level.Error(c.logger).Log(xlog.MessageKey(), "Argus responded with a non-successful status code for a RemoveItem request",
			"code", resp.Code, "ErrorHeader", resp.ArgusErrorHeader)
		return model.Item{}, fmt.Errorf(errStatusCodeFmt, resp.Code, translateNonSuccessStatusCode(resp.Code))
	}

	var item model.Item
	err = json.Unmarshal(resp.Body, &item)
	if err != nil {
		return item, fmt.Errorf("RemoveItem: %w: %s", errJSONUnmarshal, err.Error())
	}
	return item, nil
}

// Start begins listening for updates on an interval given that client configuration
// is setup correctly. If a listener process is already in progress, calling Start()
// is a NoOp. If you want to restart the current listener process, call Stop() first.
func (c *Client) Start(ctx context.Context) error {
	if c.observer == nil || c.observer.listener == nil {
		level.Warn(c.logger).Log(xlog.MessageKey(), "No listener was setup to receive updates.")
		return nil
	}
	if c.observer.ticker == nil {
		level.Error(c.logger).Log(xlog.MessageKey(), "Observer ticker is nil")
		return ErrUndefinedIntervalTicker
	}

	if !atomic.CompareAndSwapInt32(&c.observer.state, stopped, transitioning) {
		level.Error(c.logger).Log(xlog.MessageKey(), "Start called when a listener was not in stopped state", "err", ErrListenerNotStopped)
		return ErrListenerNotStopped
	}

	c.observer.ticker.Reset(c.observer.pullInterval)
	go func() {
		for {
			select {
			case <-c.observer.shutdown:
				return
			case <-c.observer.ticker.C:
				outcome := SuccessOutcome
				items, err := c.GetItems("")
				if err == nil {
					c.observer.listener.Update(items)
				} else {
					outcome = FailureOutcome
					level.Error(c.logger).Log(xlog.MessageKey(), "Failed to get items for listeners", xlog.ErrorKey(), err)
				}
				c.observer.pollCount.With(OutcomeLabel, outcome).Add(1)
			}
		}
	}()

	atomic.SwapInt32(&c.observer.state, running)
	return nil
}

// Stop requests the current listener process to stop and waits for its goroutine to complete.
// Calling Stop() when a listener is not running (or while one is getting stopped) returns an
// error.
func (c *Client) Stop(ctx context.Context) error {
	if c.observer == nil || c.observer.ticker == nil {
		return nil
	}

	if !atomic.CompareAndSwapInt32(&c.observer.state, running, transitioning) {
		level.Error(c.logger).Log(xlog.MessageKey(), "Stop called when a listener was not in running state", "err", ErrListenerNotStopped)
		return ErrListenerNotRunning
	}

	c.observer.ticker.Stop()
	c.observer.shutdown <- struct{}{}
	atomic.SwapInt32(&c.observer.state, stopped)
	return nil
}

func validatePushItemInput(owner, id string, item model.Item) error {
	if len(id) < 1 || len(item.ID) < 1 {
		return ErrItemIDEmpty
	}

	if !strings.EqualFold(id, item.ID) {
		return ErrItemIDMismatch
	}

	if len(item.Data) < 1 {
		return ErrItemDataEmpty
	}

	return nil
}

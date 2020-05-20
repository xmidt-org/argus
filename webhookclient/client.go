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

package webhookclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"io/ioutil"
	"net/http"
	"time"
)

type ClientConfig struct {
	HttpClient   *http.Client
	Bucket       string
	PullInterval time.Duration
	Address      string
	Auth         Auth
	Filters      map[string]string
	DefaultTTL   int64
}
type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type Client struct {
	client  *http.Client
	options *storeConfig
	config  ClientConfig
	ticker  *time.Ticker
	auth    acquire.Acquirer
}

func CreateClient(config ClientConfig, options ...Option) (*Client, error) {
	err := validateConfig(&config)
	if err != nil {
		return nil, err
	}
	auth, err := determineTokenAcquirer(config)
	if err != nil {
		return nil, err
	}
	clientStore := &Client{
		client: config.HttpClient,
		options: &storeConfig{
			logger: logging.DefaultLogger(),
		},
		config: config,
		ticker: time.NewTicker(config.PullInterval),
		auth:   auth,
	}
	for _, o := range options {
		o(clientStore.options)
	}
	go func() {
		for range clientStore.ticker.C {
			if clientStore.options.listener != nil {
				hooks, err := clientStore.GetWebhook("")
				if err == nil {
					clientStore.options.listener.Update(hooks)
				} else {
					logging.Error(clientStore.options.logger).Log(logging.MessageKey(), "failed to get webhooks ", logging.ErrorKey(), err)
				}
			}
		}
	}()
	return clientStore, nil
}

func validateConfig(config *ClientConfig) error {
	if config.HttpClient == nil {
		config.HttpClient = http.DefaultClient
	}
	if config.Address == "" {
		return errors.New("address can't be empty")
	}
	if config.PullInterval == 0 {
		config.PullInterval = time.Second
	}
	if config.Bucket == "" {
		config.Bucket = "testing"
	}
	if config.DefaultTTL < 1 {
		config.DefaultTTL = 300
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

func (c *Client) GetWebhook(owner string) ([]webhook.W, error) {
	hooks := []webhook.W{}
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/store/%s", c.config.Address, c.config.Bucket), nil)
	if err != nil {
		return []webhook.W{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return []webhook.W{}, err
	}
	if owner != "" {
		request.Header.Add("X-Midt-Owner", owner)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return []webhook.W{}, err
	}
	if response.StatusCode == 404 {
		return []webhook.W{}, nil
	}
	if response.StatusCode != 200 {
		return []webhook.W{}, errors.New("failed to get webhooks, non 200 statuscode")
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []webhook.W{}, err
	}
	response.Body.Close()

	body := map[string]model.Item{}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return []webhook.W{}, err
	}

	for _, value := range body {
		data, err := json.Marshal(&value.Data)
		if err != nil {
			continue
		}
		var hook webhook.W
		err = json.Unmarshal(data, &hook)
		if err != nil {
			continue
		}
		hooks = append(hooks, hook)
	}

	return hooks, nil
}

func (c *Client) Push(w webhook.W, owner string) error {
	webhookData, err := json.Marshal(&w)
	if err != nil {
		return err
	}
	webhookPayload := map[string]interface{}{}
	err = json.Unmarshal(webhookData, &webhookPayload)
	if err != nil {
		return err
	}

	var ttl int64
	if int64(w.Duration.Seconds()) < 1 {
		ttl = c.config.DefaultTTL
	} else {
		ttl = int64(w.Duration.Seconds())
	}

	item := model.Item{
		Identifier: w.ID(),
		Data:       webhookPayload,
		TTL:        ttl,
	}

	data, err := json.Marshal(&item)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/store/%s", c.config.Address, c.config.Bucket), bytes.NewReader(data))
	if err != nil {
		return err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return err
	}
	if owner != "" {
		request.Header.Add("X-Midt-Owner", owner)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("failed to push webhook, non 200 statuscode")
	}
	return nil
}

func (c *Client) Remove(id string, owner string) error {
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/store/%s/%s", c.config.Address, c.config.Bucket, id), nil)
	if err != nil {
		return err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return err
	}
	if owner != "" {
		request.Header.Add("X-Midt-Owner", owner)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("failed to delete webhook, non 200 statuscode")
	}
	return nil
}

func (c *Client) Stop(context context.Context) {
	c.ticker.Stop()
}

func (c *Client) SetListener(listener Listener) error {
	c.options.listener = listener
	return nil
}

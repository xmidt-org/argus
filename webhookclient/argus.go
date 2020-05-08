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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"io/ioutil"
	"net/http"
	"time"
)

type ArgusConfig struct {
	Client       *http.Client
	Bucket       string
	PullInterval time.Duration
	Address      string
	Auth         Auth
	Filters      map[string]string
}
type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type ArgusClient struct {
	client  *http.Client
	options *storeConfig
	config  ArgusConfig
	ticker  *time.Ticker
	auth    acquire.Acquirer
}

func CreateArgusStore(config ArgusConfig, options ...Option) (*ArgusClient, error) {
	err := validateArgusConfig(&config)
	if err != nil {
		return nil, err
	}
	auth, err := determineTokenAcquirer(config)
	if err != nil {
		return nil, err
	}
	clientStore := &ArgusClient{
		client: config.Client,
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
				hooks, err := clientStore.GetWebhook()
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

func validateArgusConfig(config *ArgusConfig) error {
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	if config.Address == "" {
		return errors.New("argus address can't be empty")
	}
	if config.PullInterval == 0 {
		config.PullInterval = time.Second
	}
	if config.Bucket == "" {
		config.Bucket = "testing"
	}
	return nil
}
func determineTokenAcquirer(config ArgusConfig) (acquire.Acquirer, error) {
	defaultAcquirer := &acquire.DefaultAcquirer{}
	if config.Auth.JWT.AuthURL != "" && config.Auth.JWT.Buffer != 0 && config.Auth.JWT.Timeout != 0 {
		return acquire.NewRemoteBearerTokenAcquirer(config.Auth.JWT)
	}

	if config.Auth.Basic != "" {
		return acquire.NewFixedAuthAcquirer(config.Auth.Basic)
	}

	return defaultAcquirer, nil
}
func createAttributeFilter(filter map[string]string) string {
	if len(filter) == 0 {
		return ""
	}
	buf := bytes.NewBufferString("?attributes=")

	for key, value := range filter {
		buf.WriteString(key + "," + value)
		buf.WriteString(",")
	}
	buf.Truncate(buf.Len() - 1)
	return buf.String()
}

func (c *ArgusClient) GetWebhook() ([]webhook.W, error) {
	hooks := []webhook.W{}
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/store/%s%s", c.config.Address, c.config.Bucket, createAttributeFilter(c.config.Filters)), nil)
	if err != nil {
		return []webhook.W{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return []webhook.W{}, err
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

	body := map[string]map[string]interface{}{}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return []webhook.W{}, err
	}

	for _, value := range body {
		data, err := json.Marshal(&value)
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

func (c *ArgusClient) Push(w webhook.W) error {
	id := base64.RawURLEncoding.EncodeToString([]byte(w.ID()))
	data, err := json.Marshal(&w)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/store/%s/%s%s", c.config.Address, c.config.Bucket, id, createAttributeFilter(c.config.Filters)), bytes.NewReader(data))
	if err != nil {
		return err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return err
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

func (c *ArgusClient) Remove(id string) error {
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/store/%s/%s", c.config.Address, c.config.Bucket, id), nil)
	if err != nil {
		return err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return err
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

func (c *ArgusClient) Stop(context context.Context) {
	c.ticker.Stop()
}

func (c *ArgusClient) SetListener(listener Listener) error {
	c.options.listener = listener
	return nil
}

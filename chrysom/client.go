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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/bascule/acquire"
)

type ClientConfig struct {
	HTTPClient *http.Client
	Bucket     string
	Address    string
	Auth       Auth
	DefaultTTL int64
	Logger     log.Logger
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
	auth                acquire.Acquirer
	bucketName          string
	remoteStoreAddress  string
	defaultStoreItemTTL int64
	loggers             loggerGroup
}

func initLoggers(logger log.Logger) loggerGroup {
	return loggerGroup{
		Info:  level.Info(logger),
		Error: level.Error(logger),
		Debug: level.Debug(logger),
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
		auth:                auth,
		loggers:             initLoggers(config.Logger),
		remoteStoreAddress:  config.Address,
		defaultStoreItemTTL: config.DefaultTTL,
		bucketName:          config.Bucket,
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
		return []model.Item{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return []model.Item{}, err
	}
	if owner != "" {
		request.Header.Add("X-Midt-Owner", owner)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return []model.Item{}, err
	}
	if response.StatusCode == 404 {
		return []model.Item{}, nil
	}
	if response.StatusCode != 200 {
		c.loggers.Error.Log("msg", "DB responded with non-200 response for request to get items", "code", response.StatusCode)
		return []model.Item{}, errors.New("failed to get items, non 200 statuscode")
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []model.Item{}, err
	}

	body := map[string]model.Item{}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return []model.Item{}, err
	}

	responseData := make([]model.Item, len(body))
	index := 0
	for _, value := range body {
		responseData[index] = value
		index++
	}
	return responseData, nil
}

func (c *Client) Push(item model.Item, owner string) (string, error) {
	if item.Identifier == "" {
		return "", errors.New("identifier can't be empty")
	}
	if item.TTL < 1 {
		item.TTL = c.defaultStoreItemTTL
	}
	data, err := json.Marshal(&item)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/store/%s", c.remoteStoreAddress, c.bucketName), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return "", err
	}
	if owner != "" {
		request.Header.Add("X-Midt-Owner", owner)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		c.loggers.Error.Log("msg", "DB responded with non-200 response for request to add/update an item", "code", response.StatusCode)
		return "", errors.New("Failed to put item as DB responded with non-200 statuscode")
	}
	responsePayload, _ := ioutil.ReadAll(response.Body)
	key := model.Key{}
	err = json.Unmarshal(responsePayload, &key)
	if err != nil {
		return "", err
	}
	return key.ID, nil
}

func (c *Client) Remove(id string, owner string) (model.Item, error) {
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/store/%s/%s", c.remoteStoreAddress, c.bucketName, id), nil)
	if err != nil {
		return model.Item{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return model.Item{}, err
	}
	if owner != "" {
		request.Header.Add("X-Midt-Owner", owner)
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

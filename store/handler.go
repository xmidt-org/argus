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

package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/themis/xlog"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	ErrNoDomainVariable = errors.New("No bucket variable in URI definition")
)

type Handler http.Handler

type KeyItemPair struct {
	Key
	Item
}

func NewHandler(e endpoint.Endpoint) Handler {
	return kithttp.NewServer(
		e,
		func(ctx context.Context, request *http.Request) (interface{}, error) {
			attributes := ToAttributes(strings.Split(request.URL.Query().Get("attributes"), ",")...)

			xlog.Get(ctx).Log(
				level.Key(), level.InfoValue(),
				xlog.MessageKey(), "request",
				"attributes", attributes,
				"mux", mux.Vars(request),
			)
			bucket, ok := mux.Vars(request)["bucket"]
			if !ok {
				return nil, ErrNoDomainVariable
			}
			key, ok := mux.Vars(request)["key"]
			if !ok {
				return KeyItemPair{
					Key: Key{
						Bucket: bucket,
					},
					Item: Item{
						Attributes: attributes,
						Value:      nil,
					},
				}, nil
			}
			itemKey := Key{
				Bucket: bucket,
				ID:     key,
			}
			if request.ContentLength == 0 {
				return itemKey, nil
			}

			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				return nil, err
			}
			value := map[string]interface{}{}
			err = json.Unmarshal(data, &value)
			if err != nil {
				return nil, err
			}
			return KeyItemPair{
				Key: itemKey,
				Item: Item{
					Attributes: attributes,
					Value:      value,
				},
			}, nil

		},
		func(ctx context.Context, response http.ResponseWriter, value interface{}) error {
			xlog.Get(ctx).Log(
				level.Key(), level.InfoValue(),
				xlog.MessageKey(), "request",
				"value", value,
			)
			if value != nil {
				if items, ok := value.(map[string]Item); ok {
					payload := map[string]map[string]interface{}{}
					for k, value := range items {
						payload[k] = value.Value
					}
					data, err := json.Marshal(&payload)
					if err != nil {
						return err
					}
					response.Header().Add("Content-Type", "application/json")
					response.Write(data)
					return nil
				}
				if item, ok := value.(Item); ok {
					data, err := json.Marshal(&item.Value)
					if err != nil {
						return err
					}
					response.Header().Add("Content-Type", "application/json")
					response.Write(data)
					return nil
				}

				data, err := json.Marshal(&value)
				if err != nil {
					return err
				}
				response.Header().Add("Content-Type", "application/json")
				response.Write(data)

			}
			return nil
		},
	)
}

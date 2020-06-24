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
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/themis/xlog"
	"io/ioutil"
	"net/http"
)

var (
	ErrNoBucketVariable = errors.New("No bucket variable in URI definition")
)

type Handler http.Handler

type KeyItemPairRequest struct {
	model.Key
	OwnableItem
	Method string
}

func NewHandler(e endpoint.Endpoint) Handler {
	return kithttp.NewServer(
		e,
		func(ctx context.Context, request *http.Request) (interface{}, error) {

			owner := request.Header.Get("X-Midt-Owner")
			xlog.Get(ctx).Log(
				level.Key(), level.InfoValue(),
				xlog.MessageKey(), "request",
				"mux", mux.Vars(request),
				"owner", owner,
			)

			bucket, ok := mux.Vars(request)["bucket"]
			if !ok {
				return nil, ErrNoBucketVariable
			}
			key, _ := mux.Vars(request)["key"]
			itemKey := model.Key{
				Bucket: bucket,
				ID:     key,
			}
			if request.ContentLength == 0 {
				return KeyItemPairRequest{
					Key: itemKey,
					OwnableItem: OwnableItem{
						Owner: owner,
					},
					Method: request.Method,
				}, nil
			}

			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				return nil, InvalidRequestError{Reason: fmt.Sprintf("failed to read body. reason: %s", err.Error())}
			}
			value := model.Item{}
			err = json.Unmarshal(data, &value)
			if err != nil {
				return nil, InvalidRequestError{Reason: fmt.Sprintf("failed to unmarshal json. reason: %s", err.Error())}
			}
			if value.TTL <= 0 {
				return nil, InvalidRequestError{Reason: "TTL must be > 0"}
			}

			return KeyItemPairRequest{
				Key: itemKey,
				OwnableItem: OwnableItem{
					Item:  value,
					Owner: owner,
				},
				Method: request.Method,
			}, nil

		},
		func(ctx context.Context, response http.ResponseWriter, value interface{}) error {
			xlog.Get(ctx).Log(
				level.Key(), level.InfoValue(),
				xlog.MessageKey(), "request",
				"value", value,
			)
			if value != nil {
				if items, ok := value.(map[string]OwnableItem); ok {
					payload := map[string]model.Item{}
					for k, value := range items {
						payload[k] = value.Item
					}
					data, err := json.Marshal(&payload)
					if err != nil {
						return err
					}
					response.Header().Add("Content-Type", "application/json")
					response.Write(data)
					return nil
				}
				if item, ok := value.(OwnableItem); ok {
					data, err := json.Marshal(&item.Item)
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
		kithttp.ServerErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Xmidt-Error", err.Error())
			if headerer, ok := err.(kithttp.Headerer); ok {
				for k, values := range headerer.Headers() {
					for _, v := range values {
						w.Header().Add(k, v)
					}
				}
			}
			code := http.StatusInternalServerError
			if sc, ok := err.(kithttp.StatusCoder); ok {
				code = sc.StatusCode()
			}
			w.WriteHeader(code)
		}),
	)
}

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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"github.com/xmidt-org/argus/model"
	"net/http"
)

type KeyNotFoundError struct {
	Key model.Key
}

func (knfe KeyNotFoundError) Error() string {
	if knfe.Key.ID == "" && knfe.Key.Bucket == "" {
		return fmt.Sprint("parameters for key not set")
	} else if knfe.Key.ID == "" && knfe.Key.Bucket != "" {
		return fmt.Sprintf("no vaule exists for bucket %s", knfe.Key.Bucket)

	}
	return fmt.Sprintf("no vaule exists with bucket: %s, id: %s", knfe.Key.Bucket, knfe.Key.ID)
}

func (knfe KeyNotFoundError) StatusCode() int {
	return http.StatusNotFound
}

type BadRequestError struct {
	Request interface{}
}

func (bre BadRequestError) Error() string {
	return fmt.Sprintf("No vaule exists with request: %#v", bre)
}

func (bre BadRequestError) StatusCode() int {
	return http.StatusBadRequest
}

func NewSetEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			kv KeyItemPairRequest
			ok bool
		)
		if kv, ok = request.(KeyItemPairRequest); !ok {
			return nil, BadRequestError{Request: request}
		}
		if kv.Identifier == "" {
			return nil, BadRequestError{Request: request}
		}
		if kv.TTL < 1 {
			kv.TTL = DefaultTTL
		}
		// Generate ID from Item identifier

		kv.ID = base64.RawURLEncoding.EncodeToString(sha256.New().Sum([]byte(kv.Identifier)))
		fmt.Println(kv)
		err := s.Push(kv.Key, kv.OwnableItem)
		return kv.Key, err
	}
}

func NewGetEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			kv KeyItemPairRequest
			ok bool
		)
		if kv, ok = request.(KeyItemPairRequest); !ok {
			return nil, BadRequestError{Request: request}
		}
		if kv.Key.Bucket == "" || kv.Key.ID == "" {
			return nil, BadRequestError{Request: request}
		}
		value, err := s.Get(kv.Key)
		if err != nil {
			return nil, err
		}
		if kv.Owner == value.Owner || kv.Owner == "" {
			if kv.Method == "DELETE" {
				value, err = s.Delete(kv.Key)
				return value, err
			}
			return value, nil
		}
		return nil, KeyNotFoundError{
			Key: kv.Key,
		}
	}
}

func NewGetAllEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			kv KeyItemPairRequest
			ok bool
		)
		if kv, ok = request.(KeyItemPairRequest); !ok {
			return nil, BadRequestError{Request: request}
		}

		value, err := s.GetAll(kv.Key.Bucket)

		return FilterOwner(value, kv.Owner), err
	}
}

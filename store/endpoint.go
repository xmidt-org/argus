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
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"net/http"
)

type KeyNotFoundError struct {
	Key Key
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
			kv KeyItemPair
			ok bool
		)
		if kv, ok = request.(KeyItemPair); !ok {
			return nil, BadRequestError{Request: request}
		}
		err := s.Push(kv.Key, kv.Item)

		return nil, err
	}
}

func NewGetEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			key Key
			ok  bool
		)
		if key, ok = request.(Key); !ok {
			return nil, BadRequestError{Request: request}
		}
		value, err := s.Get(key)

		return value, err
	}
}

func NewGetAllEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			kv KeyItemPair
			ok bool
		)
		if kv, ok = request.(KeyItemPair); !ok {
			return nil, BadRequestError{Request: request}
		}
		value, err := s.GetAll(kv.Key.Bucket)
		if len(kv.Item.Attributes) > 1 {
			return Filter(value, kv.Item.Attributes), err
		}

		return value, err
	}
}

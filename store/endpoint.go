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
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/xmidt-org/argus/model"
)

type KeyNotFoundError struct {
	Key model.Key
}

func (knfe KeyNotFoundError) Error() string {
	if knfe.Key.ID == "" && knfe.Key.Bucket == "" {
		return fmt.Sprint("parameters for key not set")
	} else if knfe.Key.ID == "" && knfe.Key.Bucket != "" {
		return fmt.Sprintf("no value exists for bucket %s", knfe.Key.Bucket)

	}
	return fmt.Sprintf("no value exists with bucket: %s, id: %s", knfe.Key.Bucket, knfe.Key.ID)
}

func (knfe KeyNotFoundError) StatusCode() int {
	return http.StatusNotFound
}

type BadRequestError struct {
	Request interface{}
}

func (bre BadRequestError) Error() string {
	return fmt.Sprintf("No value exists with request: %#v", bre)
}

func (bre BadRequestError) StatusCode() int {
	return http.StatusBadRequest
}

type InternalError struct {
	Reason    interface{}
	Retryable bool
}

func (ie InternalError) Error() string {
	return fmt.Sprintf("Request Failed: \n%#v", ie.Reason)
}

func (ie InternalError) StatusCode() int {
	return http.StatusInternalServerError
}

type InvalidRequestError struct {
	Reason string
}

func (ire InvalidRequestError) Error() string {
	return ire.Reason
}

func (ire InvalidRequestError) StatusCode() int {
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

		// Generate ID from Item identifier
		sum := sha256.Sum256([]byte(kv.Identifier))
		kv.ID = base64.RawURLEncoding.EncodeToString(sum[:])
		if len([]byte(kv.ID)) >= 1024 {
			return nil, InvalidRequestError{Reason: "identifier is too big"}
		}
		err := s.Push(kv.Key, kv.OwnableItem)
		return kv.Key, err
	}
}

func newGetItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemRequest := request.(*getItemRequest)
		itemResponse, err := s.Get(itemRequest.key)
		if err != nil {
			return nil, err
		}
		if itemRequest.owner == "" || itemRequest.owner == itemResponse.Owner {
			return itemResponse, nil
		}

		return nil, &KeyNotFoundError{Key: itemRequest.key}
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

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

	"github.com/go-kit/kit/endpoint"
	"github.com/xmidt-org/argus/model"
)

func newGetItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemRequest := request.(*getOrDeleteItemRequest)
		itemResponse, err := s.Get(itemRequest.key)
		if err != nil {
			return nil, err
		}
		if userOwnsItem(itemRequest.owner, itemResponse.Owner) {
			return &itemResponse, nil
		}

		return nil, &KeyNotFoundError{Key: itemRequest.key}
	}
}

func newDeleteItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemRequest := request.(*getOrDeleteItemRequest)
		itemResponse, err := s.Get(itemRequest.key)
		if err != nil {
			return nil, err
		}
		if userOwnsItem(itemRequest.owner, itemResponse.Owner) {
			deleteItemResp, deleteItemRespErr := s.Delete(itemRequest.key)
			if deleteItemRespErr != nil {
				return nil, deleteItemRespErr
			}
			return &deleteItemResp, nil
		}

		return nil, &KeyNotFoundError{Key: itemRequest.key}
	}
}

func newGetAllItemsEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemsRequest := request.(*getAllItemsRequest)
		items, err := s.GetAll(itemsRequest.bucket)
		if err != nil {
			return nil, err
		}

		return FilterOwner(items, itemsRequest.owner), nil
	}
}

func newPushItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		pushItemRequest := request.(*pushItemRequest)
		key := model.Key{
			Bucket: pushItemRequest.bucket,
			ID:     pushItemRequest.item.Identifier,
		}
		err := s.Push(key, pushItemRequest.item)
		if err != nil {
			return nil, err
		}
		return &key, nil
	}
}

func userOwnsItem(requestItemOwner, actualItemOwner string) bool {
	return requestItemOwner == "" || requestItemOwner == actualItemOwner
}

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
)

func newGetItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemRequest := request.(*getOrDeleteItemRequest)
		itemResponse, err := s.Get(itemRequest.key)
		if err != nil {
			return nil, err
		}
		if itemRequest.owner == itemResponse.Owner {
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

		if !authorized(itemRequest.adminMode, itemResponse.Owner, itemRequest.owner) {
			return nil, &ForbiddenRequestErr{}
		}

		deleteItemResp, deleteItemRespErr := s.Delete(itemRequest.key)
		if deleteItemRespErr != nil {
			return nil, deleteItemRespErr
		}
		return &deleteItemResp, nil
	}
}

func newGetAllItemsEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemsRequest := request.(*getAllItemsRequest)
		items, err := s.GetAll(itemsRequest.bucket)
		if err != nil {
			return nil, err
		}
		if itemsRequest.adminMode {
			return items, nil
		}
		return FilterOwner(items, itemsRequest.owner), nil
	}
}

func newSetItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		setItemRequest := request.(*setItemRequest)
		itemResponse, err := s.Get(setItemRequest.key)

		if err != nil {
			switch err.(type) {
			case KeyNotFoundError:
				err = s.Push(setItemRequest.key, setItemRequest.item)
				if err != nil {
					return nil, err
				}
				return &setItemResponse{}, nil

			default:
				return nil, err
			}
		}

		if !authorized(setItemRequest.adminMode, itemResponse.Owner, setItemRequest.item.Owner) {
			return nil, &ForbiddenRequestErr{Message: "resource owner mismatch"}
		}

		err = s.Push(setItemRequest.key, setItemRequest.item)
		if err != nil {
			return nil, err
		}

		return &setItemResponse{
			existingResource: true,
		}, nil
	}
}

func authorized(adminMode bool, resourceOwner, requestOwner string) bool {
	return adminMode || resourceOwner == requestOwner
}

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package store

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
)

var accessDeniedErr = &ForbiddenRequestErr{Message: "resource owner mismatch"}

func newGetItemEndpoint(s S) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		itemRequest := request.(*getOrDeleteItemRequest)
		itemResponse, err := s.Get(itemRequest.key)
		if err != nil {
			return nil, err
		}
		if authorized(itemRequest.adminMode, itemResponse.Owner, itemRequest.owner) {
			return &itemResponse, nil
		}

		return nil, accessDeniedErr
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
			return nil, accessDeniedErr
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
		if itemsRequest.adminMode && itemsRequest.owner == "" {
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
			if errors.Is(err, ErrItemNotFound) {
				err = s.Push(setItemRequest.key, setItemRequest.item)
				if err != nil {
					return nil, err
				}
				return &setItemResponse{}, nil
			}
			return nil, err
		}

		if !authorized(setItemRequest.adminMode, itemResponse.Owner, setItemRequest.item.Owner) {
			return nil, accessDeniedErr
		}

		setItemRequest.item.Owner = itemResponse.Owner

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

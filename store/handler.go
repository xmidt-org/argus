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
	"net/http"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/argus/model"
)

type Handler http.Handler

type KeyItemPairRequest struct {
	model.Key
	OwnableItem
	Method string
}

type ItemTTL struct {
	MaxTTL time.Duration
}

func newGetItemHandler(s S) Handler {
	return kithttp.NewServer(
		newGetItemEndpoint(s),
		decodeGetOrDeleteItemRequest,
		encodeGetOrDeleteItemResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

func newDeleteItemHandler(s S) Handler {
	return kithttp.NewServer(
		newDeleteItemEndpoint(s),
		decodeGetOrDeleteItemRequest,
		encodeGetOrDeleteItemResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

func newGetAllItemsHandler(s S) Handler {
	return kithttp.NewServer(
		newGetAllItemsEndpoint(s),
		decodeGetAllItemsRequest,
		encodeGetAllItemsResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

func newSetItemHandler(itemTTLInfo ItemTTL, s S) Handler {
	return kithttp.NewServer(
		newSetItemEndpoint(s),
		setItemRequestDecoder(itemTTLInfo),
		encodeSetItemResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

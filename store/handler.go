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

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/argus/model"
)

type Handler http.Handler

type KeyItemPairRequest struct {
	model.Key
	OwnableItem
	Method string
}

func newGetItemHandler(config *transportConfig, s S) Handler {
	return kithttp.NewServer(
		newGetItemEndpoint(s),
		getOrDeleteItemRequestDecoder(config),
		encodeGetOrDeleteItemResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

func newDeleteItemHandler(config *transportConfig, s S) Handler {
	return kithttp.NewServer(
		newDeleteItemEndpoint(s),
		getOrDeleteItemRequestDecoder(config),
		encodeGetOrDeleteItemResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

func newGetAllItemsHandler(config *transportConfig, s S) Handler {
	return kithttp.NewServer(
		newGetAllItemsEndpoint(s),
		getAllItemsRequestDecoder(config),
		encodeGetAllItemsResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

func newSetItemHandler(config *transportConfig, s S) Handler {
	return kithttp.NewServer(
		newSetItemEndpoint(s),
		setItemRequestDecoder(config),
		encodeSetItemResponse,
		kithttp.ServerErrorEncoder(encodeError),
	)
}

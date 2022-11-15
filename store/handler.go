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
	"github.com/xmidt-org/sallust"
	"go.uber.org/fx"
)

type Handler http.Handler

type KeyItemPairRequest struct {
	model.Key
	OwnableItem
	Method string
}

type handlerIn struct {
	fx.In
	GetLogger sallust.GetLoggerFunc
	Store     S
	Config    *transportConfig
}

func newGetItemHandler(in handlerIn) Handler {
	return kithttp.NewServer(
		newGetItemEndpoint(in.Store),
		getOrDeleteItemRequestDecoder(in.Config),
		encodeGetOrDeleteItemResponse,
		kithttp.ServerErrorEncoder(encodeError(in.GetLogger)),
	)
}

func newDeleteItemHandler(in handlerIn) Handler {
	return kithttp.NewServer(
		newDeleteItemEndpoint(in.Store),
		getOrDeleteItemRequestDecoder(in.Config),
		encodeGetOrDeleteItemResponse,
		kithttp.ServerErrorEncoder(encodeError(in.GetLogger)),
	)
}

func newGetAllItemsHandler(in handlerIn) Handler {
	return kithttp.NewServer(
		newGetAllItemsEndpoint(in.Store),
		getAllItemsRequestDecoder(in.Config),
		encodeGetAllItemsResponse,
		kithttp.ServerErrorEncoder(encodeError(in.GetLogger)),
	)
}

func newSetItemHandler(in handlerIn) Handler {
	return kithttp.NewServer(
		newSetItemEndpoint(in.Store),
		setItemRequestDecoder(in.Config),
		encodeSetItemResponse,
		kithttp.ServerErrorEncoder(encodeError(in.GetLogger)),
	)
}

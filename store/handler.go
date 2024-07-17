// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package store

import (
	"context"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/ancla/model"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler http.Handler

type KeyItemPairRequest struct {
	model.Key
	OwnableItem
	Method string
}

type handlerIn struct {
	fx.In
	GetLogger func(context.Context) *zap.Logger
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

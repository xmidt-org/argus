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
	"time"

	"github.com/xmidt-org/themis/config"
	"go.uber.org/fx"
)

// StoreIn is the set of dependencies for this package's components
type StoreIn struct {
	fx.In

	Store S
}

// StoreOut is the set of components emitted by this package
type StoreOut struct {
	fx.Out

	// SetItemHandler is the http.Handler to update an item in the store.
	SetItemHandler Handler `name:"setHandler"`

	// SetKeyHandler is the http.Handler to fetch an individual item from the store.
	GetItemHandler Handler `name:"getHandler"`

	// GetAllItems is the http.Handler to fetch all items from the store for a given bucket.
	GetAllItemsHandler Handler `name:"getAllHandler"`

	// DeletItems is the http.Handler to delete a certain item.
	DeleteKeyHandler Handler `name:"deleteHandler"`
}

// Provide is an uber/fx style provider for this package's components
func Provide(unmarshaller config.Unmarshaller, in StoreIn) StoreOut {
	cfg := new(requestConfig)
	unmarshaller.UnmarshalKey("request", cfg)
	validateRequestConfig(cfg)

	return StoreOut{
		SetItemHandler:     newSetItemHandler(cfg, in.Store),
		GetItemHandler:     newGetItemHandler(cfg, in.Store),
		GetAllItemsHandler: newGetAllItemsHandler(cfg, in.Store),
		DeleteKeyHandler:   newDeleteItemHandler(cfg, in.Store),
	}
}

func validateRequestConfig(cfg *requestConfig) {
	if cfg.Validation.MaxTTL <= time.Second {
		cfg.Validation.MaxTTL = YearTTL * time.Second
	}
}

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

	Unmarshaler             config.Unmarshaller
	Store                   S
	AccessLevelAttributeKey string `name:"access_level_attribute_key"`
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

// NewHandlers initializes all handlers that will be needed for the store endpoints.
func NewHandlers(in StoreIn) StoreOut {
	var itemMaxTTL time.Duration

	in.Unmarshaler.UnmarshalKey("itemMaxTTL", &itemMaxTTL)
	if itemMaxTTL == 0 {
		itemMaxTTL = DefaultMaxTTLSeconds * time.Second
	}

	config := transportConfig{
		AccessLevelAttributeKey: in.AccessLevelAttributeKey,
		ItemMaxTTL:              itemMaxTTL,
	}

	return StoreOut{
		SetItemHandler:     newSetItemHandler(config, in.Store),
		GetItemHandler:     newGetItemHandler(config, in.Store),
		GetAllItemsHandler: newGetAllItemsHandler(config, in.Store),
		DeleteKeyHandler:   newDeleteItemHandler(config, in.Store),
	}
}

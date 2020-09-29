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

	// SetItemHandler is the http.Handler to add a new item to the store.
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
	itemTTL := ItemTTL{
		DefaultTTL: DefaultTTL,
		MaxTTL:     YearTTL,
	}
	unmarshaller.UnmarshalKey("itemTTL", itemTTL)
	validateItemTTLConfig(&itemTTL)

	return StoreOut{
		SetItemHandler:     newPushItemHandler(itemTTL, in.Store),
		GetItemHandler:     newGetItemHandler(in.Store),
		GetAllItemsHandler: newGetAllItemsHandler(in.Store),
		DeleteKeyHandler:   newDeleteItemHandler(in.Store),
	}
}

func validateItemTTLConfig(ttl *ItemTTL) {
	if ttl.MaxTTL <= time.Second {
		ttl.MaxTTL = YearTTL * time.Second
	}
	if ttl.DefaultTTL <= time.Millisecond {
		ttl.DefaultTTL = DefaultTTL * time.Second
	}
}

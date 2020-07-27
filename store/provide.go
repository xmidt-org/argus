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
	"github.com/xmidt-org/themis/config"
	"go.uber.org/fx"
	"time"
)

// StoreIn is the set of dependencies for this package's components
type StoreIn struct {
	fx.In

	Store S
}

// StoreOut is the set of components emitted by this package
type StoreOut struct {
	fx.Out

	// SetKeyHandler is the http.Handler which can update the Registry
	SetKeyHandler Handler `name:"setHandler"`

	// SetKeyHandler is the http.Handler which can update the Registry
	GetKeyHandler Handler `name:"getHandler"`

	// SetKeyHandler is the http.Handler which can update the Registry
	AllKeyHandler Handler `name:"getAllHandler"`
}

// Provide is an uber/fx style provider for this package's components
func Provide(unmarshaller config.Unmarshaller, in StoreIn) StoreOut {
	itemTTL := &ItemTTL{
		DefaultTTL: DefaultTTL,
		MaxTTL:     YearTTL,
	}
	unmarshaller.UnmarshalKey("itemTTL", itemTTL)
	validateItemTTL(itemTTL)

	return StoreOut{
		SetKeyHandler: NewHandler(NewSetEndpoint(in.Store), *itemTTL),
		GetKeyHandler: NewHandler(NewGetEndpoint(in.Store), *itemTTL),
		AllKeyHandler: NewHandler(NewGetAllEndpoint(in.Store), *itemTTL),
	}
}

func validateItemTTL(ttl *ItemTTL) {
	if ttl.MaxTTL <= time.Second {
		ttl.MaxTTL = YearTTL * time.Second
	}
	if ttl.DefaultTTL <= time.Millisecond {
		ttl.DefaultTTL = DefaultTTL * time.Second
	}
}

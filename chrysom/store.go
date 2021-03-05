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

package chrysom

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/argus/model"
)

type PushReader interface {
	Pusher
	Reader
}

type Pusher interface {
	// PushItem adds the item and establishes its link to the given owner in the store.
	PushItem(owner string, item model.Item) (PushResult, error)

	// Remove will remove the matching item from the store and return it.
	RemoveItem(id, owner string) (model.Item, error)
}

type Listener interface {
	// Update is called when we get changes to our item listeners with either
	// additions, or updates.
	//
	// The list of hooks must contain only the current items.
	Update(items Items)
}

type ListenerFunc func(items Items)

func (listener ListenerFunc) Update(items Items) {
	listener(items)
}

type Reader interface {
	// GeItems returns all the items that belong to this owner.
	GetItems(owner string) (Items, error)

	// Start kicks off listening for snapshots of all items in the store.
	Start(ctx context.Context) error

	// Stop will stop the listener provided there was an active one.
	Stop(ctx context.Context) error
}

type ConfigureListener interface {
	// SetListener will attempt to set the lister.
	SetListener(listener Listener) error
}

type storeConfig struct {
	logger   log.Logger
	backend  Pusher
	listener Listener
}

// Option is the function used to configure a store.
type Option func(r *storeConfig)

// WithLogger sets a logger to use for the store.
func WithLogger(logger log.Logger) Option {
	return func(r *storeConfig) {
		if logger != nil {
			r.logger = logger
		}
	}
}

// WithStorage sets a Pusher to use for the store.
func WithStorage(pusher Pusher) Option {
	return func(r *storeConfig) {
		if pusher != nil {
			r.backend = pusher
		}
	}
}

// WithListener sets a Listener to use for the store.
func WithListener(listener Listener) Option {
	return func(r *storeConfig) {
		if listener != nil {
			r.listener = listener
		}
	}
}

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

package webhookclient

import (
	"context"
	"errors"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"sync"
	"time"
)

var (
	errNoHookAvailable = errors.New("no webhook for key")
)

type envelope struct {
	creation time.Time
	hook     webhook.W
}

type Cache struct {
	hooks   map[string]envelope
	lock    sync.RWMutex
	config  CacheConfig
	options *storeConfig
	stop    chan struct{}
}

func (cache *Cache) SetListener(listener Listener) error {
	cache.options.listener = listener
	return nil
}

func (cache *Cache) Remove(id string) error {
	// update the store if there is no backend.
	// if it is set. On List() will update the cache data set
	if cache.options.backend == nil {
		cache.lock.Lock()
		delete(cache.hooks, id)
		cache.lock.Unlock()
		// update listener
		if cache.options.listener != nil {
			hooks, _ := cache.GetWebhook()
			cache.options.listener.Update(hooks)
		}
		return nil
	}
	return cache.options.backend.Remove(id)
}

func (cache *Cache) Stop(ctx context.Context) {
	close(cache.stop)
	if cache.options.backend != nil {
		cache.options.backend.Stop(ctx)
	}
}

func (cache *Cache) GetWebhook() ([]webhook.W, error) {
	if cache.options.backend != nil {
		if reader, ok := cache.options.backend.(Reader); ok {
			return reader.GetWebhook()
		}
	}
	cache.lock.RLock()
	data := []webhook.W{}
	for _, value := range cache.hooks {
		if time.Now().Before(value.creation.Add(cache.config.TTL)) {
			data = append(data, value.hook)
		}
	}
	cache.lock.RUnlock()
	return data, nil
}

func (cache *Cache) Update(hooks []webhook.W) {
	// update cache
	if cache.options.listener != nil {
		cache.hooks = map[string]envelope{}
		for _, elem := range hooks {
			cache.hooks[elem.ID()] = envelope{
				creation: time.Now(),
				hook:     elem,
			}
		}
	}
	// notify listener
	if cache.options.listener != nil {
		cache.options.listener.Update(hooks)
	}
}

func (cache *Cache) Push(w webhook.W) error {
	// update the store if there is no backend.
	// if it is set. On List() will update the cache data set

	cache.lock.Lock()
	cache.hooks[w.ID()] = envelope{
		creation: time.Now(),
		hook:     w,
	}
	cache.lock.Unlock()
	// update listener
	if cache.options.listener != nil {
		hooks, _ := cache.GetWebhook()
		cache.options.listener.Update(hooks)
	}

	if cache.options.backend != nil {
		return cache.options.backend.Push(w)
	}
	return nil
}

// CleanUp will free remove old webhooks.
func (cache *Cache) CleanUp() {
	cache.lock.Lock()
	for key, value := range cache.hooks {
		if time.Now().After(value.creation.Add(cache.config.TTL)) {
			go cache.Remove(key)
		}
	}
	cache.lock.Unlock()
}

type CacheConfig struct {
	TTL           time.Duration
	CheckInterval time.Duration
}

const (
	defaultTTL           = time.Minute * 5
	defaultCheckInterval = time.Minute
)

func validateConfig(config CacheConfig) CacheConfig {
	if config.TTL.Nanoseconds() == 0 {
		config.TTL = defaultTTL
	}
	if config.CheckInterval.Nanoseconds() == int64(0) {
		config.CheckInterval = defaultCheckInterval
	}
	return config
}

// CreateCacheStore will create an cacheory storage that will handle ttl of webhooks.
// listner and back and optional and can be nil
func CreateCacheStore(config CacheConfig, options ...Option) *Cache {
	cache := &Cache{
		hooks:  map[string]envelope{},
		config: validateConfig(config),
		stop:   make(chan struct{}),
	}
	cache.options = &storeConfig{
		logger: logging.DefaultLogger(),
	}

	for _, o := range options {
		o(cache.options)
	}

	ticker := time.NewTicker(cache.config.CheckInterval)
	go func() {
		for {
			select {
			case <-cache.stop:
				return
			case <-ticker.C:
				cache.CleanUp()
			}
		}
	}()
	return cache
}

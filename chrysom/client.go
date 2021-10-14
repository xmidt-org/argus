/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
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
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
)

type ClientConfig struct {
	BasicClientConfig
	ListenerClientConfig
}

type BasicClientConfig struct {
	// Address is the Argus URL (i.e. https://example-argus.io:8090)
	Address string

	// Bucket partition to be used by this client.
	Bucket string

	// HTTPClient refers to the client that will be used to send requests.
	// (Optional) Defaults to http.DefaultClient.
	HTTPClient *http.Client

	// Auth provides the mechanism to add auth headers to outgoing requests.
	// (Optional) If not provided, no auth headers are added.
	Auth Auth

	// Logger to be used by the client.
	// (Optional). By default a no op logger will be used.
	Logger log.Logger
}

// ListenerConfig contains config data to enable listening for the Argus client.
type ListenerClientConfig struct {
	// Listener provides a mechanism to fetch a copy of all items within a bucket on
	// an interval.
	// (Optional). If not provided, listening won't be enabled for this client.
	Listener Listener

	// PullInterval is how often listeners should get updates.
	// (Optional). Defaults to 5 seconds.
	PullInterval time.Duration

	// Logger to be used by the client.
	// (Optional). By default a no op logger will be used.
	Logger log.Logger
}

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

package model

// Key defines the field mapping to retrieve an item from storage.
type Key struct {
	// Bucket is the name for a collection or partition to which an item belongs.
	Bucket string `json:"bucket"`

	// ID is an item's identifier. Note that different buckets may have
	// items with the same ID.
	ID string `json:"id"`
}

// Item defines the abstract item to be stored.
type Item struct {
	// ID is the unique ID identifying this item. It is recommended this value is the resulting
	// value of a SHA256 calculation, using the unique attributes of the object being represented
	// (e.g. SHA256(<common_name>)). This will be used by argus to determine uniqueness of objects being stored or updated.
	ID string `json:"id"`

	// Data is the JSON object to be stored. Opaque to argus.
	Data map[string]interface{} `json:"data"`

	// TTL is the time to live in storage, specified in seconds.
	// Optional. When not set, items don't expire.
	TTL *int64 `json:"ttl,omitempty"`
}

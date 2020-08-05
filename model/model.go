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
	// Bucket is a collection of items.
	Bucket string `json:"bucket"`

	// ID is the unique ID for an item in a bucket
	ID string `json:"id"`
}

// Item defines the abstract item to be stored.
type Item struct {
	// Identifier is how the client refers to the object.
	Identifier string `json:"identifier"`

	// Data is an abstract json object
	Data map[string]interface{} `json:"data"`

	// TTL is the time to live in storage. If not provided and if the storage requires it the default configuration will be used.
	TTL int64 `json:"ttl,omitempty"`
}

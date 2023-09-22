// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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

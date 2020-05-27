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

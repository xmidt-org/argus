package chrysom

import "github.com/xmidt-org/argus/model"

// GetItemsInput contains all the needed parameters for GetItems
type GetItemsInput struct {
	// Bucket is the name of the bucket from which to fetch the items.
	Bucket string

	// Owner is the name of the owner items should be filtered on.
	// (Optional)
	Owner string
}

// GetItemsOutput contains all the output parameters for GetItems.
// Note: errors are reported separately.
type GetItemsOutput struct {
	Items []model.Item
}

type PushItemInput struct {
	ID     string
	Bucket string
	Owner  string
	Item   model.Item
}

type PushItemOutput struct {
	Result PushResult
}

type RemoveItemInput struct {
	ID     string
	Bucket string
	Owner  string
}

type RemoveItemOutput struct {
	Item model.Item
}

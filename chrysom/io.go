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

type GetItemsOutput struct {
	Items []model.Item
}

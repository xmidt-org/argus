package chrysom

import (
	"strings"

	"github.com/xmidt-org/argus/model"
)

// GetItemsInput contains input parameters for GetItems.
type GetItemsInput struct {
	// Bucket is the name of the bucket from which to fetch the items.
	Bucket string

	// Owner is the name of the owner items should be filtered on.
	// (Optional)
	Owner string
}

// GetItemsOutput output parameters for GetItems.
// Note: errors are reported separately.
type GetItemsOutput struct {
	Items []model.Item
}

// PushItemInput contains input parameters for PushItem.
type PushItemInput struct {
	// ID is the unique identifier for the item within the given bucket.
	ID string

	// Bucket is the name of the item grouping the item to be deleted belongs to.
	Bucket string

	// Owner is the name of the owner associated to this item.
	// (Optional)
	Owner string

	// Item contains the item data to be pushed.
	Item model.Item
}

// PushItemOput contains output parameters for PushItem.
type PushItemOutput struct {
	// Result reports whether the successful item push operation was
	// an update or a creation.
	Result PushResult
}

// RemoveItemInput contains input parameters for RemoveItem.
type RemoveItemInput struct {
	ID     string
	Bucket string
	Owner  string
}

// RemoveItemOutput contains output parameters for RemoveItem.
type RemoveItemOutput struct {
	// Item is the data of the item which was deleted.
	Item model.Item
}

func validateGetItemsInput(input *GetItemsInput) error {
	if input == nil {
		return ErrUndefinedInput
	}

	if len(input.Bucket) < 1 {
		return ErrBucketEmpty
	}
	return nil
}

func validatePushItemInputByValue(input PushItemInput) error {
	if len(input.Bucket) < 1 {
		return ErrBucketEmpty
	}

	if len(input.ID) < 1 || len(input.Item.ID) < 1 {
		return ErrItemIDEmpty
	}

	if !strings.EqualFold(input.ID, input.Item.ID) {
		return ErrItemIDMismatch
	}

	// TODO: we can also validate the ID format here
	// we'll need to create an exporter validator in argus though

	if len(input.Item.Data) < 1 {
		return ErrItemDataEmpty
	}

	return nil
}
func validatePushItemInput(bucket, owner, id string, item model.Item) error {
	if len(bucket) < 1 {
		return ErrBucketEmpty
	}

	if len(id) < 1 || len(item.ID) < 1 {
		return ErrItemIDEmpty
	}

	if !strings.EqualFold(id, item.ID) {
		return ErrItemIDMismatch
	}

	// TODO: we can also validate the ID format here
	// we'll need to create an exporter validator in argus though

	if len(item.Data) < 1 {
		return ErrItemDataEmpty
	}

	return nil
}

func validateRemoveItemInput(input *RemoveItemInput) error {
	if input == nil {
		return ErrUndefinedInput
	}

	if len(input.Bucket) < 1 {
		return ErrBucketEmpty
	}

	if len(input.ID) < 1 {
		return ErrItemIDEmpty
	}
	return nil
}

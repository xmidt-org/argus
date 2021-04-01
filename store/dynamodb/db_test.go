package dynamodb

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

var (
	errInternal = errors.New("internal dummy error")
	testKey     = model.Key{}
	testItem    = store.OwnableItem{}
)

func TestPushDAO(t *testing.T) {
	tcs := []struct {
		Description string
		PushErr     error
		ExpectedErr error
	}{
		{
			Description: "push error",
			PushErr:     errInternal,
			ExpectedErr: store.SanitizedError{
				Err:     errInternal,
				ErrHTTP: store.ErrHTTPOpFailed,
			},
		},
		{
			Description: "success",
			ExpectedErr: nil,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockService)
			m.On("Push", testKey, testItem).Return(&dynamodb.ConsumedCapacity{}, tc.PushErr)
			d := dao{
				s: m,
			}
			err := d.Push(testKey, testItem)
			assert.Equal(tc.ExpectedErr, err)
		})
	}
}

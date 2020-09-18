package store

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/model"
)

func TestGetItemEndpoint(t *testing.T) {
	testCases := []struct {
		Name             string
		ItemRequest      *getOrDeleteItemRequest
		DAOResponse      OwnableItem
		DAOResponseErr   error
		ExpectedResponse *OwnableItem
		ExpectedErr      error
	}{
		{
			Name: "DAO failure",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "Arthur",
			},
			DAOResponseErr: errors.New("database failure"),
			ExpectedErr:    errors.New("database failure"),
		},
		{
			Name: "Wrong owner",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "Kirby",
				key: model.Key{
					ID: "hammer",
				},
			},
			DAOResponse: OwnableItem{
				Owner: "Yoshi",
			},
			ExpectedErr: &KeyNotFoundError{Key: model.Key{
				ID: "hammer",
			}},
		},
		{
			Name: "Success",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "Argus",
			},
			DAOResponse: OwnableItem{
				Owner: "Argus",
			},
			ExpectedResponse: &OwnableItem{
				Owner: "Argus",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(MockDAO)
			m.On("Get", testCase.ItemRequest.key).Return(testCase.DAOResponse, error(testCase.DAOResponseErr))
			endpoint := newGetItemEndpoint(m)

			resp, err := endpoint(context.Background(), testCase.ItemRequest)
			if resp == nil {
				assert.Nil(testCase.ExpectedResponse)
			} else {
				ownableItemResponse := resp.(*OwnableItem)
				assert.Equal(testCase.ExpectedResponse, ownableItemResponse)
			}
			assert.Equal(testCase.ExpectedErr, err)
		})
	}
}

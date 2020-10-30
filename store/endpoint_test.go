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
				owner: "Donkey Kong",
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
				owner: "Cloud",
			},
			DAOResponse: OwnableItem{
				Owner: "Cloud",
			},
			ExpectedResponse: &OwnableItem{
				Owner: "Cloud",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(MockDAO)
			m.On("Get", testCase.ItemRequest.key).Return(testCase.DAOResponse, error(testCase.DAOResponseErr)).Once()
			endpoint := newGetItemEndpoint(m)

			resp, err := endpoint(context.Background(), testCase.ItemRequest)
			if testCase.ExpectedResponse == nil {
				assert.Nil(resp)
			} else {
				ownableItemResponse := resp.(*OwnableItem)
				assert.Equal(testCase.ExpectedResponse, ownableItemResponse)
			}
			assert.Equal(testCase.ExpectedErr, err)
			m.AssertExpectations(t)
		})
	}
}

func TestDeleteItemEndpoint(t *testing.T) {
	testCases := []struct {
		Name                 string
		ItemRequest          *getOrDeleteItemRequest
		GetDAOResponse       OwnableItem
		GetDAOResponseErr    error
		DeleteDAOResponse    OwnableItem
		DeleteDAOResponseErr error
		ExpectedResponse     *OwnableItem
		ExpectedErr          error
	}{
		{
			Name: "Get DAO failure",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "dsl",
			},
			GetDAOResponseErr: errors.New("error fetching item"),
			ExpectedErr:       errors.New("error fetching item"),
		},
		{
			Name: "Wrong owner",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "cable",
			},
			GetDAOResponse: OwnableItem{
				Owner: "fiber",
			},
			ExpectedErr: &KeyNotFoundError{},
		},
		{
			Name: "Deletion fails",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "dsl",
			},
			GetDAOResponse: OwnableItem{
				Owner: "dsl",
			},
			DeleteDAOResponseErr: errors.New("failed to delete item"),
			ExpectedErr:          errors.New("failed to delete item"),
		},
		{
			Name: "Successful deletion",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "dial-up",
			},
			GetDAOResponse: OwnableItem{
				Owner: "dial-up",
			},
			DeleteDAOResponse: OwnableItem{
				Owner: "dial-up",
			},
			ExpectedResponse: &OwnableItem{
				Owner: "dial-up",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(MockDAO)

			m.On("Get", testCase.ItemRequest.key).Return(testCase.GetDAOResponse, error(testCase.GetDAOResponseErr)).Once()

			// verify item is not deleted by user who doesn't own it
			allowDelete := testCase.ItemRequest.owner == "" || testCase.ItemRequest.owner == testCase.GetDAOResponse.Owner

			if testCase.GetDAOResponseErr != nil || !allowDelete {
				m.AssertNotCalled(t, "Delete", testCase.ItemRequest)
			} else {
				m.On("Delete", testCase.ItemRequest.key).Return(testCase.DeleteDAOResponse, error(testCase.DeleteDAOResponseErr)).Once()
			}

			deleteEndpoint := newDeleteItemEndpoint(m)

			resp, err := deleteEndpoint(context.Background(), testCase.ItemRequest)

			if testCase.ExpectedResponse == nil {
				assert.Nil(resp)
			}

			assert.Equal(testCase.ExpectedErr, err)
			m.AssertExpectations(t)
		})
	}
}

func TestUpdateItemEndpoint(t *testing.T) {
	testCases := []struct {
		Name                 string
		ItemRequest          *pushItemRequest
		UpdateDAOResponse    OwnableItem
		UpdateDAOResponseErr error
		GetDAOResponse       OwnableItem
		GetDAOResponseErr    error
		ExpectedResponse     *model.Key
		ExpectedErr          error
	}{
		{
			Name: "Update DAO failure",
			ItemRequest: &pushItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "random-UUID",
				},
				item: OwnableItem{
					Item: model.Item{
						Identifier: "strawberry",
					},
					Owner: "Bob",
				},
			},
			GetDAOResponse: OwnableItem{
				Owner: "Bob",
			},
			UpdateDAOResponseErr: errors.New("DB failed"),
			ExpectedErr:          errors.New("DB failed"),
		},
		{
			Name: "Get DAO failure",
			ItemRequest: &pushItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "random-UUID",
				},
				item: OwnableItem{
					Item: model.Item{
						Identifier: "strawberry",
					},
					Owner: "Bob",
				},
			},
			GetDAOResponseErr: errors.New("DB failed"),
			ExpectedErr:       errors.New("DB failed"),
		},
		{
			Name: "Wrong owner",
			ItemRequest: &pushItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "random-UUID",
				},
				item: OwnableItem{
					Item:  model.Item{},
					Owner: "cable",
				},
			},
			GetDAOResponse: OwnableItem{
				Owner: "fiber",
			},
			ExpectedErr: &KeyNotFoundError{
				Key: model.Key{
					Bucket: "fruits",
					ID:     "random-UUID",
				},
			},
		},
		{
			Name: "Successful Update",
			ItemRequest: &pushItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "random-UUID",
				},
				item: OwnableItem{
					Item:  model.Item{},
					Owner: "cable",
				},
			},
			GetDAOResponse: OwnableItem{
				Owner: "cable",
			},
			UpdateDAOResponse: OwnableItem{
				Owner: "cable",
				Item:  model.Item{},
			},
			ExpectedResponse: &model.Key{
				Bucket: "fruits",
				ID:     "random-UUID",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(MockDAO)

			if testCase.UpdateDAOResponseErr == nil {
				m.On("Push", testCase.ItemRequest.key, testCase.ItemRequest.item).Return(nil).Once()
			} else {
				m.On("Push", testCase.ItemRequest.key, testCase.ItemRequest.item).Return(testCase.UpdateDAOResponseErr).Once()
			}

			m.On("Get", testCase.ItemRequest.key).Return(testCase.GetDAOResponse, error(testCase.GetDAOResponseErr)).Once()

			endpoint := newUpdateItemEndpoint(m)
			resp, err := endpoint(context.Background(), testCase.ItemRequest)

			if testCase.ExpectedErr == nil {
				assert.Nil(err)
				assert.Equal(&testCase.ItemRequest.key, resp)
				m.AssertExpectations(t)
			} else {
				assert.Equal(testCase.ExpectedErr, err)
			}

		})
	}
}

func TestGetAllItemsEndpoint(t *testing.T) {
	t.Run("DAOFails", testGetAllItemsEndpointDAOFails)
	t.Run("FilteredItems", testGetAllItemsEndpointFiltered)
}

func testGetAllItemsEndpointDAOFails(t *testing.T) {
	assert := assert.New(t)
	m := new(MockDAO)
	itemsRequest := &getAllItemsRequest{
		bucket: "sports-cars",
		owner:  "alfa-romeo",
	}
	mockedErr := errors.New("sports cars api is down")
	m.On("GetAll", "sports-cars").Return(map[string]OwnableItem{}, mockedErr).Once()

	endpoint := newGetAllItemsEndpoint(m)
	resp, err := endpoint(context.Background(), itemsRequest)

	assert.Nil(resp)
	assert.Equal(mockedErr, err)
	m.AssertExpectations(t)
}

func testGetAllItemsEndpointFiltered(t *testing.T) {
	assert := assert.New(t)
	m := new(MockDAO)
	itemsRequest := &getAllItemsRequest{
		bucket: "sports-cars",
		owner:  "alfa-romeo",
	}
	mockedItems := map[string]OwnableItem{
		"mustang": OwnableItem{
			Owner: "ford",
		},
		"4c-spider": OwnableItem{
			Owner: "alfa-romeo",
		},
		"gtr": OwnableItem{
			Owner: "nissan",
		},
		"giulia": OwnableItem{
			Owner: "alfa-romeo",
		},
	}
	m.On("GetAll", "sports-cars").Return(mockedItems, error(nil)).Once()

	endpoint := newGetAllItemsEndpoint(m)
	resp, err := endpoint(context.Background(), itemsRequest)

	expectedItems := map[string]OwnableItem{
		"4c-spider": OwnableItem{
			Owner: "alfa-romeo",
		},
		"giulia": OwnableItem{
			Owner: "alfa-romeo",
		},
	}

	assert.Equal(expectedItems, resp)
	assert.Nil(err)
	m.AssertExpectations(t)
}

func TestPushItemEndpoint(t *testing.T) {
	t.Run("DAOFails", testPushItemEndpointDAOFails)
	t.Run("Happy Path", testPushItemEndpointHappyPath)
}

func testPushItemEndpointHappyPath(t *testing.T) {
	assert := assert.New(t)
	m := new(MockDAO)
	key := model.Key{
		Bucket: "fruits",
		ID:     "strawberry",
	}

	item := OwnableItem{
		Item: model.Item{
			Identifier: "strawberry",
		},
		Owner: "Bob",
	}

	m.On("Push", key, item).Return(nil).Once()
	endpoint := newPushItemEndpoint(m)
	resp, err := endpoint(context.Background(), &pushItemRequest{
		key:  key,
		item: item,
	})
	assert.Nil(err)
	assert.Equal(&key, resp)
	m.AssertExpectations(t)
}

func testPushItemEndpointDAOFails(t *testing.T) {
	assert := assert.New(t)
	m := new(MockDAO)
	m.On("Push", model.Key{}, OwnableItem{}).Return(errors.New("DB failed")).Once()
	endpoint := newPushItemEndpoint(m)
	resp, err := endpoint(context.Background(), &pushItemRequest{})
	assert.Nil(resp)
	assert.Equal(errors.New("DB failed"), err)
	m.AssertExpectations(t)
}

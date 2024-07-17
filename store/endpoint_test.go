// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package store

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/ancla/model"
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
			ExpectedErr: accessDeniedErr,
		},

		{
			Name: "Wrong owner but admin mode",
			ItemRequest: &getOrDeleteItemRequest{
				owner: "Kirby",
				key: model.Key{
					ID: "hammer",
				},
				adminMode: true,
			},
			DAOResponse: OwnableItem{
				Owner: "Yoshi",
			},

			ExpectedResponse: &OwnableItem{
				Owner: "Yoshi",
			},
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
			m.On("Get", testCase.ItemRequest.key).Return(testCase.DAOResponse, testCase.DAOResponseErr).Once()
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
			ExpectedErr: accessDeniedErr,
		},

		{
			Name: "Wrong owner but admin mode",
			ItemRequest: &getOrDeleteItemRequest{
				owner:     "cable",
				adminMode: true,
			},
			GetDAOResponse: OwnableItem{
				Owner: "fiber",
			},

			DeleteDAOResponse: OwnableItem{
				Owner: "fiber",
			},

			ExpectedResponse: &OwnableItem{
				Owner: "fiber",
			},
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

			m.On("Get", testCase.ItemRequest.key).Return(testCase.GetDAOResponse, testCase.GetDAOResponseErr).Once()

			// verify item is not deleted by user who doesn't own it
			allowDelete := testCase.ItemRequest.adminMode || testCase.ItemRequest.owner == testCase.GetDAOResponse.Owner

			if testCase.GetDAOResponseErr != nil || !allowDelete {
				m.AssertNotCalled(t, "Delete", testCase.ItemRequest)
			} else {
				m.On("Delete", testCase.ItemRequest.key).Return(testCase.DeleteDAOResponse, testCase.DeleteDAOResponseErr).Once()
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

func TestSetItemEndpoint(t *testing.T) {
	testCases := []struct {
		Name               string
		ItemRequest        *setItemRequest
		PushDAOResponse    OwnableItem
		PushDAOResponseErr error
		GetDAOResponse     OwnableItem
		GetDAOResponseErr  error
		ExpectedResponse   *setItemResponse
		ExpectedErr        error
	}{
		{
			Name: "Push DAO failure",
			ItemRequest: &setItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
				},
				item: OwnableItem{
					Item: model.Item{
						ID: "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
					},
					Owner: "Bob",
				},
			},
			GetDAOResponse: OwnableItem{
				Owner: "Bob",
			},
			PushDAOResponseErr: errors.New("DB failed"),
			ExpectedErr:        errors.New("DB failed"),
		},
		{
			Name: "Get DAO failure",
			ItemRequest: &setItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
				},
				item: OwnableItem{
					Item: model.Item{
						ID: "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
					},
					Owner: "Bob",
				},
			},
			GetDAOResponseErr: errors.New("DB failed"),
			ExpectedErr:       errors.New("DB failed"),
		},
		{
			Name: "Wrong owner",
			ItemRequest: &setItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
				},
				item: OwnableItem{
					Item:  model.Item{},
					Owner: "cable",
				},
			},
			GetDAOResponse: OwnableItem{
				Owner: "fiber",
			},
			ExpectedErr: accessDeniedErr,
		},
		{
			Name: "Successful Update. Wrong owner but admin mode",
			ItemRequest: &setItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
				},
				item: OwnableItem{
					Item:  model.Item{},
					Owner: "shouldBeIgnored",
				},
				adminMode: true,
			},
			GetDAOResponse: OwnableItem{
				Owner: "cable",
			},
			PushDAOResponse: OwnableItem{
				Owner: "cable",
				Item:  model.Item{},
			},
			ExpectedResponse: &setItemResponse{
				existingResource: true,
			},
		},

		{
			Name: "Successful Creation",
			ItemRequest: &setItemRequest{
				key: model.Key{
					Bucket: "fruits",
					ID:     "XnN_iR2xF1RCo5_ec-UdeBpUVQbXHJVHem3rWYi9f5o",
				},
				item: OwnableItem{
					Item:  model.Item{},
					Owner: "cable",
				},
			},
			GetDAOResponseErr: ErrItemNotFound,
			PushDAOResponse: OwnableItem{
				Owner: "cable",
				Item:  model.Item{},
			},
			ExpectedResponse: &setItemResponse{
				existingResource: false,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(MockDAO)

			pushItem := OwnableItem{
				Item: model.Item{
					ID:   testCase.ItemRequest.item.ID,
					Data: testCase.ItemRequest.item.Data,
				},
				Owner: testCase.ItemRequest.item.Owner,
			}

			if testCase.ItemRequest.adminMode {
				pushItem.Owner = testCase.GetDAOResponse.Owner
			}

			m.On("Push", testCase.ItemRequest.key, pushItem).Return(testCase.PushDAOResponseErr).Once()
			m.On("Get", testCase.ItemRequest.key).Return(testCase.GetDAOResponse, testCase.GetDAOResponseErr).Once()

			endpoint := newSetItemEndpoint(m)
			resp, err := endpoint(context.Background(), testCase.ItemRequest)

			if testCase.ExpectedErr == nil {
				assert.Nil(err)
				assert.Equal(testCase.ExpectedResponse, resp)
				m.AssertExpectations(t)
			} else {
				assert.Equal(testCase.ExpectedErr, err)
			}
		})
	}
}

func TestGetAllItemsEndpoint(t *testing.T) {
	testCases := []struct {
		Name                 string
		ItemRequest          *getAllItemsRequest
		GetAllDAOResponse    map[string]OwnableItem
		GetAllDAOResponseErr error
		ExpectedResponse     map[string]OwnableItem
		ExpectedErr          error
	}{
		{
			Name: "DAO failure",
			ItemRequest: &getAllItemsRequest{
				bucket: "sports-cars",
				owner:  "alfa-romeo",
			},
			GetAllDAOResponseErr: errors.New("DB failed"),
			ExpectedErr:          errors.New("DB failed"),
		},
		{
			Name: "Filtered results",
			ItemRequest: &getAllItemsRequest{
				bucket: "sports-cars",
				owner:  "alfa-romeo",
			},
			GetAllDAOResponse: map[string]OwnableItem{
				"mustang": {
					Owner: "ford",
				},
				"4c-spider": {
					Owner: "alfa-romeo",
				},
				"gtr": {
					Owner: "nissan",
				},
				"giulia": {
					Owner: "alfa-romeo",
				},
			},

			ExpectedResponse: map[string]OwnableItem{
				"4c-spider": {
					Owner: "alfa-romeo",
				},

				"giulia": {
					Owner: "alfa-romeo",
				},
			},
		},

		{
			Name: "Filtered admin mode",
			ItemRequest: &getAllItemsRequest{
				bucket:    "sports-cars",
				owner:     "alfa-romeo",
				adminMode: true,
			},
			GetAllDAOResponse: map[string]OwnableItem{
				"mustang": {
					Owner: "ford",
				},
				"4c-spider": {
					Owner: "alfa-romeo",
				},
				"gtr": {
					Owner: "nissan",
				},
				"giulia": {
					Owner: "alfa-romeo",
				},
			},

			ExpectedResponse: map[string]OwnableItem{
				"4c-spider": {
					Owner: "alfa-romeo",
				},
				"giulia": {
					Owner: "alfa-romeo",
				},
			},
		},

		{
			Name: "Unfiltered Admin mode",
			ItemRequest: &getAllItemsRequest{
				bucket:    "sports-cars",
				owner:     "",
				adminMode: true,
			},
			GetAllDAOResponse: map[string]OwnableItem{
				"mustang": {
					Owner: "ford",
				},
				"4c-spider": {
					Owner: "alfa-romeo",
				},
				"gtr": {
					Owner: "nissan",
				},
				"giulia": {
					Owner: "alfa-romeo",
				},
			},

			ExpectedResponse: map[string]OwnableItem{
				"mustang": {
					Owner: "ford",
				},
				"4c-spider": {
					Owner: "alfa-romeo",
				},
				"gtr": {
					Owner: "nissan",
				},
				"giulia": {
					Owner: "alfa-romeo",
				},
			},
		},

		{
			Name: "Empty results",
			ItemRequest: &getAllItemsRequest{
				bucket: "sports-cars",
				owner:  "volkswagen",
			},
			GetAllDAOResponse: map[string]OwnableItem{
				"mustang": {
					Owner: "ford",
				},
				"giulia": {
					Owner: "alfa-romeo",
				},
			},

			ExpectedResponse: map[string]OwnableItem{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			m := new(MockDAO)

			m.On("GetAll", testCase.ItemRequest.bucket).Return(testCase.GetAllDAOResponse, testCase.GetAllDAOResponseErr)

			endpoint := newGetAllItemsEndpoint(m)
			resp, err := endpoint(context.Background(), testCase.ItemRequest)
			if testCase.ExpectedErr == nil {
				assert.Nil(err)
				assert.Equal(testCase.ExpectedResponse, resp)
			} else {
				assert.Equal(testCase.ExpectedErr, err)
			}
		})
	}
}

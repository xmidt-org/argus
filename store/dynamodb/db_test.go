// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package dynamodb

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
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

func TestGetDAO(t *testing.T) {
	tcs := []struct {
		Description string
		GetErr      error
		ExpectedErr error
	}{
		{
			Description: "get error",
			GetErr:      errInternal,
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
			m.On("Get", testKey).Return(testItem, &dynamodb.ConsumedCapacity{}, tc.GetErr)
			d := dao{s: m}
			item, err := d.Get(testKey)
			assert.Equal(testItem, item)
			assert.Equal(tc.ExpectedErr, err)
		})
	}
}

func TestDeleteDAO(t *testing.T) {
	tcs := []struct {
		Description string
		DeleteErr   error
		ExpectedErr error
	}{
		{
			Description: "delete error",
			DeleteErr:   errInternal,
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
			m.On("Delete", testKey).Return(testItem, &dynamodb.ConsumedCapacity{}, tc.DeleteErr)
			d := dao{s: m}
			item, err := d.Delete(testKey)
			assert.Equal(testItem, item)
			assert.Equal(tc.ExpectedErr, err)
		})
	}
}

func TestGetAllDAO(t *testing.T) {
	tcs := []struct {
		Description string
		GetAllErr   error
		ExpectedErr error
	}{
		{
			Description: "getAll error",
			GetAllErr:   errInternal,
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
	testItems := map[string]store.OwnableItem{
		"454349e422f05297191ead13e21d3db520e5abef52055e4964b82fb213f593a1": {
			Item: model.Item{
				ID: "454349e422f05297191ead13e21d3db520e5abef52055e4964b82fb213f593a1",
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockService)
			m.On("GetAll", "testBucket").Return(testItems, &dynamodb.ConsumedCapacity{}, tc.GetAllErr)
			d := dao{s: m}
			items, err := d.GetAll("testBucket")
			assert.Equal(testItems, items)
			assert.Equal(tc.ExpectedErr, err)
		})
	}
}

func TestSanitizeError(t *testing.T) {
	dynamodbValidationErr := awserr.New("ValidationException", "some dynamodb specific input validation error", errInternal)
	tcs := []struct {
		Description     string
		InputErr        error
		ExpectedErr     error
		ExpectedCode    int
		ExpectedMessage string
		ExpectedErrHTTP error
	}{
		{
			Description:     "Validation error",
			InputErr:        dynamodbValidationErr,
			ExpectedErr:     dynamodbValidationErr,
			ExpectedErrHTTP: errHTTPBadRequest,
		},
		{
			Description:     "Other error",
			InputErr:        errInternal,
			ExpectedErr:     errInternal,
			ExpectedErrHTTP: store.ErrHTTPOpFailed,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			err := sanitizeError(tc.InputErr)
			var sErr store.SanitizedError
			assert.True(errors.As(err, &sErr))
			assert.Equal(tc.ExpectedErr, sErr.Err)
			assert.EqualValues(tc.ExpectedErrHTTP, sErr.ErrHTTP)
		})
	}
}

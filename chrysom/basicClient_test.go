// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/sallust"
)

const failingURL = "nowhere://"

var (
	_ Pusher = &BasicClient{}
	_ Reader = &BasicClient{}
)

func TestValidateBasicConfig(t *testing.T) {
	type testCase struct {
		Description    string
		Input          *BasicClientConfig
		ExpectedErr    error
		ExpectedConfig *BasicClientConfig
	}

	allDefaultsCaseConfig := &BasicClientConfig{
		HTTPClient: http.DefaultClient,
		Address:    "http://awesome-argus-hostname.io",
		Bucket:     "bucket-name",
	}
	myAmazingClient := &http.Client{Timeout: time.Hour}
	allDefinedCaseConfig := &BasicClientConfig{
		HTTPClient: myAmazingClient,
		Address:    "http://legit-argus-hostname.io",
		Auth:       Auth{},
		Bucket:     "amazing-bucket",
	}

	tcs := []testCase{
		{
			Description: "No address",
			Input: &BasicClientConfig{
				Bucket: "bucket-name",
			},
			ExpectedErr: ErrAddressEmpty,
		},
		{
			Description: "No bucket",
			Input: &BasicClientConfig{
				Address: "http://awesome-argus-hostname.io",
			},
			ExpectedErr: ErrBucketEmpty,
		},
		{
			Description: "All default values",
			Input: &BasicClientConfig{
				Address: "http://awesome-argus-hostname.io",
				Bucket:  "bucket-name",
			},
			ExpectedConfig: allDefaultsCaseConfig,
		},
		{
			Description: "All defined",
			Input: &BasicClientConfig{
				Address:    "http://legit-argus-hostname.io",
				Bucket:     "amazing-bucket",
				HTTPClient: myAmazingClient,
			},
			ExpectedConfig: allDefinedCaseConfig,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			err := validateBasicConfig(tc.Input)
			assert.Equal(tc.ExpectedErr, err)
			if tc.ExpectedErr == nil {
				assert.Equal(tc.ExpectedConfig, tc.Input)
			}
		})
	}
}

func TestSendRequest(t *testing.T) {
	type testCase struct {
		Description      string
		Owner            string
		Method           string
		URL              string
		Body             []byte
		AcquirerFails    bool
		ClientDoFails    bool
		ExpectedResponse response
		ExpectedErr      error
	}

	tcs := []testCase{
		{
			Description: "New Request fails",
			Method:      "what method?",
			URL:         "http://argus-hostname.io",
			ExpectedErr: errNewRequestFailure,
		},
		{
			Description:   "Auth acquirer fails",
			Method:        http.MethodGet,
			URL:           "http://argus-hostname.io",
			AcquirerFails: true,
			ExpectedErr:   ErrAuthAcquirerFailure,
		},
		{
			Description:   "Client Do fails",
			Method:        http.MethodPut,
			ClientDoFails: true,
			ExpectedErr:   errDoRequestFailure,
		},
		{
			Description: "Happy path",
			Method:      http.MethodPut,
			URL:         "http://argus-hostname.io",
			Body:        []byte("testing"),
			Owner:       "HappyCaseOwner",
			ExpectedResponse: response{
				Code: http.StatusOK,
				Body: []byte("testing"),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			echoHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(tc.Owner, r.Header.Get(store.ItemOwnerHeaderKey))
				rw.WriteHeader(http.StatusOK)
				bodyBytes, err := io.ReadAll(r.Body)
				require.Nil(err)
				rw.Write(bodyBytes)
			})

			server := httptest.NewServer(echoHandler)
			defer server.Close()

			client, err := NewBasicClient(BasicClientConfig{
				HTTPClient: server.Client(),
				Address:    "http://argus-hostname.io",
				Bucket:     "bucket-name",
			}, sallust.Get)

			if tc.AcquirerFails {
				client.auth = acquirerFunc(failAcquirer)
			}

			var URL = server.URL
			if tc.ClientDoFails {
				URL = "http://should-definitely-fail.net"
			}

			assert.Nil(err)
			resp, err := client.sendRequest(context.TODO(), tc.Owner, tc.Method, URL, bytes.NewBuffer(tc.Body))

			if tc.ExpectedErr == nil {
				assert.Equal(http.StatusOK, resp.Code)
				assert.Equal(tc.ExpectedResponse, resp)
			} else {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
		})
	}
}

func TestGetItems(t *testing.T) {
	type testCase struct {
		Description           string
		ResponsePayload       []byte
		ResponseCode          int
		ShouldMakeRequestFail bool
		ShouldDoRequestFail   bool
		ExpectedErr           error
		ExpectedOutput        Items
	}

	tcs := []testCase{
		{

			Description:           "Make request fails",
			ShouldMakeRequestFail: true,
			ExpectedErr:           ErrAuthAcquirerFailure,
		},
		{
			Description:         "Do request fails",
			ShouldDoRequestFail: true,
			ExpectedErr:         errDoRequestFailure,
		},
		{
			Description:  "Unauthorized",
			ResponseCode: http.StatusForbidden,
			ExpectedErr:  ErrFailedAuthentication,
		},
		{
			Description:  "Bad request",
			ResponseCode: http.StatusBadRequest,
			ExpectedErr:  ErrBadRequest,
		},
		{
			Description:  "Other non-success",
			ResponseCode: http.StatusInternalServerError,
			ExpectedErr:  errNonSuccessResponse,
		},
		{
			Description:     "Payload unmarshal error",
			ResponseCode:    http.StatusOK,
			ResponsePayload: []byte("[{}"),
			ExpectedErr:     errJSONUnmarshal,
		},
		{
			Description:     "Happy path",
			ResponseCode:    http.StatusOK,
			ResponsePayload: getItemsValidPayload(),
			ExpectedOutput:  getItemsHappyOutput(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				bucket  = "bucket-name"
				owner   = "owner-name"
			)

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(http.MethodGet, r.Method)
				assert.Equal(owner, r.Header.Get(store.ItemOwnerHeaderKey))
				assert.Equal(fmt.Sprintf("%s/%s", storeAPIPath, bucket), r.URL.Path)

				rw.WriteHeader(tc.ResponseCode)
				rw.Write(tc.ResponsePayload)
			}))

			client, err := NewBasicClient(BasicClientConfig{
				HTTPClient: server.Client(),
				Address:    server.URL,
				Bucket:     bucket,
			}, sallust.Get)

			require.Nil(err)

			if tc.ShouldMakeRequestFail {
				client.auth = acquirerFunc(failAcquirer)
			}

			if tc.ShouldDoRequestFail {
				client.storeBaseURL = failingURL
			}

			output, err := client.GetItems(context.TODO(), owner)

			assert.True(errors.Is(err, tc.ExpectedErr))
			if tc.ExpectedErr == nil {
				assert.EqualValues(tc.ExpectedOutput, output)
			}
		})
	}
}

func TestPushItem(t *testing.T) {
	type testCase struct {
		Description           string
		Item                  model.Item
		Owner                 string
		ResponseCode          int
		ShouldEraseBucket     bool
		ShouldRespNonSuccess  bool
		ShouldMakeRequestFail bool
		ShouldDoRequestFail   bool
		ExpectedErr           error
		ExpectedOutput        PushResult
	}

	validItem := model.Item{
		ID: "252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
		Data: map[string]interface{}{
			"field0": float64(0),
			"nested": map[string]interface{}{
				"response": "wow",
			},
		}}

	tcs := []testCase{
		{
			Description: "Item ID Missing",
			Item:        model.Item{Data: map[string]interface{}{}},
			ExpectedErr: ErrItemIDEmpty,
		},
		{
			Description: "Item Data missing",
			Item:        model.Item{ID: validItem.ID},
			ExpectedErr: ErrItemDataEmpty,
		},
		{
			Description:           "Make request fails",
			Item:                  validItem,
			ShouldMakeRequestFail: true,
			ExpectedErr:           ErrAuthAcquirerFailure,
		},
		{
			Description:         "Do request fails",
			Item:                validItem,
			ShouldDoRequestFail: true,
			ExpectedErr:         errDoRequestFailure,
		},
		{
			Description:  "Unauthorized",
			Item:         validItem,
			ResponseCode: http.StatusForbidden,
			ExpectedErr:  ErrFailedAuthentication,
		},
		{
			Description:  "Bad request",
			Item:         validItem,
			ResponseCode: http.StatusBadRequest,
			ExpectedErr:  ErrBadRequest,
		},
		{
			Description:  "Other non-success",
			Item:         validItem,
			ResponseCode: http.StatusInternalServerError,
			ExpectedErr:  errNonSuccessResponse,
		},
		{
			Description:    "Create success",
			Item:           validItem,
			ResponseCode:   http.StatusCreated,
			ExpectedOutput: CreatedPushResult,
		},
		{
			Description:    "Update success",
			Item:           validItem,
			ResponseCode:   http.StatusOK,
			ExpectedOutput: UpdatedPushResult,
		},
		{
			Description:    "Update success with owner",
			Item:           validItem,
			ResponseCode:   http.StatusOK,
			Owner:          "owner-name",
			ExpectedOutput: UpdatedPushResult,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				bucket  = "bucket-name"
				id      = "252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111"
			)

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(fmt.Sprintf("%s/%s/%s", storeAPIPath, bucket, id), r.URL.Path)
				assert.Equal(tc.Owner, r.Header.Get(store.ItemOwnerHeaderKey))
				rw.WriteHeader(tc.ResponseCode)

				if tc.ResponseCode == http.StatusCreated || tc.ResponseCode == http.StatusOK {
					payload, err := io.ReadAll(r.Body)
					require.Nil(err)
					var item model.Item
					err = json.Unmarshal(payload, &item)
					require.Nil(err)
					assert.EqualValues(tc.Item, item)
				}
			}))

			client, err := NewBasicClient(BasicClientConfig{
				HTTPClient: server.Client(),
				Address:    server.URL,
				Bucket:     bucket,
			}, sallust.Get)

			if tc.ShouldMakeRequestFail {
				client.auth = acquirerFunc(failAcquirer)
			}

			if tc.ShouldDoRequestFail {
				client.storeBaseURL = failingURL
			}

			if tc.ShouldEraseBucket {
				bucket = ""
			}

			require.Nil(err)
			output, err := client.PushItem(context.TODO(), tc.Owner, tc.Item)

			if tc.ExpectedErr == nil {
				assert.EqualValues(tc.ExpectedOutput, output)
			} else {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
		})
	}
}

func TestRemoveItem(t *testing.T) {
	type testCase struct {
		Description           string
		ResponsePayload       []byte
		ResponseCode          int
		Owner                 string
		ShouldRespNonSuccess  bool
		ShouldMakeRequestFail bool
		ShouldDoRequestFail   bool
		ExpectedErr           error
		ExpectedOutput        model.Item
	}

	tcs := []testCase{
		{
			Description:           "Make request fails",
			ShouldMakeRequestFail: true,
			ExpectedErr:           ErrAuthAcquirerFailure,
		},
		{
			Description:         "Do request fails",
			ShouldDoRequestFail: true,
			ExpectedErr:         errDoRequestFailure,
		},
		{
			Description:  "Unauthorized",
			ResponseCode: http.StatusForbidden,
			ExpectedErr:  ErrFailedAuthentication,
		},
		{
			Description:  "Bad request",
			ResponseCode: http.StatusBadRequest,
			ExpectedErr:  ErrBadRequest,
		},
		{
			Description:  "Other non-success",
			ResponseCode: http.StatusInternalServerError,
			ExpectedErr:  errNonSuccessResponse,
		},
		{
			Description:     "Unmarshal failure",
			ResponseCode:    http.StatusOK,
			ResponsePayload: []byte("{{}"),
			ExpectedErr:     errJSONUnmarshal,
		},
		{
			Description:     "Succcess",
			ResponseCode:    http.StatusOK,
			ResponsePayload: getRemoveItemValidPayload(),
			ExpectedOutput:  getRemoveItemHappyOutput(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
				bucket  = "bucket-name"
				// nolint:gosec
				id = "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7"
			)
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(fmt.Sprintf("%s/%s/%s", storeAPIPath, bucket, id), r.URL.Path)
				assert.Equal(http.MethodDelete, r.Method)
				rw.WriteHeader(tc.ResponseCode)
				rw.Write(tc.ResponsePayload)
			}))

			client, err := NewBasicClient(BasicClientConfig{
				HTTPClient: server.Client(),
				Address:    server.URL,
				Bucket:     bucket,
			}, sallust.Get)

			if tc.ShouldMakeRequestFail {
				client.auth = acquirerFunc(failAcquirer)
			}

			if tc.ShouldDoRequestFail {
				client.storeBaseURL = failingURL
			}

			require.Nil(err)
			output, err := client.RemoveItem(context.TODO(), id, tc.Owner)

			if tc.ExpectedErr == nil {
				assert.EqualValues(tc.ExpectedOutput, output)
			} else {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
		})
	}
}

func TestTranslateStatusCode(t *testing.T) {
	type testCase struct {
		Description string
		Code        int
		ExpectedErr error
	}

	tcs := []testCase{
		{
			Code:        http.StatusForbidden,
			ExpectedErr: ErrFailedAuthentication,
		},
		{
			Code:        http.StatusUnauthorized,
			ExpectedErr: ErrFailedAuthentication,
		},
		{
			Code:        http.StatusBadRequest,
			ExpectedErr: ErrBadRequest,
		},
		{
			Code:        http.StatusInternalServerError,
			ExpectedErr: errNonSuccessResponse,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.ExpectedErr, translateNonSuccessStatusCode(tc.Code))
		})
	}
}

func failAcquirer() (string, error) {
	return "", errors.New("always fail")
}

type acquirerFunc func() (string, error)

func (a acquirerFunc) Acquire() (string, error) {
	return a()
}

func getRemoveItemValidPayload() []byte {
	return []byte(`
	{
		"id": "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
		"data": {
		  "words": [
			"Hello","World"
		  ],
		  "year": 2021
		},
		"ttl": 100
	}`)
}

func getRemoveItemHappyOutput() model.Item {
	return model.Item{
		ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
		Data: map[string]interface{}{
			"words": []interface{}{"Hello", "World"},
			"year":  float64(2021),
		},
		TTL: aws.Int64(100),
	}
}

func getItemsValidPayload() []byte {
	return []byte(`[{
    "id": "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
    "data": {
      "words": [
        "Hello","World"
      ],
      "year": 2021
    },
    "ttl": 255
  }]`)
}

func getItemsHappyOutput() Items {
	return []model.Item{
		{
			ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
			Data: map[string]interface{}{
				"words": []interface{}{"Hello", "World"},
				"year":  float64(2021),
			},
			TTL: aws.Int64(255),
		},
	}
}

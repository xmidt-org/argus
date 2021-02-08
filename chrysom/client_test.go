package chrysom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

const failingURL = "nowhere://"

func TestInterface(t *testing.T) {
	assert := assert.New(t)
	var client interface{}
	client = &Client{}
	_, ok := client.(Pusher)
	assert.True(ok, "not a pusher")
	_, ok = client.(Reader)
	assert.True(ok, "not a reader")
	_, ok = client.(PushReader)
	assert.True(ok, "not a PushReader")
}

func TestValidateConfig(t *testing.T) {
	type testCase struct {
		Description    string
		Input          *ClientConfig
		ExpectedErr    error
		ExpectedConfig *ClientConfig
	}

	allDefaultsCaseConfig := &ClientConfig{
		HTTPClient:      http.DefaultClient,
		PullInterval:    time.Second * 5,
		Logger:          log.NewNopLogger(),
		Address:         "http://awesome-argus-hostname.io",
		MetricsProvider: provider.NewDiscardProvider(),
	}

	myAmazingClient := &http.Client{Timeout: time.Hour}
	allDefinedCaseConfig := &ClientConfig{
		HTTPClient:      myAmazingClient,
		PullInterval:    time.Hour * 24,
		Address:         "http://legit-argus-hostname.io",
		Auth:            Auth{},
		MetricsProvider: provider.NewDiscardProvider(),
		Logger:          log.NewJSONLogger(ioutil.Discard),
	}

	tcs := []testCase{
		{
			Description: "All default values",
			Input: &ClientConfig{
				Address:         "http://awesome-argus-hostname.io",
				MetricsProvider: provider.NewDiscardProvider(),
			},
			ExpectedConfig: allDefaultsCaseConfig,
		},

		{
			Description: "No metrics provider",
			Input: &ClientConfig{
				Address: "http://awesome-argus-hostname.io",
			},
			ExpectedErr: ErrUndefinedMetricsProvider,
		},
		{
			Description: "No address",
			Input: &ClientConfig{
				MetricsProvider: provider.NewDiscardProvider(),
			},
			ExpectedErr: ErrAddressEmpty,
		},

		{
			Description: "All defined",
			Input: &ClientConfig{
				MetricsProvider: provider.NewDiscardProvider(),
				Address:         "http://legit-argus-hostname.io",
				HTTPClient:      myAmazingClient,
				PullInterval:    time.Hour * 24,
				Logger:          log.NewJSONLogger(ioutil.Discard),
			},
			ExpectedConfig: allDefinedCaseConfig,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			err := validateConfig(tc.Input)
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
			ExpectedErr: ErrNewRequestFailure,
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
			ExpectedErr:   ErrDoRequestFailure,
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
				bodyBytes, err := ioutil.ReadAll(r.Body)
				require.Nil(err)
				rw.Write(bodyBytes)
			})

			server := httptest.NewServer(echoHandler)
			defer server.Close()

			client, err := NewClient(ClientConfig{
				HTTPClient:      server.Client(),
				Address:         "http://argus-hostname.io",
				MetricsProvider: provider.NewDiscardProvider(),
			})

			if tc.AcquirerFails {
				client.auth = acquirerFunc(failAcquirer)
			}

			var URL = server.URL
			if tc.ClientDoFails {
				URL = "http://should-definitely-fail.net"
			}

			assert.Nil(err)
			resp, err := client.sendRequest(tc.Owner, tc.Method, URL, bytes.NewBuffer(tc.Body))

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
		ShouldRespNonSuccess  bool
		ShouldEraseBucket     bool
		ShouldMakeRequestFail bool
		ShouldDoRequestFail   bool
		ExpectedErr           error
		ExpectedOutput        Items
	}

	tcs := []testCase{
		{
			Description:       "Bucket is required",
			ShouldEraseBucket: true,
			ExpectedErr:       ErrBucketEmpty,
		},
		{

			Description:           "Make request fails",
			ShouldMakeRequestFail: true,
			ExpectedErr:           ErrAuthAcquirerFailure,
		},
		{
			Description:         "Do request fails",
			ShouldDoRequestFail: true,
			ExpectedErr:         ErrDoRequestFailure,
		},
		{
			Description:          "Non success code",
			ShouldRespNonSuccess: true,
			ExpectedErr:          ErrGetItemsFailure,
		},
		{
			Description:     "Payload unmarshal error",
			ResponsePayload: []byte("[{}"),
			ExpectedErr:     ErrJSONUnmarshal,
		},
		{
			Description:     "Happy path",
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

			if tc.ShouldEraseBucket {
				bucket = ""
			}

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(http.MethodGet, r.Method)
				assert.Equal(owner, r.Header.Get(store.ItemOwnerHeaderKey))
				assert.Equal(fmt.Sprintf("%s/%s", storeAPIPath, bucket), r.URL.Path)

				if tc.ShouldRespNonSuccess {
					rw.WriteHeader(http.StatusBadRequest)
				}

				rw.Write(tc.ResponsePayload)
			}))

			client, err := NewClient(ClientConfig{
				HTTPClient:      server.Client(),
				Address:         server.URL,
				MetricsProvider: provider.NewDiscardProvider(),
			})

			require.Nil(err)

			if tc.ShouldMakeRequestFail {
				client.auth = acquirerFunc(failAcquirer)
			}

			if tc.ShouldDoRequestFail {
				client.storeBaseURL = failingURL
			}

			output, err := client.GetItems(bucket, "owner-name")

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
		SuccessResponseCode   int
		ShouldEraseID         bool
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
			Description:       "Bucket is required",
			Item:              validItem,
			ShouldEraseBucket: true,
			ExpectedErr:       ErrBucketEmpty,
		},
		{
			Description:   "Item ID Missing",
			Item:          validItem,
			ShouldEraseID: true,
			ExpectedErr:   ErrItemIDEmpty,
		},
		{
			Description: "Item ID Missing from payload",
			Item:        model.Item{Data: validItem.Data},
			ExpectedErr: ErrItemIDEmpty,
		},
		{
			Description: "Item ID Mismatch",
			Item:        model.Item{ID: "752f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f119", Data: validItem.Data},
			ExpectedErr: ErrItemIDMismatch,
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
			ExpectedErr:         ErrDoRequestFailure,
		},
		{
			Description:          "Non success code",
			Item:                 validItem,
			ShouldRespNonSuccess: true,
			ExpectedErr:          ErrPushItemFailure,
		},
		{
			Description:         "Create success",
			Item:                validItem,
			SuccessResponseCode: http.StatusCreated,
			ExpectedOutput:      CreatedPushResult,
		},
		{
			Description:         "Update success",
			Item:                validItem,
			SuccessResponseCode: http.StatusOK,
			ExpectedOutput:      UpdatedPushResult,
		},

		{
			Description:         "Update success with owner",
			Item:                validItem,
			SuccessResponseCode: http.StatusOK,
			Owner:               "owner-name",
			ExpectedOutput:      UpdatedPushResult,
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

				if tc.ShouldRespNonSuccess {
					rw.WriteHeader(http.StatusBadRequest)
				} else {
					payload, err := ioutil.ReadAll(r.Body)
					require.Nil(err)
					var item model.Item
					err = json.Unmarshal(payload, &item)
					require.Nil(err)

					assert.EqualValues(tc.Item, item)
					rw.WriteHeader(tc.SuccessResponseCode)
				}
			}))

			client, err := NewClient(ClientConfig{
				HTTPClient:      server.Client(),
				Address:         server.URL,
				MetricsProvider: provider.NewDiscardProvider(),
			})

			if tc.ShouldMakeRequestFail {
				client.auth = acquirerFunc(failAcquirer)
			}

			if tc.ShouldDoRequestFail {
				client.storeBaseURL = failingURL
			}

			if tc.ShouldEraseBucket {
				bucket = ""
			}

			if tc.ShouldEraseID {
				id = ""
			}

			require.Nil(err)
			output, err := client.PushItem(id, bucket, tc.Owner, tc.Item)

			if tc.ExpectedErr == nil {
				assert.EqualValues(tc.ExpectedOutput, output)
			} else {
				fmt.Println(err)
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
		})
	}
}

func TestRemoveItem(t *testing.T) {
	type testCase struct {
		Description           string
		Input                 *RemoveItemInput
		ExpectedOutput        *RemoveItemOutput
		ResponsePayload       []byte
		ShouldRespNonSuccess  bool
		ShouldMakeRequestFail bool
		ShouldDoRequestFail   bool
		ExpectedErr           error
	}

	validRemoveItemInput := &RemoveItemInput{
		ID:     "252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111",
		Bucket: "bucket-name",
		Owner:  "owner-name",
	}

	tcs := []testCase{
		{
			Description: "Input is nil",
			Input:       nil,
			ExpectedErr: ErrUndefinedInput,
		},
		{
			Description: "Bucket is required",
			Input:       &RemoveItemInput{Owner: "owner-name", ID: "252f10c83610ebca1a059c0bae8255eba2f95be4d1d7bcfa89d7248a82d9f111"},
			ExpectedErr: ErrBucketEmpty,
		},
		{
			Description: "Item ID Missing",
			Input:       &RemoveItemInput{Owner: "owner-name", Bucket: "bucket-name"},
			ExpectedErr: ErrItemIDEmpty,
		},
		{
			Description:           "Make request fails",
			Input:                 validRemoveItemInput,
			ShouldMakeRequestFail: true,
			ExpectedErr:           ErrAuthAcquirerFailure,
		},
		{
			Description:         "Do request fails",
			Input:               validRemoveItemInput,
			ShouldDoRequestFail: true,
			ExpectedErr:         ErrDoRequestFailure,
		},
		{
			Description:          "Non success code",
			Input:                validRemoveItemInput,
			ShouldRespNonSuccess: true,
			ExpectedErr:          ErrRemoveItemFailure,
		},
		{
			Description:     "Unmarshal failure",
			Input:           validRemoveItemInput,
			ResponsePayload: []byte("{{}"),
			ExpectedErr:     ErrJSONUnmarshal,
		},
		{
			Description:     "Succcess",
			Input:           validRemoveItemInput,
			ResponsePayload: getRemoveItemValidPayload(),
			ExpectedOutput:  getRemoveItemHappyOutput(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(fmt.Sprintf("%s/%s/%s", storeAPIPath, tc.Input.Bucket, tc.Input.ID), r.URL.Path)
				assert.Equal(http.MethodDelete, r.Method)
				if tc.ShouldRespNonSuccess {
					rw.WriteHeader(http.StatusBadRequest)
				} else {
					rw.Write(tc.ResponsePayload)
				}
			}))

			client, err := NewClient(ClientConfig{
				HTTPClient:      server.Client(),
				Address:         server.URL,
				MetricsProvider: provider.NewDiscardProvider(),
			})

			if tc.ShouldMakeRequestFail {
				client.auth = acquirerFunc(failAcquirer)
			}

			if tc.ShouldDoRequestFail {
				client.storeBaseURL = failingURL
			}

			require.Nil(err)
			output, err := client.RemoveItem(tc.Input)

			fmt.Println(err)
			assert.True(errors.Is(err, tc.ExpectedErr))
			if tc.ExpectedErr == nil {
				assert.EqualValues(tc.ExpectedOutput, output)
			}
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

func getPushItemInputBucketMissing(validPushItemInput *PushItemInput) *PushItemInput {
	return &PushItemInput{
		ID:    validPushItemInput.ID,
		Owner: validPushItemInput.Owner,
		Item: model.Item{
			ID:   validPushItemInput.Item.ID,
			Data: validPushItemInput.Item.Data,
		},
	}
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

func getRemoveItemHappyOutput() *RemoveItemOutput {
	return &RemoveItemOutput{
		Item: model.Item{
			ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
			Data: map[string]interface{}{
				"words": []interface{}{"Hello", "World"},
				"year":  float64(2021),
			},
			TTL: aws.Int64(100),
		},
	}
}

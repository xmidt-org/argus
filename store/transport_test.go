package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/model"
)

func TestDecodeGetItemRequest(t *testing.T) {
	testCases := []struct {
		Name                   string
		URLVars                map[string]string
		Headers                map[string][]string
		ExpectedDecodedRequest interface{}
		ExpectedErr            error
	}{
		{
			Name: "Missing id",
			URLVars: map[string]string{
				"bucket": "california",
			},
			ExpectedErr: &BadRequestErr{Message: idVarMissingMsg},
		},
		{
			Name: "Missing bucket",
			URLVars: map[string]string{
				"id": "san francisco",
			},
			ExpectedErr: &BadRequestErr{Message: bucketVarMissingMsg},
		},
		{
			Name: "Happy path - No owner",
			URLVars: map[string]string{
				"bucket": "california",
				"id":     "san francisco",
			},
			ExpectedDecodedRequest: &getItemRequest{
				key: model.Key{
					Bucket: "california",
					ID:     "san francisco",
				},
			},
		},
		{
			Name: "Happy path",
			URLVars: map[string]string{
				"bucket": "california",
				"id":     "san francisco",
			},

			ExpectedDecodedRequest: &getItemRequest{
				key: model.Key{
					Bucket: "california",
					ID:     "san francisco",
				},
				owner: "SF Giants",
			},
			Headers: map[string][]string{
				ItemOwnerHeaderKey: []string{"SF Giants"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			r := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
			transferHeaders(testCase.Headers, r)

			r = mux.SetURLVars(r, testCase.URLVars)
			decodedRequest, err := decodeGetItemRequest(context.Background(), r)

			assert.Equal(testCase.ExpectedDecodedRequest, decodedRequest)
			assert.Equal(testCase.ExpectedErr, err)
		})
	}
}

func TestEncodeGetItemResponse(t *testing.T) {
	testCases := []struct {
		Name            string
		ItemResponse    interface{}
		ExpectedHeaders http.Header
		ExpectedCode    int
		ExpectedBody    string
		ExpectedErr     error
	}{
		{
			Name:            "Unexpected casting error",
			ItemResponse:    nil,
			ExpectedHeaders: make(http.Header),
			ExpectedErr:     ErrCastingEncodeGetItemResponse,
			// used due to limitations in httptest. In reality, any non-nil error promises nothing
			// would be written to the response writer
			ExpectedCode: 200,
		},
		{
			Name: "Expired item",
			ItemResponse: &OwnableItem{
				Item: model.Item{
					TTL: 0,
				},
			},
			ExpectedCode:    http.StatusNotFound,
			ExpectedHeaders: make(http.Header),
		},
		{
			Name: "Happy path",
			ItemResponse: &OwnableItem{
				Owner: "xmidt",
				Item: model.Item{
					TTL:        20,
					Identifier: "id01",
					Data: map[string]interface{}{
						"key": 10,
					},
				},
			},
			ExpectedBody: `{"identifier":"id01","data":{"key":10},"ttl":20}`,
			ExpectedCode: 200,
			ExpectedHeaders: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			recorder := httptest.NewRecorder()
			err := encodeGetItemResponse(context.Background(), recorder, testCase.ItemResponse)
			assert.Equal(testCase.ExpectedErr, err)
			assert.Equal(testCase.ExpectedBody, recorder.Body.String())
			assert.Equal(testCase.ExpectedHeaders, recorder.HeaderMap)
			assert.Equal(testCase.ExpectedCode, recorder.Code)
		})
	}
}

func transferHeaders(headers map[string][]string, r *http.Request) {
	for k, values := range headers {
		for _, value := range values {
			r.Header.Add(k, value)
		}
	}
}

package store

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/model"
)

func TestGetOrDeleteItemRequestDecoder(t *testing.T) {
	testCases := []struct {
		Name                   string
		URLVars                map[string]string
		Headers                map[string][]string
		ExpectedDecodedRequest interface{}
		ExpectedErr            error
	}{
		{
			Name: "Happy path - No owner - Normal mode",
			URLVars: map[string]string{
				"bucket": "california",
				"uuid":   "san francisco",
			},
			ExpectedDecodedRequest: &getOrDeleteItemRequest{
				key: model.Key{
					Bucket: "california",
					UUID:   "san francisco",
				},
			},
		},
		{
			Name: "Happy path - Owner - Admin mode",
			URLVars: map[string]string{
				"bucket": "california",
				"uuid":   "san francisco",
			},

			Headers: map[string][]string{
				ItemOwnerHeaderKey:  []string{"SF Giants"},
				AdminTokenHeaderKey: []string{"secretAdminToken"},
			},

			ExpectedDecodedRequest: &getOrDeleteItemRequest{
				key: model.Key{
					Bucket: "california",
					UUID:   "san francisco",
				},
				owner:     "SF Giants",
				adminMode: true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			r := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
			transferHeaders(testCase.Headers, r)

			r = mux.SetURLVars(r, testCase.URLVars)
			config := &requestConfig{
				Authorization: authorizationConfig{
					AdminToken: "secretAdminToken",
				},
			}
			decoder := getOrDeleteItemRequestDecoder(config)
			decodedRequest, err := decoder(context.Background(), r)

			assert.Equal(testCase.ExpectedDecodedRequest, decodedRequest)
			assert.Equal(testCase.ExpectedErr, err)
		})
	}
}

func TestEncodeGetOrDeleteItemResponse(t *testing.T) {
	testCases := []struct {
		Name            string
		ItemResponse    interface{}
		ExpectedHeaders http.Header
		ExpectedCode    int
		ExpectedBody    string
		ExpectedErr     error
	}{
		{
			Name: "Happy path",
			ItemResponse: &OwnableItem{
				Owner: "xmidt",
				Item: model.Item{
					UUID:       "NaYFGE961cS_3dpzJcoP3QTL4kBYcw9ua3Q6Hy5E4nI",
					TTL:        aws.Int64(20),
					Identifier: "id01",
					Data: map[string]interface{}{
						"key": 10,
					},
				},
			},
			ExpectedBody: `{"uuid":"NaYFGE961cS_3dpzJcoP3QTL4kBYcw9ua3Q6Hy5E4nI","identifier":"id01","data":{"key":10},"ttl":20}`,
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
			err := encodeGetOrDeleteItemResponse(context.Background(), recorder, testCase.ItemResponse)
			assert.Equal(testCase.ExpectedErr, err)
			assert.Equal(testCase.ExpectedBody, recorder.Body.String())
			assert.Equal(testCase.ExpectedHeaders, recorder.HeaderMap)
			assert.Equal(testCase.ExpectedCode, recorder.Code)
		})
	}
}

func TestgetAllItemsRequestDecoder(t *testing.T) {
	testCases := []struct {
		Name                   string
		URLVars                map[string]string
		Headers                map[string][]string
		ExpectedDecodedRequest interface{}
		ExpectedErr            error
	}{
		{
			Name: "Happy path - No owner - Normal mode",
			URLVars: map[string]string{
				"bucket": "california",
			},
			ExpectedDecodedRequest: &getAllItemsRequest{
				bucket: "california",
			},
		},
		{
			Name: "Happy path - Owner - Admin mode",
			URLVars: map[string]string{
				"bucket": "california",
				"uuid":   "san francisco",
			},

			Headers: map[string][]string{
				ItemOwnerHeaderKey:  []string{"SF Giants"},
				AdminTokenHeaderKey: []string{"secretAdminToken"},
			},

			ExpectedDecodedRequest: &getAllItemsRequest{
				owner:     "SF Giants",
				adminMode: true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			r := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
			transferHeaders(testCase.Headers, r)

			r = mux.SetURLVars(r, testCase.URLVars)
			config := &requestConfig{
				Authorization: authorizationConfig{
					AdminToken: "secretAdminToken",
				},
			}
			decoder := getAllItemsRequestDecoder(config)
			decodedRequest, err := decoder(context.Background(), r)

			assert.Equal(testCase.ExpectedDecodedRequest, decodedRequest)
			assert.Equal(testCase.ExpectedErr, err)
		})
	}
}

func TestEncodeGetAllItemsResponse(t *testing.T) {
	assert := assert.New(t)
	response := map[string]OwnableItem{
		"E-VG": OwnableItem{
			Item: model.Item{
				UUID:       "E-VG",
				Identifier: "fix-you",
				Data:       map[string]interface{}{},
				TTL:        aws.Int64(1),
			},
		},
		"Y9G": OwnableItem{
			Item: model.Item{
				UUID:       "Y9G",
				Identifier: "this-is-it",
				Data:       map[string]interface{}{},
			},
		},
	}
	recorder := httptest.NewRecorder()
	expectedResponseBody := `[{"uuid":"E-VG","identifier":"fix-you","data":{},"ttl":1},{"uuid":"Y9G","identifier":"this-is-it","data":{}}]`
	err := encodeGetAllItemsResponse(context.Background(), recorder, response)
	assert.Nil(err)
	assert.Equal(expectedResponseBody, recorder.Body.String())
}

func transferHeaders(headers map[string][]string, r *http.Request) {
	for k, values := range headers {
		for _, value := range values {
			r.Header.Add(k, value)
		}
	}
}

func TestsetItemRequestDecoder(t *testing.T) {
	testCases := []struct {
		Name            string
		URLVars         map[string]string
		Headers         map[string][]string
		RequestBody     string
		ExpectedErr     error
		ExpectedRequest *setItemRequest
	}{
		{
			Name:        "Bad JSON data",
			URLVars:     map[string]string{bucketVarKey: "bucketVal", uuidVarKey: "rWPSg7pI0jj8mMG9tmscdQMOGKeRAquySfkObTasRBc"},
			RequestBody: `{"validJSON": false,}`,
			ExpectedErr: &BadRequestErr{
				Message: "failed to unmarshal json",
			},
		},
		{
			Name:        "Missing data item field",
			URLVars:     map[string]string{bucketVarKey: "letters", uuidVarKey: "ypeBEsobvcr6wjGzmiPcTaeG7_gUfE5yuYB3ha_uSLs"},
			RequestBody: `{"uuid": "ypeBEsobvcr6wjGzmiPcTaeG7_gUfE5yuYB3ha_uSLs","identifier": "a"}`,
			ExpectedErr: &BadRequestErr{
				Message: "data field must be set",
			},
		},

		{
			Name:        "Capped TTL",
			URLVars:     map[string]string{bucketVarKey: "variables", uuidVarKey: "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84"},
			Headers:     map[string][]string{ItemOwnerHeaderKey: []string{"math"}},
			RequestBody: `{"uuid":"evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84", "identifier": "xyz", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 3900}`,
			ExpectedRequest: &setItemRequest{
				item: OwnableItem{
					Item: model.Item{
						UUID:       "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84",
						Identifier: "xyz",
						Data: map[string]interface{}{
							"x": float64(0),
							"y": float64(1),
							"z": float64(2),
						},
						TTL: aws.Int64(int64((time.Minute * 5).Seconds())),
					},
					Owner: "math",
				},
				key: model.Key{
					Bucket: "variables",
					UUID:   "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84",
				},
			},
		},

		{
			Name:        "UUID mismatch TTL",
			URLVars:     map[string]string{bucketVarKey: "variables", uuidVarKey: "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84"},
			RequestBody: `{"uuid":"iBCtWB5Z8rw5KLJhcHpxMI9-E56wSCA2bcTVwY2YAiU", "identifier": "xyz", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 3900}`,
			ExpectedRequest: &setItemRequest{
				item: OwnableItem{
					Item: model.Item{
						Identifier: "xyz",
						Data: map[string]interface{}{
							"x": float64(0),
							"y": float64(1),
							"z": float64(2),
						},
						TTL: aws.Int64(60),
					},
				},
				key: model.Key{
					Bucket: "variables",
				},
			},
		},

		{
			Name:        "Happy Path - Admin mode",
			URLVars:     map[string]string{bucketVarKey: "variables", uuidVarKey: "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84"},
			Headers:     map[string][]string{ItemOwnerHeaderKey: []string{"math"}, AdminTokenHeaderKey: []string{"secretAdminPassKey"}},
			RequestBody: `{"uuid":"evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84", "identifier": "xyz", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 39}`,
			ExpectedRequest: &setItemRequest{
				item: OwnableItem{
					Item: model.Item{
						UUID:       "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84",
						Identifier: "xyz",
						Data: map[string]interface{}{
							"x": float64(0),
							"y": float64(1),
							"z": float64(2),
						},
						TTL: aws.Int64(39),
					},
					Owner: "math",
				},
				key: model.Key{
					Bucket: "variables",
					UUID:   "evCz5Hw1gg-r72nMVCOSvS0PbjfDSYUXKPDGgwE1Y84",
				},
				adminMode: true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			r := httptest.NewRequest(http.MethodGet, "http://localhost", bytes.NewBufferString(testCase.RequestBody))

			r = mux.SetURLVars(r, testCase.URLVars)
			transferHeaders(testCase.Headers, r)

			config := &requestConfig{
				Authorization: authorizationConfig{
					AdminToken: "secretAdminPassKey",
				},
				Validation: validationConfig{
					MaxTTL: time.Minute * 5,
				},
			}
			decoder := setItemRequestDecoder(config)
			decodedRequest, err := decoder(context.Background(), r)
			if testCase.ExpectedRequest == nil {
				assert.Nil(decodedRequest)
			} else {
				assert.Equal(testCase.ExpectedRequest, decodedRequest)
			}
			assert.Equal(testCase.ExpectedErr, err)
		})
	}
}

func TestEncodeSetItemResponse(t *testing.T) {
	assert := assert.New(t)
	createdRecorder := httptest.NewRecorder()
	err := encodeSetItemResponse(context.Background(), createdRecorder, &setItemResponse{
		existingResource: false,
	})
	assert.Nil(err)
	assert.Equal(http.StatusCreated, createdRecorder.Code)

	updatedRecorder := httptest.NewRecorder()
	err = encodeSetItemResponse(context.Background(), updatedRecorder, &setItemResponse{
		existingResource: true,
	})
	assert.Nil(err)
	assert.Equal(http.StatusOK, updatedRecorder.Code)
}

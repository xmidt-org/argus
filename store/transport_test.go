package store

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/bascule"
)

func TestGetOrDeleteItemRequestDecoder(t *testing.T) {
	sfID := Sha256HexDigest("san francisco")
	testCases := []struct {
		Name                   string
		URLVars                map[string]string
		Owner                  string
		ExpectedDecodedRequest interface{}
		ExpectedErr            error
		ElevatedAccess         bool
	}{
		{
			Name: "Invalid ID",
			URLVars: map[string]string{
				"bucket": "california",
				"id":     "badIDabcdef",
			},
			ExpectedErr: errInvalidID,
		},

		{
			Name: "Invalid Bucket",
			URLVars: map[string]string{
				"bucket": "california?",
				"id":     sfID,
			},
			ExpectedErr: errInvalidBucket,
		},
		{
			Name: "Happy path. No owner. Normal mode",
			URLVars: map[string]string{
				"bucket": "california",
				"id":     sfID,
			},
			ExpectedDecodedRequest: &getOrDeleteItemRequest{
				key: model.Key{
					Bucket: "california",
					ID:     sfID,
				},
			},
		},
		{
			Name: "Happy path. Owner. Admin mode",
			URLVars: map[string]string{
				"bucket": "california",
				"id":     sfID,
			},

			Owner:          "SF Giants",
			ElevatedAccess: true,

			ExpectedDecodedRequest: &getOrDeleteItemRequest{
				key: model.Key{
					Bucket: "california",
					ID:     sfID,
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
			r = mux.SetURLVars(r, testCase.URLVars)

			if len(testCase.Owner) > 0 {
				r.Header.Set(ItemOwnerHeaderKey, testCase.Owner)
			}

			config := &transportConfig{
				AccessLevelAttributeKey: auth.DefaultAccessLevelAttributeKey,
				ItemMaxTTL:              time.Hour * 24,
			}

			ctx := context.Background()

			if testCase.ElevatedAccess {
				ctx = withElevatedAccess(ctx)
			}

			decoder := getOrDeleteItemRequestDecoder(config)
			decodedRequest, err := decoder(ctx, r)

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
					ID:  "NaYFGE961cS_3dpzJcoP3QTL4kBYcw9ua3Q6Hy5E4nI",
					TTL: aws.Int64(20),
					Data: map[string]interface{}{
						"key": 10,
					},
				},
			},
			ExpectedBody: `{"id":"NaYFGE961cS_3dpzJcoP3QTL4kBYcw9ua3Q6Hy5E4nI","data":{"key":10},"ttl":20}`,
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

func TestGetAllItemsRequestDecoder(t *testing.T) {
	testCases := []struct {
		Name                   string
		URLVars                map[string]string
		Owner                  string
		ElevatedAccess         bool
		ExpectedDecodedRequest interface{}
		ExpectedErr            error
	}{
		{
			Name: "Invalid bucket",
			URLVars: map[string]string{
				"bucket": "cal!fornia",
			},
			ExpectedErr: errInvalidBucket,
		},
		{
			Name: "Happy path. No owner. Normal mode",
			URLVars: map[string]string{
				"bucket": "california",
			},
			ExpectedDecodedRequest: &getAllItemsRequest{
				bucket: "california",
			},
		},
		{
			Name: "Happy path. Owner. Admin mode",
			URLVars: map[string]string{
				"bucket": "california",
				"ID":     Sha256HexDigest("san francisco"),
			},
			Owner: "SF Giants",
			ExpectedDecodedRequest: &getAllItemsRequest{
				bucket:    "california",
				owner:     "SF Giants",
				adminMode: true,
			},
			ElevatedAccess: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			r := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
			r = mux.SetURLVars(r, testCase.URLVars)
			if len(testCase.Owner) > 0 {
				r.Header.Set(ItemOwnerHeaderKey, testCase.Owner)
			}

			config := &transportConfig{
				AccessLevelAttributeKey: auth.DefaultAccessLevelAttributeKey,
				ItemMaxTTL:              time.Hour * 24,
			}

			ctx := context.Background()
			if testCase.ElevatedAccess {
				ctx = withElevatedAccess(ctx)
			}
			decoder := getAllItemsRequestDecoder(config)
			decodedRequest, err := decoder(ctx, r)

			assert.Equal(testCase.ExpectedDecodedRequest, decodedRequest)
			assert.Equal(testCase.ExpectedErr, err)
		})
	}
}

func TestEncodeGetAllItemsResponse(t *testing.T) {
	assert := assert.New(t)
	evgItemID := Sha256HexDigest("E-VG")
	y9gItemID := Sha256HexDigest("Y9G")
	response := map[string]OwnableItem{
		"E-VG": {
			Item: model.Item{
				ID:   evgItemID,
				Data: map[string]interface{}{},
				TTL:  aws.Int64(1),
			},
		},
		"Y9G": {
			Item: model.Item{
				ID:   y9gItemID,
				Data: map[string]interface{}{},
			},
		},
	}
	recorder := httptest.NewRecorder()
	expectedResponseBody := fmt.Sprintf("[{\"id\":\"%s\",\"data\":{}},{\"id\":\"%s\",\"data\":{},\"ttl\":1}]", y9gItemID, evgItemID)
	err := encodeGetAllItemsResponse(context.Background(), recorder, response)
	assert.Nil(err)
	assert.JSONEq(expectedResponseBody, recorder.Body.String())
}

func TestSetItemRequestDecoder(t *testing.T) {
	testCases := []struct {
		Name            string
		URLVars         map[string]string
		Owner           string
		ElevatedAccess  bool
		RequestBody     string
		ExpectedErr     error
		ExpectedRequest *setItemRequest
	}{
		{
			Name:        "Bad JSON data",
			URLVars:     map[string]string{bucketVarKey: "bucket-val", idVarKey: "7731f5f6fc9456d9ca274416ad66030777778026716e821f1de966bf54ab9e2e"},
			RequestBody: `{"validJSON": false,}`,
			ExpectedErr: errPayloadUnmarshalFailure,
		},
		{
			Name:        "Missing data item field",
			URLVars:     map[string]string{bucketVarKey: "letters", idVarKey: "d228667158e251494aa05b9183a5d01c0620aad791860163c7d553ce64b35fcf"},
			RequestBody: `{"id": "d228667158e251494aa05b9183a5d01c0620aad791860163c7d553ce64b35fcf"}`,
			ExpectedErr: errDataFieldMissing,
		},
		{
			Name:        "Capped TTL",
			URLVars:     map[string]string{bucketVarKey: "variables", idVarKey: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b"},
			Owner:       "math",
			RequestBody: `{"id":"4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 3900}`,
			ExpectedRequest: &setItemRequest{
				item: OwnableItem{
					Item: model.Item{
						ID: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b",
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
					ID:     "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b",
				},
			},
		},
		{
			Name:        "Invalid ID",
			URLVars:     map[string]string{bucketVarKey: "variables", idVarKey: "badID"},
			ExpectedErr: errInvalidID,
		},
		{
			Name:        "Invalid Bucket",
			URLVars:     map[string]string{bucketVarKey: "when-validation-gives-you-lemons!", idVarKey: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b"},
			ExpectedErr: errInvalidBucket,
		},
		{
			Name:        "ID mismatch",
			URLVars:     map[string]string{bucketVarKey: "variables", idVarKey: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b"},
			RequestBody: `{"id":"4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74oops", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 3900}`,
			ExpectedErr: errIDMismatch,
		},
		{
			Name:           "Happy Path. Admin mode",
			URLVars:        map[string]string{bucketVarKey: "variables", idVarKey: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b"},
			Owner:          "math",
			ElevatedAccess: true,
			RequestBody:    `{"id":"4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 39}`,
			ExpectedRequest: &setItemRequest{
				item: OwnableItem{
					Item: model.Item{
						ID: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b",
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
					ID:     "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b",
				},
				adminMode: true,
			},
		},
		{
			Name:        "Alternative ID format",
			URLVars:     map[string]string{bucketVarKey: "variables", idVarKey: "4B13653E5D6D611DE5999AB0E7C0AA67E1D83D4CBA8349A04DA0A431FB27F74B"},
			Owner:       "math",
			RequestBody: `{"id":"4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b", "data": {"x": 0, "y": 1, "z": 2}, "ttl": 39}`,
			ExpectedRequest: &setItemRequest{
				item: OwnableItem{
					Item: model.Item{
						ID: "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b",
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
					ID:     "4b13653e5d6d611de5999ab0e7c0aa67e1d83d4cba8349a04da0a431fb27f74b",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			r := httptest.NewRequest(http.MethodGet, "http://localhost", bytes.NewBufferString(testCase.RequestBody))
			r = mux.SetURLVars(r, testCase.URLVars)
			if len(testCase.Owner) > 0 {
				r.Header.Set(ItemOwnerHeaderKey, testCase.Owner)
			}

			config := &transportConfig{
				AccessLevelAttributeKey: auth.DefaultAccessLevelAttributeKey,
				ItemMaxTTL:              time.Minute * 5,
			}
			decoder := setItemRequestDecoder(config)

			ctx := context.Background()
			if testCase.ElevatedAccess {
				ctx = withElevatedAccess(ctx)
			}

			decodedRequest, err := decoder(ctx, r)
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

func TestHasElevatedAccess(t *testing.T) {
	type testCase struct {
		Description           string
		IncludeBasculeAuth    bool
		IncludeAttributeValue bool
		AttributeValue        interface{}
		Expected              bool
	}

	tcs := []testCase{
		{
			Description:        "BasculeAuthMissing",
			IncludeBasculeAuth: false,
		},
		{
			Description:           "AttributeMissing",
			IncludeBasculeAuth:    true,
			IncludeAttributeValue: false,
		},
		{
			Description:           "WrongAttributeType",
			IncludeBasculeAuth:    true,
			IncludeAttributeValue: true,
			AttributeValue:        "1",
		},
		{
			Description:           "StandardAccess",
			IncludeBasculeAuth:    true,
			IncludeAttributeValue: true,
			AttributeValue:        auth.DefaultAccessLevelAttributeValue,
		},
		{
			Description:           "ElevatedAccess",
			IncludeBasculeAuth:    true,
			IncludeAttributeValue: true,
			AttributeValue:        auth.ElevatedAccessLevelAttributeValue,
			Expected:              true,
		},
	}

	for _, tc := range tcs {

		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			attributeKey := "attrKey"
			ctx := context.Background()
			attributesMap := make(map[string]interface{})

			if tc.IncludeAttributeValue {
				attributesMap[attributeKey] = tc.AttributeValue
			}

			attributes := bascule.NewAttributes(attributesMap)

			auth := bascule.Authentication{
				Token: bascule.NewToken("Bearer", "testUser", attributes),
			}

			if tc.IncludeBasculeAuth {
				ctx = bascule.WithAuthentication(ctx, auth)
			}

			assert.Equal(tc.Expected, hasElevatedAccess(ctx, attributeKey))

		})
	}
}

func withElevatedAccess(ctx context.Context) context.Context {
	attributes := bascule.NewAttributes(map[string]interface{}{
		auth.DefaultAccessLevelAttributeKey: auth.ElevatedAccessLevelAttributeValue,
	})
	basculeAuth := bascule.Authentication{
		Authorization: bascule.Authorization("Bearer"),
		Token:         bascule.NewToken("Bearer", "testUser", attributes),
	}
	return bascule.WithAuthentication(ctx, basculeAuth)
}

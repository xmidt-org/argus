package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/bascule"
)

// request URL path keys.
const (
	bucketVarKey = "bucket"
	idVarKey     = "id"
)

// Request and Response Headers.
const (
	ItemOwnerHeaderKey  = "X-Midt-Owner"
	AdminTokenHeaderKey = "X-Midt-Admin-Token"
	XmidtErrorHeaderKey = "X-Midt-Error"
)

// ElevatedAccessLevel is the bascule attribute value found in requests that should be granted
// priviledged access to operations.
const ElevatedAccessLevel = 1

// ErrCasting indicates there was (most likely) a middleware wiring mistake with
// the go-kit style encoders/decoders.
var ErrCasting = errors.New("casting error due to middleware wiring mistake")

var (
	errBodyReadFailure         = BadRequestErr{Message: "Failed to read body."}
	errPayloadUnmarshalFailure = BadRequestErr{Message: "Failed to unmarshal json payload."}
)

type transportConfig struct {
	AccessLevelAttributeKey string
	ItemMaxTTL              time.Duration
}
type getOrDeleteItemRequest struct {
	key       model.Key
	owner     string
	adminMode bool
}

type getAllItemsRequest struct {
	bucket    string
	owner     string
	adminMode bool
}

type setItemRequest struct {
	key       model.Key
	item      OwnableItem
	adminMode bool
}

type setItemResponse struct {
	existingResource bool
}

func getAllItemsRequestDecoder(config *transportConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		bucket := mux.Vars(r)[bucketVarKey]
		if !isBucketValid(bucket) {
			return nil, errInvalidBucket
		}
		return &getAllItemsRequest{
			bucket:    bucket,
			owner:     r.Header.Get(ItemOwnerHeaderKey),
			adminMode: hasElevatedAccess(ctx, config.AccessLevelAttributeKey),
		}, nil
	}
}

func setItemRequestDecoder(config *transportConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		var (
			URLVars = mux.Vars(r)
			id      = strings.ToLower(URLVars[idVarKey])
			bucket  = URLVars[bucketVarKey]
		)

		if err := validateItemPathVars(bucket, id); err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, errBodyReadFailure
		}

		unmarshaler := validItemUnmarshaler{config: config, id: id}

		if err := json.Unmarshal(data, &unmarshaler); err != nil {
			if _, ok := err.(BadRequestErr); !ok {
				err = errPayloadUnmarshalFailure
			}
			return nil, err
		}

		return &setItemRequest{
			item: OwnableItem{
				Item:  unmarshaler.item,
				Owner: r.Header.Get(ItemOwnerHeaderKey),
			},
			key: model.Key{
				Bucket: bucket,
				ID:     id,
			},
			adminMode: hasElevatedAccess(ctx, config.AccessLevelAttributeKey),
		}, nil
	}
}

func getOrDeleteItemRequestDecoder(config *transportConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		var (
			URLVars = mux.Vars(r)
			id      = strings.ToLower(URLVars[idVarKey])
			bucket  = URLVars[bucketVarKey]
		)

		if err := validateItemPathVars(bucket, id); err != nil {
			return nil, err
		}

		return &getOrDeleteItemRequest{
			key: model.Key{
				Bucket: bucket,
				ID:     id,
			},
			adminMode: hasElevatedAccess(ctx, config.AccessLevelAttributeKey),
			owner:     r.Header.Get(ItemOwnerHeaderKey),
		}, nil
	}
}

func encodeSetItemResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	r := response.(*setItemResponse)
	if r.existingResource {
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusCreated)
	}
	return nil
}

// TODO: I noticed order of result elements get shuffled around on multiple fetches
// This is because of dynamodb. To make tests easier, results are sorted by lexicographical non-decreasing
// order of the ids.
func encodeGetAllItemsResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	items := response.(map[string]OwnableItem)
	list := []model.Item{}
	for _, value := range items {
		list = append(list, value.Item)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})

	data, err := json.Marshal(&list)
	if err != nil {
		return err
	}

	rw.Header().Add("Content-Type", "application/json")
	rw.Write(data)
	return nil
}

func encodeGetOrDeleteItemResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	item := response.(*OwnableItem)

	data, err := json.Marshal(&item.Item)
	if err != nil {
		return err
	}

	rw.Header().Add("Content-Type", "application/json")
	rw.Write(data)
	return nil
}

func encodeError(ctx context.Context, err error, w http.ResponseWriter) {
	w.Header().Set(XmidtErrorHeaderKey, err.Error())
	if headerer, ok := err.(kithttp.Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}
	code := http.StatusInternalServerError
	if sc, ok := err.(kithttp.StatusCoder); ok {
		code = sc.StatusCode()
	}
	w.WriteHeader(code)
}

// Sha256HexDigest returns the SHA-256 hex digest of the given input.
func Sha256HexDigest(message string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(message)))
}

func hasElevatedAccess(ctx context.Context, accessLevelAttributeKey string) bool {
	basculeAuth, ok := bascule.FromContext(ctx)
	if !ok {
		return false
	}
	attributes := basculeAuth.Token.Attributes()
	attribute, ok := attributes.Get(accessLevelAttributeKey)
	if !ok {
		return false
	}
	accessLevel, ok := attribute.(int)
	if !ok {
		return false
	}

	return accessLevel == auth.ElevatedAccessLevelAttributeValue
}

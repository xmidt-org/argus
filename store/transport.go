package store

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/argus/model"
)

// request URL path keys
const (
	bucketVarKey = "bucket"
	uuidVarKey   = "uuid"
)

const (
	bucketVarMissingMsg = "{bucket} URL path parameter missing"
	uuidVarMissingMsg   = "{uuid} URL path parameter missing"
)

// Request and Response Headers
const (
	ItemOwnerHeaderKey  = "X-Midt-Owner"
	AdminTokenHeaderKey = "X-Midt-Admin-Token"
	XmidtErrorHeaderKey = "X-Midt-Error"
)

// ErrCasting indicates there was a middleware wiring mistake with the go-kit style
// encoders.
var ErrCasting = errors.New("casting error due to middleware wiring mistake")

type requestConfig struct {
	Validation struct {
		MaxTTL time.Duration
	}

	Authorization struct {
		Admintoken string
	}
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
	key              model.Key
	existingResource bool
}

func getAllItemsRequestDecoder(config *requestConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		vars := mux.Vars(r)
		bucket, ok := vars[bucketVarKey]
		if !ok {
			return nil, &BadRequestErr{Message: bucketVarMissingMsg}
		}
		return &getAllItemsRequest{
			bucket:    bucket,
			owner:     r.Header.Get(ItemOwnerHeaderKey),
			adminMode: config.Authorization.Admintoken == r.Header.Get(AdminTokenHeaderKey),
		}, nil
	}
}

func setItemRequestDecoder(config *requestConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		vars := mux.Vars(r)
		bucket, ok := vars[bucketVarKey]
		if !ok {
			return nil, &BadRequestErr{Message: bucketVarMissingMsg}
		}

		uuid, ok := vars[uuidVarKey]
		if !ok {
			return nil, &BadRequestErr{Message: uuidVarMissingMsg}
		}

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, &BadRequestErr{Message: "failed to read body"}
		}

		item := model.Item{}
		err = json.Unmarshal(data, &item)
		if err != nil {
			return nil, &BadRequestErr{Message: "failed to unmarshal json"}
		}

		if len(item.Data) <= 0 {
			return nil, &BadRequestErr{Message: "data field must be set"}
		}

		validateItemTTL(&item, config.Validation.MaxTTL)

		if item.UUID != uuid {
			return nil, &BadRequestErr{Message: "UUIDs must match between URL and payload"}
		}

		return &setItemRequest{
			item: OwnableItem{
				Item:  item,
				Owner: r.Header.Get(ItemOwnerHeaderKey),
			},
			key: model.Key{
				Bucket: bucket,
				UUID:   uuid,
			},
			adminMode: config.Authorization.Admintoken == r.Header.Get(AdminTokenHeaderKey),
		}, nil
	}
}

func getOrDeleteItemRequestDecoder(config *requestConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		vars := mux.Vars(r)
		bucket, ok := vars[bucketVarKey]
		if !ok {
			return nil, &BadRequestErr{Message: bucketVarMissingMsg}
		}

		uuid, ok := vars[uuidVarKey]

		if !ok {
			return nil, &BadRequestErr{Message: uuidVarMissingMsg}
		}

		return &getOrDeleteItemRequest{
			key: model.Key{
				Bucket: bucket,
				UUID:   uuid,
			},
			adminMode: config.Authorization.Admintoken == r.Header.Get(AdminTokenHeaderKey),
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

func encodeGetAllItemsResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	items := response.(map[string]OwnableItem)
	var list []model.Item
	for _, value := range items {
		list = append(list, value.Item)
	}
	data, err := json.Marshal(&list)
	if err != nil {
		return err
	}
	rw.Header().Add("Content-Type", "application/json")
	rw.Write(data)
	return nil
}

func encodeGetOrDeleteItemResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	item, ok := response.(*OwnableItem)
	if !ok {
		return ErrCasting
	}

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

func validateItemTTL(item *model.Item, maxTTL time.Duration) {
	if item.TTL != nil {
		ttlCapSeconds := int64(maxTTL.Seconds())
		if *item.TTL > ttlCapSeconds {
			item.TTL = &ttlCapSeconds
		}
	}
}

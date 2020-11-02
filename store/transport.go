package store

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

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
	XmidtErrorHeaderKey = "X-Midt-Error"
)

// ErrCasting indicates there was a middleware wiring mistake with the go-kit style
// encoders.
var ErrCasting = errors.New("casting error due to middleware wiring mistake")

type getOrDeleteItemRequest struct {
	key   model.Key
	owner string
}

type getAllItemsRequest struct {
	bucket string
	owner  string
}

type setItemRequest struct {
	key  model.Key
	item OwnableItem
}

type setItemResponse struct {
	key              model.Key
	existingResource bool
}

func decodeGetAllItemsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	bucket, ok := vars[bucketVarKey]
	if !ok {
		return nil, &BadRequestErr{Message: bucketVarMissingMsg}
	}
	return &getAllItemsRequest{
		bucket: bucket,
		owner:  r.Header.Get(ItemOwnerHeaderKey),
	}, nil
}

func setItemRequestDecoder(itemTTLInfo ItemTTL) kithttp.DecodeRequestFunc {
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

		validateItemTTL(&item, itemTTLInfo)

		return &setItemRequest{
			item: OwnableItem{
				Item:  item,
				Owner: r.Header.Get(ItemOwnerHeaderKey),
			},
			key: model.Key{
				Bucket: bucket,
				UUID:   uuid,
			},
		}, nil
	}
}

func validateItemTTL(item *model.Item, itemTTLInfo ItemTTL) {
	if item.TTL != nil {
		ttlCapSeconds := int64(itemTTLInfo.MaxTTL.Seconds())
		if *item.TTL > ttlCapSeconds {
			item.TTL = &ttlCapSeconds
		}
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
	payload := map[string]model.Item{}
	for k, value := range items {
		payload[k] = value.Item
	}
	data, err := json.Marshal(&payload)
	if err != nil {
		return err
	}
	rw.Header().Add("Content-Type", "application/json")
	rw.Write(data)
	return nil
}

func decodeGetOrDeleteItemRequest(ctx context.Context, r *http.Request) (interface{}, error) {
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
		owner: r.Header.Get(ItemOwnerHeaderKey),
	}, nil
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

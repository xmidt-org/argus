package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/argus/model"
)

// request URL path keys
const (
	bucketVarKey = "bucket"
	idVarKey     = "id"
)

const (
	bucketVarMissingMsg = "{bucket} URL path parameter missing"
	idVarMissingMsg     = "{id} URL path parameter missing"
)

// Request and Response Headers
const (
	ItemOwnerHeaderKey  = "X-Xmidt-Owner"
	XmidtErrorHeaderKey = "X-Xmidt-Error"
)

// ErrCasting indicates there was a middleware wiring mistake with the go-kit style
// encoders.
var ErrCasting = errors.New("casting error due to middleware wiring mistake")

type getOrDeleteItemRequest struct {
	key   model.Key
	owner string
}

func decodeGetOrDeleteItemRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	bucket, ok := vars[bucketVarKey]
	if !ok {
		return nil, &BadRequestErr{Message: bucketVarMissingMsg}
	}

	id, ok := vars[idVarKey]

	if !ok {
		return nil, &BadRequestErr{Message: idVarMissingMsg}
	}

	return &getOrDeleteItemRequest{
		key: model.Key{
			Bucket: bucket,
			ID:     id,
		},
		owner: r.Header.Get(ItemOwnerHeaderKey),
	}, nil
}

func encodeGetOrDeleteItemResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	item, ok := response.(*OwnableItem)
	if !ok {
		return ErrCasting
	}

	if item.TTL <= 0 {
		rw.WriteHeader(http.StatusNotFound)
		return nil
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

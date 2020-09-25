package store

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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

type getAllItemsRequest struct {
	bucket string
	owner  string
}

type pushItemRequest struct {
	item   OwnableItem
	bucket string
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

func pushItemRequestDecoder(itemTTLInfo ItemTTL) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		vars := mux.Vars(r)
		bucket, ok := vars[bucketVarKey]
		if !ok {
			return nil, &BadRequestErr{Message: bucketVarMissingMsg}
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

		item.Identifier = transformPushItemID(item.Identifier)
		validateItemTTL(&item, itemTTLInfo)

		return &pushItemRequest{
			item: OwnableItem{
				Item:  item,
				Owner: r.Header.Get(ItemOwnerHeaderKey),
			},
			bucket: bucket,
		}, nil
	}
}

func validateItemTTL(item *model.Item, itemTTLInfo ItemTTL) {
	if item.TTL > int64(itemTTLInfo.MaxTTL.Seconds()) {
		item.TTL = int64(itemTTLInfo.MaxTTL.Seconds())
	}

	if item.TTL < 1 {
		item.TTL = int64(itemTTLInfo.DefaultTTL.Seconds())
	}
}

func transformPushItemID(ID string) string {
	var checkSum [32]byte = sha256.Sum256([]byte(ID))
	return base64.RawURLEncoding.EncodeToString(checkSum[:])
}

func encodePushItemResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	pushItemResponse := response.(*model.Key)
	data, err := json.Marshal(&pushItemResponse)
	if err != nil {
		return err
	}
	rw.Header().Add("Content-Type", "application/json")
	rw.Write(data)
	return nil
}
func encodeGetAllItemsResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	items := response.(map[string]OwnableItem)
	payload := map[string]model.Item{}
	for k, value := range items {
		if value.TTL <= 0 {
			continue
		}
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

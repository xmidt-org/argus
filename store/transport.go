package store

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/bascule"
)

// SuperUserAccessLevel is the bascule attribute value to check that a request
// should be allowed admin permissions.
const SuperUserAccessLevel = 1

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
	ItemOwnerHeaderKey  = "X-Midt-Owner"
	AdminTokenHeaderKey = "X-Midt-Admin-Token"
	XmidtErrorHeaderKey = "X-Midt-Error"
)

// ErrCasting indicates there was a middleware wiring mistake with the go-kit style
// encoders.
var ErrCasting = errors.New("casting error due to middleware wiring mistake")

type requestConfig struct {
	Validation    validationConfig
	Authorization authorizationConfig
}

type validationConfig struct {
	MaxTTL time.Duration
}

type authorizationConfig struct {
	AdminToken string
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
		return &getAllItemsRequest{
			bucket:    mux.Vars(r)[bucketVarKey],
			owner:     r.Header.Get(ItemOwnerHeaderKey),
			adminMode: isSuperUser(ctx),
		}, nil
	}
}

func setItemRequestDecoder(config *requestConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		URLVars := mux.Vars(r)
		bucket := URLVars[bucketVarKey]
		id := URLVars[idVarKey]

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

		if item.ID != id {
			return nil, &BadRequestErr{Message: "ids must match between URL and payload"}
		}

		return &setItemRequest{
			item: OwnableItem{
				Item:  item,
				Owner: r.Header.Get(ItemOwnerHeaderKey),
			},
			key: model.Key{
				Bucket: bucket,
				ID:     id,
			},
			adminMode: isSuperUser(ctx),
		}, nil
	}
}

func getOrDeleteItemRequestDecoder(config *requestConfig) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		URLVars := mux.Vars(r)

		return &getOrDeleteItemRequest{
			key: model.Key{
				Bucket: URLVars[bucketVarKey],
				ID:     URLVars[idVarKey],
			},
			adminMode: isSuperUser(ctx),
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

func validateItemTTL(item *model.Item, maxTTL time.Duration) {
	if item.TTL != nil {
		ttlCapSeconds := int64(maxTTL.Seconds())
		if *item.TTL > ttlCapSeconds {
			item.TTL = &ttlCapSeconds
		}
	}
}

func isSuperUser(ctx context.Context) bool {
	auth, ok := bascule.FromContext(ctx)
	if !ok {
		return false
	}
	attributes := auth.Token.Attributes()
	attribute, ok := attributes.Get("access-level")
	if !ok {
		return false
	}
	accessLevel, ok := attribute.(int)
	if !ok {
		return false
	}

	return accessLevel == SuperUserAccessLevel
}

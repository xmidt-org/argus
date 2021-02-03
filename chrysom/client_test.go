package chrysom

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/store"
)

func TestInterface(t *testing.T) {
	assert := assert.New(t)
	var (
		client interface{}
	)
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
		Bucket:          "testing",
		PullInterval:    time.Second * 5,
		Logger:          log.NewNopLogger(),
		Address:         "http://awesome-argus-hostname.io",
		MetricsProvider: provider.NewDiscardProvider(),
	}

	myAmazingClient := &http.Client{Timeout: time.Hour}
	allDefinedCaseConfig := &ClientConfig{
		HTTPClient:      myAmazingClient,
		Bucket:          "argus-staging",
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
				Bucket:          "argus-staging",
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

func TestDo(t *testing.T) {
	type testCase struct {
		Description      string
		ClientDoFails    bool
		ExpectedResponse *doResponse
		ExpectedErr      error
	}

	tcs := []testCase{
		{
			Description:   "Client Do fails",
			ClientDoFails: true,
			ExpectedErr:   ErrDoRequestFailure,
		},
		{
			Description: "Success",
			ExpectedResponse: &doResponse{
				code: 200,
				body: []byte("testing"),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)
			)

			echoHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusOK)
				bodyBytes, err := ioutil.ReadAll(r.Body)
				require.Nil(err)
				rw.Write(bodyBytes)
			})

			server := httptest.NewServer(echoHandler)
			defer server.Close()
			client, err := NewClient(&ClientConfig{
				HTTPClient:      server.Client(),
				MetricsProvider: provider.NewDiscardProvider(),
				Address:         server.URL,
			})
			require.Nil(err)

			var URL = server.URL

			if tc.ClientDoFails {
				URL = "http://should-definitely-fail.net"
			}

			request, err := http.NewRequest(http.MethodPut, URL, bytes.NewBufferString("testing"))
			require.Nil(err)

			resp, err := client.do(request)

			if tc.ExpectedErr == nil {
				assert.Equal(http.StatusOK, resp.code)
				assert.Equal(tc.ExpectedResponse, resp)
			} else {
				assert.True(errors.Is(err, tc.ExpectedErr))
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

func TestMakeRequest(t *testing.T) {
	type testCase struct {
		Description   string
		Owner         string
		Method        string
		URL           string
		Body          []byte
		AcquirerFails bool
		ExpectedErr   error
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
			Description: "Happy path",
			Method:      http.MethodPut,
			URL:         "http://argus-hostname.io",
			Body:        []byte("testing"),
		},

		{
			Description: "Happy path with owner",
			Method:      http.MethodPut,
			URL:         "http://argus-hostname.io",
			Owner:       "xmidt",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)

			client, err := NewClient(&ClientConfig{
				Address:         "http://argus-hostname.io",
				MetricsProvider: provider.NewDiscardProvider(),
			})

			if tc.AcquirerFails {
				client.auth = acquirerFunc(failAcquirer)
			}

			assert.Nil(err)
			r, err := client.makeRequest(tc.Owner, tc.Method, tc.URL, bytes.NewBuffer(tc.Body))

			if tc.ExpectedErr == nil {
				assert.Nil(err)
				assert.Equal(tc.URL, r.URL.String())
				assert.Equal(tc.Method, r.Method)
				assert.EqualValues(len(tc.Body), r.ContentLength)
				assert.Equal(tc.Owner, r.Header.Get(store.ItemOwnerHeaderKey))
			} else {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
		})
	}
}

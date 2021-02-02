package chrysom

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Address:         server.URL,
		MetricsProvider: provider.NewDiscardProvider(),
	})
	assert.Nil(err)

	var inputBody = []byte("this is a test")
	request, err := http.NewRequest(http.MethodPut, server.URL, bytes.NewBuffer(inputBody))
	require.Nil(err)

	resp, err := client.do(request)
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.code)
	assert.Equal(inputBody, resp.body)

}

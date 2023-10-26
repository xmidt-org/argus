// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
)

var (
	mockListener = ListenerFunc((func(_ Items) {
		fmt.Println("Doing amazing work for 100ms")
		time.Sleep(time.Millisecond * 100)
	}))
	mockMeasures = &Measures{
		Polls: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "testPollsCounter",
				Help: "testPollsCounter",
			},
			[]string{OutcomeLabel},
		)}
	happyListenerClientConfig = ListenerClientConfig{
		Listener:     mockListener,
		PullInterval: time.Second,
		Logger:       sallust.Default(),
	}
)

func TestListenerStartStopPairsParallel(t *testing.T) {
	require := require.New(t)
	client, close, err := newStartStopClient(true)
	assert.Nil(t, err)
	defer close()

	t.Run("ParallelGroup", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			testNumber := i
			t.Run(strconv.Itoa(testNumber), func(t *testing.T) {
				t.Parallel()
				assert := assert.New(t)
				fmt.Printf("%d: Start\n", testNumber)
				errStart := client.Start(context.Background())
				if errStart != nil {
					assert.Equal(ErrListenerNotStopped, errStart)
				}
				time.Sleep(time.Millisecond * 400)
				errStop := client.Stop(context.Background())
				if errStop != nil {
					assert.Equal(ErrListenerNotRunning, errStop)
				}
				fmt.Printf("%d: Done\n", testNumber)
			})
		}
	})

	require.Equal(stopped, client.observer.state)
}

func TestListenerStartStopPairsSerial(t *testing.T) {
	require := require.New(t)
	client, close, err := newStartStopClient(true)
	assert.Nil(t, err)
	defer close()

	for i := 0; i < 5; i++ {
		testNumber := i
		t.Run(strconv.Itoa(testNumber), func(t *testing.T) {
			assert := assert.New(t)
			fmt.Printf("%d: Start\n", testNumber)
			assert.Nil(client.Start(context.Background()))
			assert.Nil(client.Stop(context.Background()))
			fmt.Printf("%d: Done\n", testNumber)
		})
	}
	require.Equal(stopped, client.observer.state)
}

func TestListenerEdgeCases(t *testing.T) {
	t.Run("NoListener", func(t *testing.T) {
		_, _, err := newStartStopClient(false)
		assert.Equal(t, ErrNoListenerProvided, err)
	})

	t.Run("NilTicker", func(t *testing.T) {
		assert := assert.New(t)
		client, stopServer, err := newStartStopClient(true)
		assert.Nil(err)
		defer stopServer()
		client.observer.ticker = nil
		assert.Equal(ErrUndefinedIntervalTicker, client.Start(context.Background()))
	})
}

func newStartStopClient(includeListener bool) (*ListenerClient, func(), error) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Write(getItemsValidPayload())
	}))

	config := ListenerClientConfig{
		PullInterval: time.Millisecond * 200,
		Logger:       sallust.Default(),
	}
	if includeListener {
		config.Listener = mockListener
	}
	client, err := NewListenerClient(config, nil, mockMeasures, &BasicClient{})
	if err != nil {
		return nil, nil, err
	}

	return client, server.Close, nil
}

func TestValidateListenerConfig(t *testing.T) {
	tcs := []struct {
		desc        string
		expectedErr error
		config      ListenerClientConfig
	}{
		{
			desc:   "Happy case Success",
			config: happyListenerClientConfig,
		},
		{
			desc:        "No listener Failure",
			config:      ListenerClientConfig{},
			expectedErr: ErrNoListenerProvided,
		},
		{
			desc: "No logger and no pull interval Success",
			config: ListenerClientConfig{
				Listener: mockListener,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			c := tc.config
			err := validateListenerConfig(&c)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestNewListenerClient(t *testing.T) {
	tcs := []struct {
		desc        string
		config      ListenerClientConfig
		expectedErr error
		measures    *Measures
		reader      Reader
	}{
		{
			desc:        "Listener Config Failure",
			config:      ListenerClientConfig{},
			expectedErr: ErrNoListenerProvided,
		},
		{
			desc:        "No measures Failure",
			config:      happyListenerClientConfig,
			expectedErr: ErrNilMeasures,
		},
		{
			desc:        "No reader Failure",
			config:      happyListenerClientConfig,
			measures:    mockMeasures,
			expectedErr: ErrNoReaderProvided,
		},
		{
			desc:     "Happy case Success",
			config:   happyListenerClientConfig,
			measures: mockMeasures,
			reader:   &BasicClient{},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			_, err := NewListenerClient(tc.config, nil, tc.measures, tc.reader)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

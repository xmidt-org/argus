/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package chrysom

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/themis/xlog"
)

const (
	storeAPIPath     = "/api/v1/store"
	errWrappedFmt    = "%w: %s"
	errStatusCodeFmt = "statusCode %v: %w"
)

// Errors that can be returned by this package. Since some of these errors are returned wrapped, it
// is safest to use errors.Is() to check for them.
// Some internal errors might be unwrapped from output errors but unless these errors become exported,
// they are not part of the library API and may change in future versions.
var (
	ErrFailedAuthentication = errors.New("failed to authentication with argus")

	ErrListenerNotStopped = errors.New("listener is either running or starting")
	ErrListenerNotRunning = errors.New("listener is either stopped or stopping")
)

// listening states
const (
	stopped int32 = iota
	running
	transitioning
)

type ListenerClient struct {
	observer  *observerConfig
	logger    log.Logger
	setLogger func(context.Context, log.Logger) context.Context
}

type observerConfig struct {
	listener     Listener
	ticker       *time.Ticker
	pullInterval time.Duration
	measures     *Measures
	shutdown     chan struct{}
	state        int32
}

// Start begins listening for updates on an interval given that client configuration
// is setup correctly. If a listener process is already in progress, calling Start()
// is a NoOp. If you want to restart the current listener process, call Stop() first.
func (c *ListenerClient) Start(ctx context.Context) error {
	if c.observer == nil || c.observer.listener == nil {
		level.Warn(c.logger).Log(xlog.MessageKey(), "No listener was setup to receive updates.")
		return nil
	}
	if c.observer.ticker == nil {
		level.Error(c.logger).Log(xlog.MessageKey(), "Observer ticker is nil")
		return ErrUndefinedIntervalTicker
	}

	if !atomic.CompareAndSwapInt32(&c.observer.state, stopped, transitioning) {
		level.Error(c.logger).Log(xlog.MessageKey(), "Start called when a listener was not in stopped state", "err", ErrListenerNotStopped)
		return ErrListenerNotStopped
	}

	c.observer.ticker.Reset(c.observer.pullInterval)
	go func() {
		for {
			select {
			case <-c.observer.shutdown:
				return
			case <-c.observer.ticker.C:
				outcome := SuccessOutcome
				ctx := c.setLogger(context.Background(), c.logger)
				items, err := c.GetItems(ctx, "")
				if err == nil {
					c.observer.listener.Update(items)
				} else {
					outcome = FailureOutcome
					level.Error(c.logger).Log(xlog.MessageKey(), "Failed to get items for listeners", xlog.ErrorKey(), err)
				}
				c.observer.measures.Polls.With(prometheus.Labels{
					OutcomeLabel: outcome}).Add(1)
			}
		}
	}()

	atomic.SwapInt32(&c.observer.state, running)
	return nil
}

// Stop requests the current listener process to stop and waits for its goroutine to complete.
// Calling Stop() when a listener is not running (or while one is getting stopped) returns an
// error.
func (c *ListenerClient) Stop(ctx context.Context) error {
	if c.observer == nil || c.observer.ticker == nil {
		return nil
	}

	if !atomic.CompareAndSwapInt32(&c.observer.state, running, transitioning) {
		level.Error(c.logger).Log(xlog.MessageKey(), "Stop called when a listener was not in running state", "err", ErrListenerNotStopped)
		return ErrListenerNotRunning
	}

	c.observer.ticker.Stop()
	c.observer.shutdown <- struct{}{}
	atomic.SwapInt32(&c.observer.state, stopped)
	return nil
}

func NewListenerClient(config ListenerClientConfig,
	setLogger func(context.Context, log.Logger) context.Context,
	measures *Measures) (*ListenerClient, error) {
	if measures == nil {
		return nil, ErrNilMeasures
	}
	if config.Logger == nil {
		config.Logger = log.NewNopLogger()
	}
	if config.PullInterval == 0 {
		config.PullInterval = time.Second * 5
	}

	if setLogger == nil {
		setLogger = func(ctx context.Context, _ log.Logger) context.Context {
			return ctx
		}
	}
	return &ListenerClient{
		observer: &observerConfig{
			listener:     config.Listener,
			ticker:       time.NewTicker(config.PullInterval),
			pullInterval: config.PullInterval,
			measures:     measures,
			shutdown:     make(chan struct{}),
		},
		logger:    config.Logger,
		setLogger: setLogger,
	}, nil
}

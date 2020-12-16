/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/justinas/alice"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/themis/xlog"
	"go.uber.org/fx"
)

// GetLogger pulls the logger from the context and adds a timestamp to it.
func GetLogger(ctx context.Context) log.Logger {
	logger := log.With(xlog.GetDefault(ctx, nil), xlog.TimestampKey(), log.DefaultTimestampUTC)
	return logger
}

type logOptionsProvider struct {
	ServerName string
}

// SetLogger creates an alice constructor that sets up a logger that can be
// used for all logging related to the current request.  The logger is added to
// the request's context.
func (l logOptionsProvider) SetLogger(logger log.Logger) alice.Constructor {
	return func(delegate http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				logHeader := r.Header.Clone()
				if str := logHeader.Get("Authorization"); str != "" {
					logHeader.Del("Authorization")
					logHeader.Set("Authorization-Type", strings.Split(str, " ")[0])
				}
				r = r.WithContext(xlog.With(r.Context(), log.With(logger, "requestHeaders", logHeader, "requestURL", r.URL.EscapedPath(),
					"method", r.Method, "server", l.ServerName)))
				delegate.ServeHTTP(w, r)
			})
	}
}

// getBasculeLogger simply convers a go-kit logger to a bascule logger.  They
// are the same.
func getBasculeLogger(f func(context.Context) log.Logger) func(context.Context) bascule.Logger {
	return func(ctx context.Context) bascule.Logger {
		return bascule.Logger(f(ctx))
	}
}

// TODO: if we see the need, we could split this for each server...
func (l logOptionsProvider) provide() fx.Option {
	return fx.Options(
		fx.Supply(GetLogger),
		fx.Provide(
			fx.Annotated{
				Name:   fmt.Sprintf("%s_set_logger", l.ServerName),
				Target: l.SetLogger,
			},

			fx.Annotated{
				Group: fmt.Sprintf("%s_bascule_constructor_options", l.ServerName),
				Target: func(getLogger func(context.Context) log.Logger) basculehttp.COption {
					return basculehttp.WithCLogger(getBasculeLogger(getLogger))
				},
			},

			fx.Annotated{
				Group: fmt.Sprintf("%s_bascule_constructor_options", l.ServerName),
				Target: func(getLogger func(context.Context) log.Logger) basculehttp.EOption {
					return basculehttp.WithELogger(getBasculeLogger(getLogger))
				},
			},
		),
	)
}

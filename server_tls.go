// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/httpaux"
	serveraux "github.com/xmidt-org/httpaux/server"
)

// MtlsConfig adds themis-like mTLS controls to the server TLS config.
//
// DisableRequire and DisableVerify map to tls.Config.ClientAuth as follows:
//   - DisableRequire=true,  DisableVerify=true  => tls.RequestClientCert
//   - DisableRequire=false, DisableVerify=true  => tls.RequireAnyClientCert
//   - DisableRequire=true,  DisableVerify=false => tls.VerifyClientCertIfGiven
//   - DisableRequire=false, DisableVerify=false => tls.RequireAndVerifyClientCert
//
// ClientCACertificateFile should point to a PEM bundle for client certificate
// verification and trust.
type MtlsConfig struct {
	ClientCACertificateFile string `mapstructure:"clientCACertificateFile"`
	DisableRequire          bool   `mapstructure:"disableRequire"`
	DisableVerify           bool   `mapstructure:"disableVerify"`
}

// ArgusTLSConfig wraps arrange TLS settings and optionally adds mTLS flags.
type ArgusTLSConfig struct {
	arrangetls.Config `mapstructure:",squash"`
	Mtls              *MtlsConfig `mapstructure:"mtls"`
}

// New builds the base tls.Config via arrangetls, then applies optional mTLS overrides.
func (c *ArgusTLSConfig) New(extra ...arrangetls.PeerVerifier) (*tls.Config, error) {
	if c == nil {
		return nil, nil
	}

	tc, err := c.Config.New(extra...)
	if err != nil {
		return nil, err
	}

	return configureMtls(tc, c.Mtls)
}

func configureMtls(tc *tls.Config, mtls *MtlsConfig) (*tls.Config, error) {
	if tc == nil || mtls == nil {
		return tc, nil
	}

	tc.ClientAuth = clientAuthMode(mtls.DisableRequire, mtls.DisableVerify)

	clientCAs, err := readCertPool(mtls.ClientCACertificateFile)
	if err != nil {
		return nil, err
	}
	tc.ClientCAs = clientCAs
	return tc, nil
}

func clientAuthMode(disableRequire, disableVerify bool) tls.ClientAuthType {
	switch {
	case disableRequire && disableVerify:
		return tls.RequestClientCert
	case !disableRequire && disableVerify:
		return tls.RequireAnyClientCert
	case disableRequire && !disableVerify:
		return tls.VerifyClientCertIfGiven
	default:
		return tls.RequireAndVerifyClientCert
	}
}

func readCertPool(path string) (*x509.CertPool, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(contents) {
		return nil, fmt.Errorf("unable to add certificates from %s", path)
	}
	return pool, nil
}

// ArgusServerConfig is a custom arrangehttp server factory that supports the
// default arrangehttp settings plus themis-like TLS mTLS options.
type ArgusServerConfig struct {
	Network string
	Address string

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	KeepAlive         time.Duration

	Header http.Header
	TLS    *ArgusTLSConfig
}

func (sc ArgusServerConfig) NewServer(h http.Handler) (server *http.Server, err error) {
	header := httpaux.NewHeader(sc.Header)

	server = &http.Server{
		Addr:              sc.Address,
		Handler:           serveraux.Header(header.SetTo)(h),
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       sc.IdleTimeout,
		MaxHeaderBytes:    sc.MaxHeaderBytes,
	}

	server.TLSConfig, err = sc.TLS.New()
	return
}

func (sc ArgusServerConfig) Listen(ctx context.Context, s *http.Server) (net.Listener, error) {
	return arrangehttp.ServerConfig{
		Network:   sc.Network,
		Address:   sc.Address,
		KeepAlive: sc.KeepAlive,
	}.Listen(ctx, s)
}

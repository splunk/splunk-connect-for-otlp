// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package bearerauthextension

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	errNoAuth              = errors.New("no basic auth provided")
	errInvalidCredentials  = errors.New("invalid credentials")
	errInvalidSchemePrefix = errors.New("invalid authorization scheme prefix")
)

func newServerAuthExtension(cfg *Config) (*basicAuthServer, error) {
	return &basicAuthServer{
		tokens: cfg.Tokens,
	}, nil
}

var (
	_ extension.Extension  = (*basicAuthServer)(nil)
	_ extensionauth.Server = (*basicAuthServer)(nil)
)

type basicAuthServer struct {
	component.ShutdownFunc
	tokens []configopaque.String
}

func (ba *basicAuthServer) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (ba *basicAuthServer) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	auth := getAuthHeader(headers)
	if auth == "" {
		return ctx, errNoAuth
	}

	token, err := parseToken(auth)
	if err != nil {
		return ctx, err
	}

	for _, t := range ba.tokens {
		if string(t) == token {
			cl := client.FromContext(ctx)
			return client.NewContext(ctx, cl), nil
		}
	}

	return ctx, errInvalidCredentials
}

func getAuthHeader(h map[string][]string) string {
	const (
		canonicalHeaderKey = "Authorization"
		metadataKey        = "authorization"
	)

	authHeaders, ok := h[canonicalHeaderKey]

	if !ok {
		authHeaders, ok = h[metadataKey]
	}

	if !ok {
		for k, v := range h {
			if strings.EqualFold(k, metadataKey) {
				authHeaders = v
				break
			}
		}
	}

	if len(authHeaders) == 0 {
		return ""
	}

	return authHeaders[0]
}

func parseToken(auth string) (string, error) {
	// TODO support Basic as well.
	const prefix = "Splunk "

	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", errInvalidSchemePrefix
	}

	token := auth[len(prefix):]
	return token, nil
}

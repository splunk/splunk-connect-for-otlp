// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package splunkauthextension

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"go.opentelemetry.io/collector/config/configopaque"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.uber.org/zap"
)

var (
	_ extension.Extension  = (*splunkAuth)(nil)
	_ extensionauth.Server = (*splunkAuth)(nil)
)

type Key string

const ContextKey Key = "hec"

// BearerTokenAuth is an implementation of extensionauth interfaces. It embeds a static authorization "bearer" token in every rpc call.
type splunkAuth struct {
	authorizationValuesAtomic atomic.Value
	shutdownCH                chan struct{}
	logger                    *zap.Logger
	header                    string
	scheme                    string
	filename                  string
}

const (
	defaultHeader = "Authorization"
	defaultScheme = "Splunk"
)

func newBearerTokenAuth(cfg *Config, logger *zap.Logger) *splunkAuth {
	a := &splunkAuth{
		header: defaultHeader,
		scheme: defaultScheme,
		logger: logger,
	}
	a.setAuthorizationValues(cfg.Tokens) // Store tokens
	return a
}

// Start of BearerTokenAuth does nothing and returns nil
func (b *splunkAuth) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (b *splunkAuth) setAuthorizationValues(tokens []HecTokenConfig) {
	values := make([]HecTokenConfig, len(tokens))
	for i, token := range tokens {
		if b.scheme != "" {
			values[i].Token = configopaque.String(b.scheme + " " + string(token.Token))
		} else {
			values[i] = token
		}
	}
	b.authorizationValuesAtomic.Store(values)
}

// authorizationValues returns the Authorization header/metadata values
// to set for client auth, and expected values for server auth.
func (b *splunkAuth) authorizationValues() []HecTokenConfig {
	return b.authorizationValuesAtomic.Load().([]HecTokenConfig)
}

// Shutdown of BearerTokenAuth does nothing and returns nil
func (b *splunkAuth) Shutdown(_ context.Context) error {
	if b.filename == "" {
		return nil
	}

	if b.shutdownCH == nil {
		return errors.New("bearerToken file monitoring is not running")
	}
	b.shutdownCH <- struct{}{}
	close(b.shutdownCH)
	b.shutdownCH = nil
	return nil
}

// Authenticate checks whether the given context contains valid auth data. Validates tokens from clients trying to access the service (incoming requests)
func (b *splunkAuth) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	auth, ok := headers[strings.ToLower(b.header)]
	if !ok {
		auth, ok = headers[b.header]
	}
	if !ok || len(auth) == 0 {
		return ctx, fmt.Errorf("missing or empty authorization header: %s", b.header)
	}
	token := auth[0] // Extract token from authorization header
	expectedTokens := b.authorizationValues()
	for _, expectedToken := range expectedTokens {
		if subtle.ConstantTimeCompare([]byte(expectedToken.Token), []byte(token)) == 1 {
			return context.WithValue(ctx, ContextKey, expectedToken), nil // Authentication successful, token is valid
		}
	}
	return ctx, fmt.Errorf("scheme or token does not match: %s", token) // Token is invalid
}

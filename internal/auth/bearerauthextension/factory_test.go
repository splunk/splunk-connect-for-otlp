// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package bearerauthextension

import (
	"testing"

	"go.opentelemetry.io/collector/component"

	"go.opentelemetry.io/collector/config/configopaque"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

func TestCreateExtension_ValidConfig(t *testing.T) {
	cfg := &Config{
		Tokens: []configopaque.String{
			configopaque.String("foo"),
		},
	}

	ext, err := createExtension(t.Context(), extensiontest.NewNopSettings(extensiontest.NopType), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, ext)
}

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.NotNil(t, f)
	assert.Equal(t, f.Type(), component.MustNewType("token"))
}

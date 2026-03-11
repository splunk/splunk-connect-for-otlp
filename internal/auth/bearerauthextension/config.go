// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package bearerauthextension

import (
	"go.opentelemetry.io/collector/config/configopaque"
)

type Config struct {
	_      struct{}
	Tokens []configopaque.String
}

func (cfg *Config) Validate() error {
	return nil
}

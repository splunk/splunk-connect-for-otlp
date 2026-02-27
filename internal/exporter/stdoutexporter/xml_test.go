// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package stdoutexporter

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestXMLExporter(t *testing.T) {
	oneLog := plog.NewLogs()
	lr := oneLog.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("foo")
	buf := bytes.NewBuffer(nil)
	err := xmlExporter{}.consumeLogs(context.Background(), oneLog, zap.NewNop(), buf)
	require.NoError(t, err)
	require.Equal(t, "<stream><event><data>foo</data><host>unknown</host><index></index><source></source><sourcetype></sourcetype></event></stream>", buf.String())
}

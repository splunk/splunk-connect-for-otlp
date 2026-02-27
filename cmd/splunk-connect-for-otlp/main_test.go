// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/splunk/otlp2splunk/internal"
	"github.com/splunk/otlp2splunk/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestMainPrintsScheme(t *testing.T) {
	output := testutils.CaptureStdout(t, func() {
		originalArgs := os.Args
		os.Args = []string{"splunk-connect-for-otlp", "--scheme"}
		defer func() {
			os.Args = originalArgs
		}()

		main()
	})

	require.Equal(t, internal.Scheme+"\n", output)
}

func TestRunReturnsErrorForInvalidInput(t *testing.T) {
	restoreStdin := testutils.WriteToStdin(t, "not-xml")
	defer restoreStdin()

	err := run()
	require.Error(t, err)
}

func TestRunStartsAndStopsOnSignal(t *testing.T) {
	restoreStdin := testutils.WriteToStdin(t, `<input><configuration><stanza name="splunk-connect-for-otlp://test" app="search"><param name="grpc_port">0</param><param name="http_port">0</param><param name="listen_address">127.0.0.1</param></stanza></configuration></input>`)
	defer restoreStdin()

	done := make(chan error, 1)
	go func() {
		done <- run()
	}()

	time.AfterFunc(500*time.Millisecond, func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	})

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not complete in time")
	}
}

func TestExpected(t *testing.T) {
	tests := []struct {
		name         string
		otlpendpoint string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "metrics",
			otlpendpoint: "/v1/metrics",
			inputPath:    filepath.Join("testdata", "otlp_metrics.json"),
			expectedPath: filepath.Join("testdata", "expected_hec_metrics.json"),
		},
		{
			name:         "traces",
			otlpendpoint: "/v1/traces",
			inputPath:    filepath.Join("testdata", "otlp_traces.json"),
			expectedPath: filepath.Join("testdata", "expected_hec_traces.json"),
		},
		{
			name:         "logs",
			otlpendpoint: "/v1/logs",
			inputPath:    filepath.Join("testdata", "otlp_logs.json"),
			expectedPath: filepath.Join("testdata", "expected_logs.xml"),
		},
		{
			name:         "large file metrics",
			otlpendpoint: "/v1/metrics",
			inputPath:    filepath.Join("testdata", "otlp_metrics_big.json"),
			expectedPath: filepath.Join("testdata", "expected_hec_metrics_big.json"),
		},
		{
			name:         "large file traces",
			otlpendpoint: "/v1/traces",
			inputPath:    filepath.Join("testdata", "otlp_traces_big.json"),
			expectedPath: filepath.Join("testdata", "expected_hec_traces_big.json"),
		},
		{
			name:         "large file logs",
			otlpendpoint: "/v1/logs",
			inputPath:    filepath.Join("testdata", "otlp_logs_big.json"),
			expectedPath: filepath.Join("testdata", "expected_logs_big.xml"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			grpcPort := testutils.GetFreePort(t)
			httpPort := testutils.GetFreePort(t)

			config := fmt.Sprintf(`<input><configuration><stanza name="splunk-connect-for-otlp://test" app="search"><param name="grpc_port">%d</param><param name="http_port">%d</param><param name="listen_address">127.0.0.1</param></stanza></configuration></input>`, grpcPort, httpPort)

			restoreStdin := testutils.WriteToStdin(t, config)
			t.Cleanup(restoreStdin)

			var stdoutLines <-chan string
			var restoreStdout func()

			if strings.HasSuffix(tt.expectedPath, ".xml") {
				stdoutLines, restoreStdout = testutils.CaptureStdoutXML(t)
			} else {
				stdoutLines, restoreStdout = testutils.CaptureStdoutLines(t)
			}

			t.Cleanup(restoreStdout)

			runDone := make(chan error, 1)
			go func() {
				runDone <- run()
			}()

			payload, err := os.ReadFile(tt.inputPath)
			require.NoError(t, err)

			expected := testutils.LoadExpectedData(t, tt.expectedPath)
			expectedLines := strings.Split(strings.TrimSpace(string(expected)), "\n")
			require.NotEmpty(t, expectedLines, "%s must contain fixture data", tt.expectedPath)

			testutils.PostOTLP(t, httpPort, tt.otlpendpoint, payload)

			actual := testutils.CollectLines(t, stdoutLines, len(expectedLines))
			require.Equal(t, expectedLines, actual)

			require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
			require.NoError(t, <-runDone)
		})
	}
}

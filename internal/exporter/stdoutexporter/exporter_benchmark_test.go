package stdoutexporter

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/splunk/otlp2splunk/internal/testutils"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var testDataDir = "../../../cmd/splunk-connect-for-otlp/testdata"

func BenchmarkStdoutExporter(b *testing.B) {
	stdoutWriter = func([]byte) error { return nil }

	settings := exportertest.NewNopSettings(exportertest.NopType)
	cfg := createDefaultConfig().(*Config)
	cfg.QueueBatchConfig.GetOrInsertDefault().QueueSize = 1e6 // Set to a large value so sending queue doesn't get full
	ctx := context.Background()

	tests := []struct {
		name          string
		telType       testutils.TelemetryType
		inputFilePath string
	}{
		{
			name:          "metrics",
			telType:       testutils.TelemetryTypeMetrics,
			inputFilePath: filepath.Join(testDataDir, "otlp_metrics.json"),
		},
		{
			name:          "large metric data set",
			telType:       testutils.TelemetryTypeMetrics,
			inputFilePath: filepath.Join(testDataDir, "otlp_metrics_big.json"),
		},
		{
			name:          "traces",
			telType:       testutils.TelemetryTypeTraces,
			inputFilePath: filepath.Join(testDataDir, "otlp_traces.json"),
		},
		{
			name:          "large trace data set",
			telType:       testutils.TelemetryTypeTraces,
			inputFilePath: filepath.Join(testDataDir, "otlp_traces_big.json"),
		},
		{
			name:          "logs",
			telType:       testutils.TelemetryTypeLogs,
			inputFilePath: filepath.Join(testDataDir, "otlp_logs.json"),
		},
		{
			name:          "large log data set",
			telType:       testutils.TelemetryTypeLogs,
			inputFilePath: filepath.Join(testDataDir, "otlp_logs_big.json"),
		},
	}

	for _, tt := range tests {
		tt := tt
		b.Run(tt.name, func(b *testing.B) {
			var consume func(context.Context) error
			switch tt.telType {
			case testutils.TelemetryTypeMetrics:
				metrics := testutils.LoadMetricsFromFile(b, tt.inputFilePath)
				consume = setupMetricsExporter(b, ctx, settings, cfg, metrics)
			case testutils.TelemetryTypeTraces:
				traces := testutils.LoadTracesFromFile(b, tt.inputFilePath)
				consume = setupTracesExporter(b, ctx, settings, cfg, traces)
			case testutils.TelemetryTypeLogs:
				logs := testutils.LoadLogsFromFile(b, tt.inputFilePath)
				consume = setupLogsExporter(b, ctx, settings, cfg, logs)
			default:
				b.Fatalf("unknown telemetry type: %v", tt.telType)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := consume(ctx); err != nil {
					b.Fatalf("%s failed: %v", tt.name, err)
				}
			}
		})
	}
}

func setupMetricsExporter(b *testing.B, ctx context.Context, settings exporter.Settings, cfg *Config, metrics pmetric.Metrics) func(context.Context) error {
	b.Helper()
	exp, err := newMetricsExporter(ctx, settings, cfg)
	if err != nil {
		b.Fatalf("failed to create metrics exporter: %v", err)
	}
	if err = exp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start metrics exporter: %v", err)
	}
	return func(ctx context.Context) error {
		return exp.ConsumeMetrics(ctx, metrics)
	}
}

func setupTracesExporter(b *testing.B, ctx context.Context, settings exporter.Settings, cfg *Config, traces ptrace.Traces) func(context.Context) error {
	b.Helper()
	exp, err := newTracesExporter(ctx, settings, cfg)
	if err != nil {
		b.Fatalf("failed to create traces exporter: %v", err)
	}
	if err = exp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start traces exporter: %v", err)
	}
	return func(ctx context.Context) error {
		return exp.ConsumeTraces(ctx, traces)
	}
}

func setupLogsExporter(b *testing.B, ctx context.Context, settings exporter.Settings, cfg *Config, logs plog.Logs) func(context.Context) error {
	b.Helper()
	exp, err := newLogsExporter(ctx, settings, cfg)
	if err != nil {
		b.Fatalf("failed to create logs exporter: %v", err)
	}
	if err = exp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start logs exporter: %v", err)
	}
	return func(ctx context.Context) error {
		return exp.ConsumeLogs(ctx, logs)
	}
}

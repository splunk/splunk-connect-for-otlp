// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package stdoutexporter

import (
	"context"
	"io"
	"os"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var stdoutWriter = defaultStdoutWriter

type transformExporter interface {
	consumeLogs(ctx context.Context, ld plog.Logs, logger *zap.Logger, writer io.Writer) error
	consumeTraces(ctx context.Context, td ptrace.Traces, logger *zap.Logger, writer io.Writer) error
	consumeMetrics(ctx context.Context, md pmetric.Metrics, logger *zap.Logger, writer io.Writer) error
}

var (
	logExporter    = xmlExporter{}
	metricExporter = hecExporter{}
	traceExporter  = hecExporter{}
)

func newLogsExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Logs, error) {
	oCfg := cfg.(*Config)

	e := &stdoutExporter{}

	return exporterhelper.NewLogs(ctx, set, cfg,
		func(ctx context.Context, ld plog.Logs) error {
			return logExporter.consumeLogs(ctx, ld, set.Logger, e)
		},
		exporterhelper.WithCapabilities(consumer.Capabilities{
			MutatesData: false,
		}),
		exporterhelper.WithQueue(oCfg.QueueBatchConfig))
}

func newTracesExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Traces, error) {
	oCfg := cfg.(*Config)

	e := &stdoutExporter{}

	return exporterhelper.NewTraces(ctx, set, cfg,
		func(ctx context.Context, td ptrace.Traces) error {
			return traceExporter.consumeTraces(ctx, td, set.Logger, e)
		},
		exporterhelper.WithCapabilities(consumer.Capabilities{
			MutatesData: false,
		}),
		exporterhelper.WithQueue(oCfg.QueueBatchConfig))
}

func newMetricsExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Metrics, error) {
	oCfg := cfg.(*Config)

	e := &stdoutExporter{}

	return exporterhelper.NewMetrics(ctx, set, cfg,
		func(ctx context.Context, md pmetric.Metrics) error {
			return metricExporter.consumeMetrics(ctx, md, set.Logger, e)
		},
		exporterhelper.WithCapabilities(consumer.Capabilities{
			MutatesData: false,
		}),
		exporterhelper.WithQueue(oCfg.QueueBatchConfig))
}

type stdoutExporter struct {
	TelemetrySettings component.TelemetrySettings
}

func (se *stdoutExporter) Write(b []byte) (int, error) {
	return stdoutWriter(b)
}

func defaultStdoutWriter(b []byte) (int, error) {
	return os.Stdout.Write(b)
}

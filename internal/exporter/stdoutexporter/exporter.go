// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package stdoutexporter

import (
	"context"
	"errors"
	"os"

	"github.com/goccy/go-json"
	translator "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/splunk"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var stdoutWriter = defaultStdoutWriter

func newLogsExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Logs, error) {
	oCfg := cfg.(*Config)

	e := &stdoutExporter{}

	return exporterhelper.NewLogs(ctx, set, cfg, e.ConsumeLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{
			MutatesData: false,
		}),
		exporterhelper.WithQueue(oCfg.QueueBatchConfig))
}

func newTracesExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Traces, error) {
	oCfg := cfg.(*Config)

	e := &stdoutExporter{}

	return exporterhelper.NewTraces(ctx, set, cfg, e.ConsumeTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{
			MutatesData: false,
		}),
		exporterhelper.WithQueue(oCfg.QueueBatchConfig))
}

func newMetricsExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (exporter.Metrics, error) {
	oCfg := cfg.(*Config)

	e := &stdoutExporter{}

	return exporterhelper.NewMetrics(ctx, set, cfg, e.ConsumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{
			MutatesData: false,
		}),
		exporterhelper.WithQueue(oCfg.QueueBatchConfig))
}

type stdoutExporter struct {
	TelemetrySettings component.TelemetrySettings
}

func (se *stdoutExporter) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	toOtelAttrs := translator.DefaultHecToOtelAttrs()
	toHecAttrs := translator.DefaultOtelToHecFields()

	var errs []error
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)
		r := rl.Resource()
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				logRecord := sl.LogRecords().At(k)
				event := translator.LogToSplunkEvent(r, logRecord, toOtelAttrs, toHecAttrs, "", "", "")
				if event == nil {
					continue
				}
				b, err := json.Marshal(&event)
				if err != nil {
					errs = append(errs, err)
				} else {
					if err = se.writeToStdout(b); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (se *stdoutExporter) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	toOtelAttrs := translator.DefaultHecToOtelAttrs()

	var errs []error
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		r := rs.Resource()
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)
				b, err := json.Marshal(translator.SpanToSplunkEvent(r, span, toOtelAttrs, "", "", ""))
				if err != nil {
					errs = append(errs, err)
				} else {
					if err = se.writeToStdout(b); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (se *stdoutExporter) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	toOtelAttrs := translator.DefaultHecToOtelAttrs()

	var errs []error
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		r := rm.Resource()
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				for _, result := range translator.MetricToSplunkEvent(r, m, se.TelemetrySettings.Logger, toOtelAttrs, "", "", "") {
					b, err := json.Marshal(result)
					if err != nil {
						errs = append(errs, err)
					} else {
						if err = se.writeToStdout(b); err != nil {
							errs = append(errs, err)
						}
					}
				}

			}
		}
	}
	return errors.Join(errs...)
}

func (se *stdoutExporter) writeToStdout(b []byte) error {
	return stdoutWriter(b)
}

func defaultStdoutWriter(b []byte) error {
	_, err := os.Stdout.Write(append(b, '\n'))
	return err
}

// Copyright Splunk, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stdoutexporter

import (
	"context"
	"errors"
	"io"

	"github.com/goccy/go-json"
	translator "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/splunk"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type hecExporter struct{}

var _ transformExporter = (*hecExporter)(nil)

func (hecExporter) consumeLogs(_ context.Context, ld plog.Logs, logger *zap.Logger, writer io.Writer) error {
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
					if _, err = writer.Write(b); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (hecExporter) consumeTraces(_ context.Context, td ptrace.Traces, logger *zap.Logger, writer io.Writer) error {
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
					if _, err = writer.Write(append(b, '\n')); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (hecExporter) consumeMetrics(_ context.Context, md pmetric.Metrics, logger *zap.Logger, writer io.Writer) error {
	toOtelAttrs := translator.DefaultHecToOtelAttrs()

	var errs []error
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		r := rm.Resource()
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				for _, result := range translator.MetricToSplunkEvent(r, m, logger, toOtelAttrs, "", "", "") {
					b, err := json.Marshal(result)
					if err != nil {
						errs = append(errs, err)
					} else {
						if _, err = writer.Write(append(b, '\n')); err != nil {
							errs = append(errs, err)
						}
					}
				}

			}
		}
	}
	return errors.Join(errs...)
}

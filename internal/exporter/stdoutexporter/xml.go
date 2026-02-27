// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package stdoutexporter

import (
	"context"
	"encoding/xml"
	"errors"
	"io"

	"github.com/goccy/go-json"
	translator "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/splunk"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type xmlExporter struct{}

var _ transformExporter = (*xmlExporter)(nil)

func (xmlExporter) consumeLogs(_ context.Context, ld plog.Logs, logger *zap.Logger, writer io.Writer) error {
	toOtelAttrs := translator.DefaultHecToOtelAttrs()
	toHecAttrs := translator.DefaultOtelToHecFields()
	var errs []error

	encoder := xml.NewEncoder(writer)

	if err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "stream"}}); err != nil {
		errs = append(errs, err)
	}

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

				var evAsString string
				switch event.Event.(type) {
				case string:
					evAsString = event.Event.(string)
				default:
					evAsBytes, _ := json.Marshal(event.Event)
					evAsString = string(evAsBytes)
				}

				tokens := []xml.Token{xml.StartElement{Name: xml.Name{Local: "event"}}}

				tokens = append(tokens, attributeToXml("data", evAsString)...)
				tokens = append(tokens, attributeToXml("host", event.Host)...)
				tokens = append(tokens, attributeToXml("index", event.Index)...)
				tokens = append(tokens, attributeToXml("source", event.Source)...)
				tokens = append(tokens, attributeToXml("sourcetype", event.SourceType)...)

				tokens = append(tokens, xml.EndElement{Name: xml.Name{Local: "event"}})

				for _, t := range tokens {
					err := encoder.EncodeToken(t)
					if err != nil {
						errs = append(errs, err)
					}
				}

				if err := encoder.Flush(); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "stream"}}); err != nil {
		errs = append(errs, err)
	}

	// close to ensure tokens are written
	if err := encoder.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func attributeToXml(key, value string) []xml.Token {
	t := xml.StartElement{Name: xml.Name{Local: key}}
	return []xml.Token{t, xml.CharData(value), xml.EndElement{Name: t.Name}}
}

func (xmlExporter) consumeTraces(_ context.Context, td ptrace.Traces, logger *zap.Logger, writer io.Writer) error {
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
					if _, err = writer.Write(b); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (xmlExporter) consumeMetrics(_ context.Context, md pmetric.Metrics, logger *zap.Logger, writer io.Writer) error {
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
						if _, err = writer.Write(b); err != nil {
							errs = append(errs, err)
						}
					}
				}

			}
		}
	}
	return errors.Join(errs...)
}

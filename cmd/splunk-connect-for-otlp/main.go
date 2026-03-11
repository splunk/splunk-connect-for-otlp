// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

// Program splunk-connect-for-otlp is a binary listening for OTLP data and exporting it to stdout.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/splunk/otlp2splunk/internal"
	"github.com/splunk/otlp2splunk/internal/exporter/stdoutexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--scheme":
			fmt.Println(internal.Scheme)
		case "--validate-arguments":

		}
	} else if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			fmt.Println("Stack trace:")
			fmt.Println(string(debug.Stack()))
		}
	}()
	config, err := internal.ReadFromStdin()
	if err != nil {
		return err
	}

	logger, err := internal.CreateLogger()
	if err != nil {
		return err
	}
	logger.Info("Starting OTLP input")

	settings := component.TelemetrySettings{
		Logger:         logger,
		TracerProvider: noop.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		Resource:       pcommon.NewResource(),
	}

	grpcPort, httpPort, listeningAddress, index, source, sourcetype := config.Extract()
	stdoutCfg := stdoutexporter.NewFactory().CreateDefaultConfig().(*stdoutexporter.Config)
	stdoutCfg.Index = index
	stdoutCfg.Source = source
	stdoutCfg.Sourcetype = sourcetype

	f := stdoutexporter.NewFactory()
	ctx := context.Background()
	telemetrySettings := exporter.Settings{
		TelemetrySettings: settings,
		ID:                component.MustNewID("stdout"),
	}
	le, err := f.CreateLogs(ctx, telemetrySettings, stdoutCfg)
	if err != nil {
		return err
	}
	me, err := f.CreateMetrics(ctx, telemetrySettings, stdoutCfg)
	if err != nil {
		return err
	}
	te, err := f.CreateTraces(ctx, telemetrySettings, stdoutCfg)
	if err != nil {
		return err
	}
	logger.Info("Configured exporter")

	rf := otlpreceiver.NewFactory()
	cfg := rf.CreateDefaultConfig().(*otlpreceiver.Config)
	_ = cfg.GRPC.Unmarshal(confmap.NewFromStringMap(map[string]any{"endpoint": fmt.Sprintf("%s:%d", listeningAddress, grpcPort)}))
	_ = cfg.HTTP.Unmarshal(confmap.NewFromStringMap(map[string]any{"endpoint": fmt.Sprintf("%s:%d", listeningAddress, httpPort)}))

	if _, err = rf.CreateLogs(ctx, receiver.Settings{
		TelemetrySettings: settings,
		ID:                component.MustNewID("otlp"),
	}, cfg, le); err != nil {
		return err
	}
	if _, err = rf.CreateMetrics(ctx, receiver.Settings{
		TelemetrySettings: settings,
		ID:                component.MustNewID("otlp"),
	}, cfg, me); err != nil {
		return err
	}
	r, err := rf.CreateTraces(ctx, receiver.Settings{
		TelemetrySettings: settings,
		ID:                component.MustNewID("otlp"),
	}, cfg, te)
	if err != nil {
		return err
	}

	logger.Info("Configured OTLP receiver")

	h := &internal.TTYHost{
		ErrStatus:  make(chan error, 1),
		Extensions: map[component.ID]component.Component{},
	}
	h.Start()

	if err = le.Start(ctx, h); err != nil {
		return err
	}
	if err = me.Start(ctx, h); err != nil {
		return err
	}
	if err = te.Start(ctx, h); err != nil {
		return err
	}
	if err = r.Start(ctx, h); err != nil {
		return err
	}

	logger.Info("OTLP Input started")

	err = h.Wait()

	_ = r.Shutdown(ctx)
	_ = le.Shutdown(ctx)
	_ = te.Shutdown(ctx)
	_ = me.Shutdown(ctx)

	return err
}

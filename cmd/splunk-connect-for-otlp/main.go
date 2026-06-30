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

	"github.com/splunk/otlp2splunk/internal/auth"

	"github.com/splunk/otlp2splunk/internal"
	"github.com/splunk/otlp2splunk/internal/exporter/stdoutexporter"
	"go.opentelemetry.io/collector/component"
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

	xmlCfg := config.Extract()
	if err = xmlCfg.Validate(); err != nil {
		return err
	}
	stdoutCfg := stdoutexporter.NewFactory().CreateDefaultConfig().(*stdoutexporter.Config)
	stdoutCfg.Source = xmlCfg.Source
	stdoutCfg.Sourcetype = xmlCfg.Sourcetype

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
	cfg.GRPC.GetOrInsertDefault().NetAddr.Endpoint = fmt.Sprintf("%s:%d", xmlCfg.ListenAddress, xmlCfg.GrpcPort)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.NetAddr.Endpoint = fmt.Sprintf("%s:%d", xmlCfg.ListenAddress, xmlCfg.HTTPPort)
	extID := component.MustNewID("splunkauth")
	cfg.GRPC.Get().Auth.GetOrInsertDefault().AuthenticatorID = extID
	cfg.HTTP.Get().ServerConfig.Auth.GetOrInsertDefault().AuthenticatorID = extID

	if xmlCfg.EnableSSL {
		cfg.GRPC.Get().TLS.GetOrInsertDefault().CertFile = xmlCfg.ServerCert
		cfg.HTTP.Get().ServerConfig.TLS.GetOrInsertDefault().CertFile = xmlCfg.ServerCert
		cfg.GRPC.Get().TLS.GetOrInsertDefault().KeyFile = xmlCfg.ServerKey
		cfg.HTTP.Get().ServerConfig.TLS.GetOrInsertDefault().KeyFile = xmlCfg.ServerKey
	}

	otlpSettings := receiver.Settings{
		TelemetrySettings: settings,
		ID:                component.MustNewID("otlp"),
	}
	if _, err = rf.CreateLogs(ctx, otlpSettings, cfg, le); err != nil {
		return err
	}
	if _, err = rf.CreateMetrics(ctx, otlpSettings, cfg, me); err != nil {
		return err
	}
	r, err := rf.CreateTraces(ctx, otlpSettings, cfg, te)
	if err != nil {
		return err
	}

	logger.Info("Configured OTLP receiver")

	auth, err := auth.New(ctx, settings, xmlCfg.ServerURI, xmlCfg.SessionKey)
	if err != nil {
		return err
	}

	h := &internal.TTYHost{
		ErrStatus: make(chan error, 1),
		Extensions: map[component.ID]component.Component{
			extID: auth,
		},
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

# Splunk Connect for OTLP

This repository contains a technical addon that exposes a OTLP endpoint for consumption of logs, traces and metrics.


# Usage

## Installation

Add the .tgz file to your Splunk apps by following these instructions: [Install an add-on in a single-instance Splunk Enterprise deployment
](https://docs.splunk.com/Documentation/AddOns/released/Overview/Singleserverinstall)

## Configuration

Create a new OTLP input in the data inputs.

![Picker](docs/data_input_picker.png)

Configure the OTLP input ports, network interface and index.

![Setup](docs/data_input_setup.webp)

The input is configured as a data input in the Splunk Data Input settings.

You can set:
* The gRPC port and HTTP ports the OTLP receiver will listen on
* The network interface address on which the OTLP input will listen.

## Sending OTLP

When sending OTLP data, this input interprets resource attributes to create HEC equivalents.

This table shows the resource attributes mapping:

| Resource attribute    | HEC event field |
|-----------------------|-----------------|
| com.splunk.index      | index           |
| com.splunk.sourcetype | sourcetype      |
| com.splunk.source     | source          |
| host.name             | host            |

OpenTelemetry protocol representation of a log record contains additional fields. The table below shows the mapping of those fields to HEC event indexed fields:

| Log record field | HEC event indexed field    |
|------------------|----------------------------|
| Severity text    | `otel.log.severity.text`   |
| Severity number  | `otel.log.severity.number` |
 
All other resource and individual log record attributes are mapped to indexed fields.

This mapping follows the OpenTelemetry specification.

## Build

Prerequisites:
* Go 1.24
* Make

Run:
```shell
$> make tgz
```

This will build the binaries, and assemble the TA tar.gz archive.

The archive is created as splunk-connect-for-otlp.tgz.

## Testing

### Run a Splunk instance locally

You can run a local Splunk instance with:
```shell
make splunk
```

Log in with `admin`/`changeme`.

Install the application by going to Apps > Manage Apps > Install application from file.

### Send via collector

See the [example folder](./example) for a collector configuration.

### Telemetrygen

You can generate a payload using [telemetrygen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen.

Install telemetrygen with:
```shell
go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest
```

Try running telemetrygen with:

```shell
$> telemetrygen metrics --otlp-insecure --otlp-endpoint 0.0.0.0:4317 --metrics 100 --workers 10
$> telemetrygen logs --otlp-insecure --otlp-endpoint 0.0.0.0:4317 --logs 100 --workers 10
$> telemetrygen traces --otlp-insecure --otlp-endpoint 0.0.0.0:4317 --traces 100 --workers 10
```

To try sending data to different indexes, add a resource attribute with the `--otlp-attributes` parameter.

Example to send to the `foo` metric index:
```shell
$> telemetrygen metrics --otlp-insecure --otlp-endpoint 0.0.0.0:4317 --metrics 100 --workers 10 --otlp-attributes com.splunk.index=\"foo\"
```

# Binary File Declaration
The following binaries are written in Go present in the source repository:
- linux_x86_64/bin/splunk-connect-for-otlp
- windows_x86_64/bin/splunk-connect-for-otlp

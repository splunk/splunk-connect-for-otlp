# Splunk HEC Example

This example showcases how the collector can collect data from files and send it to Splunk Enterprise.

The example runs as a Docker Compose deployment. The collector can be configured to send logs to Splunk Enterprise.

Splunk is configured to receive data from the OpenTelemetry Collector using the Splunk Connect for OTLP technical addon.

To deploy the example, check out this git repository, open a terminal and in this directory type:
```bash
$> docker-compose up
```

Splunk will become available on port 18000. You can login on [http://localhost:18000](http://localhost:18000) with `admin` and `changeme`.

Once logged in, install the Splunk Connect for OTLP application by clicking on `Apps` > `Manage Apps` > `Install app from file` 
and selecting the technical addon archive from the last release. 

Once installed, follow the steps under `Usage` in [../README.md](the README.md) file to configure the new technical addon.

You can then visit the [search application](http://localhost:18000/en-US/app/search) to see the logs collected by Splunk.

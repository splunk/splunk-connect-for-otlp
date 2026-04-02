// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInput(t *testing.T) {
	input := `
<?xml version="1.0" encoding="UTF-8"?>
<input>
  <server_host>773c28971b2a</server_host>
  <server_uri>https://127.0.0.1:8089</server_uri>
  <session_key>OwLHq7jpfgz0WLe5t8KwZuxT4QZRggryMB2io6Phimb2zi5ErifFvx0Eu8WTmfviO^KUKEA8CsGbVltVlCDlYOBM0RE8QoOjOHZhKnHsphk20XoqaK1KXTZj1N</session_key>
  <checkpoint_dir>/opt/splunk/var/lib/splunk/modinputs/splunk-connect-for-otlp</checkpoint_dir>
  <configuration>
    <stanza name="splunk-connect-for-otlp://specialmind" app="search">
      <param name="grpc_port">4317</param>
      <param name="host">$decideOnStartup</param>
      <param name="http_port">4318</param>
      <param name="listen_address">0.0.0.0</param>
      <param name="sourcetype">_splunk-connect-for-otlp</param>
      <param name="enableSSL">1</param>
      <param name="serverCert">/var/certs/server.cert</param>
      <param name="serverKey">/var/certs/server.key</param>
      <param name="start_by_shell">false</param>
    </stanza>
  </configuration>
</input>`

	var config XMLInput
	err := xml.Unmarshal([]byte(input), &config)
	require.NoError(t, err)

	require.Equal(t, "splunk-connect-for-otlp://specialmind", config.Configuration.Stanza.Name)

	grpcPort, httpPort, listeningAddress, source, sourcetype, serverURI, sessionKey, enableSSL, serverCert, serverKey := config.Extract()

	require.Equal(t, 4317, grpcPort)
	require.Equal(t, "0.0.0.0", listeningAddress)
	require.Equal(t, 4318, httpPort)
	require.Equal(t, "", source)
	require.Equal(t, "_splunk-connect-for-otlp", sourcetype)
	require.Equal(t, "https://127.0.0.1:8089", serverURI)
	require.Equal(t, "OwLHq7jpfgz0WLe5t8KwZuxT4QZRggryMB2io6Phimb2zi5ErifFvx0Eu8WTmfviO^KUKEA8CsGbVltVlCDlYOBM0RE8QoOjOHZhKnHsphk20XoqaK1KXTZj1N", sessionKey)
	require.True(t, enableSSL)
	require.Equal(t, "/var/certs/server.cert", serverCert)
	require.Equal(t, "/var/certs/server.key", serverKey)
}

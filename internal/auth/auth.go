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

package auth

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/extension"
)

func New(ctx context.Context, settings component.TelemetrySettings, serverURI, sessionKey string) (extension.Extension, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/servicesNS/-/-/data/inputs/http", serverURI), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Splunk %s", sessionKey))
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	// Configure the TLS settings to skip certificate verification
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	// Create a new http client with the custom transport
	httpClient := &http.Client{Transport: customTransport}
	defer httpClient.CloseIdleConnections()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	f := feed{}
	err = xml.Unmarshal(body, &f)
	if err != nil {
		return nil, err
	}

	tokens := make([]configopaque.String, len(f.Entry))
	for i, entry := range f.Entry {
		for _, key := range entry.Content.Dict.Key {
			if key.Name == "token" {
				tokens[i] = configopaque.String(key.Text)
			}
		}
	}

	btae := bearertokenauthextension.NewFactory()
	authConfig := btae.CreateDefaultConfig().(*bearertokenauthextension.Config)
	authConfig.Tokens = tokens
	authConfig.Scheme = "Splunk"

	return btae.Create(ctx, extension.Settings{
		ID:                component.MustNewID("bearertokenauth"),
		TelemetrySettings: settings,
	}, authConfig)
}

// feed is generated from the atom feed XML response of the Splunk instance, per
// https://help.splunk.com/en/splunk-enterprise/leverage-rest-apis/rest-api-user-manual/9.3/rest-api-user-manual/basic-concepts-about-the-splunk-platform-rest-api
type feed struct {
	XMLName    xml.Name `xml:"feed"`
	Text       string   `xml:",chardata"`
	Xmlns      string   `xml:"xmlns,attr"`
	S          string   `xml:"s,attr"`
	Opensearch string   `xml:"opensearch,attr"`
	Title      string   `xml:"title"`
	ID         string   `xml:"id"`
	Updated    string   `xml:"updated"`
	Generator  struct {
		Text    string `xml:",chardata"`
		Build   string `xml:"build,attr"`
		Version string `xml:"version,attr"`
	} `xml:"generator"`
	Author struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
	} `xml:"author"`
	Link []struct {
		Text string `xml:",chardata"`
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
	TotalResults string `xml:"totalResults"`
	ItemsPerPage string `xml:"itemsPerPage"`
	StartIndex   string `xml:"startIndex"`
	Messages     string `xml:"messages"`
	Entry        []struct {
		Text    string `xml:",chardata"`
		Title   string `xml:"title"`
		ID      string `xml:"id"`
		Updated string `xml:"updated"`
		Link    []struct {
			Text string `xml:",chardata"`
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
		Author struct {
			Text string `xml:",chardata"`
			Name string `xml:"name"`
		} `xml:"author"`
		Content struct {
			Text string `xml:",chardata"`
			Type string `xml:"type,attr"`
			Dict struct {
				Text string `xml:",chardata"`
				Key  []struct {
					List struct {
						Text string `xml:",chardata"`
						Item string `xml:"item"`
					} `xml:"list"`
					Text string `xml:",chardata"`
					Name string `xml:"name,attr"`
					Dict struct {
						Text string `xml:",chardata"`
						Key  []struct {
							Text string `xml:",chardata"`
							Name string `xml:"name,attr"`
							Dict struct {
								Text string `xml:",chardata"`
								Key  []struct {
									Text string `xml:",chardata"`
									Name string `xml:"name,attr"`
									List struct {
										Text string `xml:",chardata"`
										Item string `xml:"item"`
									} `xml:"list"`
								} `xml:"key"`
							} `xml:"dict"`
						} `xml:"key"`
					} `xml:"dict"`
				} `xml:"key"`
			} `xml:"dict"`
		} `xml:"content"`
	} `xml:"entry"`
}

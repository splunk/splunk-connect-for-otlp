// Copyright Splunk Inc. 2025
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bufio"
	"encoding/xml"
	"os"
	"strconv"
)

const (
	DefaultGrpcPort      = 4317
	DefaultHTTPPort      = 4318
	DefaultListenAddress = "0.0.0.0"
)

type XMLInput struct {
	ServerURI     string    `xml:"server_uri"`
	SessionKey    string    `xml:"session_key"`
	Configuration XMLConfig `xml:"configuration"`
}

type XMLConfig struct {
	Stanza XMLStanza `xml:"stanza"`
}

type XMLStanza struct {
	Name   string     `xml:"name,attr"`
	App    string     `xml:"app,attr"`
	Params []XMLParam `xml:"param"`
}

type XMLParam struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",innerxml"`
}

func (x XMLInput) Extract() (int, int, string, string, string, string, string) {
	grpcPort := DefaultGrpcPort
	httpPort := DefaultHTTPPort
	listeningAddress := DefaultListenAddress
	source := ""
	sourcetype := ""

	serverURI := x.ServerURI
	sessionKey := x.SessionKey

	for _, p := range x.Configuration.Stanza.Params {
		switch p.Name {
		case "grpc_port":
			grpcPort, _ = strconv.Atoi(p.Value)
		case "http_port":
			httpPort, _ = strconv.Atoi(p.Value)
		case "listen_address":
			listeningAddress = p.Value
		case "source":
			source = p.Value
		case "sourcetype":
			sourcetype = p.Value
		}
	}

	return grpcPort, httpPort, listeningAddress, source, sourcetype, serverURI, sessionKey
}

func ReadFromStdin() (XMLInput, error) {
	scanner := bufio.NewScanner(os.Stdin)
	text := ""
	for scanner.Scan() {
		text += scanner.Text()
	}

	var config XMLInput
	err := xml.Unmarshal([]byte(text), &config)
	return config, err
}

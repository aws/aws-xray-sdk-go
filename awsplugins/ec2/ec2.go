// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package ec2

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/internal/logger"
	"github.com/aws/aws-xray-sdk-go/internal/plugins"
)

// Origin is the type of AWS resource that runs your application.
const Origin = "AWS::EC2::Instance"

type metadata struct {
	AvailabilityZone string
	ImageID          string
	InstanceID       string
	InstanceType     string
}

//Init activates EC2Plugin at runtime.
func Init() {
	if plugins.InstancePluginMetadata != nil && plugins.InstancePluginMetadata.EC2Metadata == nil {
		addPluginMetadata(plugins.InstancePluginMetadata)
	}
}

func addPluginMetadata(pluginmd *plugins.PluginMetadata) {
	var instanceData metadata
	imdsURL := "http://169.254.169.254/latest/"

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	token, err := getToken(imdsURL, client)
	if err != nil {
		logger.Debugf("Unable to fetch EC2 instance metadata token fallback to IMDS V1: %v", err)
	}

	resp, err := getMetadata(imdsURL, client, token)
	if err != nil {
		logger.Errorf("Unable to read EC2 instance metadata: %v", err)
		return
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		logger.Errorf("Error while reading data from response buffer: %v", err)
		return
	}
	metadata := buf.String()

	if err := json.Unmarshal([]byte(metadata), &instanceData); err != nil {
		logger.Errorf("Error while unmarshal operation: %v", err)
		return
	}

	pluginmd.EC2Metadata = &plugins.EC2Metadata{InstanceID: instanceData.InstanceID, AvailabilityZone: instanceData.AvailabilityZone}
	pluginmd.Origin = Origin
}

// getToken fetches token to fetch EC2 metadata
func getToken(imdsURL string, client *http.Client) (string, error) {
	ttlHeader := "X-aws-ec2-metadata-token-ttl-seconds"
	defaultTTL := "60"
	tokenURL := imdsURL + "api/token"

	req, _ := http.NewRequest("PUT", tokenURL, nil)
	req.Header.Add(ttlHeader, defaultTTL)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		logger.Errorf("Error while reading data from response buffer: %v", err)
		return "", err
	}
	token := buf.String()

	return token, err
}

// getMetadata fetches instance metadata
func getMetadata(imdsURL string, client *http.Client, token string) (*http.Response, error) {
	var metadataHeader string
	metadataURL := imdsURL + "dynamic/instance-identity/document"

	req, _ := http.NewRequest("GET", metadataURL, nil)
	if token != "" {
		metadataHeader = "X-aws-ec2-metadata-token"
		req.Header.Add(metadataHeader, token)
	}

	return client.Do(req)
}

// Metdata represents IMDS response.
//
// Deprecated: Metdata exists only for backward compatibility.
type Metdata struct {
	AvailabilityZone string
	ImageID          string
	InstanceID       string
	InstanceType     string
}

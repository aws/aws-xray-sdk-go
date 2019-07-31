// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	"github.com/aws/aws-xray-sdk-go/internal/logger"
	"github.com/aws/aws-xray-sdk-go/internal/plugins"
)

const Origin = "AWS::EC2::Instance"

func Init() {
	if plugins.InstancePluginMetadata != nil && plugins.InstancePluginMetadata.EC2Metadata == nil {
		addPluginMetadata(plugins.InstancePluginMetadata)
	}
}

func addPluginMetadata(pluginmd *plugins.PluginMetadata) {
	client := ec2metadata.New(defaults.Config())
	doc, err := client.GetInstanceIdentityDocument()
	if err != nil {
		logger.Errorf("Unable to read EC2 instance metadata: %v", err)
		return
	}

	pluginmd.EC2Metadata = &plugins.EC2Metadata{InstanceID: doc.InstanceID, AvailabilityZone: doc.AvailabilityZone}
	pluginmd.Origin = Origin
}

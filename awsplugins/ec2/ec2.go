// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package ec2

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-xray-sdk-go/internal/logger"
	"github.com/aws/aws-xray-sdk-go/internal/plugins"
)

// Origin is the type of AWS resource that runs your application.
const Origin = "AWS::EC2::Instance"

// Init activates EC2Plugin at runtime.
func Init() {
	if plugins.InstancePluginMetadata != nil && plugins.InstancePluginMetadata.EC2Metadata == nil {
		addPluginMetadata(plugins.InstancePluginMetadata)
	}
}

func addPluginMetadata(pluginmd *plugins.PluginMetadata) {
	session, e := session.NewSession()
	if e != nil {
		logger.Errorf("Unable to create a new ec2 session: %v", e)
		return
	}
	client := ec2metadata.New(session)
	doc, err := client.GetInstanceIdentityDocument()
	if err != nil {
		logger.Errorf("Unable to read EC2 instance metadata: %v", err)
		return
	}

	pluginmd.EC2Metadata = &plugins.EC2Metadata{InstanceID: doc.InstanceID, AvailabilityZone: doc.AvailabilityZone}
	pluginmd.Origin = Origin
}

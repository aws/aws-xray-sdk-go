// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package ecs

import (
	"os"

	"github.com/aws/aws-xray-sdk-go/internal/logger"
	"github.com/aws/aws-xray-sdk-go/internal/plugins"
)

// Origin is the type of AWS resource that runs your application.
const Origin = "AWS::ECS::Container"

// Init activates ECSPlugin at runtime.
func Init() {
	if plugins.InstancePluginMetadata != nil && plugins.InstancePluginMetadata.ECSMetadata == nil {
		addPluginMetadata(plugins.InstancePluginMetadata)
	}
}

func addPluginMetadata(pluginmd *plugins.PluginMetadata) {
	hostname, err := os.Hostname()

	if err != nil {
		logger.Errorf("Unable to retrieve hostname from OS. %v", err)
		return
	}

	pluginmd.ECSMetadata = &plugins.ECSMetadata{ContainerName: hostname}
	pluginmd.Origin = Origin
}

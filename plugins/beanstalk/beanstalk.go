// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package beanstalk

import (
	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-xray-sdk-go/internal/plugins"
	log "github.com/cihub/seelog"
)

const Origin = "AWS::ElasticBeanstalk::Environment"

// Init allows applications running inside Beanstalk to add
// their necessary metadata to the traces they provide
func Init() {
	if plugins.InstancePluginMetadata != nil && plugins.InstancePluginMetadata.BeanstalkMetadata == nil {
		addPluginMetadata(plugins.InstancePluginMetadata)
	}
}

func addPluginMetadata(pluginmd *plugins.PluginMetadata) {
	ebConfigPath := "/var/elasticbeanstalk/xray/environment.conf"

	rawConfig, err := ioutil.ReadFile(ebConfigPath)
	if err != nil {
		log.Errorf("Unable to read Elastic Beanstalk configuration file %s: %v", ebConfigPath, err)
		return
	}

	config := &plugins.BeanstalkMetadata{}
	err = json.Unmarshal(rawConfig, config)
	if err != nil {
		log.Errorf("Unable to unmarshal Elastic Beanstalk configuration file %s: %v", ebConfigPath, err)
		return
	}

	pluginmd.BeanstalkMetadata = config
	pluginmd.Origin = Origin
}

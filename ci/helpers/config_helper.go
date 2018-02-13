package helpers

import (
	"encoding/json"
	"io/ioutil"

	"github.com/alphagov/paas-elasticache-broker/broker"
)

func WriteCustomConfig(
	config broker.Config,
	brokerName string,
	elastiCacheSubnetGroupName string,
	ec2SecurityGroupID string,
) (broker.Config, string, error) {
	newConfig := &config

	newConfig.BrokerName = brokerName
	newConfig.CacheSubnetGroupName = elastiCacheSubnetGroupName
	newConfig.VpcSecurityGroupIds = []string{ec2SecurityGroupID}

	if err := newConfig.Validate(); err != nil {
		return broker.Config{}, "", err
	}

	configFile, err := ioutil.TempFile("", "elasticache-broker")
	if err != nil {
		return broker.Config{}, "", err
	}

	configJSON, err := json.Marshal(newConfig)
	if err != nil {
		return broker.Config{}, "", err
	}

	if err = ioutil.WriteFile(configFile.Name(), configJSON, 0644); err != nil {
		return broker.Config{}, "", err
	}

	if err = configFile.Close(); err != nil {
		return broker.Config{}, "", err
	}

	return *newConfig, configFile.Name(), err
}

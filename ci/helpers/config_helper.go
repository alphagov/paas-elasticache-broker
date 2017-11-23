package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/satori/go.uuid"
)

func WriteCustomConfig(
	config broker.Config,
	elastiCacheSubnetGroupName string,
	ec2SecurityGroupID string,
) (broker.Config, string, error) {
	newConfig := &config
	newConfig.BrokerName = fmt.Sprintf("%s-%s",
		newConfig.BrokerName,
		uuid.NewV4().String(),
	)
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

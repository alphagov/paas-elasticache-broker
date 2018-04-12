package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/brokerapi"
)

var (
	ErrNoSuchPlan = errors.New("no plan found")
)

type BindParameters struct{}

type PlanConfig struct {
	InstanceType             string            `json:"instance_type"`
	ReplicasPerNodeGroup     int64             `json:"replicas_per_node_group"`
	ShardCount               int64             `json:"shard_count"`
	SnapshotRetentionLimit   int64             `json:"snapshot_retention_limit"`
	AutomaticFailoverEnabled bool              `json:"automatic_failover_enabled"`
	Parameters               map[string]string `json:"parameters"`
}

type Config struct {
	LogLevel             string                    `json:"log_level"`
	Username             string                    `json:"username"`
	Password             string                    `json:"password"`
	Region               string                    `json:"region"`
	BrokerName           string                    `json:"broker_name"`
	AuthTokenSeed        string                    `json:"auth_token_seed"`
	CacheSubnetGroupName string                    `json:"cache_subnet_group_name"`
	VpcSecurityGroupIds  []string                  `json:"vpc_security_group_ids"`
	Catalog              brokerapi.CatalogResponse `json:"catalog"`
	PlanConfigs          map[string]PlanConfig     `json:"plan_configs"`
	KmsKeyID             string                    `json:"kms_key_id"`
}

func (c Config) GetPlanConfig(planID string) (PlanConfig, error) {
	plan, ok := c.PlanConfigs[planID]
	if !ok {
		return PlanConfig{}, ErrNoSuchPlan
	}
	return plan, nil
}

func LoadConfig(configFile string) (config Config, err error) {
	if configFile == "" {
		return config, errors.New("Must provide a config file")
	}

	file, err := os.Open(configFile)
	if err != nil {
		return config, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return config, err
	}

	if err = json.Unmarshal(bytes, &config); err != nil {
		return config, err
	}

	if err = config.Validate(); err != nil {
		return config, fmt.Errorf("Validating config contents: %s", err)
	}

	return config, nil
}

func (c Config) Validate() error {
	if c.LogLevel == "" {
		return errors.New("Must provide a non-empty log_level")
	}

	if c.Username == "" {
		return errors.New("Must provide a non-empty username")
	}

	if c.Password == "" {
		return errors.New("Must provide a non-empty password")
	}

	if c.Region == "" {
		return errors.New("Must provide a non-empty region")
	}

	if c.BrokerName == "" {
		return errors.New("Must provide a non-empty broker_name")
	}

	if c.AuthTokenSeed == "" {
		return errors.New("Must provide a non-empty auth_token_seed")
	}

	if c.CacheSubnetGroupName == "" {
		return errors.New("Must provide a cache_subnet_group_name")
	}

	if len(c.VpcSecurityGroupIds) < 1 {
		return errors.New("Must provide at least one VPC security group ID")
	}

	for _, s := range c.Catalog.Services {
		for _, p := range s.Plans {
			if !c.hasPlanConfig(p.ID) {
				return fmt.Errorf("Plan with ID %s has no PlanConfig", p.ID)
			}
		}
	}

	for k := range c.PlanConfigs {
		if !c.hasPlan(k) {
			return fmt.Errorf("PlanConfig %v not found in catalog", k)
		}
	}

	return nil
}

func (c Config) hasPlanConfig(id string) bool {
	_, ok := c.PlanConfigs[id]
	return ok
}

func (c Config) hasPlan(id string) bool {
	for _, s := range c.Catalog.Services {
		for _, p := range s.Plans {
			if p.ID == id {
				return true
			}
		}
	}
	return false
}

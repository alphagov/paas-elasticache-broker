package broker

import (
	"encoding/json"
	"fmt"
)

const ParamRestoreLatestSnapshotOf = "restore_from_latest_snapshot_of"
const ParamMaxMemoryPolicy = "maxmemory_policy"
const ParamPreferredMaintenanceWindow = "preferred_maintenance_window"

func parseProvisionParameters(data []byte) (*ProvisionParameters, error) {
	params := &ProvisionParameters{}
	err := unmarshalParameters(data, params, []string{
		ParamRestoreLatestSnapshotOf,
		ParamMaxMemoryPolicy,
		ParamPreferredMaintenanceWindow,
	})
	if err != nil {
		return nil, err
	}
	return params, nil
}

func parseUpdateParameters(data []byte) (*UpdateParameters, error) {
	params := &UpdateParameters{}
	err := unmarshalParameters(data, params, []string{
		ParamMaxMemoryPolicy,
		ParamPreferredMaintenanceWindow,
	})
	if err != nil {
		return nil, err
	}
	return params, nil
}

func unmarshalParameters(data []byte, out interface{}, validKeys []string) error {
	mapParams := map[string]interface{}{}
	err := json.Unmarshal(data, &mapParams)
	if err != nil {
		return err
	}
	for key := range mapParams {
		valid := false
		for _, validKey := range validKeys {
			if validKey == key {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unknown parameter: %s", key)
		}
	}
	return json.Unmarshal(data, out)
}

type ProvisionParameters struct {
	RestoreFromLatestSnapshotOf *string `json:"restore_from_latest_snapshot_of"`
	MaxMemoryPolicy             *string `json:"maxmemory_policy"`
	PreferredMaintenanceWindow  string  `json:"preferred_maintenance_window"`
}

type UpdateParameters struct {
	MaxMemoryPolicy            *string `json:"maxmemory_policy"`
	PreferredMaintenanceWindow string  `json:"preferred_maintenance_window"`
}

package broker

type CfProvisionParameters struct {
	RestoreFromLatestSnapshotOf *string `json:"restore_from_latest_snapshot_of"`
}

func (pp *CfProvisionParameters) Validate() error {
	return nil
}

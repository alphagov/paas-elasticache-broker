package providers

import (
	"context"
	"time"

	"github.com/pivotal-cf/brokerapi/domain"
)

// ServiceState is the state of a service instance
type ServiceState string

// Service states
const (
	Creating     ServiceState = "creating"
	Available    ServiceState = "available"
	Modifying    ServiceState = "modifying"
	Deleting     ServiceState = "deleting"
	CreateFailed ServiceState = "create-failed"
	Snapshotting ServiceState = "snapshotting"
	NonExisting  ServiceState = "non-existing"
)

type ProvisionParameters struct {
	InstanceType               string
	CacheParameterGroupFamily  string
	SecurityGroupIds           []string
	CacheSubnetGroupName       string
	PreferredMaintenanceWindow string
	ReplicasPerNodeGroup       int64
	ShardCount                 int64
	SnapshotRetentionLimit     int64
	RestoreFromSnapshot        *string
	AutomaticFailoverEnabled   bool
	MultiAZEnabled             bool
	Description                string
	Parameters                 map[string]string
	Tags                       map[string]string
	Engine                     string
	EngineVersion              string
}

type DeprovisionParameters struct {
	FinalSnapshotIdentifier string
}

type UpdateReplicationGroupParameters struct {
	PreferredMaintenanceWindow string
}

type UpdateParamGroupParameters struct {
	Parameters map[string]string
}

type SnapshotInfo struct {
	Name       string
	CreateTime time.Time
	Tags       map[string]string
}

type CacheParameter struct {
	ParameterName  string `json:"parameter_name"`
	ParameterValue string `json:"parameter_value"`
}
type InstanceParameters struct {
	PreferredMaintenanceWindow string           `json:"preferred_maintenance_window"`
	DailyBackupWindow          string           `json:"daily_backup_window"`
	MaxMemoryPolicy            string           `json:"maxmemory_policy"`
	CacheParameters            []CacheParameter `json:"cache_parameters"`
	ActiveNodes                []string         `json:"active_nodes"`
	PassiveNodes               []string         `json:"passive_nodes"`
}

type InstanceDetails struct {
	ServiceID    string             `json:"service_id"`
	PlanID       string             `json:"plan_id"`
	DashboardURL string             `json:"dashboard_url"`
	Parameters   InstanceParameters `json:"parameters"`
	domain.GetInstanceDetailsSpec
}

// Provider is a general interface to implement the broker's functionality with a specific provider
//
//counterfeiter:generate -o mocks/provider.go . Provider
type Provider interface {
	Provision(ctx context.Context, instanceID string, params ProvisionParameters) error
	UpdateReplicationGroup(ctx context.Context, instanceID string, params UpdateReplicationGroupParameters) error
	UpdateParamGroupParameters(ctx context.Context, instanceID string, params UpdateParamGroupParameters) error
	Deprovision(ctx context.Context, instanceID string, params DeprovisionParameters) error
	GetState(ctx context.Context, instanceID string) (ServiceState, string, error)
	GetInstanceParameters(ctx context.Context, instanceID string) (InstanceParameters, error)
	GetInstanceTags(ctx context.Context, instanceID string) (map[string]string, error)
	GenerateCredentials(ctx context.Context, instanceID, bindingID string) (*Credentials, error)
	RevokeCredentials(ctx context.Context, instanceID, bindingID string) error
	DeleteCacheParameterGroup(ctx context.Context, instanceID string) error
	FindSnapshots(ctx context.Context, instanceID string) ([]SnapshotInfo, error)
	ForceFailover(ctx context.Context, instanceID string) error
	AutoFailover(ctx context.Context, instanceID string, enable bool) error
}

// Credentials are the connection parameters for Redis clients
type Credentials struct {
	Host       string `json:"host"`
	Port       int64  `json:"port"`
	Name       string `json:"name"`
	Password   string `json:"password"`
	URI        string `json:"uri"`
	TLSEnabled bool   `json:"tls_enabled"`
}

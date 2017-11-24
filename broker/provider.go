package broker

import "context"

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

// Provider is a general interface to implement the broker's functionality with a specific provider
//
//go:generate counterfeiter -o mocks/provider.go . Provider
type Provider interface {
	Provision(ctx context.Context, instanceID string, params ProvisionParameters) error
	Deprovision(ctx context.Context, instanceID string, params DeprovisionParameters) error
	GetState(ctx context.Context, instanceID string) (ServiceState, string, error)
	GenerateCredentials(ctx context.Context, instanceID, bindingID string) (*Credentials, error)
	RevokeCredentials(ctx context.Context, instanceID, bindingID string) error
}

// Credentials are the connection parameters for Redis clients
type Credentials struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	Name       string `json:"name"`
	Password   string `json:"password"`
	URI        string `json:"uri"`
	TLSEnabled string `json:"tls_enabled"`
}

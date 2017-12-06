package broker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
)

// Broker is the open service broker API implementation for AWS Elasticache Redis
type Broker struct {
	config   Config
	provider Provider
	logger   lager.Logger
}

// New creates a new broker instance
func New(config Config, provider Provider, logger lager.Logger) *Broker {
	return &Broker{
		config:   config,
		provider: provider,
		logger:   logger,
	}
}

// Services returns with the provided services
func (b *Broker) Services(ctx context.Context) []brokerapi.Service {
	return b.config.Catalog.Services
}

// Provision creates a new ElastiCache replication group
func (b *Broker) Provision(ctx context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	b.logger.Debug("provision-start", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	planConfig, err := b.config.GetPlanConfig(details.PlanID)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("service plan %s: %s", details.PlanID, err)
	}

	provisionParams := ProvisionParameters{
		InstanceType:               planConfig.InstanceType,
		CacheParameterGroupName:    "default.redis3.2",
		SecurityGroupIds:           b.config.VpcSecurityGroupIds,
		CacheSubnetGroupName:       b.config.CacheSubnetGroupName,
		PreferredMaintenanceWindow: "sun:23:00-mon:01:30",
		ReplicasPerNodeGroup:       0,
		ShardCount:                 1,
		SnapshotRetentionLimit:     0,
		Description:                "Cloud Foundry service",
		AutomaticFailoverEnabled:   false,
		Parameters:                 planConfig.Parameters,
		Tags: map[string]string{
			"created-by":      b.config.BrokerName,
			"service-id":      details.ServiceID,
			"plan-id":         details.PlanID,
			"organization-id": details.OrganizationGUID,
			"space-id":        details.SpaceGUID,
		},
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	err = b.provider.Provision(providerCtx, instanceID, provisionParams)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("provider %s for plan %s: %s", "redis", details.PlanID, err)
	}

	b.logger.Debug("provision-success", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	return brokerapi.ProvisionedServiceSpec{IsAsync: true}, nil
}

// Update modifies an existing service instance
func (b *Broker) Update(ctx context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	b.logger.Debug("update", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.UpdateServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	return brokerapi.UpdateServiceSpec{IsAsync: true}, errors.New("notimp")
}

// Deprovision deletes a service instance
func (b *Broker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	b.logger.Debug("deprovision-start", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	err := b.provider.Deprovision(providerCtx, instanceID, DeprovisionParameters{})
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("provider %s for plan %s: %s", "redis", details.PlanID, err)
	}

	b.logger.Debug("deprovision-success", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	return brokerapi.DeprovisionServiceSpec{IsAsync: true}, nil
}

// Bind binds an application and a service instance
func (b *Broker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	b.logger.Debug("bind", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	credentials, err := b.provider.GenerateCredentials(ctx, instanceID, bindingID)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	return brokerapi.Binding{
		Credentials: credentials,
	}, nil
}

// Unbind removes the binding between an application and a service instance
func (b *Broker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	b.logger.Debug("unbind", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	return b.provider.RevokeCredentials(ctx, instanceID, bindingID)
}

// LastOperation returns with the last known state of the given service instance
func (b *Broker) LastOperation(ctx context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	b.logger.Debug("last-operation", lager.Data{
		"instance-id":    instanceID,
		"operation-data": operationData,
	})

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	state, stateDescription, err := b.provider.GetState(providerCtx, instanceID)
	if err != nil {
		return brokerapi.LastOperation{}, fmt.Errorf("error getting state for %s: %s", instanceID, err)
	}

	if state == NonExisting {
		return brokerapi.LastOperation{}, brokerapi.ErrInstanceDoesNotExist
	}

	lastOperationState, err := ProviderStatesMapping(state)
	if err != nil {
		b.logger.Error("last-operation", err, lager.Data{
			"instance-id": instanceID,
		})
	}

	return brokerapi.LastOperation{
		State:       lastOperationState,
		Description: stateDescription,
	}, nil
}

func ProviderStatesMapping(state ServiceState) (brokerapi.LastOperationState, error) {
	switch state {
	case Available:
		return brokerapi.Succeeded, nil
	case CreateFailed:
		return brokerapi.Failed, nil
	case Creating:
		fallthrough
	case Modifying:
		fallthrough
	case Deleting:
		fallthrough
	case Snapshotting:
		return brokerapi.InProgress, nil
	}
	return brokerapi.InProgress, fmt.Errorf("Unknown service state: %s", state)
}

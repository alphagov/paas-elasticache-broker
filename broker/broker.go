package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"

	"github.com/alphagov/paas-elasticache-broker/providers"
)

// Broker is the open service broker API implementation for AWS Elasticache Redis
type Broker struct {
	config   Config
	provider providers.Provider
	logger   lager.Logger
}

type action = string

// Operation is the operation data passed back by the provision/deprovision/update calls and received by the last
// operation call
type Operation struct {
	Action      action `json:"action"`
	PrimaryNode string `json:"primaryNode"`
	TimeOut     string `json:"timeOut"`
}

func (o Operation) String() string {
	b, _ := json.Marshal(o)
	return string(b)
}

// New creates a new broker instance
func New(config Config, provider providers.Provider, logger lager.Logger) *Broker {
	return &Broker{
		config:   config,
		provider: provider,
		logger:   logger,
	}
}

// Possible actions in the operation data
const (
	ActionProvisioning   action = "provisioning"
	ActionDeprovisioning action = "deprovisioning"
	ActionUpdating       action = "updating"
	ActionFailover       action = "failover"
	FailoverTimeout             = 45 * time.Minute
)

// Sort providers.SnapshotInfo
type ByCreateTime []providers.SnapshotInfo

func (ct ByCreateTime) Len() int           { return len(ct) }
func (ct ByCreateTime) Swap(i, j int)      { ct[i], ct[j] = ct[j], ct[i] }
func (ct ByCreateTime) Less(i, j int) bool { return ct[i].CreateTime.After(ct[j].CreateTime) }

func (b *Broker) GetBinding(ctx context.Context, first, second string) (brokerapi.GetBindingSpec, error) {
	return brokerapi.GetBindingSpec{}, fmt.Errorf("GetBinding method not implemented")
}

func (b *Broker) GetInstance(ctx context.Context, instanceID string) (brokerapi.GetInstanceDetailsSpec, error) {
	instanceParameters, err := b.provider.GetInstanceParameters(ctx, instanceID)
	if err != nil {
		return brokerapi.GetInstanceDetailsSpec{}, err
	}
	instanceTags, err := b.provider.GetInstanceTags(ctx, instanceID)
	if err != nil {
		return brokerapi.GetInstanceDetailsSpec{}, err
	}
	return brokerapi.GetInstanceDetailsSpec{
		ServiceID:    instanceTags["service-id"],
		PlanID:       instanceTags["plan-id"],
		DashboardURL: "",
		Parameters:   instanceParameters,
	}, nil
}

func (b *Broker) LastBindingOperation(ctx context.Context, first, second string, pollDetails brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, fmt.Errorf("LastBindingOperation method not implemented")
}

// Services returns with the provided services
func (b *Broker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return b.config.Catalog.Services, nil
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

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	userParameters := &ProvisionParameters{}
	if len(details.RawParameters) > 0 {
		var err error
		userParameters, err = parseProvisionParameters(details.RawParameters)
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}

	// TODO: parsing the user provided parameters should be done in the provider and not in the broker
	var restoreFromSnapshotName *string
	if userParameters.RestoreFromLatestSnapshotOf != nil {
		snapshots, err := b.provider.FindSnapshots(providerCtx, *userParameters.RestoreFromLatestSnapshotOf)
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
		if len(snapshots) == 0 {
			return brokerapi.ProvisionedServiceSpec{},
				fmt.Errorf("No snapshots found for: %s", *userParameters.RestoreFromLatestSnapshotOf)
		}
		sort.Sort(ByCreateTime(snapshots))
		latestSnapshot := snapshots[0]

		if snapshotSpaceId, ok := latestSnapshot.Tags["space-id"]; !ok || snapshotSpaceId != details.SpaceGUID {
			return brokerapi.ProvisionedServiceSpec{},
				fmt.Errorf("The service instance you are getting a snapshot from is not in the same org or space")
		}
		if snapshotOrgId, ok := latestSnapshot.Tags["organization-id"]; !ok || snapshotOrgId != details.OrganizationGUID {
			return brokerapi.ProvisionedServiceSpec{},
				fmt.Errorf("The service instance you are getting a snapshot from is not in the same org or space")
		}
		if snapshotPlanId, ok := latestSnapshot.Tags["plan-id"]; !ok || snapshotPlanId != details.PlanID {
			return brokerapi.ProvisionedServiceSpec{},
				fmt.Errorf("You must use the same plan as the service instance you are getting a snapshot from")
		}

		restoreFromSnapshotName = &snapshots[0].Name

	}

	params := make(map[string]string, len(planConfig.Parameters))
	for k, v := range planConfig.Parameters {
		params[k] = v
	}
	if userParameters.MaxMemoryPolicy != nil {
		params["maxmemory-policy"] = *userParameters.MaxMemoryPolicy
	}

	provisionParams := providers.ProvisionParameters{
		InstanceType:               planConfig.InstanceType,
		CacheParameterGroupFamily:  planConfig.CacheParameterGroupFamily,
		SecurityGroupIds:           b.config.VpcSecurityGroupIds,
		CacheSubnetGroupName:       b.config.CacheSubnetGroupName,
		PreferredMaintenanceWindow: userParameters.PreferredMaintenanceWindow,
		ReplicasPerNodeGroup:       planConfig.ReplicasPerNodeGroup,
		ShardCount:                 planConfig.ShardCount,
		SnapshotRetentionLimit:     planConfig.SnapshotRetentionLimit,
		RestoreFromSnapshot:        restoreFromSnapshotName,
		AutomaticFailoverEnabled:   planConfig.AutomaticFailoverEnabled,
		MultiAZEnabled:             planConfig.MultiAZEnabled,

		Description: "Cloud Foundry service",
		Parameters:  params,
		Tags: map[string]string{
			"created-by":        b.config.BrokerName,
			"service-id":        details.ServiceID,
			"plan-id":           details.PlanID,
			"organization-id":   details.OrganizationGUID,
			"space-id":          details.SpaceGUID,
			"instance-id":       instanceID,
			"chargeable_entity": instanceID, // 'chargeable_entity' is the configured cost allocation tag. It's supposed to be snake_case.
		},
		Engine:        planConfig.Engine,
		EngineVersion: planConfig.EngineVersion,
	}

	err = b.provider.Provision(providerCtx, instanceID, provisionParams)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("provider %s for plan %s: %s", "redis", details.PlanID, err)
	}

	b.logger.Debug("provision-success", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})
	return brokerapi.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: Operation{Action: ActionProvisioning}.String(),
	}, nil
}

// Update modifies an existing service instance.
// It can be used to update the maintenance window and / or the maxmemory policy.
// As this is a synchronous operation, if updating the maintenance window fails
// the whole operation will fail (ie. it won't try to be smart and carry on)
func (b *Broker) Update(ctx context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	b.logger.Debug("update", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.UpdateServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	if details.PlanID != details.PreviousValues.PlanID {
		return brokerapi.UpdateServiceSpec{}, fmt.Errorf("changing plans is not currently supported")
	}

	if details.ServiceID != details.PreviousValues.ServiceID {
		return brokerapi.UpdateServiceSpec{}, fmt.Errorf("changing plans is not currently supported")
	}

	userParameters := &UpdateParameters{}
	if len(details.RawParameters) > 0 {
		var err error
		userParameters, err = parseUpdateParameters(details.RawParameters)
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, err
		}
	}

	if checkIfNoUpdateParametersAreSet(userParameters) {
		return brokerapi.UpdateServiceSpec{}, fmt.Errorf("no parameters provided")
	}

	if userParameters.TestFailover != nil {

		planConfig, err := b.config.GetPlanConfig(details.PlanID)
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, errors.Wrap(err, "Failed to find service plan")
		}
		if planConfig.Parameters["cluster-enabled"] == "yes" {
			return brokerapi.UpdateServiceSpec{}, fmt.Errorf("Test failover is not supported for Redis instances in cluster mode")
		}
		if planConfig.MultiAZEnabled == false || planConfig.AutomaticFailoverEnabled == false {
			return brokerapi.UpdateServiceSpec{}, fmt.Errorf("Test failover is not supported without MultiAZEnabled and AutomaticFailoverEnabled")
		}
		if planConfig.ReplicasPerNodeGroup < 1 {
			return brokerapi.UpdateServiceSpec{}, fmt.Errorf("Test failover requires one or more replicas")
		}
		if userParameters.MaxMemoryPolicy != nil || userParameters.PreferredMaintenanceWindow != "" {
			return brokerapi.UpdateServiceSpec{}, fmt.Errorf("Test failover must be used by itself")
		}
		primaryNode, err := b.provider.StartFailoverTest(providerCtx, instanceID)
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, errors.Wrap(err, "Test failover failed: ")
		}
		return brokerapi.UpdateServiceSpec{
			IsAsync: true,
			OperationData: Operation{
				Action:      ActionFailover,
				PrimaryNode: primaryNode,
				TimeOut:     time.Now().Add(FailoverTimeout).Format(time.RFC3339),
			}.String(),
		}, nil
	}

	if userParameters.PreferredMaintenanceWindow != "" {
		err := b.provider.UpdateReplicationGroup(providerCtx, instanceID, providers.UpdateReplicationGroupParameters{
			PreferredMaintenanceWindow: userParameters.PreferredMaintenanceWindow,
		})
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, errors.Wrap(err, "Updating preferred maintenance window failed")
		}
		b.logger.Debug("update-replication-group-success", lager.Data{
			"instance-id":        instanceID,
			"details":            details,
			"accepts-incomplete": asyncAllowed,
		})
	}

	if userParameters.MaxMemoryPolicy != nil {
		params := map[string]string{}
		params["maxmemory-policy"] = *userParameters.MaxMemoryPolicy

		err := b.provider.UpdateParamGroupParameters(providerCtx, instanceID, providers.UpdateParamGroupParameters{
			Parameters: params,
		})
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, errors.Wrap(err, "Updating maxmemory policy failed")
		}
		b.logger.Debug("update-parameter-group-success", lager.Data{
			"instance-id":        instanceID,
			"details":            details,
			"accepts-incomplete": asyncAllowed,
		})
	}

	return brokerapi.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: Operation{Action: ActionUpdating}.String(),
	}, nil
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

	err := b.provider.Deprovision(providerCtx, instanceID, providers.DeprovisionParameters{})
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("provider %s for plan %s: %s", "redis", details.PlanID, err)
	}

	b.logger.Debug("deprovision-success", lager.Data{
		"instance-id":        instanceID,
		"details":            details,
		"accepts-incomplete": asyncAllowed,
	})

	return brokerapi.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: Operation{Action: ActionDeprovisioning}.String(),
	}, nil
}

// Bind binds an application and a service instance
func (b *Broker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
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
func (b *Broker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails, asyncAllowed bool) (brokerapi.UnbindSpec, error) {
	b.logger.Debug("unbind", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	return brokerapi.UnbindSpec{}, b.provider.RevokeCredentials(ctx, instanceID, bindingID)
}

// LastOperation returns with the last known state of the given service instance
func (b *Broker) LastOperation(ctx context.Context, instanceID string, pollDetails brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	b.logger.Debug("last-operation", lager.Data{
		"instance-id":    instanceID,
		"operation-data": pollDetails.OperationData,
	})

	// provider call to get operation?
	var operation Operation
	if pollDetails.OperationData != "" {
		err := json.Unmarshal([]byte(pollDetails.OperationData), &operation)
		if err != nil {
			return brokerapi.LastOperation{}, fmt.Errorf("invalid operation data: %s", pollDetails.OperationData)
		}
		if operation.Action == "" {
			return brokerapi.LastOperation{}, fmt.Errorf("invalid operation, action parameter is empty: %s", pollDetails.OperationData)
		}
	}

	if operation.TimeOut != "" {
		timeOutParsed, err := time.Parse(time.RFC3339, operation.TimeOut)
		if err != nil {
			return brokerapi.LastOperation{}, errors.Wrap(err, "Failed to parse time out string")
		}

		if time.Now().After(timeOutParsed) {
			return brokerapi.LastOperation{}, fmt.Errorf("Operation %s timed out for %s", operation.Action, instanceID)
		}
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	state, stateDescription, err := b.provider.ProgressState(providerCtx, instanceID, operation.Action, operation.PrimaryNode)
	if err != nil {
		return brokerapi.LastOperation{}, fmt.Errorf("error getting state for %s: %s", instanceID, err)
	}

	if state == providers.NonExisting {
		if operation.Action == ActionDeprovisioning {
			err = b.provider.DeleteCacheParameterGroup(providerCtx, instanceID)
			if err != nil {
				return brokerapi.LastOperation{}, fmt.Errorf("error deleting parameter group %s: %s", instanceID, err)
			}
		}
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

func ProviderStatesMapping(state providers.ServiceState) (brokerapi.LastOperationState, error) {
	switch state {
	case providers.Available:
		return brokerapi.Succeeded, nil
	case providers.CreateFailed:
		return brokerapi.Failed, nil
	case providers.Creating:
		fallthrough
	case providers.Modifying:
		fallthrough
	case providers.Deleting:
		fallthrough
	case providers.Snapshotting:
		return brokerapi.InProgress, nil
	}
	return brokerapi.InProgress, fmt.Errorf("Unknown service state: %s", state)
}

package redis

import (
	"context"
	"encoding/base32"
	"fmt"
	"hash/fnv"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

// Provider is the Redis broker provider
type Provider struct {
	aws    ElastiCache
	logger lager.Logger
}

// NewProvider creates a new Redis provider
func NewProvider(elasticache ElastiCache, logger lager.Logger) *Provider {
	return &Provider{
		aws:    elasticache,
		logger: logger,
	}
}

func (p *Provider) createCacheParameterGroup(ctx context.Context, instanceID string, params broker.ProvisionParameters) error {
	replicationGroupID := GenerateReplicationGroupName(instanceID)
	_, err := p.aws.CreateCacheParameterGroupWithContext(ctx, &elasticache.CreateCacheParameterGroupInput{
		CacheParameterGroupFamily: aws.String("redis3.2"),
		CacheParameterGroupName:   aws.String(replicationGroupID),
		Description:               aws.String("Created by Cloud Foundry"),
	})
	if err != nil {
		return err
	}

	pgParams := []*elasticache.ParameterNameValue{}
	for paramName, paramValue := range params.Parameters {
		pgParams = append(pgParams, &elasticache.ParameterNameValue{
			ParameterName:  aws.String(paramName),
			ParameterValue: aws.String(paramValue),
		})
	}

	_, err = p.aws.ModifyCacheParameterGroupWithContext(ctx, &elasticache.ModifyCacheParameterGroupInput{
		ParameterNameValues:     pgParams,
		CacheParameterGroupName: aws.String(replicationGroupID),
	})
	return err
}

// Provision creates a replication group and a cache parameter group
func (p *Provider) Provision(ctx context.Context, instanceID string, params broker.ProvisionParameters) error {
	replicationGroupID := GenerateReplicationGroupName(instanceID)

	err := p.createCacheParameterGroup(ctx, instanceID, params)
	if err != nil {
		return err
	}

	cacheParameterGroupName := replicationGroupID

	input := &elasticache.CreateReplicationGroupInput{
		Tags: []*elasticache.Tag{},
		AtRestEncryptionEnabled:     aws.Bool(true),
		TransitEncryptionEnabled:    aws.Bool(true),
		AuthToken:                   aws.String(params.AuthToken),
		AutomaticFailoverEnabled:    aws.Bool(params.AutomaticFailoverEnabled),
		CacheNodeType:               aws.String(params.InstanceType),
		CacheParameterGroupName:     aws.String(cacheParameterGroupName),
		SecurityGroupIds:            aws.StringSlice(params.SecurityGroupIds),
		CacheSubnetGroupName:        aws.String(params.CacheSubnetGroupName),
		Engine:                      aws.String("redis"),
		EngineVersion:               aws.String("3.2.6"),
		PreferredMaintenanceWindow:  aws.String(params.PreferredMaintenanceWindow),
		ReplicationGroupDescription: aws.String(params.Description),
		ReplicationGroupId:          aws.String(replicationGroupID),
		NumNodeGroups:               aws.Int64(params.ShardCount),
	}

	for tagName, tagValue := range params.Tags {
		input.Tags = append(input.Tags, &elasticache.Tag{
			Key:   aws.String(tagName),
			Value: aws.String(tagValue),
		})
	}

	if params.ShardCount == 1 {
		input.SetNumCacheClusters(params.ReplicasPerNodeGroup + 1)
	} else {
		input.SetReplicasPerNodeGroup(params.ReplicasPerNodeGroup)
	}

	if params.SnapshotRetentionLimit > 0 {
		input.SetSnapshotRetentionLimit(params.SnapshotRetentionLimit)
	}

	_, err = p.aws.CreateReplicationGroupWithContext(ctx, input)
	return err
}

// Deprovision deletes the replication group
func (p *Provider) Deprovision(ctx context.Context, instanceID string, params broker.DeprovisionParameters) error {
	replicationGroupID := GenerateReplicationGroupName(instanceID)

	input := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: aws.String(replicationGroupID),
	}
	if params.FinalSnapshotIdentifier != "" {
		input.SetFinalSnapshotIdentifier(params.FinalSnapshotIdentifier)
	}

	_, err := p.aws.DeleteReplicationGroupWithContext(ctx, input)
	return err
}

// GetState returns with the state of an existing cluster
// If the cluster doesn't exist we return with the broker.NonExisting state
func (p *Provider) GetState(ctx context.Context, instanceID string) (broker.ServiceState, string, error) {
	replicationGroupID := GenerateReplicationGroupName(instanceID)

	output, err := p.aws.DescribeReplicationGroupsWithContext(ctx, &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: aws.String(replicationGroupID),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == elasticache.ErrCodeReplicationGroupNotFoundFault {
				return broker.NonExisting, fmt.Sprintf("Replication group does not exist: %s", replicationGroupID), nil
			}
		}
		return broker.ServiceState(""), "", err
	}

	if len(output.ReplicationGroups) == 0 {
		return broker.ServiceState(""), "", fmt.Errorf("Invalid response from AWS: no cache clusters returned for %s", replicationGroupID)
	}

	if output.ReplicationGroups[0].Status == nil {
		return broker.ServiceState(""), "", fmt.Errorf("Invalid response from AWS: status is missing for %s", replicationGroupID)
	}

	message := fmt.Sprintf("ElastiCache state is %s for %s", *output.ReplicationGroups[0].Status, replicationGroupID)

	return broker.ServiceState(*output.ReplicationGroups[0].Status), message, nil
}

// GenerateReplicationGroupName generates a valid ElastiCache replication group name
// A valid name must contain between 1 and 20 alphanumeric characters or hyphens, should start with a letter, and cannot end with a hyphen or contain two consecutive hyphens.
func GenerateReplicationGroupName(instanceID string) string {
	hash := fnv.New64a()
	hash.Write([]byte(instanceID))
	out := hash.Sum([]byte{})
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	return strings.ToLower("cf-" + encoder.EncodeToString(out))
}

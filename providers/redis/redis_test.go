package redis_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/alphagov/paas-elasticache-broker/providers"
	"github.com/alphagov/paas-elasticache-broker/providers/mocks"
	. "github.com/alphagov/paas-elasticache-broker/providers/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	uuid "github.com/satori/go.uuid"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {
	var (
		mockElasticache    *mocks.FakeElastiCache
		mockSecretsManager *mocks.FakeSecretsManager
		ctx                context.Context
		instanceID         string
		replicationGroupID string
		secretsManagerPath string
		awsAccountID       string
		awsPartition       string
		awsRegion          string

		provider *RedisProvider
		kmsKeyID = "my-kms-key"
	)

	BeforeEach(func() {
		mockElasticache = &mocks.FakeElastiCache{}
		mockSecretsManager = &mocks.FakeSecretsManager{}
		ctx = context.Background()
		instanceID = "foobar"
		replicationGroupID = "cf-qwkec4pxhft6q"
		secretsManagerPath = "elasticache-broker-test"
		awsAccountID = "123456789012"
		awsPartition = "aws"
		awsRegion = "eu-west-1"
	})

	JustBeforeEach(func() {
		provider = NewProvider(
			mockElasticache,
			mockSecretsManager,
			awsAccountID,
			awsPartition,
			awsRegion,
			lager.NewLogger("logger"),
			kmsKeyID,
			secretsManagerPath,
		)
	})

	Context("GetPrimaryAndReplicaCacheClusterIds", func() {
		var (
			replicationGroup *elasticache.ReplicationGroup
		)

		BeforeEach(func() {
			// Set up dummy data for the replication group
			replicationGroup = &elasticache.ReplicationGroup{
				NodeGroups: []*elasticache.NodeGroup{
					{
						NodeGroupMembers: []*elasticache.NodeGroupMember{
							{
								CurrentRole:    aws.String("primary"),
								CacheClusterId: aws.String("primary-cluster-id"),
							},
							{
								CurrentRole:    aws.String("replica"),
								CacheClusterId: aws.String("replica-cluster-id"),
							},
						},
					},
				},
			}
		})

		It("returns the primary and replica cache cluster IDs", func() {
			primaryNode, replicaNode, err := GetPrimaryAndReplicaCacheClusterIds(replicationGroup)
			Expect(err).ToNot(HaveOccurred())
			Expect(primaryNode).To(Equal("primary-cluster-id"))
			Expect(replicaNode).To(Equal("replica-cluster-id"))
		})

		It("returns an error if the primary node cannot be determined", func() {
			// Modify the dummy data to not include a primary node
			replicationGroup.NodeGroups[0].NodeGroupMembers[0].CurrentRole = aws.String("replica")
			_, _, err := GetPrimaryAndReplicaCacheClusterIds(replicationGroup)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Unable to determine primary node"))
		})

		It("returns an error if the replica node cannot be determined", func() {
			// Modify the dummy data to not include a replica node
			replicationGroup.NodeGroups[0].NodeGroupMembers[1].CurrentRole = aws.String("primary")
			_, _, err := GetPrimaryAndReplicaCacheClusterIds(replicationGroup)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Unable to determine replica node"))
		})
	})

	Context("when provisioning", func() {
		var (
			provisionParams providers.ProvisionParameters
			provisionErr    error
		)

		BeforeEach(func() {
			provisionParams = providers.ProvisionParameters{
				InstanceType:               "test instance type",
				CacheParameterGroupFamily:  "redis4.0",
				SecurityGroupIds:           []string{"test sg1"},
				CacheSubnetGroupName:       "test subnet group",
				PreferredMaintenanceWindow: "test maintenance window",
				ReplicasPerNodeGroup:       0,
				ShardCount:                 1,
				SnapshotRetentionLimit:     7,
				AutomaticFailoverEnabled:   true,
				MultiAZEnabled:             true,
				Description:                "test desc",
				Parameters: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Tags:          map[string]string{},
				Engine:        "redis",
				EngineVersion: "4.0.10",
			}
		})

		JustBeforeEach(func() {
			provisionErr = provider.Provision(ctx, instanceID, provisionParams)
		})

		It("succeeds", func() {
			Expect(provisionErr).ToNot(HaveOccurred())
		})

		It("creates a cache parameter group and sets the parameters", func() {
			Expect(mockElasticache.CreateCacheParameterGroupWithContextCallCount()).To(Equal(1))
			receivedCtx, receivedInput, _ := mockElasticache.CreateCacheParameterGroupWithContextArgsForCall(0)
			Expect(receivedCtx).To(Equal(ctx))
			Expect(receivedInput).To(Equal(&elasticache.CreateCacheParameterGroupInput{
				CacheParameterGroupFamily: aws.String("redis4.0"),
				CacheParameterGroupName:   aws.String(replicationGroupID),
				Description:               aws.String("Created by Cloud Foundry"),
			}))

			Expect(mockElasticache.ModifyCacheParameterGroupWithContextCallCount()).To(Equal(1))
			receivedCtx, paramGroupInput, _ := mockElasticache.ModifyCacheParameterGroupWithContextArgsForCall(0)
			Expect(receivedCtx).To(Equal(ctx))
			Expect(paramGroupInput.CacheParameterGroupName).To(Equal(aws.String(replicationGroupID)))
			Expect(paramGroupInput.ParameterNameValues).To(ConsistOf([]*elasticache.ParameterNameValue{
				{
					ParameterName:  aws.String("cluster-enabled"),
					ParameterValue: aws.String("no"),
				},
				{
					ParameterName:  aws.String("key1"),
					ParameterValue: aws.String("value1"),
				},
				{
					ParameterName:  aws.String("key2"),
					ParameterValue: aws.String("value2"),
				},
			}))
		})

		Context("when cluster mode is enabled", func() {
			BeforeEach(func() {
				provisionParams.Parameters["cluster-enabled"] = "yes"
			})

			It("creates a cache parameter group with cluster mode enabled", func() {
				_, paramGroupInput, _ := mockElasticache.ModifyCacheParameterGroupWithContextArgsForCall(0)
				Expect(paramGroupInput.ParameterNameValues).To(ContainElement(&elasticache.ParameterNameValue{
					ParameterName:  aws.String("cluster-enabled"),
					ParameterValue: aws.String("yes"),
				}))
			})
		})

		Context("when creating a parameter group fails", func() {
			var createErr = errors.New("some error")

			BeforeEach(func() {
				mockElasticache.CreateCacheParameterGroupWithContextReturnsOnCall(0, nil, createErr)
			})

			It("returns the error", func() {
				Expect(provisionErr).To(MatchError(createErr))
				Expect(mockElasticache.ModifyCacheParameterGroupWithContextCallCount()).To(Equal(0))
				Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(0))
			})
		})

		Context("when modifying a parameter group fails", func() {
			var modifyErr = errors.New("some error")

			BeforeEach(func() {
				mockElasticache.ModifyCacheParameterGroupWithContextReturnsOnCall(0, nil, modifyErr)
			})

			It("returns the error", func() {
				Expect(provisionErr).To(MatchError(modifyErr))
				Expect(mockElasticache.CreateCacheParameterGroupWithContextCallCount()).To(Equal(1))
				Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(0))
			})
		})

		It("saves the auth token in the secrets manager", func() {
			Expect(mockSecretsManager.CreateSecretWithContextCallCount()).To(Equal(1))
			passedCtx, input, _ := mockSecretsManager.CreateSecretWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(input.Name).To(Equal(aws.String("elasticache-broker-test/foobar/auth-token")))
			Expect(*input.SecretString).ToNot(BeEmpty())
			Expect(input.KmsKeyId).To(Equal(aws.String("my-kms-key")))
		})

		It("tags the secret in the secrets manager with the cost allocation tag", func() {
			Expect(mockSecretsManager.CreateSecretWithContextCallCount()).To(Equal(1))
			_, input, _ := mockSecretsManager.CreateSecretWithContextArgsForCall(0)
			Expect(input.Tags).To(ContainElement(&secretsmanager.Tag{
				Key:   aws.String("chargeable_entity"),
				Value: aws.String(instanceID),
			}))
		})

		Context("when the secrets manager path contains a trailing /", func() {
			BeforeEach(func() {
				secretsManagerPath = "elasticache-broker-test/"
			})

			It("strips it from the path", func() {
				_, input, _ := mockSecretsManager.CreateSecretWithContextArgsForCall(0)
				Expect(input.Name).To(Equal(aws.String("elasticache-broker-test/foobar/auth-token")))
			})
		})

		Context("when creating the auth token in the secrets manager fails", func() {
			var createErr = errors.New("error in secrets manager")

			BeforeEach(func() {
				mockSecretsManager.CreateSecretWithContextReturnsOnCall(0, nil, createErr)
			})

			It("returns with an error", func() {
				Expect(provisionErr).To(MatchError("failed to create auth token: " + createErr.Error()))
			})
		})

		It("creates the replication group", func() {
			Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(1))

			passedCtx, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInput).To(Equal(&elasticache.CreateReplicationGroupInput{
				Tags:                        []*elasticache.Tag{},
				AtRestEncryptionEnabled:     aws.Bool(true),
				TransitEncryptionEnabled:    aws.Bool(true),
				AuthToken:                   passedInput.AuthToken,
				AutomaticFailoverEnabled:    aws.Bool(true),
				MultiAZEnabled:              aws.Bool(true),
				CacheNodeType:               aws.String("test instance type"),
				CacheParameterGroupName:     aws.String(replicationGroupID),
				SecurityGroupIds:            aws.StringSlice([]string{"test sg1"}),
				CacheSubnetGroupName:        aws.String("test subnet group"),
				Engine:                      aws.String("redis"),
				EngineVersion:               aws.String("4.0.10"),
				PreferredMaintenanceWindow:  aws.String("test maintenance window"),
				ReplicationGroupDescription: aws.String("test desc"),
				ReplicationGroupId:          aws.String(replicationGroupID),
				ReplicasPerNodeGroup:        aws.Int64(0),
				NumNodeGroups:               aws.Int64(1),
				SnapshotRetentionLimit:      aws.Int64(7),
				SnapshotWindow:              aws.String("02:00-05:00"),
			}))
		})

		It("has an auth token with the right length", func() {
			_, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
			Expect(*passedInput.AuthToken).To(HaveLen(PasswordLength))
		})

		Context("when tags are set", func() {
			BeforeEach(func() {
				provisionParams.Tags = map[string]string{"tag1": "tag value1", "tag2": "tag value2"}
			})

			It("should pass them correctly", func() {
				_, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
				Expect(passedInput.Tags).To(ConsistOf([]*elasticache.Tag{
					{Key: aws.String("tag1"), Value: aws.String("tag value1")},
					{Key: aws.String("tag2"), Value: aws.String("tag value2")},
				}))
			})
		})

		Context("when backup and failover is not enabled", func() {
			BeforeEach(func() {
				provisionParams.SnapshotRetentionLimit = 0
				provisionParams.AutomaticFailoverEnabled = false
				provisionParams.MultiAZEnabled = false
			})

			It("creates a replication group without backup or failover", func() {
				Expect(provisionErr).NotTo(HaveOccurred())
				passedCtx, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
				Expect(passedCtx).To(Equal(ctx))
				Expect(passedInput.SnapshotRetentionLimit).To(BeNil())
				Expect(passedInput.SnapshotWindow).To(BeNil())
				Expect(*passedInput.AutomaticFailoverEnabled).To(BeFalse())
				Expect(*passedInput.MultiAZEnabled).To(BeFalse())
			})
		})

		It("doesn't clean up accidentally", func() {
			Expect(mockElasticache.DeleteCacheParameterGroupWithContextCallCount()).To(Equal(0))
			Expect(mockSecretsManager.DeleteSecretWithContextCallCount()).To(Equal(0))
		})

		Context("when provision fails", func() {
			var createErr = errors.New("some err")

			BeforeEach(func() {
				mockElasticache.CreateReplicationGroupWithContextReturnsOnCall(0, nil, createErr)
			})

			It("returns with the error", func() {
				Expect(provisionErr).To(MatchError(createErr))
			})

			It("deletes the auth token from the Secrets Manager", func() {
				Expect(mockSecretsManager.DeleteSecretWithContextCallCount()).To(Equal(1))
				passedCtx, input, _ := mockSecretsManager.DeleteSecretWithContextArgsForCall(0)
				Expect(passedCtx).To(Equal(ctx))
				Expect(input.SecretId).To(Equal(aws.String("elasticache-broker-test/foobar/auth-token")))
				Expect(*input.RecoveryWindowInDays).To(Equal(int64(7)))
			})

			Context("if deleting the auth token fails", func() {
				BeforeEach(func() {
					mockSecretsManager.DeleteSecretWithContextReturns(nil, errors.New("some error"))
				})
				It("is ignored", func() {
					Expect(provisionErr).To(MatchError(createErr))
				})
			})

			It("deletes the cache parameter group", func() {
				Expect(mockElasticache.DeleteCacheParameterGroupWithContextCallCount()).To(Equal(1))
				passedCtx, receivedInput, _ := mockElasticache.DeleteCacheParameterGroupWithContextArgsForCall(0)
				Expect(passedCtx).To(Equal(ctx))
				Expect(receivedInput).To(Equal(&elasticache.DeleteCacheParameterGroupInput{
					CacheParameterGroupName: aws.String("cf-qwkec4pxhft6q"),
				}))
			})

			Context("if deleting the cache parameter group fails", func() {
				BeforeEach(func() {
					mockElasticache.DeleteCacheParameterGroupWithContextReturns(nil, errors.New("some error"))
				})
				It("is ignored", func() {
					Expect(provisionErr).To(MatchError(createErr))
				})
			})
		})

		Context("when restoring from a snapshot", func() {
			var snapshotToRestore = "automatic.cf-1234567890"

			BeforeEach(func() {
				provisionParams.RestoreFromSnapshot = aws.String(snapshotToRestore)
			})

			It("passes the snapshot name to AWS", func() {
				Expect(provisionErr).NotTo(HaveOccurred())
				Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(1))

				passedCtx, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
				Expect(passedCtx).To(Equal(ctx))
				Expect(passedInput.SnapshotName).To(Equal(aws.String(snapshotToRestore)))
			})
		})
	})

	Context("when deprovisioning", func() {

		var (
			deprovisionParams providers.DeprovisionParameters
			deprovisionErr    error
		)

		BeforeEach(func() {
			deprovisionParams = providers.DeprovisionParameters{}
		})

		JustBeforeEach(func() {
			deprovisionErr = provider.Deprovision(ctx, instanceID, deprovisionParams)
		})

		It("succeeds", func() {
			Expect(deprovisionErr).ToNot(HaveOccurred())
		})

		It("deletes the replication group", func() {
			Expect(mockElasticache.DeleteReplicationGroupWithContextCallCount()).To(Equal(1))
			passedCtx, passedInput, _ := mockElasticache.DeleteReplicationGroupWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInput).To(Equal(&elasticache.DeleteReplicationGroupInput{
				ReplicationGroupId: aws.String(replicationGroupID),
			}))
		})

		It("deletes the auth token from the secrets manager", func() {
			Expect(mockSecretsManager.DeleteSecretWithContextCallCount()).To(Equal(1))
			ctx, input, _ := mockSecretsManager.DeleteSecretWithContextArgsForCall(0)
			Expect(ctx).To(Equal(ctx))
			Expect(input.SecretId).To(Equal(aws.String("elasticache-broker-test/foobar/auth-token")))
			Expect(*input.RecoveryWindowInDays).To(Equal(int64(30)))
		})

		Context("when deleting the auth token fails", func() {
			var deleteErr = errors.New("error in secrets manager")
			BeforeEach(func() {
				mockSecretsManager.DeleteSecretWithContextReturnsOnCall(0, nil, deleteErr)
			})

			It("returns an error", func() {
				Expect(deprovisionErr).To(MatchError(deleteErr))
			})
		})

		Context("when the final snapshot name is set", func() {
			BeforeEach(func() {
				deprovisionParams = providers.DeprovisionParameters{
					FinalSnapshotIdentifier: "test snapshot",
				}
			})

			It("sets a parameter for creating a final snapshot", func() {
				_, passedInput, _ := mockElasticache.DeleteReplicationGroupWithContextArgsForCall(0)
				Expect(passedInput.FinalSnapshotIdentifier).To(Equal(aws.String("test snapshot")))
			})
		})

		Context("if deleting the replication group fails", func() {
			var deleteErr = errors.New("some error")

			BeforeEach(func() {
				mockElasticache.DeleteReplicationGroupWithContextReturnsOnCall(0, nil, deleteErr)
			})

			It("returns with the error", func() {
				Expect(deprovisionErr).To(MatchError(deleteErr))
			})

			It("does not delete the auth token from the Secrets Manager", func() {
				Expect(mockSecretsManager.DeleteSecretWithContextCallCount()).To(Equal(0))
			})
		})
	})

	Context("when getting the status of the cluster", func() {

		It("returns with the current status for an existing cluster", func() {
			replicationGroupID := "cf-qwkec4pxhft6q"

			awsOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
					{
						Status: aws.String("test status"),
					},
				},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(awsOutput, nil)

			instanceID := "foobar"
			ctx := context.Background()

			state, stateMessage, stateErr := provider.ProgressState(ctx, instanceID, "", "")

			Expect(state).To(Equal(providers.ServiceState("test status")))
			Expect(stateMessage).To(ContainSubstring("status               : test status"))
			Expect(stateErr).ToNot(HaveOccurred())

			Expect(mockElasticache.DescribeReplicationGroupsWithContextCallCount()).To(Equal(1))
			passedCtx, passedInput, _ := mockElasticache.DescribeReplicationGroupsWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInput).To(Equal(&elasticache.DescribeReplicationGroupsInput{
				ReplicationGroupId: aws.String(replicationGroupID),
			}))
		})

		Describe("ProgressState", func() {

			var (
				snapshotWindow     = aws.String("05:01-09:01")
				maintenanceWindow  = aws.String("sun:23:01-mon:01:31")
				replicationGroupID = "cf-qwkec4pxhft6q"
				cacheClusterId     = replicationGroupID + "-001-001"
				instanceID         = "foobar"
				autoFailover       = aws.String("enabled")
				primaryNode        = aws.String("cf-qwkec4pxhft6q-001")
				replicaNode        = aws.String("cf-qwkec4pxhft6q-002")
			)

			JustBeforeEach(func() {
				mockElasticache.DescribeReplicationGroupsWithContextReturns(&elasticache.DescribeReplicationGroupsOutput{
					ReplicationGroups: []*elasticache.ReplicationGroup{
						{
							ReplicationGroupId: aws.String(replicationGroupID),
							Status:             aws.String("available"),
							AutomaticFailover:  autoFailover,
							MemberClusters: []*string{
								aws.String(cacheClusterId),
							},
							SnapshotWindow: snapshotWindow,

							NodeGroups: []*elasticache.NodeGroup{
								{
									NodeGroupMembers: []*elasticache.NodeGroupMember{
										{
											CurrentRole:    aws.String("primary"),
											CacheClusterId: primaryNode,
										},
										{
											CurrentRole:    aws.String("replica"),
											CacheClusterId: replicaNode,
										},
									},
								},
							},
						},
					},
				}, nil)

				mockElasticache.DescribeCacheClustersWithContextReturns(&elasticache.DescribeCacheClustersOutput{
					CacheClusters: []*elasticache.CacheCluster{
						{
							CacheClusterId:             aws.String(cacheClusterId),
							PreferredMaintenanceWindow: maintenanceWindow,
							EngineVersion:              aws.String("9.9.9"),
						},
					},
				}, nil)

				mockElasticache.DescribeCacheParametersWithContextReturns(&elasticache.DescribeCacheParametersOutput{
					Parameters: []*elasticache.Parameter{
						{
							ParameterName:  aws.String("maxmemory-policy"),
							ParameterValue: aws.String("test-ttl"),
						},
						{
							ParameterName:  aws.String("some-other-param"),
							ParameterValue: aws.String("some-other-value"),
						},
						{
							ParameterName:  aws.String("cluster-enabled"),
							ParameterValue: aws.String("yes"),
						},
					},
				}, nil)
			})

			It("returns a message with details for useful configuration values", func() {
				_, stateMessage, stateErr := provider.ProgressState(context.Background(), instanceID, "", "")
				Expect(stateErr).ToNot(HaveOccurred())

				Expect(mockElasticache.DescribeCacheParametersWithContextCallCount()).To(Equal(1))
				Expect(stateMessage).To(ContainSubstring("status               : available"))
				Expect(stateMessage).To(ContainSubstring("engine version       : 9.9.9"))
				Expect(stateMessage).To(ContainSubstring("maxmemory policy     : test-ttl"))
				Expect(stateMessage).To(ContainSubstring("daily backup window  : 05:01-09:01"))
				Expect(stateMessage).To(ContainSubstring("maintenance window   : sun:23:01-mon:01:31"))
				Expect(stateMessage).To(ContainSubstring("cluster enabled      : yes"))
				Expect(stateMessage).To(ContainSubstring("automatic failover   : enabled"))
			})

			Context("when it doesn't have automated backup", func() {
				BeforeEach(func() {
					snapshotWindow = nil
				})
				It("won't return the daily backup window in the message", func() {
					_, stateMessage, stateErr := provider.ProgressState(context.Background(), instanceID, "", "")
					Expect(stateErr).ToNot(HaveOccurred())
					Expect(stateMessage).ToNot(ContainSubstring("daily backup window"))
				})
			})

			Context("when it doesn't have a preferred maintenance window", func() {
				BeforeEach(func() {
					maintenanceWindow = nil
				})
				It("won't return the field", func() {
					_, stateMessage, stateErr := provider.ProgressState(context.Background(), instanceID, "", "")
					Expect(stateErr).ToNot(HaveOccurred())
					Expect(stateMessage).ToNot(ContainSubstring("maintenance window"))
				})
			})

			Context("when operation is failover", func() {
				BeforeEach(func() {
					autoFailover = aws.String("disabled")
				})
				It("will trigger node failover", func() {
					mockElasticache.ModifyReplicationGroupWithContextReturns(&elasticache.ModifyReplicationGroupOutput{}, nil)
					_, stateMessage, stateErr := provider.ProgressState(
						context.Background(),
						instanceID, "failover",
						"cf-qwkec4pxhft6q-001",
					)

					_, input, _ := mockElasticache.ModifyReplicationGroupWithContextArgsForCall(0)
					Expect(input.ApplyImmediately).To(Equal(aws.Bool(true)))
					Expect(input.ReplicationGroupId).To(Equal(aws.String(replicationGroupID)))
					Expect(input.PrimaryClusterId).To(Equal(replicaNode))

					Expect(stateErr).ToNot(HaveOccurred())
					Expect(stateMessage).To(ContainSubstring("status               : modifying"))
					Expect(stateMessage).To(ContainSubstring("automatic failover   : disabled"))
					Expect(mockElasticache.ModifyReplicationGroupWithContextCallCount()).To(Equal(1))
				})
				It("will trigger autofailover enable", func() {
					mockElasticache.ModifyReplicationGroupWithContextReturns(&elasticache.ModifyReplicationGroupOutput{}, nil)
					_, stateMessage, stateErr := provider.ProgressState(
						context.Background(),
						instanceID,
						"failover",
						"cf-qwkec4pxhft6q-002",
					)

					_, input, _ := mockElasticache.ModifyReplicationGroupWithContextArgsForCall(0)
					Expect(input.ApplyImmediately).To(Equal(aws.Bool(true)))
					Expect(input.ReplicationGroupId).To(Equal(aws.String(replicationGroupID)))
					Expect(input.MultiAZEnabled).To(Equal(aws.Bool(true)))
					Expect(input.AutomaticFailoverEnabled).To(Equal(aws.Bool(true)))

					Expect(stateErr).ToNot(HaveOccurred())
					Expect(stateMessage).To(ContainSubstring("status               : modifying"))
					Expect(stateMessage).To(ContainSubstring("automatic failover   : disabled"))
					Expect(mockElasticache.ModifyReplicationGroupWithContextCallCount()).To(Equal(1))
				})
			})
		})

		It("handles errors from the AWS API", func() {
			describeErr := errors.New("some error")
			mockElasticache.DescribeReplicationGroupsWithContextReturns(nil, describeErr)

			_, _, stateErr := provider.ProgressState(context.Background(), "foobar", "", "")
			Expect(stateErr).To(MatchError(describeErr))
		})

		It("returns deleted status for a non-existing replication group", func() {
			describeErr := awserr.New(elasticache.ErrCodeReplicationGroupNotFoundFault, "some message", nil)
			mockElasticache.DescribeReplicationGroupsWithContextReturns(nil, describeErr)

			state, stateMessage, stateErr := provider.ProgressState(context.Background(), "foobar", "", "")
			Expect(state).To(Equal(providers.NonExisting))
			Expect(stateMessage).To(Equal("Replication group does not exist: cf-qwkec4pxhft6q"))
			Expect(stateErr).ToNot(HaveOccurred())
		})

		It("handles an empty replication group list from AWS", func() {
			describeOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeOutput, nil)
			_, _, stateErr := provider.ProgressState(context.Background(), "foobar", "", "")
			Expect(stateErr).To(MatchError("Invalid response from AWS: no replication groups returned for cf-qwkec4pxhft6q"))
		})

		It("handles empty status from AWS", func() {
			describeOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{{}},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeOutput, nil)
			_, _, stateErr := provider.ProgressState(context.Background(), "foobar", "", "")
			Expect(stateErr).To(MatchError("Invalid response from AWS: status is missing for cf-qwkec4pxhft6q"))
		})

	})

	Context("when generating replication group names", func() {
		It("should generate valid and unique values", func() {
			n := 20000
			collisionCount := 0
			instanceIDs := make(map[string]int, n)
			nameRegexp := regexp.MustCompile(`[a-z][a-z0-9\-]*`)
			for i := 0; i < n; i++ {
				instanceID := uuid.NewV4().String()
				name := GenerateReplicationGroupName(instanceID)
				Expect(len(name)).To(BeNumerically(">", 0))
				Expect(len(name)).To(BeNumerically("<=", 20))
				Expect(nameRegexp.MatchString(name)).To(BeTrue())
				Expect(name[len(name)-1]).ToNot(Equal(byte('-')))
				Expect(name).ToNot(ContainSubstring("--"))
				instanceIDs[name]++
				if instanceIDs[name] > 1 {
					collisionCount++
				}
			}
			Expect(collisionCount).To(BeNumerically("<=", 2))
		})
	})

	Describe("GenerateCredentials", func() {

		var (
			bindingID                       string
			credentials                     *providers.Credentials
			generateErr                     error
			describeReplicationGroupsOutput *elasticache.DescribeReplicationGroupsOutput
			describeReplicationGroupsErr    error
			getSecretValueOutput            *secretsmanager.GetSecretValueOutput
			getSecretValueErr               error
		)

		BeforeEach(func() {
			bindingID = "test-binding"

			describeReplicationGroupsOutput = &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
					{
						ConfigurationEndpoint: &elasticache.Endpoint{
							Address: aws.String("test-host"),
							Port:    aws.Int64(1234),
						},
					},
				},
			}
			describeReplicationGroupsErr = nil

			getSecretValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("Jc9xP_jNPaWtqIry7D-EuRlsm_z_-D_dtIVQhEv6oR4="),
			}
			getSecretValueErr = nil
		})

		JustBeforeEach(func() {
			mockElasticache.DescribeReplicationGroupsWithContextReturns(
				describeReplicationGroupsOutput,
				describeReplicationGroupsErr,
			)

			mockSecretsManager.GetSecretValueWithContextReturns(getSecretValueOutput, getSecretValueErr)

			credentials, generateErr = provider.GenerateCredentials(ctx, instanceID, bindingID)
		})

		It("should return no error", func() {
			Expect(generateErr).ToNot(HaveOccurred())
		})

		It("gets the auth token from the Secrets Manager", func() {
			Expect(mockSecretsManager.GetSecretValueWithContextCallCount()).To(Equal(1))
			passedCtx, passedSecretsManagerInput, _ := mockSecretsManager.GetSecretValueWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedSecretsManagerInput).To(Equal(&secretsmanager.GetSecretValueInput{
				SecretId: aws.String("elasticache-broker-test/foobar/auth-token"),
			}))
		})

		Context("when getting the auth token fails with a general error", func() {
			BeforeEach(func() {
				getSecretValueOutput = nil
				getSecretValueErr = errors.New("some error")
			})
			It("should return the error", func() {
				Expect(generateErr).To(Equal(getSecretValueErr))
			})
		})

		Context("when getting the auth token fails with an AWS error (e.g. failed request)", func() {
			BeforeEach(func() {
				getSecretValueOutput = nil
				getSecretValueErr = awserr.New(secretsmanager.ErrCodeInternalServiceError, "x", nil)
			})
			It("should return the error", func() {
				Expect(generateErr).To(Equal(getSecretValueErr))
			})
		})

		It("should return with credentials", func() {
			Expect(credentials).To(Equal(&providers.Credentials{
				Host:       "test-host",
				Port:       1234,
				Name:       "cf-qwkec4pxhft6q",
				Password:   "Jc9xP_jNPaWtqIry7D-EuRlsm_z_-D_dtIVQhEv6oR4=",
				URI:        "rediss://:Jc9xP_jNPaWtqIry7D-EuRlsm_z_-D_dtIVQhEv6oR4=@test-host:1234",
				TLSEnabled: true,
			}))
		})

		It("calls DescribeReplicationGroups with the right parameters", func() {
			Expect(mockElasticache.DescribeReplicationGroupsWithContextCallCount()).To(Equal(1))
			passedCtx, passedElasticacheInput, _ := mockElasticache.DescribeReplicationGroupsWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedElasticacheInput).To(Equal(&elasticache.DescribeReplicationGroupsInput{
				ReplicationGroupId: aws.String(replicationGroupID),
			}))
		})

		Context("when no configuration endpoint is provided (cluster mode is disabled)", func() {
			BeforeEach(func() {
				describeReplicationGroupsOutput = &elasticache.DescribeReplicationGroupsOutput{
					ReplicationGroups: []*elasticache.ReplicationGroup{
						{
							NodeGroups: []*elasticache.NodeGroup{
								{
									PrimaryEndpoint: &elasticache.Endpoint{
										Address: aws.String("test-host"),
										Port:    aws.Int64(1234),
									},
								},
							},
						},
					},
				}
			})

			It("should provide the primary endpoint in the credentials", func() {
				Expect(credentials).To(Equal(&providers.Credentials{
					Host:       "test-host",
					Port:       1234,
					Name:       "cf-qwkec4pxhft6q",
					Password:   "Jc9xP_jNPaWtqIry7D-EuRlsm_z_-D_dtIVQhEv6oR4=",
					URI:        "rediss://:Jc9xP_jNPaWtqIry7D-EuRlsm_z_-D_dtIVQhEv6oR4=@test-host:1234",
					TLSEnabled: true,
				}))
			})
		})

		Context("when no node groups are returned", func() {
			BeforeEach(func() {
				describeReplicationGroupsOutput = &elasticache.DescribeReplicationGroupsOutput{
					ReplicationGroups: []*elasticache.ReplicationGroup{
						{
							NodeGroups: []*elasticache.NodeGroup{},
						},
					},
				}
			})

			It("should return an error", func() {
				Expect(generateErr).To(HaveOccurred())
			})
		})

		Context("when no replication groups are returned", func() {
			BeforeEach(func() {
				describeReplicationGroupsOutput = &elasticache.DescribeReplicationGroupsOutput{
					ReplicationGroups: []*elasticache.ReplicationGroup{},
				}
			})

			It("should return an error", func() {
				Expect(generateErr).To(HaveOccurred())
			})
		})

		Context("when no endpoints groups are returned", func() {
			BeforeEach(func() {
				describeReplicationGroupsOutput = &elasticache.DescribeReplicationGroupsOutput{
					ReplicationGroups: []*elasticache.ReplicationGroup{
						{},
					},
				}
			})

			It("should return an error", func() {
				Expect(generateErr).To(HaveOccurred())
			})
		})

		Context("when the cluster does not exist", func() {
			BeforeEach(func() {
				describeReplicationGroupsOutput = nil
				describeReplicationGroupsErr = awserr.New(elasticache.ErrCodeReplicationGroupNotFoundFault, "some message", nil)
			})
			It("should return an error", func() {
				Expect(generateErr).To(HaveOccurred())
			})
		})

		Context("when there is no auth token in Secrets Manager", func() {
			BeforeEach(func() {
				getSecretValueOutput = nil
				getSecretValueErr = awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "x", nil)
			})

			It("should return with the error", func() {
				Expect(generateErr).To(MatchError(getSecretValueErr))
			})
		})
	})

	Context("when deleting a parameter group", func() {
		It("should delete the parameter group successfully", func() {
			ctx := context.Background()
			err := provider.DeleteCacheParameterGroup(ctx, "foobar")

			Expect(mockElasticache.DeleteCacheParameterGroupWithContextCallCount()).To(Equal(1))
			receivedCtx, receivedInput, _ := mockElasticache.DeleteCacheParameterGroupWithContextArgsForCall(0)
			Expect(receivedCtx).To(Equal(ctx))
			Expect(receivedInput).To(Equal(&elasticache.DeleteCacheParameterGroupInput{
				CacheParameterGroupName: aws.String("cf-qwkec4pxhft6q"),
			}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return no error if parameter group does not exist", func() {
			deleteErr := awserr.New(elasticache.ErrCodeCacheParameterGroupNotFoundFault, "some message", nil)
			mockElasticache.DeleteCacheParameterGroupWithContextReturns(nil, deleteErr)
			err := provider.DeleteCacheParameterGroup(context.Background(), "foobar")

			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error if deleting the parameter group fails", func() {
			deleteErr := errors.New("A really really really bad error")
			mockElasticache.DeleteCacheParameterGroupWithContextReturns(nil, deleteErr)
			err := provider.DeleteCacheParameterGroup(context.Background(), "foobar")

			Expect(err).To(MatchError(deleteErr))
		})
	})

	Context("when revoking credentials from an app", func() {
		It("should return no error", func() {
			instanceID := "foobar"
			bindingID := "test-binding"
			ctx := context.Background()
			err := provider.RevokeCredentials(ctx, instanceID, bindingID)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when listing existing snapshots", func() {
		var (
			describeSnapshotOutputToReturn []elasticache.DescribeSnapshotsOutput
			errorToReturn                  error
		)
		BeforeEach(func() {
			describeSnapshotOutputToReturn = []elasticache.DescribeSnapshotsOutput{}
			errorToReturn = nil

			mockElasticache.ListTagsForResourceWithContextReturns(&elasticache.TagListMessage{}, nil)

			mockElasticache.DescribeSnapshotsPagesWithContextStub =
				func(ctx aws.Context, input *elasticache.DescribeSnapshotsInput,
					fn func(*elasticache.DescribeSnapshotsOutput, bool) bool, opts ...request.Option) error {
					for i, s := range describeSnapshotOutputToReturn {
						continue_ := fn(&s, i == len(describeSnapshotOutputToReturn)-1)
						if !continue_ {
							break
						}
					}
					return errorToReturn
				}
		})

		It("returns the list of existing snapshots", func() {
			instanceID := "foobar"

			mockElasticache.ListTagsForResourceWithContextReturns(
				&elasticache.TagListMessage{
					TagList: []*elasticache.Tag{
						{
							Key:   aws.String("instance-id"),
							Value: aws.String(instanceID),
						},
						{
							Key:   aws.String("Tag1"),
							Value: aws.String("Val1"),
						},
						{
							Key:   aws.String("Tag2"),
							Value: aws.String("Val2"),
						},
					},
				},
				nil,
			)

			now := time.Now()
			describeSnapshotOutputToReturn = []elasticache.DescribeSnapshotsOutput{
				{
					Snapshots: []*elasticache.Snapshot{
						{
							SnapshotName: aws.String("snapshot1"),
							NodeSnapshots: []*elasticache.NodeSnapshot{
								{
									CacheClusterId:     &instanceID,
									SnapshotCreateTime: aws.Time(now.Add(-2 * 24 * time.Hour)),
								},
							},
						},
						{
							SnapshotName: aws.String("snapshot2"),
							NodeSnapshots: []*elasticache.NodeSnapshot{
								{
									CacheClusterId:     &instanceID,
									SnapshotCreateTime: aws.Time(now.Add(-1 * 24 * time.Hour)),
								},
							},
						},
					},
				},
			}

			ctx := context.Background()
			snapshots, err := provider.FindSnapshots(ctx, instanceID)

			Expect(err).ToNot(HaveOccurred())
			Expect(snapshots).To(HaveLen(2))
			Expect(snapshots).To(ConsistOf(
				providers.SnapshotInfo{
					Name:       "snapshot1",
					CreateTime: now.Add(-2 * 24 * time.Hour),
					Tags: map[string]string{
						"Tag1":        "Val1",
						"Tag2":        "Val2",
						"instance-id": instanceID,
					},
				},
				providers.SnapshotInfo{
					Name:       "snapshot2",
					CreateTime: now.Add(-1 * 24 * time.Hour),
					Tags: map[string]string{
						"Tag1":        "Val1",
						"Tag2":        "Val2",
						"instance-id": instanceID,
					},
				},
			))
		})

		It("returns and empty list if there are no snapshots", func() {
			instanceID := "foobar"

			ctx := context.Background()
			snapshots, err := provider.FindSnapshots(ctx, instanceID)

			Expect(err).ToNot(HaveOccurred())
			Expect(snapshots).To(HaveLen(0))
		})

		It("returns error if the call to AWS to list snapshots fails", func() {
			instanceID := "foobar"

			mockElasticache.DescribeSnapshotsPagesWithContextReturns(
				fmt.Errorf("HORRIBLE ERROR DESCRIBING SNAPSHOTS"),
			)

			ctx := context.Background()
			_, err := provider.FindSnapshots(ctx, instanceID)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("HORRIBLE ERROR DESCRIBING SNAPSHOTS")))
		})

		It("returns error if the call to AWS to list tags fails", func() {
			instanceID := "foobar"

			now := time.Now()
			describeSnapshotOutputToReturn = []elasticache.DescribeSnapshotsOutput{
				{
					Snapshots: []*elasticache.Snapshot{

						{
							SnapshotName: aws.String("snapshot1"),
							NodeSnapshots: []*elasticache.NodeSnapshot{
								{
									CacheClusterId:     &instanceID,
									SnapshotCreateTime: aws.Time(now.Add(-2 * 24 * time.Hour)),
								},
							},
						},
					},
				},
			}

			mockElasticache.ListTagsForResourceWithContextReturns(
				&elasticache.TagListMessage{},
				fmt.Errorf("HORRIBLE ERROR LISTING TAGS"),
			)

			ctx := context.Background()
			_, err := provider.FindSnapshots(ctx, instanceID)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("HORRIBLE ERROR LISTING TAGS")))
		})
		It("returns error if the snapshots miss some data", func() {
			instanceID := "foobar"

			describeSnapshotOutputToReturn = []elasticache.DescribeSnapshotsOutput{
				{
					Snapshots: []*elasticache.Snapshot{
						{},
					},
				},
			}

			ctx := context.Background()
			_, err := provider.FindSnapshots(ctx, instanceID)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(
				ContainSubstring("Invalid response from AWS: Missing values for snapshot for elasticache cluster")),
			)
		})
	})

	Context("when updating", func() {
		It("should update the cache parameter group", func() {
			replicationGroupID := "cf-qwkec4pxhft6q"
			instanceID := "foobar"

			ctx := context.Background()

			err := provider.UpdateParamGroupParameters(ctx, instanceID, providers.UpdateParamGroupParameters{
				Parameters: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(mockElasticache.ModifyCacheParameterGroupWithContextCallCount()).To(Equal(1))
			receivedCtx, paramGroupInput, _ := mockElasticache.ModifyCacheParameterGroupWithContextArgsForCall(0)
			Expect(receivedCtx).To(Equal(ctx))
			Expect(paramGroupInput.CacheParameterGroupName).To(Equal(aws.String(replicationGroupID)))
			Expect(paramGroupInput.ParameterNameValues).To(ConsistOf([]*elasticache.ParameterNameValue{
				{
					ParameterName:  aws.String("key1"),
					ParameterValue: aws.String("value1"),
				},
				{
					ParameterName:  aws.String("key2"),
					ParameterValue: aws.String("value2"),
				},
			}))
		})

		It("should not modify the cache parameter group if no params are passed", func() {
			err := provider.UpdateParamGroupParameters(context.Background(), "foobar", providers.UpdateParamGroupParameters{
				Parameters: map[string]string{},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(mockElasticache.ModifyCacheParameterGroupWithContextCallCount()).To(Equal(0))
		})

		It("should return with error if AWS API returns with an error", func() {
			awsError := errors.New("some error")
			mockElasticache.ModifyCacheParameterGroupWithContextReturnsOnCall(0, nil, awsError)

			err := provider.UpdateParamGroupParameters(context.Background(), "foobar", providers.UpdateParamGroupParameters{
				Parameters: map[string]string{
					"key1": "value1",
				},
			})
			Expect(err).To(MatchError(awsError))
		})

		It("should update the replication group", func() {
			replicationGroupID := "cf-qwkec4pxhft6q"
			instanceID := "foobar"

			ctx := context.Background()

			err := provider.UpdateReplicationGroup(ctx, instanceID, providers.UpdateReplicationGroupParameters{
				PreferredMaintenanceWindow: "asdf",
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(mockElasticache.ModifyReplicationGroupWithContextCallCount()).To(Equal(1))
			receivedCtx, replicationGroupInput, _ := mockElasticache.ModifyReplicationGroupWithContextArgsForCall(0)
			Expect(receivedCtx).To(Equal(ctx))
			Expect(replicationGroupInput.ReplicationGroupId).To(Equal(aws.String(replicationGroupID)))
			Expect(*replicationGroupInput.PreferredMaintenanceWindow).To(Equal("asdf"))
		})

		It("should not update the replication group if no preferredMaintenanceWindow is passed", func() {
			instanceID := "foobar"

			ctx := context.Background()

			err := provider.UpdateReplicationGroup(ctx, instanceID, providers.UpdateReplicationGroupParameters{
				PreferredMaintenanceWindow: "",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(mockElasticache.ModifyReplicationGroupWithContextCallCount()).To(Equal(0))
		})
	})

	Describe("replicationGroupARN", func() {
		It("should generate the correct ARN string", func() {
			arn := ExportReplicationGroupARN(provider, replicationGroupID)
			Expect(arn).To(Equal(fmt.Sprintf("arn:%s:elasticache:%s:%s:replicationgroup:%s", awsPartition, awsRegion, awsAccountID, replicationGroupID)))
		})
	})

	Describe("GetInstanceTags", func() {
		It("Should produce a key:value list of AWS tags on the instance", func() {
			expectedTags := map[string]string{
				"testkey1": "testvalue1",
				"testkey2": "testvalue2",
			}
			tagListMessage := elasticache.TagListMessage{
				TagList: []*elasticache.Tag{
					{
						Key:   aws.String("testkey1"),
						Value: aws.String("testvalue1"),
					},
					{
						Key:   aws.String("testkey2"),
						Value: aws.String("testvalue2"),
					},
				},
			}
			mockElasticache.ListTagsForResourceWithContextReturns(&tagListMessage, nil)

			tags, _ := provider.GetInstanceTags(ctx, instanceID)
			Expect(tags).To(Equal(expectedTags))
		})
		It("should return the error if an error occurs", func() {
			errorReturn := errors.New("there was an error")
			mockElasticache.ListTagsForResourceWithContextReturns(&elasticache.TagListMessage{}, errorReturn)
			tags, err := provider.GetInstanceTags(ctx, instanceID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errorReturn))
			Expect(tags).To(BeNil())
		})
	})

	Describe("Test Failover", func() {
		Context("when performing complete failover", func() {
			BeforeEach(func() {
				describeReplicationGroupsOutput := &elasticache.DescribeReplicationGroupsOutput{
					ReplicationGroups: []*elasticache.ReplicationGroup{
						{
							SnapshotWindow: aws.String("some-snapshot-window"),

							Status: aws.String("available"),

							AutomaticFailover: aws.String("enabled"),
							MultiAZ:           aws.String("enabled"),

							NodeGroups: []*elasticache.NodeGroup{
								{
									NodeGroupMembers: []*elasticache.NodeGroupMember{
										{
											CurrentRole:    aws.String("primary"),
											CacheClusterId: aws.String("cf-qwkec4pxhft6q-001"),
										},
										{
											CurrentRole:    aws.String("replica"),
											CacheClusterId: aws.String("cf-qwkec4pxhft6q-002"),
										},
									},
								},
							},
						},
					},
				}

				replicationGroup := describeReplicationGroupsOutput.ReplicationGroups[0]
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(0,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				replicationGroup.Status = aws.String("modifying")
				replicationGroup.AutomaticFailover = aws.String("disabling")
				replicationGroup.MultiAZ = aws.String("disabling")
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(1,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				replicationGroup.Status = aws.String("available")
				replicationGroup.AutomaticFailover = aws.String("disabled")
				replicationGroup.MultiAZ = aws.String("disabled")
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(2,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				replicationGroup.Status = aws.String("modifying")
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(3,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				replicationGroup.Status = aws.String("available")
				replicationGroup.NodeGroups[0].NodeGroupMembers[0].CurrentRole = aws.String("replica")
				replicationGroup.NodeGroups[0].NodeGroupMembers[1].CurrentRole = aws.String("primary")
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(4,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				replicationGroup.Status = aws.String("modifying")
				replicationGroup.AutomaticFailover = aws.String("enabling")
				replicationGroup.MultiAZ = aws.String("enabling")
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(5,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				replicationGroup.Status = aws.String("available")
				replicationGroup.AutomaticFailover = aws.String("enabled")
				replicationGroup.MultiAZ = aws.String("enabled")
				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(5,
					awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)

				mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(6,
					&elasticache.DescribeReplicationGroupsOutput{}, fmt.Errorf("We shouldn't have gotten this far"))
			})

			It("Fails over successfully", func() {

				primaryNode, err := provider.StartFailoverTest(ctx, instanceID)
				Expect(primaryNode).To(Equal("cf-qwkec4pxhft6q-001"))
				Expect(err).ToNot(HaveOccurred())
				Expect(mockElasticache.ModifyReplicationGroupWithContextCallCount()).To(Equal(1))
				Expect(mockElasticache.DescribeReplicationGroupsWithContextCallCount()).To(Equal(1))

				receivedCtx, replicationGroupInput, _ := mockElasticache.ModifyReplicationGroupWithContextArgsForCall(0)
				Expect(receivedCtx).To(Equal(ctx))
				Expect(replicationGroupInput.ApplyImmediately).To(Equal(aws.Bool(true)))
				Expect(replicationGroupInput.AutomaticFailoverEnabled).To(Equal(aws.Bool(false)))
				Expect(replicationGroupInput.MultiAZEnabled).To(Equal(aws.Bool(false)))
				Expect(replicationGroupInput.ReplicationGroupId).To(Equal(aws.String(replicationGroupID)))

				var message string
				Eventually(
					func() providers.ServiceState {
						state, message2, err := provider.ProgressState(
							ctx,
							instanceID,
							"failover",
							"cf-qwkec4pxhft6q-001",
						)
						message = message2
						Expect(err).ToNot(HaveOccurred())
						return state
					},
					10*time.Second,
					0,
				).Should(
					Equal(providers.ServiceState("available")),
				)

				Expect(message).To(ContainSubstring("automatic failover   : enabled"))
				Expect(mockElasticache.ModifyReplicationGroupWithContextCallCount()).To(Equal(3))

				receivedCtx, replicationGroupInput, _ = mockElasticache.ModifyReplicationGroupWithContextArgsForCall(1)
				Expect(receivedCtx).To(Equal(ctx))
				Expect(replicationGroupInput.ApplyImmediately).To(Equal(aws.Bool(true)))
				Expect(replicationGroupInput.PrimaryClusterId).To(Equal(aws.String("cf-qwkec4pxhft6q-002")))
				Expect(replicationGroupInput.ReplicationGroupId).To(Equal(aws.String(replicationGroupID)))

				receivedCtx, replicationGroupInput, _ = mockElasticache.ModifyReplicationGroupWithContextArgsForCall(2)
				Expect(receivedCtx).To(Equal(ctx))
				Expect(replicationGroupInput.ApplyImmediately).To(Equal(aws.Bool(true)))
				Expect(replicationGroupInput.AutomaticFailoverEnabled).To(Equal(aws.Bool(true)))
				Expect(replicationGroupInput.MultiAZEnabled).To(Equal(aws.Bool(true)))
				Expect(replicationGroupInput.ReplicationGroupId).To(Equal(aws.String(replicationGroupID)))
			})
		})

		It("fails if primary node can not be determined", func() {
			describeReplicationGroupsOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
					{
						NodeGroups: []*elasticache.NodeGroup{
							{
								NodeGroupMembers: []*elasticache.NodeGroupMember{
									{
										CurrentRole:    aws.String("replica"),
										CacheClusterId: aws.String("cf-qwkec4pxhft6q-001"),
									},
									{
										CurrentRole:    aws.String("replica"),
										CacheClusterId: aws.String("cf-qwkec4pxhft6q-002"),
									},
								},
							},
						},
					},
				},
			}

			mockElasticache.DescribeReplicationGroupsWithContextReturnsOnCall(0,
				awsutil.CopyOf(describeReplicationGroupsOutput).(*elasticache.DescribeReplicationGroupsOutput), nil)
			_, err := provider.StartFailoverTest(ctx, instanceID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errors.New("Unable to determine primary node")))
		})

	})

	Describe("GetInstanceParameters", func() {
		BeforeEach(func() {

			describeReplicationGroupsOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
					{
						MemberClusters: []*string{aws.String("some-cluster-id")},
						SnapshotWindow: aws.String("some-snapshot-window"),

						AutomaticFailover: aws.String("enabled"),

						NodeGroups: []*elasticache.NodeGroup{
							{
								NodeGroupMembers: []*elasticache.NodeGroupMember{
									{
										CurrentRole:    aws.String("primary"),
										CacheClusterId: aws.String("cf-qwkec4pxhft6q-001"),
									},
									{
										CurrentRole:    aws.String("replica"),
										CacheClusterId: aws.String("cf-qwkec4pxhft6q-002"),
									},
								},
							},
						},
					},
				},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeReplicationGroupsOutput, nil)

			describeCacheClustersOutput := &elasticache.DescribeCacheClustersOutput{
				CacheClusters: []*elasticache.CacheCluster{
					{
						CacheClusterId:             aws.String("some-cluster-id"),
						PreferredMaintenanceWindow: aws.String("some-maintenance-window"),
					},
				},
			}
			mockElasticache.DescribeCacheClustersWithContextReturns(describeCacheClustersOutput, nil)

			describeCacheParametersOutput := &elasticache.DescribeCacheParametersOutput{
				Parameters: []*elasticache.Parameter{
					{
						ParameterName:  aws.String("some-parameter-name"),
						ParameterValue: aws.String("some-parameter-value"),
					},
					{
						ParameterName:  aws.String("maxmemory-policy"),
						ParameterValue: aws.String("some-maxmemory-policy"),
					},
				},
			}
			mockElasticache.DescribeCacheParametersWithContextReturns(describeCacheParametersOutput, nil)
		})
		It("returns the instance parameters for an instance", func() {
			instanceParams, err := provider.GetInstanceParameters(ctx, instanceID)
			Expect(err).ToNot(HaveOccurred())

			Expect(instanceParams).To(Equal(providers.InstanceParameters{
				PreferredMaintenanceWindow: "some-maintenance-window",
				DailyBackupWindow:          "some-snapshot-window",
				MaxMemoryPolicy:            "some-maxmemory-policy",
				ActiveNodes:                []string{"cf-qwkec4pxhft6q-001"},
				PassiveNodes:               []string{"cf-qwkec4pxhft6q-002"},
				AutoFailover:               true,
				CacheParameters: []providers.CacheParameter{
					{
						ParameterName:  "some-parameter-name",
						ParameterValue: "some-parameter-value",
					},
					{
						ParameterName:  "maxmemory-policy",
						ParameterValue: "some-maxmemory-policy",
					},
				},
			}))
		})
		It("handles missing maintenance window", func() {
			describeCacheClustersOutput := &elasticache.DescribeCacheClustersOutput{
				CacheClusters: []*elasticache.CacheCluster{
					{
						CacheClusterId:             aws.String("some-cluster-id"),
						PreferredMaintenanceWindow: nil,
					},
				},
			}
			mockElasticache.DescribeCacheClustersWithContextReturns(describeCacheClustersOutput, nil)
			instanceParams, err := provider.GetInstanceParameters(ctx, instanceID)
			Expect(err).ToNot(HaveOccurred())

			Expect(instanceParams).To(Equal(providers.InstanceParameters{
				PreferredMaintenanceWindow: "",
				DailyBackupWindow:          "some-snapshot-window",
				MaxMemoryPolicy:            "some-maxmemory-policy",
				ActiveNodes:                []string{"cf-qwkec4pxhft6q-001"},
				PassiveNodes:               []string{"cf-qwkec4pxhft6q-002"},
				AutoFailover:               true,
				CacheParameters: []providers.CacheParameter{
					{
						ParameterName:  "some-parameter-name",
						ParameterValue: "some-parameter-value",
					},
					{
						ParameterName:  "maxmemory-policy",
						ParameterValue: "some-maxmemory-policy",
					},
				},
			}))

		})
		It("handles missing backup window", func() {
			describeReplicationGroupsOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
					{
						MemberClusters: []*string{aws.String("some-cluster-id")},
						SnapshotWindow: nil,
					},
				},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeReplicationGroupsOutput, nil)

			instanceParams, err := provider.GetInstanceParameters(ctx, instanceID)
			Expect(err).ToNot(HaveOccurred())

			Expect(instanceParams).To(Equal(providers.InstanceParameters{
				PreferredMaintenanceWindow: "some-maintenance-window",
				DailyBackupWindow:          "",
				MaxMemoryPolicy:            "some-maxmemory-policy",
				CacheParameters: []providers.CacheParameter{
					{
						ParameterName:  "some-parameter-name",
						ParameterValue: "some-parameter-value",
					},
					{
						ParameterName:  "maxmemory-policy",
						ParameterValue: "some-maxmemory-policy",
					},
				},
			}))

		})
		It("throws an error if the replication group has no members", func() {
			describeReplicationGroupsOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{{MemberClusters: nil}},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeReplicationGroupsOutput, nil)
			_, err := provider.GetInstanceParameters(ctx, instanceID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(fmt.Errorf("Replication group does not have any member clusters: %s", replicationGroupID)))
		})
		It("returns the correct error if the replication group does not exist", func() {
			mockElasticache.DescribeReplicationGroupsWithContextReturns(nil, awserr.New(elasticache.ErrCodeReplicationGroupNotFoundFault, "", errors.New("")))
			_, err := provider.GetInstanceParameters(ctx, instanceID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(fmt.Errorf("Replication group does not exist: %s", replicationGroupID)))
		})
		It("returns the error if another error occurs", func() {
			thrownError := errors.New("this is an error")
			mockElasticache.DescribeReplicationGroupsWithContextReturns(nil, thrownError)
			_, err := provider.GetInstanceParameters(ctx, instanceID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(thrownError))
		})
	})
})

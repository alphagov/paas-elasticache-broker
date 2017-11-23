package redis_test

import (
	"context"
	"errors"
	"regexp"

	"github.com/alphagov/paas-elasticache-broker/broker"
	. "github.com/alphagov/paas-elasticache-broker/redis"
	"github.com/alphagov/paas-elasticache-broker/redis/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	uuid "github.com/satori/go.uuid"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {
	var (
		mockElasticache *mocks.FakeElastiCache
		provider        *Provider
	)

	BeforeEach(func() {
		mockElasticache = &mocks.FakeElastiCache{}
		provider = NewProvider(mockElasticache, lager.NewLogger("logger"))
	})

	Context("when provisioning", func() {
		It("creates a cache parameter group and sets the parameters", func() {
			replicationGroupID := "cf-qwkec4pxhft6q"
			instanceID := "foobar"

			ctx := context.Background()

			provider.Provision(ctx, instanceID, broker.ProvisionParameters{
				Parameters: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			})

			Expect(mockElasticache.CreateCacheParameterGroupWithContextCallCount()).To(Equal(1))
			receivedCtx, receivedInput, _ := mockElasticache.CreateCacheParameterGroupWithContextArgsForCall(0)
			Expect(receivedCtx).To(Equal(ctx))
			Expect(receivedInput).To(Equal(&elasticache.CreateCacheParameterGroupInput{
				CacheParameterGroupFamily: aws.String("redis3.2"),
				CacheParameterGroupName:   aws.String(replicationGroupID),
				Description:               aws.String("Created by Cloud Foundry"),
			}))

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

		It("handles errors during parameter group creation", func() {
			createErr := errors.New("some error")
			mockElasticache.CreateCacheParameterGroupWithContextReturnsOnCall(0, nil, createErr)

			provisionErr := provider.Provision(context.Background(), "foobar", broker.ProvisionParameters{})
			Expect(provisionErr).To(MatchError(createErr))

			Expect(mockElasticache.ModifyCacheParameterGroupWithContextCallCount()).To(Equal(0))
			Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(0))
		})

		It("handles errors during parameter group update", func() {
			modifyErr := errors.New("some error")
			mockElasticache.ModifyCacheParameterGroupWithContextReturnsOnCall(0, nil, modifyErr)

			provisionErr := provider.Provision(context.Background(), "foobar", broker.ProvisionParameters{})
			Expect(provisionErr).To(MatchError(modifyErr))

			Expect(mockElasticache.CreateCacheParameterGroupWithContextCallCount()).To(Equal(1))
			Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(0))
		})

		It("creates the replication group", func() {
			replicationGroupID := "cf-qwkec4pxhft6q"
			ctx := context.Background()
			instanceID := "foobar"
			params := broker.ProvisionParameters{
				AuthToken:                  "test auth token",
				InstanceType:               "test instance type",
				CacheParameterGroupName:    replicationGroupID,
				SecurityGroupIds:           []string{"test sg1"},
				CacheSubnetGroupName:       "test subnet group",
				PreferredMaintenanceWindow: "test maintenance window",
				ReplicasPerNodeGroup:       2,
				ShardCount:                 1,
				SnapshotRetentionLimit:     0,
				Description:                "test desc",
				AutomaticFailoverEnabled:   true,
				Parameters:                 map[string]string{},
				Tags:                       map[string]string{},
			}
			provisionErr := provider.Provision(ctx, instanceID, params)
			Expect(provisionErr).NotTo(HaveOccurred())
			Expect(mockElasticache.CreateReplicationGroupWithContextCallCount()).To(Equal(1))

			passedCtx, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInput).To(Equal(&elasticache.CreateReplicationGroupInput{
				Tags: []*elasticache.Tag{},
				AtRestEncryptionEnabled:     aws.Bool(true),
				TransitEncryptionEnabled:    aws.Bool(true),
				AuthToken:                   aws.String("test auth token"),
				AutomaticFailoverEnabled:    aws.Bool(true),
				CacheNodeType:               aws.String("test instance type"),
				CacheParameterGroupName:     aws.String(replicationGroupID),
				SecurityGroupIds:            aws.StringSlice([]string{"test sg1"}),
				CacheSubnetGroupName:        aws.String("test subnet group"),
				Engine:                      aws.String("redis"),
				EngineVersion:               aws.String("3.2.6"),
				PreferredMaintenanceWindow:  aws.String("test maintenance window"),
				ReplicationGroupDescription: aws.String("test desc"),
				ReplicationGroupId:          aws.String(replicationGroupID),
				NumNodeGroups:               aws.Int64(1),
				NumCacheClusters:            aws.Int64(3),
			}))
		})

		It("sets the tags properly", func() {
			params := broker.ProvisionParameters{
				Tags: map[string]string{"tag1": "tag value1", "tag2": "tag value2"},
			}
			provider.Provision(context.Background(), "foobar", params)
			_, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
			Expect(passedInput.Tags).To(ConsistOf([]*elasticache.Tag{
				&elasticache.Tag{Key: aws.String("tag1"), Value: aws.String("tag value1")},
				&elasticache.Tag{Key: aws.String("tag2"), Value: aws.String("tag value2")},
			}))
		})

		It("uses ReplicasPerNodeGroup instead of NumCacheClusters if shard count is greater than one", func() {
			params := broker.ProvisionParameters{
				ShardCount:           2,
				ReplicasPerNodeGroup: 3,
			}
			provider.Provision(context.Background(), "foobar", params)

			_, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
			Expect(passedInput.NumCacheClusters).To(BeNil())
			Expect(passedInput.ReplicasPerNodeGroup).To(Equal(aws.Int64(3)))
		})

		It("sets the snapshot retention if it's greater than zero", func() {
			params := broker.ProvisionParameters{
				SnapshotRetentionLimit: 1,
			}
			provider.Provision(context.Background(), "foobar", params)

			_, passedInput, _ := mockElasticache.CreateReplicationGroupWithContextArgsForCall(0)
			Expect(passedInput.SnapshotRetentionLimit).To(Equal(aws.Int64(1)))
		})

		It("handles errors during replication group creation", func() {
			createErr := errors.New("some err")
			mockElasticache.CreateReplicationGroupWithContextReturnsOnCall(0, nil, createErr)

			provisionErr := provider.Provision(context.Background(), "foobar", broker.ProvisionParameters{})
			Expect(provisionErr).To(MatchError(createErr))
		})

	})

	Context("when deprovisioning", func() {

		It("deletes the replication group", func() {
			replicationGroupID := "cf-qwkec4pxhft6q"
			ctx := context.Background()
			instanceID := "foobar"
			params := broker.DeprovisionParameters{}
			deprovisionErr := provider.Deprovision(ctx, instanceID, params)
			Expect(deprovisionErr).ToNot(HaveOccurred())

			Expect(mockElasticache.DeleteReplicationGroupWithContextCallCount()).To(Equal(1))
			passedCtx, passedInput, _ := mockElasticache.DeleteReplicationGroupWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInput).To(Equal(&elasticache.DeleteReplicationGroupInput{
				ReplicationGroupId: aws.String(replicationGroupID),
			}))
		})

		It("sets a parameter for creating a final snapshot if final snapshot name is set", func() {
			params := broker.DeprovisionParameters{
				FinalSnapshotIdentifier: "test snapshot",
			}
			provider.Deprovision(context.Background(), "foobar", params)
			_, passedInput, _ := mockElasticache.DeleteReplicationGroupWithContextArgsForCall(0)
			Expect(passedInput.FinalSnapshotIdentifier).To(Equal(aws.String("test snapshot")))
		})

		It("handles errors during deleting the replication group", func() {
			deleteErr := errors.New("some error")
			mockElasticache.DeleteReplicationGroupWithContextReturnsOnCall(0, nil, deleteErr)

			deprovisionErr := provider.Deprovision(context.Background(), "foobar", broker.DeprovisionParameters{})
			Expect(deprovisionErr).To(MatchError(deleteErr))
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

			state, stateMessage, stateErr := provider.GetState(ctx, instanceID)

			Expect(state).To(Equal(broker.ServiceState("test status")))
			Expect(stateMessage).To(Equal("ElastiCache state is test status for cf-qwkec4pxhft6q"))
			Expect(stateErr).ToNot(HaveOccurred())

			Expect(mockElasticache.DescribeReplicationGroupsWithContextCallCount()).To(Equal(1))
			passedCtx, passedInput, _ := mockElasticache.DescribeReplicationGroupsWithContextArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInput).To(Equal(&elasticache.DescribeReplicationGroupsInput{
				ReplicationGroupId: aws.String(replicationGroupID),
			}))
		})

		It("handles errors from the AWS API", func() {
			describeErr := errors.New("some error")
			mockElasticache.DescribeReplicationGroupsWithContextReturns(nil, describeErr)

			_, _, stateErr := provider.GetState(context.Background(), "foobar")
			Expect(stateErr).To(MatchError(describeErr))
		})

		It("returns deleted status for a non-existing replication group", func() {
			describeErr := awserr.New(elasticache.ErrCodeReplicationGroupNotFoundFault, "some message", nil)
			mockElasticache.DescribeReplicationGroupsWithContextReturns(nil, describeErr)

			state, stateMessage, stateErr := provider.GetState(context.Background(), "foobar")
			Expect(state).To(Equal(broker.NonExisting))
			Expect(stateMessage).To(Equal("Replication group does not exist: cf-qwkec4pxhft6q"))
			Expect(stateErr).ToNot(HaveOccurred())
		})

		It("handles an empty replication group list from AWS", func() {
			describeOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeOutput, nil)
			_, _, stateErr := provider.GetState(context.Background(), "foobar")
			Expect(stateErr).To(MatchError("Invalid response from AWS: no cache clusters returned for cf-qwkec4pxhft6q"))
		})

		It("handles empty status from AWS", func() {
			describeOutput := &elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{{}},
			}
			mockElasticache.DescribeReplicationGroupsWithContextReturns(describeOutput, nil)
			_, _, stateErr := provider.GetState(context.Background(), "foobar")
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
})

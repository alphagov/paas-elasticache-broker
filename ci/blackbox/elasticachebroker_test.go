package integration_aws_test

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	redisclient "github.com/garyburd/redigo/redis"
	uuid "github.com/satori/go.uuid"

	"github.com/alphagov/paas-elasticache-broker/ci/helpers"
	"github.com/alphagov/paas-elasticache-broker/providers/redis"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	INSTANCE_CREATE_TIMEOUT = 30 * time.Minute
	SERVICE_REDIS_UUID      = "d235edcf-8790-444a-b6e1-35e3c91a82c0"
	PLAN_MICRO_UUID         = "94767b71-2b9c-4960-a4f8-77b81a96f7e0"
)

var _ = Describe("ElastiCache Broker Daemon", func() {

	Describe("Broker lifecycle", func() {

		var (
			planID             = PLAN_MICRO_UUID
			serviceID          = SERVICE_REDIS_UUID
			instanceID         string
			appID              string
			bindingID          string
			snapshotName       string
			restoredInstanceID string
			credentials        map[string]interface{}
			conn               redisclient.Conn
		)

		// FIXME: if the broker ends up destroying parameter groups itself, this can be removed.
		AfterEach(func() {
			if instanceID != "" {
				paramGroupName := redis.GenerateReplicationGroupName(instanceID)
				helpers.DestroyParameterGroup(&paramGroupName, awsSession)
			}
		})

		It("should provision and deprovision redis", func() {
			instanceID = uuid.NewV4().String()
			appID = uuid.NewV4().String()
			bindingID = uuid.NewV4().String()
			snapshotName = fmt.Sprintf("test-%s-snapshot", instanceID)
			restoredInstanceID = uuid.NewV4().String()

			brokerAPIClient.AcceptsIncomplete = true

			elasticacheService := elasticache.New(awsSession)

			By("provisioning", func() {
				code, operation, err := brokerAPIClient.ProvisionInstance(instanceID, serviceID, planID, "{}")
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(202))
				state := pollForOperationCompletion(instanceID, serviceID, planID, operation, "succeeded")
				Expect(state).To(Equal("succeeded"))
			})

			defer By("deprovisioning", func() {
				code, operation, err := brokerAPIClient.DeprovisionInstance(instanceID, serviceID, planID)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(202))
				state := pollForOperationCompletion(instanceID, serviceID, planID, operation, "gone")
				Expect(state).To(Equal("gone"))
			})

			By("checking that the new replication group has the right tags", func() {
				replicationGroupID := redis.GenerateReplicationGroupName(instanceID)
				replicationGroupARN, err := helpers.ReplicationGroupARN(awsSession, replicationGroupID)
				Expect(err).ToNot(HaveOccurred())

				tagList, err := elasticacheService.ListTagsForResource(
					&elasticache.ListTagsForResourceInput{
						ResourceName: aws.String(replicationGroupARN),
					})
				Expect(err).ToNot(HaveOccurred())

				Expect(tagList.TagList).To(ConsistOf(
					&elasticache.Tag{
						Key:   aws.String("created-by"),
						Value: aws.String(brokerName),
					},
					&elasticache.Tag{
						Key:   aws.String("service-id"),
						Value: aws.String(serviceID),
					},
					&elasticache.Tag{
						Key:   aws.String("plan-id"),
						Value: aws.String(planID),
					},
					&elasticache.Tag{
						Key:   aws.String("organization-id"),
						Value: aws.String(brokerAPIClient.DefaultOrganizationID),
					},
					&elasticache.Tag{
						Key:   aws.String("space-id"),
						Value: aws.String(brokerAPIClient.DefaultSpaceID),
					},
					&elasticache.Tag{
						Key:   aws.String("instance-id"),
						Value: aws.String(instanceID),
					},
				))
			})

			By("checking that the cache parameter group has been set", func() {
				replicationGroupID := redis.GenerateReplicationGroupName(instanceID)
				res, err := elasticacheService.DescribeCacheParameters(&elasticache.DescribeCacheParametersInput{
					CacheParameterGroupName: aws.String(replicationGroupID),
				})
				Expect(err).ToNot(HaveOccurred())
				found := 0
				for _, p := range res.Parameters {
					if *p.ParameterName == "maxmemory-policy" {
						found++
						Expect(*p.ParameterValue).To(Equal("volatile-lru"))
					}
					if *p.ParameterName == "reserved-memory" {
						found++
						Expect(*p.ParameterValue).To(Equal("0"))
					}
				}
				Expect(found).To(Equal(2))
			})

			By("updating parameters", func() {
				updateParams := fmt.Sprintf(`{"maxmemory_policy": "noeviction"}`)
				oldPlanID := planID
				oldServiceID := serviceID
				code, _, err := brokerAPIClient.UpdateInstance(instanceID, serviceID, planID, oldPlanID, oldServiceID, brokerAPIClient.DefaultOrganizationID, brokerAPIClient.DefaultSpaceID, updateParams)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(200))
			})

			By("checking that the cache parameter group has been updated", func() {
				replicationGroupID := redis.GenerateReplicationGroupName(instanceID)
				res, err := elasticacheService.DescribeCacheParameters(&elasticache.DescribeCacheParametersInput{
					CacheParameterGroupName: aws.String(replicationGroupID),
				})
				Expect(err).ToNot(HaveOccurred())
				found := 0
				for _, p := range res.Parameters {
					if *p.ParameterName == "maxmemory-policy" {
						found++
						Expect(*p.ParameterValue).To(Equal("noeviction"))
					}
				}
				Expect(found).To(Equal(1))
			})

			By("binding a resource to the service", func() {
				code, bindingResponse, err := brokerAPIClient.Bind(instanceID, serviceID, planID, appID, bindingID)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(201))
				Expect(bindingResponse).ToNot(BeNil())
				credentials = bindingResponse.Credentials
				Expect(credentials).ToNot(BeNil())
				Expect(credentials).To(HaveKey("host"))
				Expect(credentials).To(HaveKey("port"))
				Expect(credentials).To(HaveKey("name"))
				Expect(credentials).To(HaveKey("password"))
				Expect(credentials).To(HaveKey("uri"))
				Expect(credentials).To(HaveKeyWithValue("tls_enabled", true))
			})

			defer By("unbinding the service", func() {
				code, err := brokerAPIClient.Unbind(instanceID, serviceID, planID, bindingID)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(200))
			})

			By("ensuring binding credentials allow connecting to the service with the url", func() {
				uri, ok := credentials["uri"].(string)
				Expect(ok).To(BeTrue(), fmt.Sprintf("uri is invalid: %v", credentials["uri"]))

				var err error
				conn, err = redisclient.DialURL(uri)
				Expect(err).ToNot(HaveOccurred())
			})

			defer By("disconnecting from the service", func() {
				err := conn.Close()
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring binding credentials allow writing data", func() {
				_, err := conn.Do("SET", "hello", "world")
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring binding credentials allow reading data", func() {
				s, err := redisclient.String(conn.Do("GET", "hello"))
				Expect(err).ToNot(HaveOccurred())
				Expect(s).To(Equal("world"))
			})

			By("ensuring binding credentials allow deleting data", func() {
				_, err := conn.Do("DEL", "hello")
				Expect(err).ToNot(HaveOccurred())
				ok, _ := redisclient.Bool(conn.Do("EXISTS", "hello"))
				Expect(ok).To(Equal(false))
			})

			By("ensuring binding credentials allow connecting to the service with the connection parameters", func() {
				host, ok := credentials["host"].(string)
				Expect(ok).To(BeTrue(), fmt.Sprintf("host is invalid: %v", credentials["host"]))
				port, ok := convertInterfaceToInt64(credentials["port"])
				Expect(ok).To(BeTrue(), fmt.Sprintf("port is invalid: %v", credentials["port"]))
				password, ok := credentials["password"].(string)
				Expect(ok).To(BeTrue(), fmt.Sprintf("password is invalid: %v", credentials["password"]))

				conn2, err := redisclient.Dial(
					"tcp",
					fmt.Sprintf("%s:%d", host, port),
					redisclient.DialPassword(password),
					redisclient.DialUseTLS(true),
				)
				Expect(err).ToNot(HaveOccurred())

				err = conn2.Close()
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring a client is not able to connect without TLS", func() {
				host, ok := credentials["host"].(string)
				Expect(ok).To(BeTrue(), fmt.Sprintf("host is invalid: %v", credentials["host"]))
				port, ok := convertInterfaceToInt64(credentials["port"])
				Expect(ok).To(BeTrue(), fmt.Sprintf("port is invalid: %v", credentials["port"]))
				password, ok := credentials["password"].(string)
				Expect(ok).To(BeTrue(), fmt.Sprintf("password is invalid: %v", credentials["password"]))

				_, err := redisclient.Dial(
					"tcp",
					fmt.Sprintf("%s:%d", host, port),
					redisclient.DialPassword(password),
					redisclient.DialUseTLS(false),
				)
				Expect(err).To(HaveOccurred())
			})

			By("preparing a snapshot", func() {
				_, err := conn.Do("SET", "should-be-present-on-restored-instance", "yup")
				Expect(err).ToNot(HaveOccurred())

				replicationGroupID := redis.GenerateReplicationGroupName(instanceID)
				_, err = elasticacheService.CreateSnapshot(&elasticache.CreateSnapshotInput{
					ReplicationGroupId: aws.String(replicationGroupID),
					SnapshotName:       aws.String(snapshotName),
				})
				Expect(err).ToNot(HaveOccurred())

				fmt.Fprint(GinkgoWriter, "Polling for snapshot preparation to complete")
				Eventually(func() string {
					fmt.Fprint(GinkgoWriter, ".")
					describeSnapshotsOutput, err := elasticacheService.DescribeSnapshots(&elasticache.DescribeSnapshotsInput{
						ReplicationGroupId: aws.String(replicationGroupID),
						SnapshotName:       aws.String(snapshotName),
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(describeSnapshotsOutput.Snapshots).To(HaveLen(1))
					snapshotState := describeSnapshotsOutput.Snapshots[0].SnapshotStatus
					return aws.StringValue(snapshotState)
				}, 10*time.Minute, 10*time.Second).Should(Equal("available"))
				fmt.Fprintf(GinkgoWriter, "done.\n")
			})

			By("provisioning from a snapshot", func() {
				provisionParams := fmt.Sprintf(`{"restore_from_latest_snapshot_of": "%s"}`, instanceID)
				code, operation, err := brokerAPIClient.ProvisionInstance(restoredInstanceID, serviceID, planID, provisionParams)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(202))
				state := pollForOperationCompletion(restoredInstanceID, serviceID, planID, operation, "succeeded")
				Expect(state).To(Equal("succeeded"))
			})

			defer By("deprovisioning the restored instance", func() {
				code, operation, err := brokerAPIClient.DeprovisionInstance(restoredInstanceID, serviceID, planID)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(202))
				state := pollForOperationCompletion(restoredInstanceID, serviceID, planID, operation, "gone")
				Expect(state).To(Equal("gone"))
			})

			By("checking that the restored replication group has the right tags", func() {
				replicationGroupID := redis.GenerateReplicationGroupName(restoredInstanceID)
				replicationGroupARN, err := helpers.ReplicationGroupARN(awsSession, replicationGroupID)
				Expect(err).ToNot(HaveOccurred())

				tagList, err := elasticacheService.ListTagsForResource(
					&elasticache.ListTagsForResourceInput{
						ResourceName: aws.String(replicationGroupARN),
					})
				Expect(err).ToNot(HaveOccurred())

				Expect(tagList.TagList).To(ConsistOf(
					&elasticache.Tag{
						Key:   aws.String("created-by"),
						Value: aws.String(brokerName),
					},
					&elasticache.Tag{
						Key:   aws.String("service-id"),
						Value: aws.String(serviceID),
					},
					&elasticache.Tag{
						Key:   aws.String("plan-id"),
						Value: aws.String(planID),
					},
					&elasticache.Tag{
						Key:   aws.String("organization-id"),
						Value: aws.String(brokerAPIClient.DefaultOrganizationID),
					},
					&elasticache.Tag{
						Key:   aws.String("space-id"),
						Value: aws.String(brokerAPIClient.DefaultSpaceID),
					},
					&elasticache.Tag{
						Key:   aws.String("instance-id"),
						Value: aws.String(restoredInstanceID),
					},
				))
			})

			By("having restored data in the restored instance", func() {
				code, bindingResponse, err := brokerAPIClient.Bind(restoredInstanceID, serviceID, planID, appID, bindingID)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(201))
				Expect(bindingResponse).ToNot(BeNil())
				credentials2 := bindingResponse.Credentials

				uri, _ := credentials2["uri"].(string)
				conn2, err := redisclient.DialURL(uri)
				Expect(err).ToNot(HaveOccurred())

				s, err := redisclient.String(conn2.Do("GET", "should-be-present-on-restored-instance"))
				Expect(err).ToNot(HaveOccurred())
				Expect(s).To(Equal("yup"))

				err = conn2.Close()
				Expect(err).ToNot(HaveOccurred())
			})

			defer By("unbinding the restored instance", func() {
				code, err := brokerAPIClient.Unbind(restoredInstanceID, serviceID, planID, bindingID)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(200))
			})

		})

	})

})

func pollForOperationCompletion(instanceID, serviceID, planID, operation, desiredState string) string {
	var state string
	var err error

	fmt.Fprint(GinkgoWriter, "Polling for Instance Operation to complete")
	Eventually(
		func() string {
			fmt.Fprint(GinkgoWriter, ".")
			state, err = brokerAPIClient.GetLastOperationState(instanceID, serviceID, planID, operation)
			Expect(err).ToNot(HaveOccurred())
			return state
		},
		INSTANCE_CREATE_TIMEOUT,
		15*time.Second,
	).Should(
		SatisfyAny(
			Equal(desiredState),
			Equal("failed"),
		),
	)

	fmt.Fprintf(GinkgoWriter, "done. Final state: %s.\n", state)
	return state
}

func convertInterfaceToInt64(val interface{}) (int64, bool) {
	switch v := val.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

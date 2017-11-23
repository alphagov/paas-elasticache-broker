package integration_aws_test

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-elasticache-broker/ci/helpers"
	"github.com/alphagov/paas-elasticache-broker/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	uuid "github.com/satori/go.uuid"

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
			planID     = PLAN_MICRO_UUID
			serviceID  = SERVICE_REDIS_UUID
			instanceID string
		)

		// FIXME: if the broker ends up destroying parameter groups itself, this can be removed.
		AfterEach(func() {
			if instanceID != "" {
				awsSession := session.Must(session.NewSession(&aws.Config{
					Region: aws.String(elastiCacheBrokerConfig.Region)},
				))
				paramGroupName := redis.GenerateReplicationGroupName(instanceID)
				helpers.DestroyParameterGroup(&paramGroupName, awsSession)
			}
		})

		It("should provision and deprovision redis", func() {
			instanceID = uuid.NewV4().String()

			brokerAPIClient.AcceptsIncomplete = true

			By("provisioning")
			code, operation, err := brokerAPIClient.ProvisionInstance(instanceID, serviceID, planID, "{}")
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(202))
			state := pollForOperationCompletion(instanceID, serviceID, planID, operation, "succeeded")
			Expect(state).To(Equal("succeeded"))

			By("deprovisioning")
			code, operation, err = brokerAPIClient.DeprovisionInstance(instanceID, serviceID, planID)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(202))
			state = pollForOperationCompletion(instanceID, serviceID, planID, operation, "gone")
			Expect(state).To(Equal("gone"))
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

package integration_aws_test

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-elasticache-broker/ci/helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	redisclient "github.com/garyburd/redigo/redis"
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
			planID      = PLAN_MICRO_UUID
			serviceID   = SERVICE_REDIS_UUID
			instanceID  string
			appID       string
			bindingID   string
			credentials map[string]interface{}
			conn        redisclient.Conn
		)

		// FIXME: if the broker ends up destroying parameter groups itself, this can be removed.
		AfterEach(func() {
			if instanceID != "" {
				awsSession := session.Must(session.NewSession(&aws.Config{
					Region: aws.String(elastiCacheBrokerConfig.Region)},
				))
				paramGroupName := providers.GenerateReplicationGroupName(instanceID)
				helpers.DestroyParameterGroup(&paramGroupName, awsSession)
			}
		})

		It("should provision and deprovision redis", func() {
			instanceID = uuid.NewV4().String()
			appID = uuid.NewV4().String()
			bindingID = uuid.NewV4().String()

			brokerAPIClient.AcceptsIncomplete = true

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

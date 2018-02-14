package integration_aws_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/phayes/freeport"

	"github.com/alphagov/paas-elasticache-broker/broker"
	. "github.com/alphagov/paas-elasticache-broker/ci/helpers"
)

const clusterNamePrefix = "cf-broker-test"

var (
	elastiCacheBrokerPort    int
	elastiCacheBrokerUrl     string
	elastiCacheBrokerSession *gexec.Session

	brokerAPIClient *BrokerAPIClient

	elastiCacheBrokerConfig broker.Config

	awsSession                 *session.Session
	elastiCacheSubnetGroupName *string
	ec2SecurityGroupID         *string
	brokerName                 string
)

func TestSuite(t *testing.T) {
	BeforeSuite(func() {
		cmdPath, err := gexec.Build("github.com/alphagov/paas-elasticache-broker")
		Expect(err).NotTo(HaveOccurred())

		fmt.Fprintln(GinkgoWriter, os.Environ())

		originalConfig, err := broker.LoadConfig("./config.json")

		awsSession = session.Must(session.NewSession(&aws.Config{
			Region: aws.String(originalConfig.Region)},
		))

		elastiCacheSubnetGroupName, err = CreateSubnetGroup(
			clusterNamePrefix,
			awsSession,
		)
		Expect(err).NotTo(HaveOccurred())

		ec2SecurityGroupID, err = CreateSecurityGroup(
			clusterNamePrefix,
			awsSession,
		)
		Expect(err).NotTo(HaveOccurred())

		brokerName = fmt.Sprintf("%s-%s",
			originalConfig.BrokerName,
			uuid.NewV4().String(),
		)

		var configFileName string
		elastiCacheBrokerConfig, configFileName, err = WriteCustomConfig(
			originalConfig,
			brokerName,
			*elastiCacheSubnetGroupName,
			*ec2SecurityGroupID,
		)
		defer os.Remove(configFileName)

		elastiCacheBrokerPort = freeport.GetPort()
		command := exec.Command(
			cmdPath,
			fmt.Sprintf("-port=%d", elastiCacheBrokerPort),
			fmt.Sprintf("-config=%s", configFileName),
		)
		elastiCacheBrokerSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(elastiCacheBrokerSession, 10*time.Second).
			Should(gbytes.Say(fmt.Sprintf("ElastiCache Service Broker started on port %d", elastiCacheBrokerPort)))

		elastiCacheBrokerUrl = fmt.Sprintf("http://localhost:%d", elastiCacheBrokerPort)

		brokerAPIClient = NewBrokerAPIClient(
			elastiCacheBrokerUrl,
			elastiCacheBrokerConfig.Username,
			elastiCacheBrokerConfig.Password,
			"test-organization-id",
			"space-id",
		)
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()

		awsSession := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(elastiCacheBrokerConfig.Region)},
		))
		if ec2SecurityGroupID != nil {
			Expect(DestroySecurityGroup(ec2SecurityGroupID, awsSession)).To(Succeed())
		}
		if elastiCacheSubnetGroupName != nil {
			Expect(DestroySubnetGroup(elastiCacheSubnetGroupName, awsSession)).To(Succeed())
		}
		elastiCacheBrokerSession.Kill()
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "ElastiCache Broker Integration Suite")
}

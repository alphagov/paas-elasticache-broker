package integration_aws_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/phayes/freeport"

	"github.com/alphagov/paas-elasticache-broker/broker"
	. "github.com/alphagov/paas-elasticache-broker/ci/helpers"
	test "github.com/alphagov/paas-elasticache-broker/test"
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

		certPEM, keyPEM, caPEM, err := test.GenerateTestCert()
		Expect(err).NotTo(HaveOccurred())

		configBase, err := test.SetTLSConfigOptions("./config.json", certPEM, keyPEM)
		defer os.Remove(configBase)

		originalConfig, err := broker.LoadConfig(configBase)
		Expect(err).NotTo(HaveOccurred())

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
		elastiCacheBrokerConfig, configFileName, _ = WriteCustomConfig(
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

		elastiCacheBrokerUrl = fmt.Sprintf("https://localhost:%d", elastiCacheBrokerPort)

		brokerAPIClient = NewBrokerAPIClient(
			elastiCacheBrokerUrl,
			elastiCacheBrokerConfig.Username,
			elastiCacheBrokerConfig.Password,
			"test-organization-id",
			"space-id",
			caPEM,
		)
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()

		awsSession := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(elastiCacheBrokerConfig.Region)},
		))
		if elastiCacheSubnetGroupName != nil {
			Expect(DestroySubnetGroup(elastiCacheSubnetGroupName, awsSession)).To(Succeed())
		}

		fmt.Println("Waiting for network interfaces associated with security group to be deleted")
		Eventually(func() int {
			count, err := CountSGAssociatedInterfaces(ec2SecurityGroupID, awsSession)
			Expect(err).ToNot(HaveOccurred())
			fmt.Printf("Number of network interfaces associated with SG %s: %d\n", *ec2SecurityGroupID, count)
			return count
		}, 60 * time.Minute, 10 * time.Second).Should(Equal(0))

		if ec2SecurityGroupID != nil {
			Expect(DestroySecurityGroup(ec2SecurityGroupID, awsSession)).To(Succeed())
		}

		if elastiCacheBrokerSession != nil {
			elastiCacheBrokerSession.Kill()
		}
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "ElastiCache Broker Integration Suite")
}

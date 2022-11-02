package broker_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/alphagov/paas-elasticache-broker/broker"
)

var _ = Describe("Config", func() {
	var (
		config Config

		validConfig = Config{
			LogLevel:             "log_level",
			Username:             "username",
			Password:             "password",
			Region:               "region",
			BrokerName:           "broker_name",
			CacheSubnetGroupName: "cache_subnet_group_name",
			VpcSecurityGroupIds:  []string{"vpc_security_group_id"},
			Catalog: brokerapi.CatalogResponse{
				Services: []brokerapi.Service{
					{
						ID: "service1",
						Plans: []brokerapi.ServicePlan{
							{ID: "plan1"},
						},
					},
				},
			},
			PlanConfigs: map[string]PlanConfig{
				"plan1": {},
			},
			KmsKeyID:           "my-kms-key",
			SecretsManagerPath: "elasticache-broker-test",
		}
	)

	Describe("Validate", func() {
		BeforeEach(func() {
			config = validConfig
		})

		It("does not return error if all sections are valid", func() {
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("requires a log level", func() {
			config.LogLevel = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a username", func() {
			config.Username = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a password", func() {
			config.Password = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a region", func() {
			config.Region = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a broker name", func() {
			config.BrokerName = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires at least one VPC security group ID", func() {
			config.VpcSecurityGroupIds = []string{}
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a cache subnet group name", func() {
			config.CacheSubnetGroupName = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a kms key id", func() {
			config.KmsKeyID = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		It("requires a secrets manager path", func() {
			config.SecretsManagerPath = ""
			Expect(config.Validate()).NotTo(Succeed())
		})

		Context("mapping PlanConfigs to Plans", func() {
			It("errors if the plan config ID does not map to a plan ID in the catalog", func() {
				config.PlanConfigs["this-is-not-in-the-catalog"] = PlanConfig{}

				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("PlanConfig this-is-not-in-the-catalog not found in catalog"))
			})

			It("errors if a plan in the catalog doesn't have a plan config", func() {
				planWithoutConfig := brokerapi.ServicePlan{ID: "plan-without-config"}
				config.Catalog.Services[0].Plans = append(config.Catalog.Services[0].Plans, planWithoutConfig)

				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Plan with ID plan-without-config has no PlanConfig"))
			})
		})
	})
})

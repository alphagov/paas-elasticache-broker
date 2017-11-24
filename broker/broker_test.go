package broker_test

import (
	"context"
	"errors"
	"fmt"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/alphagov/paas-elasticache-broker/broker/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Broker", func() {
	var validConfig broker.Config

	BeforeEach(func() {
		validConfig = broker.Config{
			AuthTokenSeed:        "super-secret",
			BrokerName:           "Broker McBrokerface",
			VpcSecurityGroupIds:  []string{"vpc_security_group_id"},
			CacheSubnetGroupName: "cache-subnet-group-name",
			Catalog: brokerapi.CatalogResponse{
				Services: []brokerapi.Service{
					brokerapi.Service{
						ID:   "service1",
						Name: "service1",
						Plans: []brokerapi.ServicePlan{
							brokerapi.ServicePlan{
								ID:   "plan1",
								Name: "plan1",
							},
						},
					},
				},
			},
			PlanConfigs: map[string]broker.PlanConfig{
				"plan1": broker.PlanConfig{
					InstanceType: "t2.micro",
					Parameters: map[string]string{
						"maxmemory-policy": "volatile-lru",
						"reserved-memory":  "0",
					},
				},
			},
		}
	})

	Describe("Provision", func() {
		var validProvisionDetails brokerapi.ProvisionDetails

		BeforeEach(func() {
			validProvisionDetails = brokerapi.ProvisionDetails{
				ServiceID:        "service1",
				PlanID:           "plan1",
				OrganizationGUID: "org-guid",
				SpaceGUID:        "space-guid",
			}
		})

		It("logs a debug message when provision begins", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := broker.New(validConfig, &mocks.FakeProvider{}, logger)

			b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(log).To(gbytes.Say("provision-start"))
		})

		It("errors if async isn't allowed", func() {
			b := broker.New(broker.Config{}, &mocks.FakeProvider{}, lager.NewLogger("logger"))
			asyncAllowed := false

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, asyncAllowed)

			Expect(err).To(MatchError("This service plan requires client support for asynchronous service operations."))
		})

		It("errors if the plan config cannot be retrieved", func() {
			b := broker.New(broker.Config{}, &mocks.FakeProvider{}, lager.NewLogger("logger"))

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(err).To(MatchError("service plan plan1: no plan found"))
		})

		It("sets a deadline by which the AWS request should complete", func() {
			fakeProvider := &mocks.FakeProvider{}
			logger := lager.NewLogger("logger")
			b := broker.New(validConfig, fakeProvider, logger)

			b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			receivedContext, _, _ := fakeProvider.ProvisionArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct parameters to the Provider", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			_, instanceID, params := fakeProvider.ProvisionArgsForCall(0)

			expectedParams := broker.ProvisionParameters{
				InstanceType:               validConfig.PlanConfigs["plan1"].InstanceType,
				CacheParameterGroupName:    "default.redis3.2",
				SecurityGroupIds:           validConfig.VpcSecurityGroupIds,
				CacheSubnetGroupName:       validConfig.CacheSubnetGroupName,
				PreferredMaintenanceWindow: "sun:23:00-mon:01:30",
				ReplicasPerNodeGroup:       0,
				ShardCount:                 1,
				SnapshotRetentionLimit:     0,
				Description:                "Cloud Foundry service",
				AutomaticFailoverEnabled:   false,
				Parameters:                 validConfig.PlanConfigs["plan1"].Parameters,
				Tags: map[string]string{
					"created-by":      validConfig.BrokerName,
					"service-id":      validProvisionDetails.ServiceID,
					"plan-id":         validProvisionDetails.PlanID,
					"organization-id": validProvisionDetails.OrganizationGUID,
					"space-id":        validProvisionDetails.SpaceGUID,
				},
			}

			Expect(instanceID).To(Equal("instanceid"))
			Expect(params).To(Equal(expectedParams))
		})

		It("errors if provisioning fails", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			fakeProvider.ProvisionReturns(errors.New("ERROR PROVISIONING"))

			_, err := b.Provision(context.Background(), "myinstance", validProvisionDetails, true)

			Expect(err).To(MatchError("provider redis for plan plan1: ERROR PROVISIONING"))
		})

		It("logs a debug message when provisioning succeeds", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := broker.New(validConfig, &mocks.FakeProvider{}, logger)

			b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(log).To(gbytes.Say("provision-success"))
		})

		It("returns the provisioned service spec", func() {
			b := broker.New(validConfig, &mocks.FakeProvider{}, lager.NewLogger("logger"))
			Expect(b.Provision(context.Background(), "instanceid", validProvisionDetails, true)).
				To(Equal(brokerapi.ProvisionedServiceSpec{IsAsync: true}))
		})
	})

	Describe("Deprovision", func() {
		var validDeprovisionDetails brokerapi.DeprovisionDetails

		BeforeEach(func() {
			validDeprovisionDetails = brokerapi.DeprovisionDetails{
				PlanID:    "myplan-id",
				ServiceID: "myservice-id",
			}
		})

		It("logs a debug message when deprovisioning begins", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := broker.New(validConfig, &mocks.FakeProvider{}, logger)

			b.Deprovision(context.Background(), "instanceid", brokerapi.DeprovisionDetails{}, true)

			Expect(log).To(gbytes.Say("deprovision-start"))
		})

		It("errors if async isn't allowed", func() {
			b := broker.New(broker.Config{}, &mocks.FakeProvider{}, lager.NewLogger("logger"))
			asyncAllowed := false

			_, err := b.Deprovision(context.Background(), "instanceid", brokerapi.DeprovisionDetails{}, asyncAllowed)

			Expect(err).To(MatchError("This service plan requires client support for asynchronous service operations."))
		})

		It("sets a deadline by which the AWS request should complete", func() {
			fakeProvider := &mocks.FakeProvider{}
			logger := lager.NewLogger("logger")
			b := broker.New(validConfig, fakeProvider, logger)

			b.Deprovision(context.Background(), "instanceid", brokerapi.DeprovisionDetails{}, true)

			Expect(fakeProvider.DeprovisionCallCount()).To(Equal(1))
			receivedContext, _, _ := fakeProvider.DeprovisionArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct parameters to the Provider", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			b.Deprovision(context.Background(), "instanceid", validDeprovisionDetails, true)

			Expect(fakeProvider.DeprovisionCallCount()).To(Equal(1))
			_, instanceID, params := fakeProvider.DeprovisionArgsForCall(0)

			expectedParams := broker.DeprovisionParameters{}

			Expect(instanceID).To(Equal("instanceid"))
			Expect(params).To(Equal(expectedParams))
		})

		It("errors if deprovisioning fails", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			fakeProvider.DeprovisionReturns(errors.New("foobar"))

			_, err := b.Deprovision(context.Background(), "myinstance", validDeprovisionDetails, true)

			Expect(err).To(MatchError("provider redis for plan myplan-id: foobar"))
		})

		It("logs a debug message when deprovisioning succeeds", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := broker.New(validConfig, &mocks.FakeProvider{}, logger)

			b.Deprovision(context.Background(), "instanceid", brokerapi.DeprovisionDetails{}, true)

			Expect(log).To(gbytes.Say("deprovision-success"))
		})

		It("returns the deprovisioned service spec", func() {
			b := broker.New(validConfig, &mocks.FakeProvider{}, lager.NewLogger("logger"))
			Expect(b.Deprovision(context.Background(), "instanceid", validDeprovisionDetails, true)).
				To(Equal(brokerapi.DeprovisionServiceSpec{IsAsync: true}))
		})
	})

	Describe("LastOperation", func() {
		It("returns last operation data when the instance is available", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns(broker.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			Expect(b.LastOperation(context.Background(), "instanceid", "tellmeaboutprovision")).
				To(Equal(brokerapi.LastOperation{
					State:       brokerapi.Succeeded,
					Description: "i love brokers",
				}))
		})

		It("sets a deadline by which the AWS request should complete", func() {
			fakeProvider := &mocks.FakeProvider{}
			logger := lager.NewLogger("logger")
			b := broker.New(validConfig, fakeProvider, logger)

			b.LastOperation(context.Background(), "instanceid", "plztellme")

			Expect(fakeProvider.GetStateCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.GetStateArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("logs a debug message when starting to get the last operation", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := broker.New(validConfig, &mocks.FakeProvider{}, logger)

			b.LastOperation(context.Background(), "instanceid", "tellmeaboutprovision")

			Expect(log).To(gbytes.Say("last-operation"))
		})

		It("errors if getting the state fails", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			fakeProvider.GetStateReturns("", "", errors.New("foobar"))

			_, err := b.LastOperation(context.Background(), "myinstance", "opdata")

			Expect(err).To(MatchError("error getting state for myinstance: foobar"))
		})

		It("errors if the instance doesn't exist", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			fakeProvider.GetStateReturns(broker.NonExisting, "it'sgoneya'll", nil)

			_, err := b.LastOperation(context.Background(), "myinstance", "opdata")

			Expect(err).To(MatchError("instance does not exist"))
		})

		It("logs an error and reports 'in progress' if it receives an unknown state from the provider", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.ERROR))
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns("some-unknown-state", "", nil)
			b := broker.New(validConfig, fakeProvider, logger)

			b.LastOperation(context.Background(), "instanceid", "tellmeaboutprovision")

			Expect(log).To(gbytes.Say("Unknown service state: some-unknown-state"))
		})
	})

	Describe("state mapping from AWS to brokerapi package", func() {
		DescribeTable("known states",
			func(from broker.ServiceState, to brokerapi.LastOperationState) {
				Expect(broker.ProviderStatesMapping(from)).To(Equal(to))
			},
			Entry("available => succeeded", broker.Available, brokerapi.Succeeded),
			Entry("create-failed => failure", broker.CreateFailed, brokerapi.Failed),
			Entry("creating => in progress", broker.Creating, brokerapi.InProgress),
			Entry("modifying => in progress", broker.Modifying, brokerapi.InProgress),
			Entry("deleting => in progress", broker.Deleting, brokerapi.InProgress),
			Entry("snapshotting => in progress", broker.Snapshotting, brokerapi.InProgress),
		)

		It("errors on unknown state and returns 'in progress'", func() {
			brokerAPIState, err := broker.ProviderStatesMapping("some-unknown-state")
			Expect(err).To(HaveOccurred())
			Expect(brokerAPIState).To(Equal(brokerapi.InProgress))
		})
	})

	Context("when binding a service instance", func() {
		It("gets credentials from the provider", func() {
			ctx := context.Background()
			instanceID := "test-instance"
			bindingID := "test-binding"
			expectedCredentials := &broker.Credentials{
				Host: "test-host",
			}

			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GenerateCredentialsReturnsOnCall(0, expectedCredentials, nil)

			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			binding, err := b.Bind(ctx, instanceID, bindingID, brokerapi.BindDetails{})

			Expect(err).ToNot(HaveOccurred())
			Expect(binding).To(Equal(brokerapi.Binding{Credentials: expectedCredentials}))

			Expect(fakeProvider.GenerateCredentialsCallCount()).To(Equal(1))
			passedCtx, passedInstanceId, passedBindingId := fakeProvider.GenerateCredentialsArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInstanceId).To(Equal(instanceID))
			Expect(passedBindingId).To(Equal(bindingID))
		})

		It("handles errors from the provider", func() {
			bindErr := fmt.Errorf("some error")
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GenerateCredentialsReturnsOnCall(0, nil, bindErr)

			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			binding, err := b.Bind(context.Background(), "test-instance", "test-binding", brokerapi.BindDetails{})

			Expect(err).To(MatchError(bindErr))
			Expect(binding).To(Equal(brokerapi.Binding{}))
		})
	})

	Context("when unbinding a service instance", func() {
		It("revokes the credentials in the provider", func() {
			ctx := context.Background()
			instanceID := "test-instance"
			bindingID := "test-binding"

			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.RevokeCredentialsReturnsOnCall(0, nil)

			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			err := b.Unbind(ctx, instanceID, bindingID, brokerapi.UnbindDetails{})

			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.RevokeCredentialsCallCount()).To(Equal(1))
			passedCtx, passedInstanceId, passedBindingId := fakeProvider.RevokeCredentialsArgsForCall(0)
			Expect(passedCtx).To(Equal(ctx))
			Expect(passedInstanceId).To(Equal(instanceID))
			Expect(passedBindingId).To(Equal(bindingID))
		})

		It("handles errors from the provuder", func() {
			unbindErr := fmt.Errorf("some error")
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.RevokeCredentialsReturnsOnCall(0, unbindErr)

			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			err := b.Unbind(context.Background(), "test-instance", "test-binding", brokerapi.UnbindDetails{})

			Expect(err).To(MatchError(unbindErr))
		})
	})
})

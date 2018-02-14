package broker_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/alphagov/paas-elasticache-broker/providers"
	"github.com/alphagov/paas-elasticache-broker/providers/mocks"
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
					ShardCount:   1,
					Parameters: map[string]string{
						"maxmemory-policy": "volatile-lru",
						"reserved-memory":  "0",
					},
				},
			},
		}
	})

	Describe("Provision", func() {
		var (
			validProvisionDetails brokerapi.ProvisionDetails
			fakeProvider          *mocks.FakeProvider
		)

		BeforeEach(func() {
			validProvisionDetails = brokerapi.ProvisionDetails{
				ServiceID:        "service1",
				PlanID:           "plan1",
				OrganizationGUID: "org-guid",
				SpaceGUID:        "space-guid",
			}
			fakeProvider = &mocks.FakeProvider{}
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

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			_, instanceID, params := fakeProvider.ProvisionArgsForCall(0)

			expectedParams := providers.ProvisionParameters{
				InstanceType:               validConfig.PlanConfigs["plan1"].InstanceType,
				CacheParameterGroupName:    "default.redis3.2",
				SecurityGroupIds:           validConfig.VpcSecurityGroupIds,
				CacheSubnetGroupName:       validConfig.CacheSubnetGroupName,
				PreferredMaintenanceWindow: "sun:23:00-mon:01:30",
				ReplicasPerNodeGroup:       0,
				ShardCount:                 1,
				SnapshotRetentionLimit:     0,
				Description:                "Cloud Foundry service",
				Parameters:                 validConfig.PlanConfigs["plan1"].Parameters,
				Tags: map[string]string{
					"created-by":      validConfig.BrokerName,
					"service-id":      validProvisionDetails.ServiceID,
					"plan-id":         validProvisionDetails.PlanID,
					"organization-id": validProvisionDetails.OrganizationGUID,
					"space-id":        validProvisionDetails.SpaceGUID,
					"instance-id":     "instanceid",
				},
			}

			Expect(instanceID).To(Equal("instanceid"))
			Expect(params).To(Equal(expectedParams))
		})

		It("passes the user provided parameters", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			validProvisionDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			_, _, params := fakeProvider.ProvisionArgsForCall(0)

			expectedParameters := map[string]string{
				"reserved-memory":  "0",
				"maxmemory-policy": "noeviction",
			}

			Expect(params.Parameters).To(Equal(expectedParameters))
		})

		Context("given an unknown user provided parameter", func() {
			It("should return with error", func() {
				fakeProvider := &mocks.FakeProvider{}
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

				validProvisionDetails.RawParameters = []byte(`{"unknown-foo": "bar"}`)

				_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, true)
				Expect(err).To(MatchError("unknown parameter: unknown-foo"))
			})
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
				To(Equal(brokerapi.ProvisionedServiceSpec{
					IsAsync:       true,
					OperationData: broker.Operation{Action: broker.ActionProvisioning}.String(),
				}))
		})

		Context("when restoring from a snapshot", func() {
			var (
				restoreFromSnapshotInstanceGUID string
				expectedRestoreFromSnapshotName string
				snapshotOrgId                   string
				snapshotSpaceId                 string
				snapshotPlanId                  string
			)

			BeforeEach(func() {
				snapshotOrgId = validProvisionDetails.OrganizationGUID
				snapshotSpaceId = validProvisionDetails.SpaceGUID
				snapshotPlanId = validProvisionDetails.PlanID
			})

			JustBeforeEach(func() {
				restoreFromSnapshotInstanceGUID = "origin-instanceid"

				validProvisionDetails.RawParameters = json.RawMessage(
					`{"restore_from_latest_snapshot_of": "` + restoreFromSnapshotInstanceGUID + `"}`,
				)

				fakeProvider.FindSnapshotsReturns(
					[]providers.SnapshotInfo{
						providers.SnapshotInfo{
							Name:       restoreFromSnapshotInstanceGUID + "-snapshot-name-2-day-old",
							CreateTime: time.Now().Add(-2 * 24 * time.Hour),
							Tags: map[string]string{
								"created-by":      validConfig.BrokerName,
								"service-id":      validProvisionDetails.ServiceID,
								"plan-id":         snapshotPlanId,
								"organization-id": snapshotOrgId,
								"space-id":        snapshotSpaceId,
								"instance-id":     "instanceid",
							},
						},
						providers.SnapshotInfo{
							Name:       restoreFromSnapshotInstanceGUID + "-snapshot-name-1-day-old",
							CreateTime: time.Now().Add(-1 * 24 * time.Hour),
							Tags: map[string]string{
								"created-by":      validConfig.BrokerName,
								"service-id":      validProvisionDetails.ServiceID,
								"plan-id":         snapshotPlanId,
								"organization-id": snapshotOrgId,
								"space-id":        snapshotSpaceId,
								"instance-id":     "instanceid",
							},
						},
					},
					nil,
				)

				expectedRestoreFromSnapshotName = restoreFromSnapshotInstanceGUID + "-snapshot-name-1-day-old"

			})

			Context("and no snapshots are found", func() {
				JustBeforeEach(func() {
					fakeProvider.FindSnapshotsReturns(
						[]providers.SnapshotInfo{},
						nil,
					)

					expectedRestoreFromSnapshotName = restoreFromSnapshotInstanceGUID + "snapshot-name-1-day-old"
				})
				It("returns the correct error", func() {
					b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

					_, err := b.Provision(context.Background(), "myinstance", validProvisionDetails, true)

					Expect(err).To(MatchError("No snapshots found for: origin-instanceid"))
					Expect(fakeProvider.FindSnapshotsCallCount()).To(Equal(1))
				})
			})

			Context("when querying the snapshots fails", func() {
				JustBeforeEach(func() {
					fakeProvider.FindSnapshotsReturns(
						[]providers.SnapshotInfo{},
						errors.New("ERROR GETTING SNAPSHOTS"),
					)
				})
				It("returns the correct error", func() {
					b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

					_, err := b.Provision(context.Background(), "myinstance", validProvisionDetails, true)

					Expect(err).To(MatchError("ERROR GETTING SNAPSHOTS"))
					Expect(fakeProvider.FindSnapshotsCallCount()).To(Equal(1))
				})
			})

			It("passes the correct parameters to the Provider with the latest snapshot", func() {
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

				b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

				Expect(fakeProvider.FindSnapshotsCallCount()).To(Equal(1))

				Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
				_, instanceID, params := fakeProvider.ProvisionArgsForCall(0)

				expectedParams := providers.ProvisionParameters{
					InstanceType:               validConfig.PlanConfigs["plan1"].InstanceType,
					CacheParameterGroupName:    "default.redis3.2",
					SecurityGroupIds:           validConfig.VpcSecurityGroupIds,
					CacheSubnetGroupName:       validConfig.CacheSubnetGroupName,
					PreferredMaintenanceWindow: "sun:23:00-mon:01:30",
					ReplicasPerNodeGroup:       0,
					ShardCount:                 1,
					SnapshotRetentionLimit:     0,
					RestoreFromSnapshot:        &expectedRestoreFromSnapshotName,
					Description:                "Cloud Foundry service",
					Parameters:                 validConfig.PlanConfigs["plan1"].Parameters,
					Tags: map[string]string{
						"created-by":      validConfig.BrokerName,
						"service-id":      validProvisionDetails.ServiceID,
						"plan-id":         validProvisionDetails.PlanID,
						"organization-id": validProvisionDetails.OrganizationGUID,
						"space-id":        validProvisionDetails.SpaceGUID,
						"instance-id":     "instanceid",
					},
				}

				Expect(instanceID).To(Equal("instanceid"))
				Expect(params).To(Equal(expectedParams))
			})

			Context("when the snapshot is in a different space", func() {
				BeforeEach(func() {
					snapshotSpaceId = "other-space-id"
				})

				It("should fail to restore", func() {
					b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

					_, err := b.Provision(context.Background(), "myinstance", validProvisionDetails, true)

					Expect(err).To(MatchError("The service instance you are getting a snapshot from is not in the same org or space"))
					Expect(fakeProvider.FindSnapshotsCallCount()).To(Equal(1))
					Expect(fakeProvider.ProvisionCallCount()).To(Equal(0))
				})
			})

			Context("when the snapshot is in a different org", func() {
				BeforeEach(func() {
					snapshotOrgId = "other-org-id"
				})

				It("should fail to restore", func() {
					b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

					_, err := b.Provision(context.Background(), "myinstance", validProvisionDetails, true)

					Expect(err).To(MatchError("The service instance you are getting a snapshot from is not in the same org or space"))
					Expect(fakeProvider.FindSnapshotsCallCount()).To(Equal(1))
					Expect(fakeProvider.ProvisionCallCount()).To(Equal(0))
				})
			})

			Context("if it is using a different plan", func() {
				BeforeEach(func() {
					snapshotPlanId = "other-plan-id"
				})

				It("should fail to restore", func() {
					b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

					_, err := b.Provision(context.Background(), "myinstance", validProvisionDetails, true)

					Expect(err).To(MatchError("You must use the same plan as the service instance you are getting a snapshot from"))
					Expect(fakeProvider.FindSnapshotsCallCount()).To(Equal(1))
					Expect(fakeProvider.ProvisionCallCount()).To(Equal(0))
				})
			})

		})

	})

	Describe("Update", func() {
		var (
			validUpdateDetails brokerapi.UpdateDetails
			fakeProvider       *mocks.FakeProvider
			b                  *broker.Broker
		)

		BeforeEach(func() {
			validUpdateDetails = brokerapi.UpdateDetails{
				ServiceID: "service1",
				PlanID:    "plan1",
				PreviousValues: brokerapi.PreviousValues{
					ServiceID: "service1",
					PlanID:    "plan1",
					OrgID:     "org1",
					SpaceID:   "space1",
				},
			}
			fakeProvider = &mocks.FakeProvider{}
			b = broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
		})

		It("updates the service through the Provider", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.UpdateCallCount()).To(Equal(1))
			_, id, params := fakeProvider.UpdateArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			expectedParameters := providers.UpdateParameters{
				Parameters: map[string]string{
					"maxmemory-policy": "noeviction",
				},
			}

			Expect(params).To(Equal(expectedParameters))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       false,
				OperationData: broker.Operation{Action: broker.ActionUpdating}.String(),
			}))
		})

		It("should return an error when no parameters are provided", func() {
			validUpdateDetails.RawParameters = []byte(`{}`)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("no parameters provided"))

			Expect(fakeProvider.UpdateCallCount()).To(Equal(0))
		})

		It("rejects unknown parameters", func() {
			validUpdateDetails.RawParameters = []byte(`{"unknown_foo": "bar"}`)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("unknown parameter: unknown_foo"))

			Expect(fakeProvider.UpdateCallCount()).To(Equal(0))
		})

		It("should return an error when provider fails to update the service", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			providerErr := errors.New("provider-err")
			fakeProvider.UpdateReturnsOnCall(0, providerErr)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError(providerErr))
		})

		It("should return error when attempting to change plan id", func() {
			validUpdateDetails.PlanID = "plan2"

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("changing plans is not currently supported"))

			Expect(fakeProvider.UpdateCallCount()).To(Equal(0))
		})

		It("should return error when attempting to change service id", func() {
			validUpdateDetails.ServiceID = "service2"

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("changing plans is not currently supported"))

			Expect(fakeProvider.UpdateCallCount()).To(Equal(0))
		})

		It("sets a deadline by which the Provider request should complete", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.UpdateCallCount()).To(Equal(1))
			receivedContext, _, _ := fakeProvider.UpdateArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
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

			expectedParams := providers.DeprovisionParameters{}

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
				To(Equal(brokerapi.DeprovisionServiceSpec{
					IsAsync:       true,
					OperationData: broker.Operation{Action: broker.ActionDeprovisioning}.String(),
				}))
		})
	})

	Describe("LastOperation", func() {
		It("returns last operation data when the instance is available", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			Expect(b.LastOperation(context.Background(), "instanceid", `{"action": "provisioning"}`)).
				To(Equal(brokerapi.LastOperation{
					State:       brokerapi.Succeeded,
					Description: "i love brokers",
				}))
		})

		It("sets a deadline by which the AWS request should complete", func() {
			fakeProvider := &mocks.FakeProvider{}
			logger := lager.NewLogger("logger")
			b := broker.New(validConfig, fakeProvider, logger)

			b.LastOperation(context.Background(), "instanceid", `{"action": "provisioning"}`)

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

			b.LastOperation(context.Background(), "instanceid", `{"action": "provisioning"}`)

			Expect(log).To(gbytes.Say("last-operation"))
		})

		It("errors if getting the state fails", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			fakeProvider.GetStateReturns("", "", errors.New("foobar"))

			_, err := b.LastOperation(context.Background(), "myinstance", `{"action": "provisioning"}`)

			Expect(err).To(MatchError("error getting state for myinstance: foobar"))
		})

		It("accepts empty operation data temporarily", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if last operation data is not json", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", "I am not JSON")
			Expect(err).To(MatchError("invalid operation data: I am not JSON"))
		})

		It("returns an error if last operation data does not contain an action", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", "{}")
			Expect(err).To(MatchError("invalid operation, action parameter is empty: {}"))
		})

		Context("When provisioning", func() {
			It("returns ErrInstanceDoesNotExist when instance does not exist", func() {
				fakeProvider := &mocks.FakeProvider{}
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
				fakeProvider.GetStateReturns(providers.NonExisting, "it'sgoneya'll", nil)

				_, err := b.LastOperation(context.Background(), "myinstance", `{"action": "provisioning"}`)
				Expect(fakeProvider.DeleteCacheParameterGroupCallCount()).To(Equal(0))
				Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
			})
		})

		Context("When deprovisioning", func() {
			It("deletes the cache parameter group if the instance doesn't exist and returns ErrInstanceDoesNotExist", func() {
				fakeProvider := &mocks.FakeProvider{}
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
				fakeProvider.GetStateReturns(providers.NonExisting, "it'sgoneya'll", nil)
				ctx := context.Background()
				_, err := b.LastOperation(ctx, "myinstance", `{"action": "deprovisioning"}`)

				Expect(fakeProvider.DeleteCacheParameterGroupCallCount()).To(Equal(1))
				receivedContext, receivedInstanceID := fakeProvider.DeleteCacheParameterGroupArgsForCall(0)
				_, hasDeadline := receivedContext.Deadline()
				Expect(hasDeadline).To(BeTrue())
				Expect(receivedInstanceID).To(Equal("myinstance"))
				Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
			})

			It("returns an error if deleting the cache parameter group fails", func() {
				fakeProvider := &mocks.FakeProvider{}
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
				fakeProvider.GetStateReturns(providers.NonExisting, "it'sgoneya'll", nil)
				deleteError := errors.New("this is an error")
				fakeProvider.DeleteCacheParameterGroupReturns(deleteError)
				ctx := context.Background()
				_, err := b.LastOperation(ctx, "myinstance", `{"action": "deprovisioning"}`)

				Expect(err).To(MatchError("error deleting parameter group myinstance: this is an error"))
			})
		})

		It("logs an error and reports 'in progress' if it receives an unknown state from the provider", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.ERROR))
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetStateReturns("some-unknown-state", "", nil)
			b := broker.New(validConfig, fakeProvider, logger)

			b.LastOperation(context.Background(), "instanceid", `{"action": "provisioning"}`)

			Expect(log).To(gbytes.Say("Unknown service state: some-unknown-state"))
		})

	})

	Describe("state mapping from AWS to brokerapi package", func() {
		DescribeTable("known states",
			func(from providers.ServiceState, to brokerapi.LastOperationState) {
				Expect(broker.ProviderStatesMapping(from)).To(Equal(to))
			},
			Entry("available => succeeded", providers.Available, brokerapi.Succeeded),
			Entry("create-failed => failure", providers.CreateFailed, brokerapi.Failed),
			Entry("creating => in progress", providers.Creating, brokerapi.InProgress),
			Entry("modifying => in progress", providers.Modifying, brokerapi.InProgress),
			Entry("deleting => in progress", providers.Deleting, brokerapi.InProgress),
			Entry("snapshotting => in progress", providers.Snapshotting, brokerapi.InProgress),
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
			expectedCredentials := &providers.Credentials{
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

		It("handles errors from the provider", func() {
			unbindErr := fmt.Errorf("some error")
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.RevokeCredentialsReturnsOnCall(0, unbindErr)

			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			err := b.Unbind(context.Background(), "test-instance", "test-binding", brokerapi.UnbindDetails{})

			Expect(err).To(MatchError(unbindErr))
		})
	})
})

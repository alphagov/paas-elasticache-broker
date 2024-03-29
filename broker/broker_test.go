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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Broker", func() {
	var validConfig broker.Config

	BeforeEach(func() {
		validConfig = broker.Config{
			BrokerName:           "Broker McBrokerface",
			VpcSecurityGroupIds:  []string{"vpc_security_group_id"},
			CacheSubnetGroupName: "cache-subnet-group-name",
			Catalog: brokerapi.CatalogResponse{
				Services: []brokerapi.Service{
					{
						ID:   "service1",
						Name: "service1",
						Plans: []brokerapi.ServicePlan{
							{
								ID:   "plan1",
								Name: "plan1",
							},
						},
					},
				},
			},
			PlanConfigs: map[string]broker.PlanConfig{
				"plan1": {
					InstanceType:         "t2.micro",
					ShardCount:           1,
					ReplicasPerNodeGroup: 1,
					Parameters: map[string]string{
						"maxmemory-policy":             "volatile-lru",
						"reserved-memory":              "0",
						"preferred-maintenance-window": "sun:23:00-mon:01:30",
					},
					AutomaticFailoverEnabled:  true,
					MultiAZEnabled:            true,
					Engine:                    "redis",
					EngineVersion:             "4.0.10",
					CacheParameterGroupFamily: "default.redis4.0",
				},
				"plan2": {
					InstanceType:         "t2.micro",
					ShardCount:           1,
					ReplicasPerNodeGroup: 0,
					Parameters: map[string]string{
						"maxmemory-policy":             "volatile-lru",
						"reserved-memory":              "0",
						"preferred-maintenance-window": "sun:23:00-mon:01:30",
					},
					AutomaticFailoverEnabled:  true,
					MultiAZEnabled:            true,
					Engine:                    "redis",
					EngineVersion:             "4.0.10",
					CacheParameterGroupFamily: "default.redis4.0",
				},
				"plan3": {
					InstanceType:         "t2.micro",
					ShardCount:           1,
					ReplicasPerNodeGroup: 1,
					Parameters: map[string]string{
						"maxmemory-policy":             "volatile-lru",
						"reserved-memory":              "0",
						"preferred-maintenance-window": "sun:23:00-mon:01:30",
						"cluster-enabled":              "yes",
					},
					AutomaticFailoverEnabled:  true,
					MultiAZEnabled:            true,
					Engine:                    "redis",
					EngineVersion:             "4.0.10",
					CacheParameterGroupFamily: "default.redis4.0",
				},
				"plan4": {
					InstanceType:         "t2.micro",
					ShardCount:           1,
					ReplicasPerNodeGroup: 1,
					Parameters: map[string]string{
						"maxmemory-policy":             "volatile-lru",
						"reserved-memory":              "0",
						"preferred-maintenance-window": "sun:23:00-mon:01:30",
						"cluster-enabled":              "yes",
					},
					AutomaticFailoverEnabled:  false,
					MultiAZEnabled:            true,
					Engine:                    "redis",
					EngineVersion:             "4.0.10",
					CacheParameterGroupFamily: "default.redis4.0",
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
				CacheParameterGroupFamily:  "default.redis4.0",
				SecurityGroupIds:           validConfig.VpcSecurityGroupIds,
				CacheSubnetGroupName:       validConfig.CacheSubnetGroupName,
				PreferredMaintenanceWindow: "",
				ReplicasPerNodeGroup:       1,
				ShardCount:                 1,
				SnapshotRetentionLimit:     0,
				AutomaticFailoverEnabled:   validConfig.PlanConfigs["plan1"].AutomaticFailoverEnabled,
				MultiAZEnabled:             validConfig.PlanConfigs["plan1"].MultiAZEnabled,
				Description:                "Cloud Foundry service",
				Parameters:                 validConfig.PlanConfigs["plan1"].Parameters,
				Tags: map[string]string{
					"created-by":        validConfig.BrokerName,
					"service-id":        validProvisionDetails.ServiceID,
					"plan-id":           validProvisionDetails.PlanID,
					"organization-id":   validProvisionDetails.OrganizationGUID,
					"space-id":          validProvisionDetails.SpaceGUID,
					"instance-id":       "instanceid",
					"chargeable_entity": "instanceid",
				},
				Engine:        "redis",
				EngineVersion: "4.0.10",
			}

			Expect(instanceID).To(Equal("instanceid"))
			Expect(params).To(BeEquivalentTo(expectedParams))
		})

		It("passes the user provided parameters", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			validProvisionDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction", "preferred_maintenance_window": "sun:23:00-mon:01:30"}`)

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			_, _, params := fakeProvider.ProvisionArgsForCall(0)

			expectedParameters := map[string]string{
				"reserved-memory":              "0",
				"maxmemory-policy":             "noeviction",
				"preferred-maintenance-window": "sun:23:00-mon:01:30",
			}

			Expect(params.Parameters).To(Equal(expectedParameters))
		})

		It("sets a cost allocation tag with a value matching the instance id", func() {
			instanceId := "instance-123"
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.Provision(context.Background(), instanceId, validProvisionDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			_, _, callParams := fakeProvider.ProvisionArgsForCall(0)

			Expect(callParams.Tags).To(HaveKeyWithValue("chargeable_entity", instanceId))
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
						{
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
						{
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
					CacheParameterGroupFamily:  "default.redis4.0",
					SecurityGroupIds:           validConfig.VpcSecurityGroupIds,
					CacheSubnetGroupName:       validConfig.CacheSubnetGroupName,
					PreferredMaintenanceWindow: "",
					ReplicasPerNodeGroup:       1,
					ShardCount:                 1,
					SnapshotRetentionLimit:     0,
					RestoreFromSnapshot:        &expectedRestoreFromSnapshotName,
					AutomaticFailoverEnabled:   true,
					MultiAZEnabled:             true,
					Description:                "Cloud Foundry service",
					Parameters:                 validConfig.PlanConfigs["plan1"].Parameters,
					Tags: map[string]string{
						"created-by":        validConfig.BrokerName,
						"service-id":        validProvisionDetails.ServiceID,
						"plan-id":           validProvisionDetails.PlanID,
						"organization-id":   validProvisionDetails.OrganizationGUID,
						"space-id":          validProvisionDetails.SpaceGUID,
						"instance-id":       "instanceid",
						"chargeable_entity": "instanceid",
					},
					Engine:        "redis",
					EngineVersion: "4.0.10",
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

		It("updates the redis parameter group through the Provider", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(1))
			_, id, params := fakeProvider.UpdateParamGroupParametersArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			expectedParameters := providers.UpdateParamGroupParameters{
				Parameters: map[string]string{
					"maxmemory-policy": "noeviction",
				},
			}

			Expect(params).To(Equal(expectedParameters))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       true,
				OperationData: broker.Operation{Action: broker.ActionUpdating}.String(),
			}))
		})

		It("updates the redis replication group through the Provider", func() {
			validUpdateDetails.RawParameters = []byte(`{"preferred_maintenance_window": "mon:23:00-tue:01:30"}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.UpdateReplicationGroupCallCount()).To(Equal(1))
			_, id, params := fakeProvider.UpdateReplicationGroupArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			expectedParameters := providers.UpdateReplicationGroupParameters{
				PreferredMaintenanceWindow: "mon:23:00-tue:01:30",
			}

			Expect(params).To(Equal(expectedParameters))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       true,
				OperationData: broker.Operation{Action: broker.ActionUpdating}.String(),
			}))

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(BeZero())
		})

		It("triggers a redis test failover through the Provider", func() {

			validUpdateDetails.RawParameters = []byte(`{"test_failover": true}`)

			fakeProvider.StartFailoverTestReturns("primarynode", nil)
			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.StartFailoverTestCallCount()).To(Equal(1))
			_, id := fakeProvider.StartFailoverTestArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			Expect(spec.IsAsync).To(Equal(true))

			operation := broker.Operation{}
			err = json.Unmarshal([]byte(spec.OperationData), &operation)
			Expect(err).ToNot(HaveOccurred())

			parsedTime, err := time.Parse(time.RFC3339, operation.TimeOut)
			Expect(operation.Action).To(Equal(broker.ActionFailover))
			Expect(operation.PrimaryNode).To(Equal("primarynode"))
			Expect(parsedTime).To(BeTemporally(">", time.Now().Add(44*time.Minute)))
		})

		It("fails a test_failover with multiple raw parameters used", func() {

			validUpdateDetails.RawParameters = []byte(`{"test_failover": true, "preferred_maintenance_window": "mon:23:00-tue:01:30"}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(Equal("Test failover must be used by itself"))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  "",
				OperationData: "",
			}))
		})

		It("fails a test_failover for a cluster_mode enabled plan", func() {

			validUpdateDetails = brokerapi.UpdateDetails{
				ServiceID: "service1",
				PlanID:    "noplan",
				PreviousValues: brokerapi.PreviousValues{
					ServiceID: "service1",
					PlanID:    "noplan",
					OrgID:     "org1",
					SpaceID:   "space1",
				},
			}

			validUpdateDetails.RawParameters = []byte(`{"test_failover": true}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(Equal("Failed to find service plan: no plan found"))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  "",
				OperationData: "",
			}))
		})

		It("fails a test_failover for a cluster_mode enabled plan", func() {

			validUpdateDetails = brokerapi.UpdateDetails{
				ServiceID: "service1",
				PlanID:    "plan3",
				PreviousValues: brokerapi.PreviousValues{
					ServiceID: "service1",
					PlanID:    "plan3",
					OrgID:     "org1",
					SpaceID:   "space1",
				},
			}

			validUpdateDetails.RawParameters = []byte(`{"test_failover": true}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(HaveOccurred())

			brokerErr := errors.New("Test failover is not supported for Redis instances in cluster mode")
			Expect(err).To(MatchError(brokerErr))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  "",
				OperationData: "",
			}))
		})

		It("fails a test_failover for a cluster without HA enabled", func() {

			validUpdateDetails = brokerapi.UpdateDetails{
				ServiceID: "service1",
				PlanID:    "plan4",
				PreviousValues: brokerapi.PreviousValues{
					ServiceID: "service1",
					PlanID:    "plan4",
					OrgID:     "org1",
					SpaceID:   "space1",
				},
			}

			validUpdateDetails.RawParameters = []byte(`{"test_failover": true}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(HaveOccurred())

			brokerErr := errors.New("Test failover is not supported for Redis instances in cluster mode")
			Expect(err).To(MatchError(brokerErr))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  "",
				OperationData: "",
			}))
		})

		It("fails a test_failover for a cluster without replicas", func() {

			validUpdateDetails = brokerapi.UpdateDetails{
				ServiceID: "service1",
				PlanID:    "plan2",
				PreviousValues: brokerapi.PreviousValues{
					ServiceID: "service1",
					PlanID:    "plan2",
					OrgID:     "org1",
					SpaceID:   "space1",
				},
			}

			validUpdateDetails.RawParameters = []byte(`{"test_failover": true}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(HaveOccurred())

			brokerErr := errors.New("Test failover requires one or more replicas")
			Expect(err).To(MatchError(brokerErr))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  "",
				OperationData: "",
			}))
		})

		It("updates both the redis replication group and the redis parameter group through the Provider", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction", "preferred_maintenance_window": "mon:23:00-tue:01:30"}`)

			spec, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(1))
			_, id, paramParams := fakeProvider.UpdateParamGroupParametersArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			paramExpectedParameters := providers.UpdateParamGroupParameters{
				Parameters: map[string]string{
					"maxmemory-policy": "noeviction",
				},
			}

			Expect(paramParams).To(Equal(paramExpectedParameters))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       true,
				OperationData: broker.Operation{Action: broker.ActionUpdating}.String(),
			}))

			Expect(fakeProvider.UpdateReplicationGroupCallCount()).To(Equal(1))
			_, id, repParams := fakeProvider.UpdateReplicationGroupArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			repExpectedParameters := providers.UpdateReplicationGroupParameters{
				PreferredMaintenanceWindow: "mon:23:00-tue:01:30",
			}

			Expect(repParams).To(Equal(repExpectedParameters))

			Expect(spec).To(Equal(brokerapi.UpdateServiceSpec{
				IsAsync:       true,
				OperationData: broker.Operation{Action: broker.ActionUpdating}.String(),
			}))
		})

		It("updates the redis replication group when the redis parameter group fails through the Provider", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "everything", "preferred_maintenance_window": "mon:23:00-tue:01:30"}`)

			providerErr := errors.New("some-maxmemory-policy-error")
			fakeProvider.UpdateParamGroupParametersReturnsOnCall(0, providerErr)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError(providerErr))

			Expect(fakeProvider.UpdateReplicationGroupCallCount()).To(Equal(1))
			_, id, repParams := fakeProvider.UpdateReplicationGroupArgsForCall(0)

			Expect(id).To(Equal("instanceid"))

			repExpectedParameters := providers.UpdateReplicationGroupParameters{
				PreferredMaintenanceWindow: "mon:23:00-tue:01:30",
			}

			Expect(repParams).To(Equal(repExpectedParameters))
		})

		It("does not update the redis parameter group if the preferred-maintenance-window update fails", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction", "preferred_maintenance_window": "mon:23:00-tuesday:01:30"}`)

			providerErr := errors.New("some-replication-group-error")
			fakeProvider.UpdateReplicationGroupReturnsOnCall(0, providerErr)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError(providerErr))

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(BeZero())
		})

		It("should return an error when no parameters are provided", func() {
			validUpdateDetails.RawParameters = []byte(``)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("no parameters provided"))

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(0))
		})

		It("rejects unknown parameters", func() {
			validUpdateDetails.RawParameters = []byte(`{"unknown_foo": "bar"}`)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("unknown parameter: unknown_foo"))

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(0))
		})

		It("should return an error when provider fails to update the service", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			providerErr := errors.New("some-error-from-provider")
			fakeProvider.UpdateParamGroupParametersReturnsOnCall(0, providerErr)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError(providerErr))
		})

		It("should return error when attempting to change plan id", func() {
			validUpdateDetails.PlanID = "plan2"

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("changing plans is not currently supported"))

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(0))
		})

		It("should return error when attempting to change service id", func() {
			validUpdateDetails.ServiceID = "service2"

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).To(MatchError("changing plans is not currently supported"))

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(0))
		})

		It("sets a deadline by which the Provider request should complete", func() {
			validUpdateDetails.RawParameters = []byte(`{"maxmemory_policy": "noeviction"}`)

			_, err := b.Update(context.Background(), "instanceid", validUpdateDetails, true)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeProvider.UpdateParamGroupParametersCallCount()).To(Equal(1))
			receivedContext, _, _ := fakeProvider.UpdateParamGroupParametersArgsForCall(0)

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
			fakeProvider.ProgressStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			Expect(b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: `{"action": "provisioning"}`})).
				To(Equal(brokerapi.LastOperation{
					State:       brokerapi.Succeeded,
					Description: "i love brokers",
				}))
		})

		It("sets a deadline by which the AWS request should complete", func() {
			fakeProvider := &mocks.FakeProvider{}
			logger := lager.NewLogger("logger")
			b := broker.New(validConfig, fakeProvider, logger)

			_, err := b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: `{"action": "provisioning"}`})

			Expect(err).NotTo(HaveOccurred())
			Expect(fakeProvider.ProgressStateCallCount()).To(Equal(1))
			receivedContext, _, _, _ := fakeProvider.ProgressStateArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("logs a debug message when starting to get the last operation", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := broker.New(validConfig, &mocks.FakeProvider{}, logger)

			b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: `{"action": "provisioning"}`})

			Expect(log).To(gbytes.Say("last-operation"))
		})

		It("errors if getting the state fails", func() {
			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			fakeProvider.ProgressStateReturns("", "", errors.New("foobar"))

			_, err := b.LastOperation(context.Background(), "myinstance", brokerapi.PollDetails{OperationData: `{"action": "provisioning"}`})

			Expect(err).To(MatchError("error getting state for myinstance: foobar"))
		})

		It("accepts empty operation data temporarily", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.ProgressStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: ""})
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if last operation data is not json", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.ProgressStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: "I am not JSON"})
			Expect(err).To(MatchError("invalid operation data: I am not JSON"))
		})

		It("returns an error if last operation data does not contain an action", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.ProgressStateReturns(providers.Available, "i love brokers", nil)
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: "{}"})
			Expect(err).To(MatchError("invalid operation, action parameter is empty: {}"))
		})

		Context("When provisioning", func() {
			It("returns ErrInstanceDoesNotExist when instance does not exist", func() {
				fakeProvider := &mocks.FakeProvider{}
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
				fakeProvider.ProgressStateReturns(providers.NonExisting, "it'sgoneya'll", nil)

				_, err := b.LastOperation(context.Background(), "myinstance", brokerapi.PollDetails{OperationData: `{"action": "provisioning"}`})
				Expect(fakeProvider.DeleteCacheParameterGroupCallCount()).To(Equal(0))
				Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
			})
		})

		Context("When deprovisioning", func() {
			It("deletes the cache parameter group if the instance doesn't exist and returns ErrInstanceDoesNotExist", func() {
				fakeProvider := &mocks.FakeProvider{}
				b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
				fakeProvider.ProgressStateReturns(providers.NonExisting, "it'sgoneya'll", nil)
				ctx := context.Background()
				_, err := b.LastOperation(ctx, "myinstance", brokerapi.PollDetails{OperationData: `{"action": "deprovisioning"}`})

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
				fakeProvider.ProgressStateReturns(providers.NonExisting, "it'sgoneya'll", nil)
				deleteError := errors.New("this is an error")
				fakeProvider.DeleteCacheParameterGroupReturns(deleteError)
				ctx := context.Background()
				_, err := b.LastOperation(ctx, "myinstance", brokerapi.PollDetails{OperationData: `{"action": "deprovisioning"}`})

				Expect(err).To(MatchError("error deleting parameter group myinstance: this is an error"))
			})
		})

		It("logs an error and reports 'in progress' if it receives an unknown state from the provider", func() {
			logger := lager.NewLogger("logger")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.ERROR))
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.ProgressStateReturns("some-unknown-state", "", nil)
			b := broker.New(validConfig, fakeProvider, logger)

			_, err := b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: `{"action": "provisioning"}`})

			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(gbytes.Say("Unknown service state: some-unknown-state"))
		})

		It("will error if operation is timed out", func() {

			fakeProvider := &mocks.FakeProvider{}
			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))

			_, err := b.LastOperation(context.Background(), "instanceid", brokerapi.PollDetails{OperationData: `{"action": "failover", "timeOut": "2022-12-09T00:00:00Z"}`})

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("Operation failover timed out for instanceid")))

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
			binding, err := b.Bind(ctx, instanceID, bindingID, brokerapi.BindDetails{}, false)

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
			binding, err := b.Bind(context.Background(), "test-instance", "test-binding", brokerapi.BindDetails{}, false)

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
			_, err := b.Unbind(ctx, instanceID, bindingID, brokerapi.UnbindDetails{}, false)

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
			_, err := b.Unbind(context.Background(), "test-instance", "test-binding", brokerapi.UnbindDetails{}, false)

			Expect(err).To(MatchError(unbindErr))
		})
	})
	Describe("GetInstance", func() {
		// I feel a bit like I'm testing my mocks here...
		It("returns the instance details", func() {
			fakeProvider := &mocks.FakeProvider{}
			fakeProvider.GetInstanceParametersReturnsOnCall(0, providers.InstanceParameters{
				PreferredMaintenanceWindow: "1234",
				DailyBackupWindow:          "5678",
			}, nil)
			fakeProvider.GetInstanceTagsReturnsOnCall(0, map[string]string{"service-id": "test-service-id", "plan-id": "test-plan-id"}, nil)

			b := broker.New(validConfig, fakeProvider, lager.NewLogger("logger"))
			instance, err := b.GetInstance(context.Background(), "test-instance")

			Expect(err).ToNot(HaveOccurred())
			Expect(instance).To(Equal(brokerapi.GetInstanceDetailsSpec{
				ServiceID:    "test-service-id",
				PlanID:       "test-plan-id",
				DashboardURL: "",
				Parameters: providers.InstanceParameters{
					PreferredMaintenanceWindow: "1234",
					DailyBackupWindow:          "5678",
				},
			}))

			Expect(fakeProvider.GetInstanceParametersCallCount()).To(Equal(1))
			passedCtx, passedInstanceId := fakeProvider.GetInstanceParametersArgsForCall(0)
			Expect(passedCtx).To(Equal(context.Background()))
			Expect(passedInstanceId).To(Equal("test-instance"))
		})
	})
})

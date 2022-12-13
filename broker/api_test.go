package broker_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/alphagov/paas-elasticache-broker/providers"
	"github.com/alphagov/paas-elasticache-broker/providers/mocks"
	"github.com/pivotal-cf/brokerapi"

	"errors"

	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

func NewRequest(method, path string, body io.Reader, username, password string, params url.Values) *http.Request {
	brokerUrl := "http://127.0.0.1:8080" + path
	req := httptest.NewRequest(method, brokerUrl, body)
	req.Header.Set("X-Broker-API-Version", "2.14")
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	req.URL.RawQuery = params.Encode()
	return req
}

func DoRequest(server http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

var _ = Describe("Broker", func() {
	var (
		brokerImpl   *broker.Broker
		brokerAPI    http.Handler
		validConfig  broker.Config
		logger       lager.Logger
		fakeProvider *mocks.FakeProvider
		credentials  brokerapi.BrokerCredentials
	)

	BeforeEach(func() {
		validConfig = broker.Config{
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
				"plan1": {},
			},
		}
		logger = lager.NewLogger("elasticache-broker")
		fakeProvider = &mocks.FakeProvider{}
		brokerImpl = broker.New(validConfig, fakeProvider, logger)
		credentials = brokerapi.BrokerCredentials{
			Username: "username",
			Password: "password",
		}
		brokerAPI = brokerapi.New(brokerImpl, logger, credentials)
	})

	Describe("Services", func() {
		It("serves the catalog endpoint", func() {
			resp := DoRequest(brokerAPI, NewRequest(
				"GET",
				"/v2/catalog",
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{},
			))
			Expect(resp.Code).To(Equal(200))
			var catalogResponse brokerapi.CatalogResponse
			err := json.NewDecoder(resp.Body).Decode(&catalogResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(catalogResponse.Services).To(HaveLen(1))
			Expect(catalogResponse.Services[0].Plans).To(HaveLen(1))
		})
	})

	Describe("Provision", func() {
		It("accepts a provision request", func() {
			instanceID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": {}
				}`),
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"true"}},
			))

			Expect(resp.Code).To(Equal(202))
		})

		It("responds with a 500 when an unknown provisioning error occurs", func() {
			instanceID := uuid.NewV4().String()
			fakeProvider.ProvisionReturns(errors.New("bad stuff"))

			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": {}
				}`),
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"true"}},
			))

			Expect(resp.Code).To(Equal(500))
		})

		It("translates known errors into Open Service Broker API errors", func() {
			instanceID := uuid.NewV4().String()

			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": {}
				}`),
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"false"}},
			))

			Expect(resp.Code).To(Equal(422))

		})
	})

	Describe("Deprovision", func() {
		It("accepts a deprovision request", func() {
			instanceID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"DELETE",
				"/v2/service_instances/"+instanceID,
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"true"}, "service_id": []string{uuid.NewV4().String()}, "plan_id": []string{uuid.NewV4().String()}},
			))

			Expect(resp.Code).To(Equal(202))
		})

		It("responds with a 500 when an unknown deprovisioning error occurs", func() {
			instanceID := uuid.NewV4().String()
			fakeProvider.DeprovisionReturns(errors.New("bad stuff"))

			resp := DoRequest(brokerAPI, NewRequest(
				"DELETE",
				"/v2/service_instances/"+instanceID,
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"true"}, "service_id": []string{uuid.NewV4().String()}, "plan_id": []string{uuid.NewV4().String()}},
			))

			Expect(resp.Code).To(Equal(500))
		})

		It("translates known errors into Open Service Broker API errors", func() {
			instanceID := uuid.NewV4().String()

			resp := DoRequest(brokerAPI, NewRequest(
				"DELETE",
				"/v2/service_instances/"+instanceID,
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"false"}, "service_id": []string{uuid.NewV4().String()}, "plan_id": []string{uuid.NewV4().String()}},
			))

			Expect(resp.Code).To(Equal(422))

		})
	})

	Describe("LastOperation", func() {
		It("responds with 200 when the instance is available", func() {
			instanceID := uuid.NewV4().String()
			fakeProvider.ProgressStateReturns(providers.Available, "", nil)
			resp := DoRequest(brokerAPI, NewRequest(
				"GET",
				"/v2/service_instances/"+instanceID+"/last_operation",
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{},
			))

			Expect(resp.Code).To(Equal(200))
		})

		It("responds with a 500 when the state can't be retrieved", func() {
			instanceID := uuid.NewV4().String()
			fakeProvider.ProgressStateReturns("", "", errors.New("ohai"))

			resp := DoRequest(brokerAPI, NewRequest(
				"GET",
				"/v2/service_instances/"+instanceID+"/last_operation",
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{},
			))

			Expect(resp.Code).To(Equal(500))
		})

		It("responds with 410 when the instance doesn't exist", func() {
			instanceID := uuid.NewV4().String()
			fakeProvider.ProgressStateReturns(providers.NonExisting, "", nil)
			resp := DoRequest(brokerAPI, NewRequest(
				"GET",
				"/v2/service_instances/"+instanceID+"/last_operation",
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{},
			))

			Expect(resp.Code).To(Equal(410))
		})
	})

	Describe("Update", func() {
		It("accepts an update request", func() {
			instanceID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PATCH",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"previous_values": {
						"plan_id": "plan1",
						"service_id": "service1",
						"org_id": "test-organization-id",
						"space_id": "space-id"
					},
					"parameters": {"maxmemory_policy": "inexhaustible"}
				}`),
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"true"}},
			))

			Expect(resp.Code).To(Equal(202))
		})

		It("responds with a 500 when an unknown deprovisioning error occurs", func() {
			instanceID := uuid.NewV4().String()
			fakeProvider.UpdateParamGroupParametersReturns(errors.New("bad stuff"))

			resp := DoRequest(brokerAPI, NewRequest(
				"PATCH",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"previous_values": {
						"plan_id": "plan1",
						"service_id": "service1",
						"org_id": "test-organization-id",
						"space_id": "space-id"
					},
					"parameters": {"maxmemory_policy": "inexhaustible"}
				}`),
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"true"}},
			))

			Expect(resp.Code).To(Equal(500))
		})

		It("translates known errors into Open Service Broker API errors", func() {
			instanceID := uuid.NewV4().String()

			resp := DoRequest(brokerAPI, NewRequest(
				"PATCH",
				"/v2/service_instances/"+instanceID,
				nil,
				credentials.Username,
				credentials.Password,
				url.Values{"accepts_incomplete": []string{"false"}},
			))

			Expect(resp.Code).To(Equal(422))

		})
	})
})

package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/onsi/ginkgo"
	"github.com/pivotal-cf/brokerapi"
)

type ByServiceID []brokerapi.Service

func (a ByServiceID) Len() int           { return len(a) }
func (a ByServiceID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByServiceID) Less(i, j int) bool { return a[i].ID < a[j].ID }

type ProvisioningResponse struct {
	DashboardURL string `json:"dashboard_url,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

type BindingResponse struct {
	Credentials map[string]interface{} `json:"credentials,omitempty"`
}

type LastOperationResponse struct {
	State       string `json:"state,omitempty"`
	Description string `json:"description,omitempty"`
}

type uriParam struct {
	key   string
	value string
}

func BodyBytes(resp *http.Response) ([]byte, error) {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

type BrokerAPIClient struct {
	URL                   string
	Username              string
	Password              string
	DefaultOrganizationID string
	DefaultSpaceID        string
	AcceptsIncomplete     bool
}

func NewBrokerAPIClient(url string, username string, password string, defaultOrganizationID string, defaultSpaceID string) *BrokerAPIClient {
	return &BrokerAPIClient{
		URL:                   url,
		Username:              username,
		Password:              password,
		DefaultOrganizationID: defaultOrganizationID,
		DefaultSpaceID:        defaultSpaceID,
	}
}

func (b *BrokerAPIClient) doRequest(action string, path string, body io.Reader, params ...uriParam) (*http.Response, error) {

	client := &http.Client{}

	req, err := http.NewRequest(action, b.URL+path, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(b.Username, b.Password)

	q := req.URL.Query()
	for _, p := range params {
		q.Add(p.key, p.value)
	}
	req.URL.RawQuery = q.Encode()

	return client.Do(req)
}

func (b *BrokerAPIClient) GetCatalog() (brokerapi.CatalogResponse, error) {

	catalog := brokerapi.CatalogResponse{}

	resp, err := b.doRequest("GET", "/v2/catalog", nil)
	if err != nil {
		return catalog, err
	}
	if resp.StatusCode != 200 {
		return catalog, fmt.Errorf("Invalid catalog response %v", resp)
	}

	body, err := BodyBytes(resp)
	if err != nil {
		return catalog, err
	}

	err = json.Unmarshal(body, &catalog)

	return catalog, err
}

func (b *BrokerAPIClient) DoProvisionRequest(instanceID, serviceID, planID string, paramJSON string) (*http.Response, error) {
	path := "/v2/service_instances/" + instanceID

	provisionDetailsJSON := []byte(fmt.Sprintf(`
		{
			"service_id": "%s",
			"plan_id": "%s",
			"organization_guid": "%s",
			"space_guid": "%s",
			"parameters": %s
		}
	`, serviceID, planID, b.DefaultOrganizationID, b.DefaultSpaceID, paramJSON))

	return b.doRequest(
		"PUT",
		path,
		bytes.NewBuffer(provisionDetailsJSON),
		uriParam{key: "accepts_incomplete", value: strconv.FormatBool(b.AcceptsIncomplete)},
	)
}

func (b *BrokerAPIClient) DoDeprovisionRequest(instanceID, serviceID, planID string) (*http.Response, error) {
	path := fmt.Sprintf("/v2/service_instances/%s", instanceID)

	return b.doRequest(
		"DELETE",
		path,
		nil,
		uriParam{key: "service_id", value: serviceID},
		uriParam{key: "plan_id", value: planID},
		uriParam{key: "accepts_incomplete", value: strconv.FormatBool(b.AcceptsIncomplete)},
	)
}

func (b *BrokerAPIClient) DoUpdateRequest(instanceID, serviceID, planID, oldPlanID, oldServiceID, orgID, spaceID, paramJSON string) (*http.Response, error) {
	path := fmt.Sprintf("/v2/service_instances/%s", instanceID)

	provisionDetailsJSON := []byte(fmt.Sprintf(`
		{
			"service_id": "%s",
			"plan_id": "%s",
			"previous_values": {
				"plan_id": "%s",
				"service_id": "%s",
				"org_id": "%s",
				"space_id": "%s"
			},
			"parameters": %s
		}
	`, serviceID, planID, oldPlanID, oldServiceID, orgID, spaceID, paramJSON))

	return b.doRequest(
		"PATCH",
		path,
		bytes.NewBuffer(provisionDetailsJSON),
		uriParam{key: "accepts_incomplete", value: strconv.FormatBool(b.AcceptsIncomplete)},
	)
}

func (b *BrokerAPIClient) ProvisionInstance(instanceID, serviceID, planID string, paramJSON string) (responseCode int, operation string, err error) {
	resp, err := b.DoProvisionRequest(instanceID, serviceID, planID, paramJSON)
	if err != nil {
		return 0, "", err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		return resp.StatusCode, "", nil
	}

	provisioningResponse := ProvisioningResponse{}

	body, err := BodyBytes(resp)
	if err != nil {
		return resp.StatusCode, "", err
	}

	err = json.Unmarshal(body, &provisioningResponse)
	if err != nil {
		return resp.StatusCode, "", err
	}

	return resp.StatusCode, provisioningResponse.Operation, nil
}

func (b *BrokerAPIClient) DeprovisionInstance(instanceID, serviceID, planID string) (responseCode int, operation string, err error) {
	resp, err := b.DoDeprovisionRequest(instanceID, serviceID, planID)
	if err != nil {
		return 0, "", err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		return resp.StatusCode, "", nil
	}

	provisioningResponse := ProvisioningResponse{}

	body, err := BodyBytes(resp)
	if err != nil {
		return resp.StatusCode, "", err
	}

	err = json.Unmarshal(body, &provisioningResponse)
	if err != nil {
		return resp.StatusCode, "", err
	}

	return resp.StatusCode, provisioningResponse.Operation, nil
}

func (b *BrokerAPIClient) UpdateInstance(instanceID, serviceID, planID, oldPlanID, oldServiceID, orgID, spaceID, paramJSON string) (responseCode int, operation string, err error) {
	resp, err := b.DoUpdateRequest(instanceID, serviceID, planID, oldPlanID, oldServiceID, orgID, spaceID, paramJSON)
	if err != nil {
		return 0, "", err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		return resp.StatusCode, "", nil
	}

	provisioningResponse := ProvisioningResponse{}

	body, err := BodyBytes(resp)
	if err != nil {
		return resp.StatusCode, "", err
	}

	err = json.Unmarshal(body, &provisioningResponse)
	if err != nil {
		return resp.StatusCode, "", err
	}

	return resp.StatusCode, provisioningResponse.Operation, nil
}

func (b *BrokerAPIClient) DoLastOperationRequest(instanceID, serviceID, planID, operation string) (*http.Response, error) {
	path := fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)

	return b.doRequest(
		"GET",
		path,
		nil,
		uriParam{key: "service_id", value: serviceID},
		uriParam{key: "plan_id", value: planID},
		uriParam{key: "operation", value: operation},
	)
}

func (b *BrokerAPIClient) GetLastOperationState(instanceID, serviceID, planID, operation string) (string, error) {
	resp, err := b.DoLastOperationRequest(instanceID, serviceID, planID, operation)
	if err != nil {
		return "", err
	}
	body, err := BodyBytes(resp)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(ginkgo.GinkgoWriter, "%v", string(body))
	fmt.Fprintf(os.Stdout, ".")
	switch resp.StatusCode {
	case 410:
		return "gone", nil
	case 200:
		lastOperationResponse := LastOperationResponse{}

		err = json.Unmarshal(body, &lastOperationResponse)
		if err != nil {
			return "", err
		}
		return lastOperationResponse.State, nil
	default:
		return "", fmt.Errorf("Unknown code %d: %s", resp.StatusCode, string(body))
	}
}

func (b *BrokerAPIClient) DoBindRequest(instanceID, serviceID, planID, appGUID, bindingID string) (*http.Response, error) {
	path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)

	bindingDetailsJSON := []byte(fmt.Sprintf(`
		{
			"service_id": "%s",
			"plan_id": "%s",
			"bind_resource": {
				"app_guid": "%s"
			},
			"parameters": {}
		}`,
		serviceID,
		planID,
		appGUID,
	))

	return b.doRequest(
		"PUT",
		path,
		bytes.NewBuffer(bindingDetailsJSON),
	)
}

func (b *BrokerAPIClient) DoUnbindRequest(instanceID, serviceID, planID, bindingID string) (*http.Response, error) {
	path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)

	return b.doRequest(
		"DELETE",
		path,
		nil,
		uriParam{key: "service_id", value: serviceID},
		uriParam{key: "plan_id", value: planID},
	)
}

func (b *BrokerAPIClient) Bind(instanceID, serviceID, planID, appGUID, bindingID string) (int, *BindingResponse, error) {
	resp, err := b.DoBindRequest(instanceID, serviceID, planID, appGUID, bindingID)
	if err != nil {
		return 0, nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return resp.StatusCode, nil, nil
	}

	bindingResponse := &BindingResponse{}

	body, err := BodyBytes(resp)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	err = json.Unmarshal(body, bindingResponse)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, bindingResponse, nil
}

func (b *BrokerAPIClient) Unbind(instanceID, serviceID, planID, bindingID string) (int, error) {
	resp, err := b.DoUnbindRequest(instanceID, serviceID, planID, bindingID)
	if err != nil {
		return 0, err
	}

	return resp.StatusCode, nil
}

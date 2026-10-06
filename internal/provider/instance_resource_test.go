package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

func TestInstanceReadRefreshesState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vms/42" {
			t.Errorf("request = %s %s, want GET /vms/42", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"id":42,"hostname":"web-01","status":"running","plan":"nano","region":"montreal","image":"debian-13","ipv4":null,"billing":{"mode":"hourly"}}}`))
	}))
	defer server.Close()

	instance := &InstanceResource{client: testAPIClient(server)}
	state := newInstanceState(t, InstanceResourceModel{
		ID:       types.Int64Value(42),
		Hostname: types.StringValue("stale-hostname"),
		Plan:     types.StringValue("nano"),
		Region:   types.StringValue("montreal"),
		Image:    types.StringValue("debian-13"),
		Billing:  types.StringValue("hourly"),
		IPv4:     types.StringValue("192.0.2.4"),
	})
	response := resource.ReadResponse{State: state}

	instance.Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", response.Diagnostics)
	}

	var got InstanceResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("read refreshed state: %v", response.Diagnostics)
	}
	if got.Hostname.ValueString() != "web-01" {
		t.Errorf("Read() Hostname = %q, want web-01", got.Hostname.ValueString())
	}
	if !got.IPv4.IsNull() {
		t.Errorf("Read() IPv4 = %v, want null", got.IPv4)
	}
}

func TestInstanceReadRemovesMissingVMFromState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	instance := &InstanceResource{client: testAPIClient(server)}
	state := newInstanceState(t, InstanceResourceModel{
		ID:       types.Int64Value(42),
		Hostname: types.StringValue("web-01"),
		Plan:     types.StringValue("nano"),
		Region:   types.StringValue("montreal"),
		Image:    types.StringValue("debian-13"),
		Billing:  types.StringValue("hourly"),
		IPv4:     types.StringNull(),
	})
	response := resource.ReadResponse{State: state}

	instance.Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", response.Diagnostics)
	}
	if !response.State.Raw.IsNull() {
		t.Fatal("Read() should remove the missing VM from state")
	}
}

func TestInstanceUpdateChangesHostname(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/vms/42" {
			t.Errorf("request = %s %s, want PATCH /vms/42", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"id":42,"hostname":"web-02","status":"running","plan":"nano","region":"montreal","image":"debian-13","ipv4":"192.0.2.4","billing":{"mode":"hourly"}}}`))
	}))
	defer server.Close()

	instance := &InstanceResource{client: testAPIClient(server)}
	state := newInstanceState(t, InstanceResourceModel{
		ID:       types.Int64Value(42),
		Hostname: types.StringValue("web-01"),
		Plan:     types.StringValue("nano"),
		Region:   types.StringValue("montreal"),
		Image:    types.StringValue("debian-13"),
		Billing:  types.StringValue("hourly"),
		IPv4:     types.StringValue("192.0.2.4"),
	})
	plan := newInstancePlan(t, InstanceResourceModel{
		ID:       types.Int64Value(42),
		Hostname: types.StringValue("web-02"),
		Plan:     types.StringValue("nano"),
		Region:   types.StringValue("montreal"),
		Image:    types.StringValue("debian-13"),
		Billing:  types.StringValue("hourly"),
		IPv4:     types.StringValue("192.0.2.4"),
	})
	response := resource.UpdateResponse{State: state}

	instance.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics = %v", response.Diagnostics)
	}

	var got InstanceResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("read updated state: %v", response.Diagnostics)
	}
	if got.Hostname.ValueString() != "web-02" {
		t.Errorf("Update() Hostname = %q, want web-02", got.Hostname.ValueString())
	}
}

func TestInstanceConfigureSetsClient(t *testing.T) {
	instance := &InstanceResource{}
	apiClient := client.NewClient("https://example.test/v1", "token")
	instance.Configure(context.Background(), resource.ConfigureRequest{ProviderData: apiClient}, &resource.ConfigureResponse{})
	if instance.client != apiClient {
		t.Fatal("Configure() did not set the API client")
	}
	instance.Configure(context.Background(), resource.ConfigureRequest{}, &resource.ConfigureResponse{})
}

func TestInstanceCreateSetsStateWithoutAction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vms" {
			t.Errorf("request = %s %s, want POST /vms", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got == "" {
			t.Error("Create() should send an idempotency key")
		}
		var request client.CreateVmRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode Create() request: %v", err)
		} else if request.Billing != "hourly" {
			t.Errorf("Create() billing = %q, want hourly default", request.Billing)
		}
		_, _ = w.Write([]byte(`{"data":{"vm":{"id":42,"hostname":"web-01","status":"pending_payment","ipv4":"192.0.2.4"},"action":{"id":0}}}`))
	}))
	defer server.Close()

	instance := &InstanceResource{client: testAPIClient(server)}
	plan := newInstancePlan(t, InstanceResourceModel{
		ID:       types.Int64Null(),
		Hostname: types.StringValue("web-01"),
		Plan:     types.StringValue("nano"),
		Region:   types.StringValue("montreal"),
		Image:    types.StringValue("debian-13"),
		Billing:  types.StringNull(),
		IPv4:     types.StringUnknown(),
	})
	state := newInstanceState(t, InstanceResourceModel{
		ID:       types.Int64Null(),
		Hostname: types.StringNull(),
		Plan:     types.StringNull(),
		Region:   types.StringNull(),
		Image:    types.StringNull(),
		Billing:  types.StringNull(),
		IPv4:     types.StringNull(),
	})
	response := resource.CreateResponse{State: state}

	instance.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}
	var got InstanceResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("read created state: %v", response.Diagnostics)
	}
	if got.ID.ValueInt64() != 42 || got.IPv4.ValueString() != "192.0.2.4" {
		t.Errorf("Create() state = %#v, want VM 42 and assigned IPv4", got)
	}
}

func TestInstanceCreateReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	instance := &InstanceResource{client: testAPIClient(server)}
	plan := newInstancePlan(t, InstanceResourceModel{
		ID:       types.Int64Null(),
		Hostname: types.StringValue("web-01"),
		Plan:     types.StringValue("nano"),
		Region:   types.StringValue("montreal"),
		Image:    types.StringValue("debian-13"),
		Billing:  types.StringValue("hourly"),
		IPv4:     types.StringUnknown(),
	})
	response := resource.CreateResponse{State: newInstanceState(t, InstanceResourceModel{
		ID: types.Int64Null(), Hostname: types.StringNull(), Plan: types.StringNull(),
		Region: types.StringNull(), Image: types.StringNull(), Billing: types.StringNull(), IPv4: types.StringNull(),
	})}

	instance.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("Create() should report an API error")
	}
}

func TestInstanceDeleteHandlesAPIResponses(t *testing.T) {
	for _, statusCode := range []int{http.StatusAccepted, http.StatusNotFound} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/vms/42" {
					t.Errorf("request = %s %s, want DELETE /vms/42", r.Method, r.URL.Path)
				}
				if got := r.URL.Query().Get("confirm"); got != "web-01" {
					t.Errorf("confirm = %q, want web-01", got)
				}
				w.WriteHeader(statusCode)
			}))
			defer server.Close()

			instance := &InstanceResource{client: testAPIClient(server)}
			state := newInstanceState(t, InstanceResourceModel{
				ID:       types.Int64Value(42),
				Hostname: types.StringValue("web-01"),
				Plan:     types.StringValue("nano"),
				Region:   types.StringValue("montreal"),
				Image:    types.StringValue("debian-13"),
				Billing:  types.StringValue("hourly"),
				IPv4:     types.StringNull(),
			})
			response := resource.DeleteResponse{}
			instance.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
			if response.Diagnostics.HasError() != (statusCode == http.StatusNotFound) {
				t.Errorf("Delete() diagnostics error = %t, want %t", response.Diagnostics.HasError(), statusCode == http.StatusNotFound)
			}
		})
	}
}

func newInstanceState(t *testing.T, model InstanceResourceModel) tfsdk.State {
	t.Helper()

	instance := NewInstanceResource()
	var schemaResponse resource.SchemaResponse
	instance.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)

	state := tfsdk.State{Schema: schemaResponse.Schema}
	diagnostics := state.Set(context.Background(), &model)
	if diagnostics.HasError() {
		t.Fatalf("initialize resource state: %v", diagnostics)
	}
	return state
}

func newInstancePlan(t *testing.T, model InstanceResourceModel) tfsdk.Plan {
	t.Helper()

	instance := NewInstanceResource()
	var schemaResponse resource.SchemaResponse
	instance.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)

	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	diagnostics := plan.Set(context.Background(), &model)
	if diagnostics.HasError() {
		t.Fatalf("initialize resource plan: %v", diagnostics)
	}
	return plan
}

func testAPIClient(server *httptest.Server) *client.Client {
	apiClient := client.NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	return apiClient
}

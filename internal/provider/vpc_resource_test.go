package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVPCConfigureSetsClient(t *testing.T) {
	vpc := &VPCResource{}
	apiClient := client.NewClient("https://example.test/v1", "token")
	vpc.Configure(context.Background(), resource.ConfigureRequest{ProviderData: apiClient}, &resource.ConfigureResponse{})
	if vpc.client != apiClient {
		t.Fatal("Configure() did not set the API client")
	}
	vpc.Configure(context.Background(), resource.ConfigureRequest{}, &resource.ConfigureResponse{})
}

func TestVPCCreateSetsState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vpcs" {
			t.Errorf("request = %s %s, want POST /vpcs", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		var request client.CreateVPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode Create() request: %v", err)
		} else if request != (client.CreateVPCRequest{Name: "production", CIDR: "10.0.0.0/24", Region: "montreal"}) {
			t.Errorf("Create() request = %#v, unexpected values", request)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":17,"name":"production","cidr":"10.0.0.0/24","region":"montreal","status":"active","gateway":"10.0.0.1","internet_gateway":true,"created_at":"2026-10-05T12:00:00Z"}}`))
	}))
	defer server.Close()

	vpc := &VPCResource{client: testAPIClient(server)}
	plan := newVPCPlan(t, VPCResourceModel{
		ID: types.Int64Null(), Name: types.StringValue("production"), CIDR: types.StringValue("10.0.0.0/24"),
		Region: types.StringValue("montreal"), Status: types.StringUnknown(), Gateway: types.StringUnknown(),
		InternetGateway: types.BoolUnknown(), CreatedAt: types.StringUnknown(),
	})
	response := resource.CreateResponse{State: newVPCState(t, VPCResourceModel{
		ID: types.Int64Null(), Name: types.StringNull(), CIDR: types.StringNull(), Region: types.StringNull(),
		Status: types.StringNull(), Gateway: types.StringNull(), InternetGateway: types.BoolNull(), CreatedAt: types.StringNull(),
	})}

	vpc.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}
	var got VPCResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("read created state: %v", response.Diagnostics)
	}
	if got.ID.ValueInt64() != 17 || got.Gateway.ValueString() != "10.0.0.1" || !got.InternetGateway.ValueBool() {
		t.Errorf("Create() state = %#v, want populated VPC outputs", got)
	}
}

func TestVPCReadRefreshesState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vpcs/17" {
			t.Errorf("request = %s %s, want GET /vpcs/17", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"id":17,"name":"production","cidr":"10.0.0.0/24","region":"montreal","status":"active","gateway":"10.0.0.1","internet_gateway":true,"created_at":"2026-10-05T12:00:00Z"}}`))
	}))
	defer server.Close()

	state := newVPCState(t, VPCResourceModel{
		ID: types.Int64Value(17), Name: types.StringValue("stale"), CIDR: types.StringValue("10.0.0.0/24"),
		Region: types.StringValue("montreal"), Status: types.StringValue("pending"), Gateway: types.StringNull(),
		InternetGateway: types.BoolValue(false), CreatedAt: types.StringNull(),
	})
	response := resource.ReadResponse{State: state}
	(&VPCResource{client: testAPIClient(server)}).Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", response.Diagnostics)
	}
	var got VPCResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("read refreshed state: %v", response.Diagnostics)
	}
	if got.Name.ValueString() != "production" || got.Status.ValueString() != "active" || !got.InternetGateway.ValueBool() {
		t.Errorf("Read() state = %#v, want refreshed VPC values", got)
	}
}

func TestVPCReadRemovesMissingVPCFromState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	state := newVPCState(t, VPCResourceModel{
		ID: types.Int64Value(17), Name: types.StringValue("production"), CIDR: types.StringValue("10.0.0.0/24"),
		Region: types.StringValue("montreal"), Status: types.StringValue("active"), Gateway: types.StringValue("10.0.0.1"),
		InternetGateway: types.BoolValue(true), CreatedAt: types.StringValue("2026-10-05T12:00:00Z"),
	})
	response := resource.ReadResponse{State: state}
	(&VPCResource{client: testAPIClient(server)}).Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", response.Diagnostics)
	}
	if !response.State.Raw.IsNull() {
		t.Fatal("Read() should remove the missing VPC from state")
	}
}

func TestVPCDeleteHandlesSuccessAndMissingNetwork(t *testing.T) {
	for _, statusCode := range []int{http.StatusNoContent, http.StatusNotFound} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/vpcs/17" {
					t.Errorf("request = %s %s, want DELETE /vpcs/17", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Errorf("Authorization = %q, want Bearer test-token", got)
				}
				w.WriteHeader(statusCode)
			}))
			defer server.Close()
			state := newVPCState(t, VPCResourceModel{
				ID: types.Int64Value(17), Name: types.StringValue("production"), CIDR: types.StringValue("10.0.0.0/24"),
				Region: types.StringValue("montreal"), Status: types.StringValue("active"), Gateway: types.StringValue("10.0.0.1"),
				InternetGateway: types.BoolValue(true), CreatedAt: types.StringValue("2026-10-05T12:00:00Z"),
			})
			response := resource.DeleteResponse{}
			(&VPCResource{client: testAPIClient(server)}).Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
			if response.Diagnostics.HasError() {
				t.Errorf("Delete() diagnostics = %v", response.Diagnostics)
			}
		})
	}
}

func newVPCState(t *testing.T, model VPCResourceModel) tfsdk.State {
	t.Helper()
	vpc := NewVPCResource()
	var schemaResponse resource.SchemaResponse
	vpc.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diagnostics := state.Set(context.Background(), &model); diagnostics.HasError() {
		t.Fatalf("initialize VPC state: %v", diagnostics)
	}
	return state
}

func newVPCPlan(t *testing.T, model VPCResourceModel) tfsdk.Plan {
	t.Helper()
	vpc := NewVPCResource()
	var schemaResponse resource.SchemaResponse
	vpc.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diagnostics := plan.Set(context.Background(), &model); diagnostics.HasError() {
		t.Fatalf("initialize VPC plan: %v", diagnostics)
	}
	return plan
}

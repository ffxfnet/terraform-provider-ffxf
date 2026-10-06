package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFirewallResourceLifecycle(t *testing.T) {
	const firewallJSON = `{"data":{"id":2,"name":"web","rules":[{"direction":"in","protocol":"tcp","ports":"443","sources":["203.0.113.0/24"],"description":"https"}],"members":[{"vm":7,"hostname":"web-01","status":"applied"}],"created_at":"2026-10-06T12:00:00Z"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /firewalls":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(firewallJSON))
		case "GET /firewalls/2", "PUT /firewalls/2":
			_, _ = w.Write([]byte(firewallJSON))
		case "DELETE /firewalls/2":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	firewall := NewFirewallResource().(*FirewallResource)
	apiClient := testAPIClient(server)
	firewall.Configure(ctx, resource.ConfigureRequest{ProviderData: apiClient}, &resource.ConfigureResponse{})
	firewall.Configure(ctx, resource.ConfigureRequest{}, &resource.ConfigureResponse{})
	var schemaResponse resource.SchemaResponse
	firewall.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", schemaResponse.Diagnostics)
	}
	var metadata resource.MetadataResponse
	firewall.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_firewall" {
		t.Fatalf("Metadata() TypeName = %q", metadata.TypeName)
	}

	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	planned := FirewallResourceModel{ID: types.Int64Null(), Name: types.StringValue("web"), Rules: firewallTestRules(t), Members: types.ListUnknown(firewallMemberObjectType()), CreatedAt: types.StringUnknown()}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("set create plan: %v", diags)
	}
	state := tfsdk.State{Schema: schemaResponse.Schema}
	createResponse := resource.CreateResponse{State: state}
	firewall.Create(ctx, resource.CreateRequest{Plan: plan}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", createResponse.Diagnostics)
	}
	var created FirewallResourceModel
	createResponse.Diagnostics.Append(createResponse.State.Get(ctx, &created)...)
	if createResponse.Diagnostics.HasError() || created.ID.ValueInt64() != 2 || len(created.Members.Elements()) != 1 {
		t.Fatalf("Create() state = %#v, diagnostics = %v", created, createResponse.Diagnostics)
	}

	readResponse := resource.ReadResponse{State: createResponse.State}
	firewall.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", readResponse.Diagnostics)
	}
	updatePlan := tfsdk.Plan{Schema: schemaResponse.Schema}
	planned.Name = types.StringValue("web-renamed")
	if diags := updatePlan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("set update plan: %v", diags)
	}
	updateResponse := resource.UpdateResponse{State: readResponse.State}
	firewall.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResponse.State}, &updateResponse)
	if updateResponse.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics = %v", updateResponse.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	firewall.Delete(ctx, resource.DeleteRequest{State: updateResponse.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete() diagnostics = %v", deleteResponse.Diagnostics)
	}
}

func TestFirewallMemberResourceLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /firewalls/2/members":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"vm":7,"hostname":"web-01","status":"applied"}}`))
		case "GET /firewalls/2":
			_, _ = w.Write([]byte(`{"data":{"id":2,"name":"web","rules":[],"members":[{"vm":7,"hostname":"web-01","status":"applied"}]}}`))
		case "DELETE /firewalls/2/members/7":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	member := NewFirewallMemberResource().(*FirewallMemberResource)
	member.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	member.Configure(ctx, resource.ConfigureRequest{}, &resource.ConfigureResponse{})
	var schemaResponse resource.SchemaResponse
	member.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	var metadata resource.MetadataResponse
	member.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_firewall_member" {
		t.Fatalf("Metadata() TypeName = %q", metadata.TypeName)
	}
	model := FirewallMemberResourceModel{FirewallID: types.Int64Value(2), VMID: types.Int64Value(7), Hostname: types.StringUnknown(), Status: types.StringUnknown()}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	member.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", response.Diagnostics)
	}
	readResponse := resource.ReadResponse{State: response.State}
	member.Read(ctx, resource.ReadRequest{State: response.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", readResponse.Diagnostics)
	}
	var got FirewallMemberResourceModel
	readResponse.Diagnostics.Append(readResponse.State.Get(ctx, &got)...)
	if got.Hostname.ValueString() != "web-01" || got.Status.ValueString() != "applied" {
		t.Fatalf("member state = %#v", got)
	}
	updateResponse := resource.UpdateResponse{}
	member.Update(ctx, resource.UpdateRequest{}, &updateResponse)
	if !updateResponse.Diagnostics.HasError() {
		t.Fatal("Update() should require replacement")
	}
	deleteResponse := resource.DeleteResponse{}
	member.Delete(ctx, resource.DeleteRequest{State: readResponse.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete() diagnostics = %v", deleteResponse.Diagnostics)
	}
}

func TestLoadBalancerResourceLifecycle(t *testing.T) {
	const lbJSON = `{"data":{"id":8,"name":"edge","status":"active","vpc":12,"ipv4":"192.0.2.8","ipv6":"2001:db8::8","dns_name":"edge.example.test","config_applied":true,"last_seen_at":"2026-10-06T12:00:00Z","last_error":null,"pools":[],"listeners":[],"certificates":[{"hostname":"app.example.test","status":"active","expires_at":"2026-11-06T12:00:00Z","error":null}],"billing":{"mode":"hourly","rate":"0.010","currency":"CAD"},"created_at":"2026-10-06T12:00:00Z"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /load-balancers":
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("Create() should provide Idempotency-Key")
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(lbJSON))
		case "GET /load-balancers/8", "PATCH /load-balancers/8":
			_, _ = w.Write([]byte(lbJSON))
		case "DELETE /load-balancers/8":
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	lb := NewLoadBalancerResource().(*LoadBalancerResource)
	lb.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	lb.Configure(ctx, resource.ConfigureRequest{}, &resource.ConfigureResponse{})
	var schemaResponse resource.SchemaResponse
	lb.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", schemaResponse.Diagnostics)
	}
	var metadata resource.MetadataResponse
	lb.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_load_balancer" {
		t.Fatalf("Metadata() TypeName = %q", metadata.TypeName)
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	planned := LoadBalancerResourceModel{
		ID: types.Int64Null(), Name: types.StringValue("edge"), VPC: types.Int64Value(12),
		Certificates: types.ListUnknown(types.ObjectType{AttrTypes: map[string]attr.Type{
			"hostname": types.StringType, "status": types.StringType, "expires_at": types.StringType, "error": types.StringType,
		}}),
		Billing: types.ObjectUnknown(map[string]attr.Type{"mode": types.StringType, "rate": types.StringType, "currency": types.StringType}),
	}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("set create plan: %v", diags)
	}
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	lb.Create(ctx, resource.CreateRequest{Plan: plan}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", createResponse.Diagnostics)
	}
	var created LoadBalancerResourceModel
	createResponse.Diagnostics.Append(createResponse.State.Get(ctx, &created)...)
	if created.ID.ValueInt64() != 8 || created.Billing.IsNull() || len(created.Certificates.Elements()) != 1 {
		t.Fatalf("Create() state = %#v, diagnostics=%v", created, createResponse.Diagnostics)
	}
	readResponse := resource.ReadResponse{State: createResponse.State}
	lb.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", readResponse.Diagnostics)
	}
	updatePlan := tfsdk.Plan{Schema: schemaResponse.Schema}
	created.Name = types.StringValue("edge-renamed")
	if diags := updatePlan.Set(ctx, &created); diags.HasError() {
		t.Fatalf("set update plan: %v", diags)
	}
	updateResponse := resource.UpdateResponse{State: readResponse.State}
	lb.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResponse.State}, &updateResponse)
	if updateResponse.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics = %v", updateResponse.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	lb.Delete(ctx, resource.DeleteRequest{State: updateResponse.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete() diagnostics = %v", deleteResponse.Diagnostics)
	}
}

func firewallTestRules(t *testing.T) types.List {
	t.Helper()
	ctx := context.Background()
	sources, diags := types.ListValueFrom(ctx, types.StringType, []string{"203.0.113.0/24"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	destinations, diags := types.ListValueFrom(ctx, types.StringType, []string{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	attrTypes := map[string]attr.Type{
		"direction": types.StringType, "protocol": types.StringType, "ports": types.StringType,
		"sources": types.ListType{ElemType: types.StringType}, "destinations": types.ListType{ElemType: types.StringType}, "description": types.StringType,
	}
	list, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, []FirewallRuleModel{{
		Direction: types.StringValue("in"), Protocol: types.StringValue("tcp"), Ports: types.StringValue("443"),
		Sources: sources, Destinations: destinations, Description: types.StringValue("https"),
	}})
	if listDiags.HasError() {
		t.Fatal(listDiags)
	}
	return list
}

func firewallMemberObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"vm": types.Int64Type, "hostname": types.StringType, "status": types.StringType}}
}

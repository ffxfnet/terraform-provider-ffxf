package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountAreaSSHKeyLifecycleAndNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /ssh-keys":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["name"] != "deploy" || body["public_key"] != "ssh-ed25519 public" {
				t.Errorf("body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":9,"name":"deploy","fingerprint":"SHA256:abc","public_key":"ssh-ed25519 public","created_at":"2026-01-01T00:00:00Z"}}`))
		case "GET /ssh-keys":
			_, _ = w.Write([]byte(`{"data":[{"id":9,"name":"deploy","fingerprint":"SHA256:abc","public_key":"ssh-ed25519 public","created_at":"2026-01-01T00:00:00Z"}]}`))
		case "DELETE /ssh-keys/9":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	res := NewAccountAreaSSHKeyResource().(*AccountAreaSSHKeyResource)
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	var sch resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &sch)
	var meta resource.MetadataResponse
	res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &meta)
	if meta.TypeName != "ffxf_ssh_key" {
		t.Fatal(meta.TypeName)
	}
	plan := accountAreaResourcePlan(t, sch.Schema, &AccountAreaSSHKeyResourceModel{Name: types.StringValue("deploy"), PublicKey: types.StringValue("ssh-ed25519 public")})
	created := resource.CreateResponse{State: tfsdk.State{Schema: sch.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	var got AccountAreaSSHKeyResourceModel
	created.Diagnostics.Append(created.State.Get(ctx, &got)...)
	if created.Diagnostics.HasError() || got.ID.ValueInt64() != 9 || got.Fingerprint.ValueString() != "SHA256:abc" {
		t.Fatalf("created key = %#v; %v", got, created.Diagnostics)
	}
	read := resource.ReadResponse{State: created.State}
	res.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	del := resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: read.State}, &del)
	if del.Diagnostics.HasError() {
		t.Fatal(del.Diagnostics)
	}
	absent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":[]}`)) }))
	defer absent.Close()
	missingState := accountAreaResourceState(t, sch.Schema, &AccountAreaSSHKeyResourceModel{ID: types.Int64Value(9), Name: types.StringValue("deploy"), PublicKey: types.StringValue("ssh-ed25519 public"), Fingerprint: types.StringValue("f"), CreatedAt: types.StringValue("now")})
	missingRead := resource.ReadResponse{State: missingState}
	(&AccountAreaSSHKeyResource{client: testAPIClient(absent)}).Read(ctx, resource.ReadRequest{State: missingState}, &missingRead)
	if !missingRead.State.Raw.IsNull() {
		t.Fatal("missing SSH key should be removed")
	}
}

func TestAccountAreaVPCMemberLifecycleMoveAndRemoval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /vpcs/5/members":
			var body map[string]int64
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["vm"] != 12 {
				t.Errorf("body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"vm":12,"hostname":"web","ip":"10.0.0.12","configuration":"automatic"}}`))
		case "GET /vpcs/5":
			_, _ = w.Write([]byte(`{"data":{"id":5,"members":[{"vm":12,"hostname":"web","ip":"10.0.0.12","configuration":"automatic"}]}}`))
		case "POST /vpcs/5/members/12/move":
			var body map[string]int64
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["to"] != 6 {
				t.Errorf("move body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"vm":12,"hostname":"web","ip":"10.0.1.12","configuration":"automatic"}}`))
		case "DELETE /vpcs/6/members/12":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	res := NewAccountAreaVPCMemberResource().(*AccountAreaVPCMemberResource)
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	var sch resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &sch)
	var metadata resource.MetadataResponse
	res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_vpc_member" {
		t.Fatal(metadata.TypeName)
	}
	planModel := &AccountAreaVPCMemberResourceModel{VPCID: types.Int64Value(5), VMID: types.Int64Value(12)}
	plan := accountAreaResourcePlan(t, sch.Schema, planModel)
	created := resource.CreateResponse{State: tfsdk.State{Schema: sch.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	read := resource.ReadResponse{State: created.State}
	res.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	updatedModel := &AccountAreaVPCMemberResourceModel{ID: types.StringUnknown(), VPCID: types.Int64Value(6), VMID: types.Int64Value(12)}
	updatePlan := accountAreaResourcePlan(t, sch.Schema, updatedModel)
	updated := resource.UpdateResponse{State: read.State}
	res.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: read.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	var moved AccountAreaVPCMemberResourceModel
	updated.Diagnostics.Append(updated.State.Get(ctx, &moved)...)
	if updated.Diagnostics.HasError() || moved.VPCID.ValueInt64() != 6 || moved.IP.ValueString() != "10.0.1.12" {
		t.Fatalf("moved member = %#v; %v", moved, updated.Diagnostics)
	}
	deleted := resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: updated.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatal(deleted.Diagnostics)
	}
	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer missing.Close()
	missingState := accountAreaResourceState(t, sch.Schema, &AccountAreaVPCMemberResourceModel{ID: types.StringValue("5-12"), VPCID: types.Int64Value(5), VMID: types.Int64Value(12)})
	missingRead := resource.ReadResponse{State: missingState}
	(&AccountAreaVPCMemberResource{client: testAPIClient(missing)}).Read(ctx, resource.ReadRequest{State: missingState}, &missingRead)
	if !missingRead.State.Raw.IsNull() {
		t.Fatal("missing VPC should remove member state")
	}
}

func TestAccountAreaResourceCreateErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	defer server.Close()
	ctx := context.Background()
	ssh := NewAccountAreaSSHKeyResource().(*AccountAreaSSHKeyResource)
	ssh.client = testAPIClient(server)
	var sshSchema resource.SchemaResponse
	ssh.Schema(ctx, resource.SchemaRequest{}, &sshSchema)
	sshPlan := accountAreaResourcePlan(t, sshSchema.Schema, &AccountAreaSSHKeyResourceModel{Name: types.StringValue("x"), PublicKey: types.StringValue("public")})
	sshResp := resource.CreateResponse{State: tfsdk.State{Schema: sshSchema.Schema}}
	ssh.Create(ctx, resource.CreateRequest{Plan: sshPlan}, &sshResp)
	if !sshResp.Diagnostics.HasError() {
		t.Fatal("SSH create should report API failure")
	}
}

func accountAreaResourcePlan(t *testing.T, schema resourceSchema.Schema, model any) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if d := plan.Set(context.Background(), model); d.HasError() {
		t.Fatalf("set plan: %v", d)
	}
	return plan
}
func accountAreaResourceState(t *testing.T, schema resourceSchema.Schema, model any) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	if d := state.Set(context.Background(), model); d.HasError() {
		t.Fatalf("set state: %v", d)
	}
	return state
}

func TestAccountAreaResourceErrorBranches(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer server.Close()
	api := testAPIClient(server)
	ssh := NewAccountAreaSSHKeyResource().(*AccountAreaSSHKeyResource)
	ssh.client = api
	var sshSchema resource.SchemaResponse
	ssh.Schema(ctx, resource.SchemaRequest{}, &sshSchema)
	sshState := accountAreaResourceState(t, sshSchema.Schema, &AccountAreaSSHKeyResourceModel{ID: types.Int64Value(4), Name: types.StringValue("key"), PublicKey: types.StringValue("public"), Fingerprint: types.StringValue("f"), CreatedAt: types.StringValue("now")})
	read := resource.ReadResponse{State: sshState}
	ssh.Read(ctx, resource.ReadRequest{State: sshState}, &read)
	if !read.Diagnostics.HasError() {
		t.Fatal("SSH-key read should report an API error")
	}
	deleteResponse := resource.DeleteResponse{}
	ssh.Delete(ctx, resource.DeleteRequest{State: sshState}, &deleteResponse)
	if !deleteResponse.Diagnostics.HasError() {
		t.Fatal("SSH-key delete should report an API error")
	}
	updateResponse := resource.UpdateResponse{}
	ssh.Update(ctx, resource.UpdateRequest{}, &updateResponse)
	if !updateResponse.Diagnostics.HasError() {
		t.Fatal("SSH-key update should require replacement")
	}

	member := NewAccountAreaVPCMemberResource().(*AccountAreaVPCMemberResource)
	member.client = api
	var memberSchema resource.SchemaResponse
	member.Schema(ctx, resource.SchemaRequest{}, &memberSchema)
	memberState := accountAreaResourceState(t, memberSchema.Schema, &AccountAreaVPCMemberResourceModel{ID: types.StringValue("5-12"), VPCID: types.Int64Value(5), VMID: types.Int64Value(12), Hostname: types.StringValue("web"), IP: types.StringValue("10.0.0.12"), Configuration: types.StringValue("automatic")})
	createPlan := accountAreaResourcePlan(t, memberSchema.Schema, &AccountAreaVPCMemberResourceModel{VPCID: types.Int64Value(5), VMID: types.Int64Value(12)})
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: memberSchema.Schema}}
	member.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResponse)
	if !createResponse.Diagnostics.HasError() {
		t.Fatal("VPC-member create should report an API error")
	}
	memberRead := resource.ReadResponse{State: memberState}
	member.Read(ctx, resource.ReadRequest{State: memberState}, &memberRead)
	if !memberRead.Diagnostics.HasError() {
		t.Fatal("VPC-member read should report an API error")
	}
	movePlan := accountAreaResourcePlan(t, memberSchema.Schema, &AccountAreaVPCMemberResourceModel{VPCID: types.Int64Value(6), VMID: types.Int64Value(12)})
	moveResponse := resource.UpdateResponse{State: memberState}
	member.Update(ctx, resource.UpdateRequest{Plan: movePlan, State: memberState}, &moveResponse)
	if !moveResponse.Diagnostics.HasError() {
		t.Fatal("VPC-member move should report an API error")
	}
	memberDelete := resource.DeleteResponse{}
	member.Delete(ctx, resource.DeleteRequest{State: memberState}, &memberDelete)
	if !memberDelete.Diagnostics.HasError() {
		t.Fatal("VPC-member delete should report an API error")
	}

	absentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":{"id":5,"members":[]}}`)) }))
	defer absentServer.Close()
	absentRead := resource.ReadResponse{State: memberState}
	(&AccountAreaVPCMemberResource{client: testAPIClient(absentServer)}).Read(ctx, resource.ReadRequest{State: memberState}, &absentRead)
	if !absentRead.State.Raw.IsNull() {
		t.Fatal("member absent from VPC should be removed")
	}
}

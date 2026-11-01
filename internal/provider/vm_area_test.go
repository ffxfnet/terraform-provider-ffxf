package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVMAreaConstructorsExposeMetadataSchemaAndConfigure(t *testing.T) {
	ctx := context.Background()
	client := client.NewClient("https://example.test/v1", "secret")
	resources := []struct {
		name string
		new  func() resource.Resource
	}{
		{"ffxf_vm_ipv6_address", NewVMAreaIPv6AddressResource}, {"ffxf_vm_rdns", NewVMAreaRDNSResource},
		{"ffxf_vm_power_state", NewVMAreaPowerStateResource}, {"ffxf_vm_reinstall", NewVMAreaReinstallResource},
		{"ffxf_vm_backup_restore", NewVMAreaBackupRestoreResource},
	}
	for _, test := range resources {
		t.Run(test.name, func(t *testing.T) {
			r := test.new()
			var m resource.MetadataResponse
			r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &m)
			if m.TypeName != test.name {
				t.Fatalf("type=%q", m.TypeName)
			}
			var s resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &s)
			if s.Schema.Description == "" {
				t.Fatal("missing schema description")
			}
			for key, a := range s.Schema.Attributes {
				if a.GetDescription() == "" {
					t.Errorf("attribute %q has no description", key)
				}
			}
			r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &resource.ConfigureResponse{})
		})
	}
	datasources := []struct {
		name string
		new  func() datasource.DataSource
	}{
		{"ffxf_vms", NewVMAreaVMsDataSource}, {"ffxf_vm_network", NewVMAreaVMNetworkDataSource}, {"ffxf_vm_metrics", NewVMAreaVMMetricsDataSource}, {"ffxf_vm_backups", NewVMAreaVMBackupsDataSource}, {"ffxf_vm_actions", NewVMAreaVMActionsDataSource},
	}
	for _, test := range datasources {
		t.Run(test.name, func(t *testing.T) {
			d := test.new()
			var m datasource.MetadataResponse
			d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &m)
			if m.TypeName != test.name {
				t.Fatalf("type=%q", m.TypeName)
			}
			var s datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &s)
			if s.Schema.Description == "" {
				t.Fatal("missing schema description")
			}
			for key, a := range s.Schema.Attributes {
				if a.GetDescription() == "" {
					t.Errorf("attribute %q has no description", key)
				}
			}
			d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, &datasource.ConfigureResponse{})
			d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{}, &datasource.ConfigureResponse{})
		})
	}
}

func TestVMAreaVMsDataSourceReadsPages(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/vms" || r.URL.Query().Get("status") != "running" || r.URL.Query().Get("region") != "montreal" || r.URL.Query().Get("billing") != "hourly" {
			t.Errorf("request=%s %v", r.URL.Path, r.URL.Query())
		}
		if calls == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":1,"hostname":"web-1","status":"running","plan":"nano","region":"montreal","image":"debian-13","ipv4":"192.0.2.1","ipv6":null,"billing":{"mode":"hourly"}}],"meta":{"has_more":true,"next_cursor":"next"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":2,"hostname":"web-2","status":"running","plan":"nano","region":"montreal","image":"debian-13","ipv4":null,"ipv6":"2001:db8::2","billing":{"mode":"hourly"}}],"meta":{"has_more":false,"next_cursor":null}}`))
	}))
	defer server.Close()
	d := NewVMAreaVMsDataSource()
	d.(datasource.DataSourceWithConfigure).Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
	var s datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &s)
	model := VMAreaVMsModel{Status: types.StringValue("running"), Region: types.StringValue("montreal"), Billing: types.StringValue("hourly"), VMs: types.ListNull(vmAreaVMItemObjectType())}
	configState := tfsdk.State{Schema: s.Schema}
	if ds := configState.Set(context.Background(), &model); ds.HasError() {
		t.Fatal(ds)
	}
	config := tfsdk.Config{Schema: s.Schema, Raw: configState.Raw}
	state := tfsdk.State{Schema: s.Schema}
	response := datasource.ReadResponse{State: state}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var got VMAreaVMsModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var items []VMAreaVMItemModel
	response.Diagnostics.Append(got.VMs.ElementsAs(context.Background(), &items, false)...)
	if response.Diagnostics.HasError() || len(items) != 2 || items[1].ID.ValueInt64() != 2 || !items[0].IPv6.IsNull() {
		t.Fatalf("items=%#v diags=%v", items, response.Diagnostics)
	}
}

func TestVMAreaResourcesBasicLifecycleRequests(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("authorization=%q", r.Header.Get("Authorization"))
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /vms/7/network/ipv6":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["address"] != "2001:db8::7" {
				t.Errorf("ipv6 body=%v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"address":"2001:db8::7","version":6}}`))
		case "GET /vms/7/network":
			_, _ = w.Write([]byte(`{"data":{"ipv6":[{"address":"2001:db8::7","version":6}],"ipv4":[]}}`))
		case "DELETE /vms/7/network/ipv6/2001:db8::7":
			w.WriteHeader(http.StatusNoContent)
		case "PUT /vms/7/network/rdns":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["address"] != "2001:db8::7" {
				t.Errorf("rdns body=%v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"address":"2001:db8::7","rdns":"vm.example.net"}}`))
		case "POST /vms/7/actions":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["type"] == "" {
				t.Error("action type missing")
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
		case "GET /vms/7":
			_, _ = w.Write([]byte(`{"data":{"id":7,"hostname":"vm-7","status":"running","plan":"nano","region":"montreal","image":"debian-13","billing":{"mode":"hourly"}}}`))
		case "POST /vms/7/backups/backup-7/restore":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	api := testAPIClient(server)

	ipv6 := NewVMAreaIPv6AddressResource()
	ipv6.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	im := VMAreaIPv6AddressModel{VMID: types.Int64Value(7), Address: types.StringValue("2001:db8::7")}
	plan := vmAreaResourcePlan(t, ipv6, &im)
	resp := resource.CreateResponse{State: vmAreaResourceState(t, ipv6, &VMAreaIPv6AddressModel{})}
	ipv6.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	readState := resp.State
	readResp := resource.ReadResponse{State: readState}
	ipv6.Read(ctx, resource.ReadRequest{State: readState}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	del := resource.DeleteResponse{}
	ipv6.Delete(ctx, resource.DeleteRequest{State: readState}, &del)
	if del.Diagnostics.HasError() {
		t.Fatal(del.Diagnostics)
	}

	rdns := NewVMAreaRDNSResource()
	rdns.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	rm := VMAreaRDNSModel{VMID: types.Int64Value(7), Address: types.StringValue("2001:db8::7"), Hostname: types.StringValue("vm.example.net")}
	rp := vmAreaResourcePlan(t, rdns, &rm)
	rr := resource.CreateResponse{State: vmAreaResourceState(t, rdns, &VMAreaRDNSModel{})}
	rdns.Create(ctx, resource.CreateRequest{Plan: rp}, &rr)
	if rr.Diagnostics.HasError() {
		t.Fatal(rr.Diagnostics)
	}
	rdns.Update(ctx, resource.UpdateRequest{Plan: rp, State: rr.State}, &resource.UpdateResponse{State: rr.State})
	rd := resource.DeleteResponse{}
	rdns.Delete(ctx, resource.DeleteRequest{State: rr.State}, &rd)
	if rd.Diagnostics.HasError() {
		t.Fatal(rd.Diagnostics)
	}

	power := NewVMAreaPowerStateResource()
	power.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	pm := VMAreaPowerStateModel{VMID: types.Int64Value(7), State: types.StringValue("running"), Method: types.StringValue("graceful"), RebootTrigger: types.StringNull()}
	pp := vmAreaResourcePlan(t, power, &pm)
	pr := resource.CreateResponse{State: vmAreaResourceState(t, power, &VMAreaPowerStateModel{})}
	power.Create(ctx, resource.CreateRequest{Plan: pp}, &pr)
	if pr.Diagnostics.HasError() {
		t.Fatal(pr.Diagnostics)
	}
	pstate := pr.State
	pread := resource.ReadResponse{State: pstate}
	power.Read(ctx, resource.ReadRequest{State: pstate}, &pread)
	if pread.Diagnostics.HasError() {
		t.Fatal(pread.Diagnostics)
	}
	power.Delete(ctx, resource.DeleteRequest{State: pstate}, &resource.DeleteResponse{})

	restore := NewVMAreaBackupRestoreResource()
	restore.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	bm := VMAreaBackupRestoreModel{VMID: types.Int64Value(7), Backup: types.StringValue("backup-7")}
	bp := vmAreaResourcePlan(t, restore, &bm)
	br := resource.CreateResponse{State: vmAreaResourceState(t, restore, &VMAreaBackupRestoreModel{})}
	restore.Create(ctx, resource.CreateRequest{Plan: bp}, &br)
	if br.Diagnostics.HasError() {
		t.Fatal(br.Diagnostics)
	}
}

func vmAreaResourcePlan(t *testing.T, r resource.Resource, model any) tfsdk.Plan {
	t.Helper()
	var schema resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schema)
	p := tfsdk.Plan{Schema: schema.Schema}
	if d := p.Set(context.Background(), model); d.HasError() {
		t.Fatal(d)
	}
	return p
}
func vmAreaResourceState(t *testing.T, r resource.Resource, model any) tfsdk.State {
	t.Helper()
	var schema resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schema)
	s := tfsdk.State{Schema: schema.Schema}
	if d := s.Set(context.Background(), model); d.HasError() {
		t.Fatal(d)
	}
	return s
}

func vmAreaVMItemObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.Int64Type, "hostname": types.StringType, "status": types.StringType, "plan": types.StringType, "region": types.StringType, "image": types.StringType, "ipv4": types.StringType, "ipv6": types.StringType, "billing_mode": types.StringType}}
}

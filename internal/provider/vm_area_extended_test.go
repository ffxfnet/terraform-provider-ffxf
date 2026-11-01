package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVMAreaDetailDataSourcesReadAndReportErrors(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("authorization header missing")
		}
		switch r.URL.Path {
		case "/vms/7/network":
			_, _ = w.Write([]byte(`{"data":{"ipv4":[{"address":"192.0.2.7","version":4,"primary":true,"gateway":"192.0.2.1","prefix_length":24,"rdns":null}],"ipv6":[{"address":"2001:db8::7","version":6,"primary":false,"gateway":null,"prefix_length":64,"rdns":"vm.example.net"}],"ipv6_prefix":"2001:db8::/64","private":[{"vpc":4,"name":"private","address":"10.0.0.2","prefix_length":24,"configuration":"automatic"}],"port_mbps":100}}`))
		case "/vms/7/metrics":
			if r.URL.Query().Get("timeframe") != "hour" {
				t.Errorf("timeframe = %q", r.URL.Query().Get("timeframe"))
			}
			_, _ = w.Write([]byte(`{"data":{"timeframe":"hour","step_seconds":60,"series":[{"time":"2026-01-01T00:00:00Z","cpu":0.25,"memory_bytes":null}]}}`))
		case "/vms/7/backups":
			_, _ = w.Write([]byte(`{"data":[{"id":"backup-1","type":"manual","created_at":"2026-01-01T00:00:00Z","size_bytes":10,"restorable":true},{"id":"backup-2","type":"automatic","size_bytes":null,"restorable":false}],"meta":{"automatic_enabled":true,"retention":5}}`))
		case "/actions":
			if r.URL.Query().Get("status") != "error" {
				t.Errorf("status = %q", r.URL.Query().Get("status"))
			}
			_, _ = w.Write([]byte(`{"data":[{"id":9,"type":"reinstall","status":"error","vm_id":7,"source":"api","created_at":"2026-01-01T00:00:00Z","error":{"message":"failed"}}],"meta":{"has_more":false,"next_cursor":null}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	api := testAPIClient(server)

	network := NewVMAreaVMNetworkDataSource()
	network.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	networkModel := VMAreaVMNetworkModel{VMID: types.Int64Value(7), IPv4: types.ListNull(vmAreaAddressObjectType()), IPv6: types.ListNull(vmAreaAddressObjectType()), Private: types.ListNull(vmAreaPrivateObjectType())}
	networkConfig := vmAreaDataSourceConfig(t, network, &networkModel)
	networkResponse := datasource.ReadResponse{State: tfsdk.State{Schema: networkConfig.Schema}}
	network.Read(ctx, datasource.ReadRequest{Config: networkConfig}, &networkResponse)
	if networkResponse.Diagnostics.HasError() {
		t.Fatal(networkResponse.Diagnostics)
	}
	var networkResult VMAreaVMNetworkModel
	networkResponse.Diagnostics.Append(networkResponse.State.Get(ctx, &networkResult)...)
	if networkResponse.Diagnostics.HasError() || networkResult.PortMbps.ValueInt64() != 100 || networkResult.Private.IsNull() || networkResult.IPv4.IsNull() {
		t.Fatalf("network state=%#v diagnostics=%v", networkResult, networkResponse.Diagnostics)
	}

	metrics := NewVMAreaVMMetricsDataSource()
	metrics.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	metricsModel := VMAreaMetricsModel{VMID: types.Int64Value(7), Timeframe: types.StringNull(), Series: types.ListNull(vmAreaMetricObjectType())}
	metricsConfig := vmAreaDataSourceConfig(t, metrics, &metricsModel)
	metricsResponse := datasource.ReadResponse{State: tfsdk.State{Schema: metricsConfig.Schema}}
	metrics.Read(ctx, datasource.ReadRequest{Config: metricsConfig}, &metricsResponse)
	if metricsResponse.Diagnostics.HasError() {
		t.Fatal(metricsResponse.Diagnostics)
	}
	var metricsResult VMAreaMetricsModel
	metricsResponse.Diagnostics.Append(metricsResponse.State.Get(ctx, &metricsResult)...)
	if metricsResponse.Diagnostics.HasError() || metricsResult.Timeframe.ValueString() != "hour" || metricsResult.StepSeconds.ValueInt64() != 60 {
		t.Fatalf("metrics state=%#v diagnostics=%v", metricsResult, metricsResponse.Diagnostics)
	}

	backups := NewVMAreaVMBackupsDataSource()
	backups.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	backupsModel := VMAreaBackupsModel{VMID: types.Int64Value(7), Backups: types.ListNull(vmAreaBackupObjectType())}
	backupsConfig := vmAreaDataSourceConfig(t, backups, &backupsModel)
	backupsResponse := datasource.ReadResponse{State: tfsdk.State{Schema: backupsConfig.Schema}}
	backups.Read(ctx, datasource.ReadRequest{Config: backupsConfig}, &backupsResponse)
	if backupsResponse.Diagnostics.HasError() {
		t.Fatal(backupsResponse.Diagnostics)
	}
	var backupsResult VMAreaBackupsModel
	backupsResponse.Diagnostics.Append(backupsResponse.State.Get(ctx, &backupsResult)...)
	if backupsResponse.Diagnostics.HasError() || !backupsResult.AutomaticEnabled.ValueBool() || backupsResult.Retention.ValueInt64() != 5 {
		t.Fatalf("backups state=%#v diagnostics=%v", backupsResult, backupsResponse.Diagnostics)
	}

	actions := NewVMAreaVMActionsDataSource()
	actions.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	actionsModel := VMAreaActionsModel{VMID: types.Int64Null(), Status: types.StringValue("error"), Actions: types.ListNull(vmAreaActionObjectType())}
	actionsConfig := vmAreaDataSourceConfig(t, actions, &actionsModel)
	actionsResponse := datasource.ReadResponse{State: tfsdk.State{Schema: actionsConfig.Schema}}
	actions.Read(ctx, datasource.ReadRequest{Config: actionsConfig}, &actionsResponse)
	if actionsResponse.Diagnostics.HasError() {
		t.Fatal(actionsResponse.Diagnostics)
	}
	var actionsResult VMAreaActionsModel
	actionsResponse.Diagnostics.Append(actionsResponse.State.Get(ctx, &actionsResult)...)
	if actionsResponse.Diagnostics.HasError() || actionsResult.Actions.IsNull() {
		t.Fatalf("actions state=%#v diagnostics=%v", actionsResult, actionsResponse.Diagnostics)
	}

	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer badServer.Close()
	metrics.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(badServer)}, &datasource.ConfigureResponse{})
	badResponse := datasource.ReadResponse{State: tfsdk.State{Schema: metricsConfig.Schema}}
	metrics.Read(ctx, datasource.ReadRequest{Config: metricsConfig}, &badResponse)
	if !badResponse.Diagnostics.HasError() {
		t.Fatal("expected metrics API error diagnostic")
	}
	for _, test := range []struct {
		name   string
		ds     datasource.DataSource
		config tfsdk.Config
	}{
		{"network", NewVMAreaVMNetworkDataSource(), vmAreaDataSourceConfig(t, NewVMAreaVMNetworkDataSource(), &VMAreaVMNetworkModel{VMID: types.Int64Value(7), IPv4: types.ListNull(vmAreaAddressObjectType()), IPv6: types.ListNull(vmAreaAddressObjectType()), Private: types.ListNull(vmAreaPrivateObjectType())})},
		{"backups", NewVMAreaVMBackupsDataSource(), vmAreaDataSourceConfig(t, NewVMAreaVMBackupsDataSource(), &VMAreaBackupsModel{VMID: types.Int64Value(7), Backups: types.ListNull(vmAreaBackupObjectType())})},
		{"actions", NewVMAreaVMActionsDataSource(), vmAreaDataSourceConfig(t, NewVMAreaVMActionsDataSource(), &VMAreaActionsModel{VMID: types.Int64Null(), Status: types.StringNull(), Actions: types.ListNull(vmAreaActionObjectType())})},
	} {
		test.ds.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(badServer)}, &datasource.ConfigureResponse{})
		response := datasource.ReadResponse{State: tfsdk.State{Schema: test.config.Schema}}
		test.ds.Read(ctx, datasource.ReadRequest{Config: test.config}, &response)
		if !response.Diagnostics.HasError() {
			t.Errorf("%s should report an API error", test.name)
		}
	}

	conflictModel := VMAreaActionsModel{VMID: types.Int64Value(7), Status: types.StringValue("error"), Actions: types.ListNull(vmAreaActionObjectType())}
	conflictConfig := vmAreaDataSourceConfig(t, actions, &conflictModel)
	conflictResponse := datasource.ReadResponse{State: tfsdk.State{Schema: conflictConfig.Schema}}
	actions.Read(ctx, datasource.ReadRequest{Config: conflictConfig}, &conflictResponse)
	if !conflictResponse.Diagnostics.HasError() {
		t.Fatal("expected unsupported vm_id + status diagnostic")
	}
}

func TestVMAreaOneShotResourcesAndPowerUpdates(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /vms/7/actions":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["type"] == "reinstall" {
				if body["image"] != "debian-13" {
					t.Errorf("reinstall body=%v", body)
				}
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
				return
			}
			if body["type"] != "start" && body["type"] != "shutdown" && body["type"] != "stop" && body["type"] != "reboot" {
				t.Errorf("power action=%v", body)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
		case "POST /vms/7/backups/backup-2/restore":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	api := testAPIClient(server)

	reinstall := NewVMAreaReinstallResource()
	reinstall.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	keys, _ := types.SetValue(types.StringType, []attr.Value{types.StringValue("key-id")})
	reinstallModel := VMAreaReinstallModel{VMID: types.Int64Value(7), Image: types.StringValue("debian-13"), SSHKeys: keys}
	reinstallPlan := vmAreaResourcePlan(t, reinstall, &reinstallModel)
	reinstallResponse := resource.CreateResponse{State: vmAreaResourceState(t, reinstall, &reinstallModel)}
	reinstall.Create(ctx, resource.CreateRequest{Plan: reinstallPlan}, &reinstallResponse)
	if reinstallResponse.Diagnostics.HasError() {
		t.Fatal(reinstallResponse.Diagnostics)
	}
	reinstallRead := resource.ReadResponse{State: reinstallResponse.State}
	reinstall.Read(ctx, resource.ReadRequest{State: reinstallResponse.State}, &reinstallRead)
	if reinstallRead.Diagnostics.HasError() {
		t.Fatal(reinstallRead.Diagnostics)
	}
	reinstall.Update(ctx, resource.UpdateRequest{}, &resource.UpdateResponse{})
	if !vmAreaResourceUpdateDiagnostics(reinstall) {
		t.Fatal("reinstall update should be rejected")
	}
	reinstall.Delete(ctx, resource.DeleteRequest{}, &resource.DeleteResponse{})

	restore := NewVMAreaBackupRestoreResource()
	restore.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	restoreModel := VMAreaBackupRestoreModel{VMID: types.Int64Value(7), Backup: types.StringValue("backup-2")}
	restorePlan := vmAreaResourcePlan(t, restore, &restoreModel)
	restoreResponse := resource.CreateResponse{State: vmAreaResourceState(t, restore, &restoreModel)}
	restore.Create(ctx, resource.CreateRequest{Plan: restorePlan}, &restoreResponse)
	if restoreResponse.Diagnostics.HasError() {
		t.Fatal(restoreResponse.Diagnostics)
	}
	restoreRead := resource.ReadResponse{State: restoreResponse.State}
	restore.Read(ctx, resource.ReadRequest{State: restoreResponse.State}, &restoreRead)
	if restoreRead.Diagnostics.HasError() {
		t.Fatal(restoreRead.Diagnostics)
	}
	restoreUpdate := resource.UpdateResponse{}
	restore.Update(ctx, resource.UpdateRequest{}, &restoreUpdate)
	if !restoreUpdate.Diagnostics.HasError() {
		t.Fatal("restore update should be rejected")
	}
	restore.Delete(ctx, resource.DeleteRequest{}, &resource.DeleteResponse{})

	power := NewVMAreaPowerStateResource()
	power.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	old := VMAreaPowerStateModel{VMID: types.Int64Value(7), State: types.StringValue("running"), Method: types.StringValue("graceful"), RebootTrigger: types.StringValue("old")}
	oldState := vmAreaResourceState(t, power, &old)
	for _, next := range []VMAreaPowerStateModel{
		{VMID: types.Int64Value(7), State: types.StringValue("stopped"), Method: types.StringValue("forced"), RebootTrigger: types.StringValue("old")},
		{VMID: types.Int64Value(7), State: types.StringValue("running"), Method: types.StringValue("graceful"), RebootTrigger: types.StringValue("new")},
	} {
		plan := vmAreaResourcePlan(t, power, &next)
		response := resource.UpdateResponse{State: oldState}
		power.Update(ctx, resource.UpdateRequest{Plan: plan, State: oldState}, &response)
		if response.Diagnostics.HasError() {
			t.Fatal(response.Diagnostics)
		}
	}
}

func TestVMAreaIPv6ImportMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	resourceImpl := NewVMAreaIPv6AddressResource()
	valid := resource.ImportStateResponse{State: vmAreaResourceState(t, resourceImpl, &VMAreaIPv6AddressModel{})}
	resourceImpl.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "42/2001:db8::42"}, &valid)
	if valid.Diagnostics.HasError() {
		t.Fatal(valid.Diagnostics)
	}
	for _, id := range []string{"bad", "0/address", "42/"} {
		invalid := resource.ImportStateResponse{State: vmAreaResourceState(t, resourceImpl, &VMAreaIPv6AddressModel{})}
		resourceImpl.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: id}, &invalid)
		if !invalid.Diagnostics.HasError() {
			t.Errorf("import %q should fail", id)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":{"ipv4":[],"ipv6":[]}}`)) }))
	defer server.Close()
	resourceImpl.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	model := VMAreaIPv6AddressModel{ID: types.StringValue("42/2001:db8::42"), VMID: types.Int64Value(42), Address: types.StringValue("2001:db8::42")}
	state := vmAreaResourceState(t, resourceImpl, &model)
	read := resource.ReadResponse{State: state}
	resourceImpl.Read(ctx, resource.ReadRequest{State: state}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("missing address state not removed: %v", read.Diagnostics)
	}
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	defer badServer.Close()
	resourceImpl.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(badServer)}, &resource.ConfigureResponse{})
	plan := vmAreaResourcePlan(t, resourceImpl, &model)
	create := resource.CreateResponse{State: state}
	resourceImpl.Create(ctx, resource.CreateRequest{Plan: plan}, &create)
	if !create.Diagnostics.HasError() {
		t.Fatal("expected IPv6 API error")
	}
}

func TestVMAreaValidatorRejectsInvalidValues(t *testing.T) {
	v := VMAreaStringOneOfValidator{Values: []string{"now", "period_end"}}
	response := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("destroy_when"), ConfigValue: types.StringValue("later")}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected invalid enum diagnostic")
	}
	response = validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("destroy_when"), ConfigValue: types.StringNull()}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal("null values should be left to schema requiredness")
	}
}

func vmAreaDataSourceConfig(t *testing.T, d datasource.DataSource, model any) tfsdk.Config {
	t.Helper()
	var schema datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	if diags := state.Set(context.Background(), model); diags.HasError() {
		t.Fatal(diags)
	}
	return tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}
}

func vmAreaAddressObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"address": types.StringType, "version": types.Int64Type, "primary": types.BoolType, "gateway": types.StringType, "prefix_length": types.Int64Type, "rdns": types.StringType}}
}
func vmAreaPrivateObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"vpc": types.Int64Type, "name": types.StringType, "address": types.StringType, "prefix_length": types.Int64Type, "configuration": types.StringType}}
}
func vmAreaMetricObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"time": types.StringType, "cpu": types.Float64Type, "memory_bytes": types.Float64Type, "disk_read_bytes": types.Float64Type, "disk_write_bytes": types.Float64Type, "net_in_bytes": types.Float64Type, "net_out_bytes": types.Float64Type}}
}
func vmAreaBackupObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.StringType, "type": types.StringType, "created_at": types.StringType, "size_bytes": types.Int64Type, "restorable": types.BoolType}}
}
func vmAreaActionObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.Int64Type, "type": types.StringType, "status": types.StringType, "vm_id": types.Int64Type, "source": types.StringType, "created_at": types.StringType, "started_at": types.StringType, "finished_at": types.StringType, "error": types.StringType}}
}
func vmAreaResourceUpdateDiagnostics(r resource.Resource) bool {
	response := resource.UpdateResponse{}
	r.Update(context.Background(), resource.UpdateRequest{}, &response)
	return response.Diagnostics.HasError()
}

func TestVMAreaResourceErrorAndReadBranches(t *testing.T) {
	ctx := context.Background()
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	defer badServer.Close()
	bad := testAPIClient(badServer)
	ipv6 := NewVMAreaIPv6AddressResource()
	ipv6.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: bad}, &resource.ConfigureResponse{})
	im := VMAreaIPv6AddressModel{ID: types.StringValue("7/2001:db8::7"), VMID: types.Int64Value(7), Address: types.StringValue("2001:db8::7")}
	is := vmAreaResourceState(t, ipv6, &im)
	ip := vmAreaResourcePlan(t, ipv6, &im)
	ic := resource.CreateResponse{State: is}
	ipv6.Create(ctx, resource.CreateRequest{Plan: ip}, &ic)
	if !ic.Diagnostics.HasError() {
		t.Fatal("expected IPv6 create error")
	}
	iu := resource.UpdateResponse{}
	ipv6.Update(ctx, resource.UpdateRequest{}, &iu)
	if !iu.Diagnostics.HasError() {
		t.Fatal("IPv6 update should be rejected")
	}
	id := resource.DeleteResponse{}
	ipv6.Delete(ctx, resource.DeleteRequest{State: is}, &id)
	if !id.Diagnostics.HasError() {
		t.Fatal("expected IPv6 delete error")
	}
	ir := resource.ReadResponse{State: is}
	ipv6.Read(ctx, resource.ReadRequest{State: is}, &ir)
	if !ir.Diagnostics.HasError() {
		t.Fatal("expected IPv6 read error")
	}

	rdns := NewVMAreaRDNSResource()
	rdns.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: bad}, &resource.ConfigureResponse{})
	rm := VMAreaRDNSModel{VMID: types.Int64Value(7), Address: types.StringValue("2001:db8::7"), Hostname: types.StringValue("vm.example.net")}
	rs := vmAreaResourceState(t, rdns, &rm)
	rp := vmAreaResourcePlan(t, rdns, &rm)
	rc := resource.CreateResponse{State: rs}
	rdns.Create(ctx, resource.CreateRequest{Plan: rp}, &rc)
	if !rc.Diagnostics.HasError() {
		t.Fatal("expected rDNS create error")
	}
	ru := resource.UpdateResponse{State: rs}
	rdns.Update(ctx, resource.UpdateRequest{Plan: rp, State: rs}, &ru)
	if !ru.Diagnostics.HasError() {
		t.Fatal("expected rDNS update error")
	}
	rd := resource.DeleteResponse{}
	rdns.Delete(ctx, resource.DeleteRequest{State: rs}, &rd)
	if !rd.Diagnostics.HasError() {
		t.Fatal("expected rDNS delete error")
	}
	rr := resource.ReadResponse{State: rs}
	rdns.Read(ctx, resource.ReadRequest{State: rs}, &rr)
	if !rr.Diagnostics.HasError() {
		t.Fatal("expected rDNS read error")
	}

	power := NewVMAreaPowerStateResource()
	power.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: bad}, &resource.ConfigureResponse{})
	pm := VMAreaPowerStateModel{VMID: types.Int64Value(7), State: types.StringValue("stopped"), Method: types.StringValue("graceful"), RebootTrigger: types.StringNull()}
	ps := vmAreaResourceState(t, power, &pm)
	pp := vmAreaResourcePlan(t, power, &pm)
	pc := resource.CreateResponse{State: ps}
	power.Create(ctx, resource.CreateRequest{Plan: pp}, &pc)
	if !pc.Diagnostics.HasError() {
		t.Fatal("expected power create error")
	}
	pu := resource.UpdateResponse{State: ps}
	power.Update(ctx, resource.UpdateRequest{Plan: pp, State: ps}, &pu)
	if !pu.Diagnostics.HasError() {
		t.Fatal("expected power update error")
	}
	pr := resource.ReadResponse{State: ps}
	power.Read(ctx, resource.ReadRequest{State: ps}, &pr)
	if !pr.Diagnostics.HasError() {
		t.Fatal("expected power read error")
	}
	power.Delete(ctx, resource.DeleteRequest{State: ps}, &resource.DeleteResponse{})

	reinstall := NewVMAreaReinstallResource()
	reinstall.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: bad}, &resource.ConfigureResponse{})
	keys := types.SetNull(types.StringType)
	imodel := VMAreaReinstallModel{VMID: types.Int64Value(7), Image: types.StringValue("debian-13"), SSHKeys: keys}
	rschema := vmAreaResourceState(t, reinstall, &imodel)
	rplan := vmAreaResourcePlan(t, reinstall, &imodel)
	rcreate := resource.CreateResponse{State: rschema}
	reinstall.Create(ctx, resource.CreateRequest{Plan: rplan}, &rcreate)
	if !rcreate.Diagnostics.HasError() {
		t.Fatal("expected reinstall API error")
	}
	rread := resource.ReadResponse{State: rschema}
	reinstall.Read(ctx, resource.ReadRequest{State: rschema}, &rread)
	if rread.Diagnostics.HasError() {
		t.Fatal(rread.Diagnostics)
	}
	rupdate := resource.UpdateResponse{}
	reinstall.Update(ctx, resource.UpdateRequest{}, &rupdate)
	if !rupdate.Diagnostics.HasError() {
		t.Fatal("reinstall update should be rejected")
	}
	reinstall.Delete(ctx, resource.DeleteRequest{State: rschema}, &resource.DeleteResponse{})

	restore := NewVMAreaBackupRestoreResource()
	restore.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: bad}, &resource.ConfigureResponse{})
	bm := VMAreaBackupRestoreModel{VMID: types.Int64Value(7), Backup: types.StringValue("backup-1")}
	bs := vmAreaResourceState(t, restore, &bm)
	bp := vmAreaResourcePlan(t, restore, &bm)
	bc := resource.CreateResponse{State: bs}
	restore.Create(ctx, resource.CreateRequest{Plan: bp}, &bc)
	if !bc.Diagnostics.HasError() {
		t.Fatal("expected backup restore error")
	}
	br := resource.ReadResponse{State: bs}
	restore.Read(ctx, resource.ReadRequest{State: bs}, &br)
	if br.Diagnostics.HasError() {
		t.Fatal(br.Diagnostics)
	}
	bu := resource.UpdateResponse{}
	restore.Update(ctx, resource.UpdateRequest{}, &bu)
	if !bu.Diagnostics.HasError() {
		t.Fatal("restore update should be rejected")
	}
	restore.Delete(ctx, resource.DeleteRequest{State: bs}, &resource.DeleteResponse{})

	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer missing.Close()
	ipv6.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(missing)}, &resource.ConfigureResponse{})
	missingResponse := resource.ReadResponse{State: is}
	ipv6.Read(ctx, resource.ReadRequest{State: is}, &missingResponse)
	if missingResponse.Diagnostics.HasError() || !missingResponse.State.Raw.IsNull() {
		t.Fatalf("IPv6 404 not removed: %v", missingResponse.Diagnostics)
	}
	missingResponse = resource.ReadResponse{State: rs}
	rdns.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(missing)}, &resource.ConfigureResponse{})
	rdns.Read(ctx, resource.ReadRequest{State: rs}, &missingResponse)
	if !missingResponse.Diagnostics.HasError() {
		t.Fatal("rDNS 404 should report an API error")
	}
}

func TestVMAreaRegistrationNullableHelpersAndNotFoundPaths(t *testing.T) {
	resources := vmAreaResources()
	datasources := vmAreaDataSources()
	if len(resources) != 5 || len(datasources) != 5 {
		t.Fatalf("registered resources=%d data sources=%d", len(resources), len(datasources))
	}
	for _, factory := range append(resources, func() resource.Resource { return NewVMAreaIPv6AddressResource() }, func() resource.Resource { return NewVMAreaRDNSResource() }, func() resource.Resource { return NewVMAreaPowerStateResource() }, func() resource.Resource { return NewVMAreaReinstallResource() }, func() resource.Resource { return NewVMAreaBackupRestoreResource() }) {
		if factory() == nil {
			t.Fatal("nil VM resource constructor")
		}
	}
	for _, factory := range datasources {
		if factory() == nil {
			t.Fatal("nil VM data source constructor")
		}
	}
	if !vmAreaNullableInt(nil).IsNull() {
		t.Fatal("nullable integer helper should preserve null")
	}
	v := 4
	if got := vmAreaNullableInt(&v).ValueInt64(); got != 4 {
		t.Fatalf("nullable integer=%d", got)
	}
	validator := VMAreaStringOneOfValidator{Values: []string{"now"}}
	if validator.Description(context.Background()) == "" || validator.MarkdownDescription(context.Background()) == "" {
		t.Fatal("validator descriptions must be available")
	}

	ctx := context.Background()
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer notFound.Close()
	api := testAPIClient(notFound)
	ipv6 := NewVMAreaIPv6AddressResource()
	ipv6.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	im := VMAreaIPv6AddressModel{ID: types.StringValue("7/2001:db8::7"), VMID: types.Int64Value(7), Address: types.StringValue("2001:db8::7")}
	state := vmAreaResourceState(t, ipv6, &im)
	read := resource.ReadResponse{State: state}
	ipv6.Read(ctx, resource.ReadRequest{State: state}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("IPv6 404 should remove state: %v", read.Diagnostics)
	}
	del := resource.DeleteResponse{}
	ipv6.Delete(ctx, resource.DeleteRequest{State: state}, &del)
	if del.Diagnostics.HasError() {
		t.Fatalf("deleting already-missing IPv6 should succeed: %v", del.Diagnostics)
	}

	rdns := NewVMAreaRDNSResource()
	rdns.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	rm := VMAreaRDNSModel{VMID: types.Int64Value(7), Address: types.StringValue("192.0.2.7"), Hostname: types.StringValue("old.example")}
	rs := vmAreaResourceState(t, rdns, &rm)
	rr := resource.ReadResponse{State: rs}
	rdns.Read(ctx, resource.ReadRequest{State: rs}, &rr)
	if !rr.Diagnostics.HasError() {
		t.Fatal("missing rDNS network should report API error")
	}

	found := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ipv4":[{"address":"192.0.2.7","version":4,"rdns":"fresh.example"}],"ipv6":[]}}`))
	}))
	defer found.Close()
	rdns.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(found)}, &resource.ConfigureResponse{})
	rr = resource.ReadResponse{State: rs}
	rdns.Read(ctx, resource.ReadRequest{State: rs}, &rr)
	if rr.Diagnostics.HasError() {
		t.Fatal(rr.Diagnostics)
	}
	var refreshed VMAreaRDNSModel
	rr.Diagnostics.Append(rr.State.Get(ctx, &refreshed)...)
	if rr.Diagnostics.HasError() || refreshed.Hostname.ValueString() != "fresh.example" {
		t.Fatalf("rDNS refresh=%#v %v", refreshed, rr.Diagnostics)
	}

	power := NewVMAreaPowerStateResource()
	power.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: api}, &resource.ConfigureResponse{})
	pm := VMAreaPowerStateModel{VMID: types.Int64Value(7), State: types.StringValue("running"), Method: types.StringValue("graceful"), RebootTrigger: types.StringNull()}
	ps := vmAreaResourceState(t, power, &pm)
	pr := resource.ReadResponse{State: ps}
	power.Read(ctx, resource.ReadRequest{State: ps}, &pr)
	if pr.Diagnostics.HasError() || !pr.State.Raw.IsNull() {
		t.Fatalf("missing VM should remove power state: %v", pr.Diagnostics)
	}
}

func TestVMAreaInstanceSSHKeysAndDestroyPolicy(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("authorization missing")
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /vms":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			keys, ok := body["ssh_keys"].([]any)
			if !ok || len(keys) != 1 || keys[0] != "key-fingerprint" {
				t.Errorf("create body=%v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"vm":{"id":77,"hostname":"host & name","status":"running","ipv4":"192.0.2.77"},"action":{"id":0}}}`))
		case "DELETE /vms/77":
			if r.URL.Query().Get("confirm") != "host & name" || r.URL.Query().Get("when") != "period_end" {
				t.Errorf("delete query=%v", r.URL.Query())
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	instance := &InstanceResource{client: testAPIClient(server)}
	keys := []string{"key-fingerprint"}
	model := InstanceResourceModel{Hostname: types.StringValue("host & name"), Plan: types.StringValue("nano"), Region: types.StringValue("montreal"), Image: types.StringValue("debian-13"), Billing: types.StringValue("hourly"), VMAreaSSHKeys: keys, VMAreaDestroyWhen: types.StringValue("period_end"), IPv4: types.StringUnknown()}
	plan := newInstancePlan(t, model)
	response := resource.CreateResponse{State: newInstanceState(t, InstanceResourceModel{Hostname: types.StringNull(), Plan: types.StringNull(), Region: types.StringNull(), Image: types.StringNull(), Billing: types.StringNull(), IPv4: types.StringNull()})}
	instance.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var created InstanceResourceModel
	response.Diagnostics.Append(response.State.Get(ctx, &created)...)
	if response.Diagnostics.HasError() || created.ID.ValueInt64() != 77 || created.VMAreaDestroyWhen.ValueString() != "period_end" {
		t.Fatalf("instance state=%#v %v", created, response.Diagnostics)
	}
	instance.Delete(ctx, resource.DeleteRequest{State: response.State}, &resource.DeleteResponse{})
}

func TestVMAreaInstanceDeleteWaitsForAction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "DELETE /vms/88":
			_, _ = w.Write([]byte(`{"data":{"action":{"id":92}}}`))
		case "GET /actions/92":
			_, _ = w.Write([]byte(`{"data":{"id":92,"status":"completed"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	instance := &InstanceResource{client: testAPIClient(server)}
	model := InstanceResourceModel{ID: types.Int64Value(88), Hostname: types.StringValue("host-88"), Plan: types.StringValue("nano"), Region: types.StringValue("montreal"), Image: types.StringValue("debian-13"), Billing: types.StringValue("hourly"), IPv4: types.StringNull(), VMAreaDestroyWhen: types.StringValue("now")}
	state := newInstanceState(t, model)
	response := resource.DeleteResponse{}
	instance.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}

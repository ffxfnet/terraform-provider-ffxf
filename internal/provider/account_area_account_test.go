package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountAreaAccountAndUsageDataSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/account":
			_, _ = w.Write([]byte(`{"data":{"currency":"CAD","billing_mode":"prepaid","credit":{"balance":"120.00","paid":"100","gifted":"20","gifted_expires_at":null},"hourly":{"burn_rate":"0.5","runway_hours":240,"active_machines":2},"budget":{"monthly":"300","month_to_date":"50","percent":16.7,"cap_percent":100},"quota":{"limit":10,"used":2,"remaining":8},"auto_topup":{"enabled":true,"threshold":"10","amount":"50"}}}`))
		case "/usage":
			q := r.URL.Query()
			if q.Get("from") != "2026-01-01T00:00:00Z" || q.Get("to") != "2026-02-01T00:00:00Z" || q.Get("vm_id") != "42" || q.Get("group_by") != "vm" {
				t.Errorf("usage query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"vm_id":42,"hostname":"web","kind":"vm","period_start":"2026-01-01T00:00:00Z","hours":3,"rate":"0.5","rate_unit":"hour","amount":"1.5","currency":"CAD"}],"meta":{"next_cursor":null,"has_more":false,"total":"1.5","currency":"CAD"}}`))
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	api := testAPIClient(server)
	account := NewAccountAreaAccountDataSource().(*AccountAreaAccountDataSource)
	account.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var sch datasource.SchemaResponse
	account.Schema(ctx, datasource.SchemaRequest{}, &sch)
	var meta datasource.MetadataResponse
	account.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &meta)
	if meta.TypeName != "ffxf_account" {
		t.Fatal(meta.TypeName)
	}
	read := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
	account.Read(ctx, datasource.ReadRequest{}, &read)
	var model AccountAreaAccountModel
	read.Diagnostics.Append(read.State.Get(ctx, &model)...)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	credit := model.Account.Attributes()["credit"].(types.Object)
	if credit.Attributes()["balance"].(types.String).ValueString() != "120.00" {
		t.Fatalf("credit = %#v", credit)
	}
	usage := NewAccountAreaUsageDataSource().(*AccountAreaUsageDataSource)
	usage.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var usageSchema datasource.SchemaResponse
	usage.Schema(ctx, datasource.SchemaRequest{}, &usageSchema)
	var usageMeta datasource.MetadataResponse
	usage.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &usageMeta)
	if usageMeta.TypeName != "ffxf_usage" {
		t.Fatal(usageMeta.TypeName)
	}
	config := accountAreaTestConfig(t, usageSchema.Schema, map[string]any{"from": "2026-01-01T00:00:00Z", "to": "2026-02-01T00:00:00Z", "vm_id": int64(42), "group_by": "vm"})
	usageRead := datasource.ReadResponse{State: tfsdk.State{Schema: usageSchema.Schema}}
	usage.Read(ctx, datasource.ReadRequest{Config: config}, &usageRead)
	var usageState AccountAreaUsageModel
	usageRead.Diagnostics.Append(usageRead.State.Get(ctx, &usageState)...)
	if usageRead.Diagnostics.HasError() || usageState.Total.ValueString() != "1.5" || len(usageState.Records.Elements()) != 1 {
		t.Fatalf("usage = %#v; %v", usageState, usageRead.Diagnostics)
	}
}

func TestAccountAreaUsageRejectsInvalidGroupAndSurfacesAPIError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		group  string
	}{{"group", http.StatusOK, "day"}, {"api", http.StatusBadGateway, "hour"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status) }))
			defer server.Close()
			ds := NewAccountAreaUsageDataSource().(*AccountAreaUsageDataSource)
			ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
			var sch datasource.SchemaResponse
			ds.Schema(context.Background(), datasource.SchemaRequest{}, &sch)
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
			ds.Read(context.Background(), datasource.ReadRequest{Config: accountAreaTestConfig(t, sch.Schema, map[string]any{"group_by": tc.group})}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected diagnostic")
			}
		})
	}
}

func TestAccountAreaBudgetResourceLifecycle(t *testing.T) {
	monthly := "75.00"
	putCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "PUT /account/budget":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			putCount++
			if putCount == 1 && body["monthly"] != monthly {
				t.Errorf("budget body = %#v", body)
			}
			if putCount == 2 && body["monthly"] != "90.00" {
				t.Errorf("budget update body = %#v", body)
			}
			if putCount > 2 && body["monthly"] != nil {
				t.Errorf("budget removal body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"monthly":"75.00","month_to_date":"12.00","percent":16,"cap_percent":100}}`))
		case "GET /account":
			_, _ = w.Write([]byte(`{"data":{"budget":{"monthly":"75.00","month_to_date":"12.00","percent":16,"cap_percent":100}}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	res := NewAccountAreaBudgetResource().(*AccountAreaBudgetResource)
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	var sch resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &sch)
	var metadata resource.MetadataResponse
	res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_account_budget" {
		t.Fatal(metadata.TypeName)
	}
	plan := tfsdk.Plan{Schema: sch.Schema}
	planned := AccountAreaBudgetModel{ID: types.StringUnknown(), Monthly: types.StringValue(monthly), MonthToDate: types.StringUnknown(), Percent: types.Float64Unknown(), CapPercent: types.Int64Unknown()}
	if d := plan.Set(ctx, &planned); d.HasError() {
		t.Fatal(d)
	}
	var decodedPlan AccountAreaBudgetModel
	if d := plan.Get(ctx, &decodedPlan); d.HasError() || decodedPlan.Monthly.ValueString() != monthly {
		t.Fatalf("plan monthly = %#v; %v", decodedPlan.Monthly, d)
	}
	created := resource.CreateResponse{State: tfsdk.State{Schema: sch.Schema}}
	res.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	var state AccountAreaBudgetModel
	created.Diagnostics.Append(created.State.Get(ctx, &state)...)
	if created.Diagnostics.HasError() || state.ID.ValueString() != "account-budget" || state.Monthly.ValueString() != monthly {
		t.Fatalf("created budget = %#v; %v", state, created.Diagnostics)
	}
	read := resource.ReadResponse{State: created.State}
	res.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	updatedPlan := tfsdk.Plan{Schema: sch.Schema}
	updateModel := AccountAreaBudgetModel{ID: types.StringUnknown(), Monthly: types.StringValue("90.00"), MonthToDate: types.StringUnknown(), Percent: types.Float64Unknown(), CapPercent: types.Int64Unknown()}
	if d := updatedPlan.Set(ctx, &updateModel); d.HasError() {
		t.Fatal(d)
	}
	updated := resource.UpdateResponse{State: read.State}
	res.Update(ctx, resource.UpdateRequest{Plan: updatedPlan, State: read.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{State: updated.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatal(deleteResponse.Diagnostics)
	}
}

func TestAccountAreaBudgetAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	res := NewAccountAreaBudgetResource().(*AccountAreaBudgetResource)
	res.Configure(context.Background(), resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	var sch resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &sch)
	plan := tfsdk.Plan{Schema: sch.Schema}
	model := AccountAreaBudgetModel{Monthly: types.StringValue("1")}
	if d := plan.Set(context.Background(), &model); d.HasError() {
		t.Fatal(d)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: sch.Schema}}
	res.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("Create should report API error")
	}
}

func TestAccountAreaAccountReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer server.Close()
	ds := NewAccountAreaAccountDataSource().(*AccountAreaAccountDataSource)
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
	var sch datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &sch)
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
	ds.Read(context.Background(), datasource.ReadRequest{}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("account read should report an API error")
	}
}

func TestAccountAreaBudgetReadRemovesUnsetAndReportsErrors(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		removed bool
	}{{"unset", http.StatusOK, `{"data":{"budget":{"monthly":null}}}`, true}, {"api", http.StatusBadGateway, "", false}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			res := NewAccountAreaBudgetResource().(*AccountAreaBudgetResource)
			res.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
			var sch resource.SchemaResponse
			res.Schema(ctx, resource.SchemaRequest{}, &sch)
			state := tfsdk.State{Schema: sch.Schema}
			model := AccountAreaBudgetModel{ID: types.StringValue("account-budget"), Monthly: types.StringValue("10"), MonthToDate: types.StringValue("1"), Percent: types.Float64Value(10), CapPercent: types.Int64Value(100)}
			if d := state.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			response := resource.ReadResponse{State: state}
			res.Read(ctx, resource.ReadRequest{State: state}, &response)
			if tc.removed && !response.State.Raw.IsNull() {
				t.Fatal("unset budget should be removed")
			}
			if !tc.removed && !response.Diagnostics.HasError() {
				t.Fatal("API error should be diagnosed")
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	res := NewAccountAreaBudgetResource().(*AccountAreaBudgetResource)
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	deleteResp := resource.DeleteResponse{}
	res.Delete(ctx, resource.DeleteRequest{}, &deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("budget delete should report API failure")
	}
}

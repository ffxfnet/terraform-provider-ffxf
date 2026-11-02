package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDatabaseResourceCreatePollsAndMapsState(t *testing.T) {
	oldInterval, oldTimeout := databasePollInterval, databasePollTimeout
	databasePollInterval, databasePollTimeout = time.Millisecond, time.Second
	defer func() { databasePollInterval, databasePollTimeout = oldInterval, oldTimeout }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing authorization header")
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /databases":
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("missing create idempotency key")
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			if body["name"] != "orders" || body["vpc"] != float64(8) {
				t.Errorf("create body = %#v", body)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":12,"name":"orders","status":"pending","engine":"postgresql","version":"18","vpc":8,"size_gb":2}}`))
		case "GET /databases/12":
			_, _ = w.Write([]byte(`{"data":{"id":12,"name":"orders","status":"active","engine":"postgresql","version":"18","engine_version":"18.1","region":"montreal","vpc":8,"size_gb":2,"maintenance":{"day":0,"hour":3},"billing":{"mode":"hourly"},"connection":{"host":"db.example","port":5432},"public_access":{"enabled":false,"sources":[]},"disk":{"cap_gb":10},"users":[],"databases":[],"created_at":"2026-01-01T00:00:00Z"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	resourceImpl := &databaseResource{client: testAPIClient(server)}
	plan := newDatabasePlan(t, databaseResourceModel{Name: types.StringValue("orders"), VPC: types.Int64Value(8), Engine: types.StringNull(), Version: types.StringNull(), SizeGB: types.Int64Null(), Billing: types.StringNull(), MaintenanceDay: types.Int64Null(), MaintenanceHour: types.Int64Null(), DiskCapGB: types.Int64Null(), EvictionPolicy: types.StringNull()})
	response := resource.CreateResponse{State: newDatabaseState(t, databaseResourceModel{})}
	resourceImpl.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create diagnostics: %v", response.Diagnostics)
	}
	var got databaseResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("read created state: %v", response.Diagnostics)
	}
	if got.ID.ValueInt64() != 12 || got.Status.ValueString() != "active" || got.MaintenanceDay.ValueInt64() != 0 || got.Billing.ValueString() != "hourly" {
		t.Fatalf("created model = %#v", got)
	}
}

func TestDatabaseResourceReadAndDeleteNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer server.Close()
	r := &databaseResource{client: testAPIClient(server)}
	state := newDatabaseState(t, databaseResourceModel{ID: types.Int64Value(42), Name: types.StringValue("gone"), VPC: types.Int64Value(8)})
	read := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("Read should remove missing database: %v", read.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete should tolerate 404: %v", deleteResponse.Diagnostics)
	}
}

func TestDatabasePollStopsOnFailureAndTimeout(t *testing.T) {
	oldInterval, oldTimeout := databasePollInterval, databasePollTimeout
	databasePollInterval, databasePollTimeout = time.Millisecond, 8*time.Millisecond
	defer func() { databasePollInterval, databasePollTimeout = oldInterval, oldTimeout }()
	for _, tc := range []struct{ name, status string }{{"failure", "error"}, {"timeout", "pending"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"data":{"id":7,"status":"` + tc.status + `"}}`))
			}))
			defer server.Close()
			_, err := databasePoll(context.Background(), testAPIClient(server), 7)
			if err == nil {
				t.Fatal("databasePoll() should fail for non-ready state")
			}
		})
	}
}

func TestDatabaseResourceSchemaAndConfigure(t *testing.T) {
	r := &databaseResource{}
	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_database" {
		t.Fatalf("type name = %q", metadata.TypeName)
	}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	for name, attr := range schemaResponse.Schema.Attributes {
		if attr.GetDescription() == "" {
			t.Errorf("attribute %q has no description", name)
		}
	}
	apiClient := client.NewClient("https://example.test/v1", "token")
	var configure resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: apiClient}, &configure)
	if configure.Diagnostics.HasError() || r.client != apiClient {
		t.Fatalf("Configure: %v", configure.Diagnostics)
	}
}

func newDatabaseState(t *testing.T, model databaseResourceModel) tfsdk.State {
	t.Helper()
	var response resource.SchemaResponse
	(&databaseResource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	state := tfsdk.State{Schema: response.Schema}
	if diagnostics := state.Set(context.Background(), &model); diagnostics.HasError() {
		t.Fatalf("initialize database state: %v", diagnostics)
	}
	return state
}
func newDatabasePlan(t *testing.T, model databaseResourceModel) tfsdk.Plan {
	t.Helper()
	var response resource.SchemaResponse
	(&databaseResource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	plan := tfsdk.Plan{Schema: response.Schema}
	if diagnostics := plan.Set(context.Background(), &model); diagnostics.HasError() {
		t.Fatalf("initialize database plan: %v", diagnostics)
	}
	return plan
}

func TestDatabaseAttributeMapConversion(t *testing.T) {
	var response resource.SchemaResponse
	(&databaseChildResource{kind: "logical_database"}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	objectType := response.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9))
	values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "appdb")
	plan := tfsdk.Plan{Schema: response.Schema, Raw: tftypes.NewValue(objectType, values)}
	var decoded types.Object
	if diags := plan.Get(context.Background(), &decoded); diags.HasError() {
		t.Fatalf("decode plan object: %v", diags)
	}
	if got := decoded.Attributes()["name"].(types.String).ValueString(); got != "appdb" {
		t.Fatalf("name = %q", got)
	}
}

func TestDatabaseChildResourcesCreateReadAndDelete(t *testing.T) {
	tests := []struct {
		kind                                               string
		createMethod, createPath, deleteMethod, deletePath string
		createStatus, deleteStatus                         int
		setAttrs                                           func(map[string]tftypes.Value, tftypes.Object)
		body                                               string
	}{
		{kind: "user", createMethod: http.MethodPost, createPath: "/databases/9/users", deleteMethod: http.MethodDelete, deletePath: "/databases/9/users/app", createStatus: http.StatusCreated, deleteStatus: http.StatusNoContent, body: `{"data":{"username":"app","password":"secret"}}`},
		{kind: "logical_database", createMethod: http.MethodPost, createPath: "/databases/9/databases", deleteMethod: http.MethodDelete, deletePath: "/databases/9/databases/analytics", createStatus: http.StatusCreated, deleteStatus: http.StatusNoContent},
		{kind: "extension", createMethod: http.MethodPost, createPath: "/databases/9/databases/defaultdb/extensions", deleteMethod: http.MethodDelete, deletePath: "/databases/9/databases/defaultdb/extensions/vector", createStatus: http.StatusOK, deleteStatus: http.StatusOK},
		{kind: "public_access", createMethod: http.MethodPut, createPath: "/databases/9/public-access", deleteMethod: http.MethodPut, deletePath: "/databases/9/public-access", createStatus: http.StatusOK, deleteStatus: http.StatusOK},
		{kind: "backup", createMethod: http.MethodPost, createPath: "/databases/9/backups", createStatus: http.StatusAccepted},
	}
	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing authorization")
				}
				calls++
				if calls == 1 {
					if r.Method != tc.createMethod || r.URL.Path != tc.createPath {
						t.Errorf("create = %s %s", r.Method, r.URL.Path)
					}
					w.WriteHeader(tc.createStatus)
					if tc.body != "" {
						_, _ = w.Write([]byte(tc.body))
					}
					return
				}
				if calls == 2 {
					if r.Method != http.MethodGet || r.URL.Path != "/databases/9" {
						t.Errorf("read = %s %s", r.Method, r.URL.Path)
					}
					_, _ = w.Write([]byte(`{"data":{"id":9,"status":"active","users":[{"username":"app","admin":false,"databases":["analytics"],"read_only":true}],"databases":[{"name":"analytics","extensions":[]},{"name":"defaultdb","extensions":["vector"]}],"public_access":{"enabled":true,"sources":["192.0.2.1"]}}}`))
					return
				}
				if tc.kind == "backup" {
					t.Errorf("backup should not make read/delete API calls, got %s %s", r.Method, r.URL.Path)
					return
				}
				if r.Method != tc.deleteMethod || r.URL.Path != tc.deletePath {
					t.Errorf("delete = %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.deleteStatus)
			}))
			defer server.Close()
			r := &databaseChildResource{client: testAPIClient(server), kind: tc.kind}
			var schemaResponse resource.SchemaResponse
			r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
			var metadata resource.MetadataResponse
			r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
			if metadata.TypeName != "ffxf_database_"+tc.kind {
				t.Fatalf("metadata type = %q", metadata.TypeName)
			}
			for name, attribute := range schemaResponse.Schema.Attributes {
				if attribute.GetDescription() == "" {
					t.Errorf("attribute %q has no description", name)
				}
			}
			objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, typ := range objectType.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9))
			switch tc.kind {
			case "user":
				values["username"] = tftypes.NewValue(objectType.AttributeTypes["username"], "app")
				values["read_only"] = tftypes.NewValue(objectType.AttributeTypes["read_only"], true)
				values["password_rotation_trigger"] = tftypes.NewValue(objectType.AttributeTypes["password_rotation_trigger"], "first")
			case "logical_database":
				values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "analytics")
			case "extension":
				values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "defaultdb")
				values["extension"] = tftypes.NewValue(objectType.AttributeTypes["extension"], "vector")
			case "public_access":
				values["enabled"] = tftypes.NewValue(objectType.AttributeTypes["enabled"], true)
			}
			plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, values)}
			createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResponse)
			if createResponse.Diagnostics.HasError() {
				t.Fatalf("Create: %v", createResponse.Diagnostics)
			}
			var created types.Object
			createResponse.Diagnostics.Append(createResponse.State.Get(context.Background(), &created)...)
			if createResponse.Diagnostics.HasError() {
				t.Fatalf("read created state: %v", createResponse.Diagnostics)
			}
			if tc.kind == "user" && created.Attributes()["password"].(types.String).ValueString() != "secret" {
				t.Fatal("user password was not populated")
			}
			readResponse := resource.ReadResponse{State: createResponse.State}
			r.Read(context.Background(), resource.ReadRequest{State: createResponse.State}, &readResponse)
			if readResponse.Diagnostics.HasError() {
				t.Fatalf("Read: %v", readResponse.Diagnostics)
			}
			deleteResponse := resource.DeleteResponse{}
			r.Delete(context.Background(), resource.DeleteRequest{State: readResponse.State}, &deleteResponse)
			if deleteResponse.Diagnostics.HasError() {
				t.Fatalf("Delete: %v", deleteResponse.Diagnostics)
			}
		})
	}
}

func TestDatabaseUserPasswordRotation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/databases/9/users/app/reset-password" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"password":"rotated"}}`))
	}))
	defer server.Close()
	r := &databaseChildResource{client: testAPIClient(server), kind: "user"}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	stateValues := map[string]tftypes.Value{}
	planValues := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		stateValues[name] = tftypes.NewValue(typ, nil)
		planValues[name] = tftypes.NewValue(typ, nil)
	}
	for _, values := range []map[string]tftypes.Value{stateValues, planValues} {
		values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9))
		values["username"] = tftypes.NewValue(objectType.AttributeTypes["username"], "app")
	}
	stateValues["id"] = tftypes.NewValue(objectType.AttributeTypes["id"], "9/app")
	stateValues["password_rotation_trigger"] = tftypes.NewValue(objectType.AttributeTypes["password_rotation_trigger"], "old")
	stateValues["password"] = tftypes.NewValue(objectType.AttributeTypes["password"], "original")
	planValues["password_rotation_trigger"] = tftypes.NewValue(objectType.AttributeTypes["password_rotation_trigger"], "new")
	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, planValues)}
	state := tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, stateValues)}
	response := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Update: %v", response.Diagnostics)
	}
	var got types.Object
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if got.Attributes()["password"].(types.String).ValueString() != "rotated" {
		t.Fatal("password rotation result not stored")
	}
}

func TestDatabaseForkRestoreAndDetachResources(t *testing.T) {
	for _, kind := range []string{"fork", "restore"} {
		t.Run(kind, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := "/databases/9/forks"
				if kind == "restore" {
					path = "/databases/9/restore"
				}
				if r.Method != http.MethodPost || r.URL.Path != path {
					t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, path)
				}
				if r.Header.Get("Idempotency-Key") == "" {
					t.Error("missing Idempotency-Key")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				if body["name"] != "clone" {
					t.Errorf("body = %#v", body)
				}
				if kind == "restore" && body["at"] != "2026-10-14T09:30:00Z" {
					t.Errorf("restore at = %#v", body["at"])
				}
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"data":{"id":55,"name":"clone","status":"active","engine":"postgresql","version":"18","vpc":9,"size_gb":2}}`))
			}))
			defer server.Close()
			r := &databaseDerivedResource{client: testAPIClient(server), kind: kind}
			var schemaResponse resource.SchemaResponse
			r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
			objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, typ := range objectType.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			values["source_database_id"] = tftypes.NewValue(objectType.AttributeTypes["source_database_id"], int64(9))
			values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "clone")
			if kind == "restore" {
				values["at"] = tftypes.NewValue(objectType.AttributeTypes["at"], "2026-10-14T09:30:00Z")
			}
			plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, values)}
			response := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("Create: %v", response.Diagnostics)
			}
			var state types.Object
			response.Diagnostics.Append(response.State.Get(context.Background(), &state)...)
			if state.Attributes()["id"].(types.Int64).ValueInt64() != 55 {
				t.Fatalf("created id = %v", state.Attributes()["id"])
			}
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/databases/9/detach" {
			t.Errorf("detach request = %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	detach := &databaseDetachResource{client: testAPIClient(server)}
	var detachSchema resource.SchemaResponse
	detach.Schema(context.Background(), resource.SchemaRequest{}, &detachSchema)
	objectType := detachSchema.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{"id": tftypes.NewValue(objectType.AttributeTypes["id"], nil), "database_id": tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9))}
	plan := tfsdk.Plan{Schema: detachSchema.Schema, Raw: tftypes.NewValue(objectType, values)}
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: detachSchema.Schema}}
	detach.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatalf("detach Create: %v", createResponse.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	detach.Delete(context.Background(), resource.DeleteRequest{}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("detach Delete: %v", deleteResponse.Diagnostics)
	}
}

func TestDatabaseConfigValidatorRejectsUnsupportedValues(t *testing.T) {
	var resourceSchema resource.SchemaResponse
	(&databaseResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resourceSchema)
	objectType := resourceSchema.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "orders")
	values["vpc"] = tftypes.NewValue(objectType.AttributeTypes["vpc"], int64(8))
	values["engine"] = tftypes.NewValue(objectType.AttributeTypes["engine"], "oracle")
	config := tfsdk.Config{Schema: resourceSchema.Schema, Raw: tftypes.NewValue(objectType, values)}
	response := resource.ValidateConfigResponse{}
	databaseConfigValidator{kind: "database"}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: config}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("unsupported database engine should fail validation")
	}
}

func TestDatabaseResourceUpdatePatchesAndResizes(t *testing.T) {
	oldInterval := databasePollInterval
	databasePollInterval = time.Millisecond
	defer func() { databasePollInterval = oldInterval }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "PATCH /databases/12":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("patch body: %v", err)
			}
			if body["name"] != "renamed" || body["maintenance_day"] != float64(2) {
				t.Errorf("patch body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"data":{"id":12,"status":"active"}}`))
		case "POST /databases/12/resize":
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("resize missing idempotency key")
			}
			var body map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["size_gb"] != 4 {
				t.Errorf("resize body = %#v, error %v", body, err)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":12,"status":"maintenance"}}`))
		case "GET /databases/12":
			_, _ = w.Write([]byte(`{"data":{"id":12,"name":"renamed","status":"active","engine":"postgresql","version":"18","vpc":8,"size_gb":4,"created_at":"2026-01-01T00:00:00Z"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &databaseResource{client: testAPIClient(server)}
	stateModel := databaseResourceModel{ID: types.Int64Value(12), Name: types.StringValue("orders"), VPC: types.Int64Value(8), Engine: types.StringValue("postgresql"), Version: types.StringValue("18"), SizeGB: types.Int64Value(2), Billing: types.StringValue("hourly"), MaintenanceDay: types.Int64Value(1), MaintenanceHour: types.Int64Value(3), DiskCapGB: types.Int64Value(10), EvictionPolicy: types.StringNull()}
	planModel := stateModel
	planModel.Name = types.StringValue("renamed")
	planModel.SizeGB = types.Int64Value(4)
	planModel.MaintenanceDay = types.Int64Value(2)
	state, plan := newDatabaseState(t, stateModel), newDatabasePlan(t, planModel)
	response := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Update: %v", response.Diagnostics)
	}
	var got databaseResourceModel
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if got.Name.ValueString() != "renamed" || got.SizeGB.ValueInt64() != 4 {
		t.Fatalf("updated state = %#v", got)
	}
}

func TestDatabaseDerivedReadDeleteAndDetachRead(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.Method + " " + r.URL.Path {
		case "GET /databases/55":
			_, _ = w.Write([]byte(`{"data":{"id":55,"status":"active","engine":"postgresql","version":"18"}}`))
		case "DELETE /databases/55":
			w.WriteHeader(http.StatusAccepted)
		case "GET /databases/43":
			_, _ = w.Write([]byte(`{"data":{"id":43,"status":"active"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &databaseDerivedResource{client: testAPIClient(server), kind: "fork"}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["id"] = tftypes.NewValue(objectType.AttributeTypes["id"], int64(55))
	values["source_database_id"] = tftypes.NewValue(objectType.AttributeTypes["source_database_id"], int64(9))
	values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "clone")
	state := tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, values)}
	read := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &read)
	if read.Diagnostics.HasError() {
		t.Fatalf("derived Read: %v", read.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: read.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("derived Delete: %v", deleteResponse.Diagnostics)
	}

	detach := &databaseDetachResource{client: testAPIClient(server)}
	var detachSchema resource.SchemaResponse
	detach.Schema(context.Background(), resource.SchemaRequest{}, &detachSchema)
	detachType := detachSchema.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	detachRaw := tftypes.NewValue(detachType, map[string]tftypes.Value{"id": tftypes.NewValue(detachType.AttributeTypes["id"], "43"), "database_id": tftypes.NewValue(detachType.AttributeTypes["database_id"], int64(43))})
	detachState := tfsdk.State{Schema: detachSchema.Schema, Raw: detachRaw}
	detachRead := resource.ReadResponse{State: detachState}
	detach.Read(context.Background(), resource.ReadRequest{State: detachState}, &detachRead)
	if detachRead.Diagnostics.HasError() {
		t.Fatalf("detach Read: %v", detachRead.Diagnostics)
	}
	detach.Delete(context.Background(), resource.DeleteRequest{}, &resource.DeleteResponse{})
	if calls != 3 {
		t.Fatalf("API calls = %d, want 3", calls)
	}
}

func TestDatabaseUserUpdateChangesAccessAndRotates(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.Method + " " + r.URL.Path {
		case "PATCH /databases/9/users/app":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["read_only"] != false || body["databases"] == nil {
				t.Errorf("patch body = %#v, error %v", body, err)
			}
			w.WriteHeader(http.StatusOK)
		case "POST /databases/9/users/app/reset-password":
			_, _ = w.Write([]byte(`{"data":{"password":"new-password"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &databaseChildResource{client: testAPIClient(server), kind: "user"}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	stateValues, planValues := map[string]tftypes.Value{}, map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		stateValues[name] = tftypes.NewValue(typ, nil)
		planValues[name] = tftypes.NewValue(typ, nil)
	}
	for _, values := range []map[string]tftypes.Value{stateValues, planValues} {
		values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9))
		values["username"] = tftypes.NewValue(objectType.AttributeTypes["username"], "app")
	}
	stateValues["id"] = tftypes.NewValue(objectType.AttributeTypes["id"], "9/app")
	stateValues["read_only"] = tftypes.NewValue(objectType.AttributeTypes["read_only"], true)
	planValues["read_only"] = tftypes.NewValue(objectType.AttributeTypes["read_only"], false)
	stateValues["databases"] = tftypes.NewValue(objectType.AttributeTypes["databases"], []tftypes.Value{tftypes.NewValue(tftypes.String, "analytics")})
	planValues["databases"] = tftypes.NewValue(objectType.AttributeTypes["databases"], []tftypes.Value{tftypes.NewValue(tftypes.String, "otherdb")})
	stateValues["password_rotation_trigger"] = tftypes.NewValue(objectType.AttributeTypes["password_rotation_trigger"], "old")
	planValues["password_rotation_trigger"] = tftypes.NewValue(objectType.AttributeTypes["password_rotation_trigger"], "new")
	stateValues["password"] = tftypes.NewValue(objectType.AttributeTypes["password"], "old-password")
	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, planValues)}
	state := tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, stateValues)}
	response := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("user Update: %v", response.Diagnostics)
	}
	var got types.Object
	response.Diagnostics.Append(response.State.Get(context.Background(), &got)...)
	if got.Attributes()["password"].(types.String).ValueString() != "new-password" || calls != 2 {
		t.Fatalf("updated user state/calls = %#v/%d", got.Attributes(), calls)
	}
}

func TestDatabaseAreaRegistrationsExposeTypesAndSchemas(t *testing.T) {
	resources := databaseAreaResources()
	wantResources := map[string]bool{"ffxf_database": true, "ffxf_database_user": true, "ffxf_database_logical_database": true, "ffxf_database_extension": true, "ffxf_database_public_access": true, "ffxf_database_backup": true, "ffxf_database_fork": true, "ffxf_database_restore": true, "ffxf_database_detach": true}
	apiClient := client.NewClient("https://example.test/v1", "token")
	for _, constructor := range resources {
		r := constructor()
		var metadata resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		if !wantResources[metadata.TypeName] {
			t.Errorf("unexpected resource %q", metadata.TypeName)
		}
		delete(wantResources, metadata.TypeName)
		var schemaResponse resource.SchemaResponse
		r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
		if schemaResponse.Schema.Description == "" {
			t.Errorf("%s has no resource description", metadata.TypeName)
		}
		for name, attribute := range schemaResponse.Schema.Attributes {
			if attribute.GetDescription() == "" {
				t.Errorf("%s.%s has no description", metadata.TypeName, name)
			}
		}
		if configurable, ok := r.(resource.ResourceWithConfigure); ok {
			var configure resource.ConfigureResponse
			configurable.Configure(context.Background(), resource.ConfigureRequest{ProviderData: apiClient}, &configure)
			if configure.Diagnostics.HasError() {
				t.Errorf("%s Configure: %v", metadata.TypeName, configure.Diagnostics)
			}
		}
		if configurable, ok := r.(resource.ResourceWithConfigValidators); ok {
			_ = configurable.ConfigValidators(context.Background())
		}
	}
	for name := range wantResources {
		t.Errorf("missing resource %q", name)
	}

	dataSources := databaseAreaDataSources()
	wantDataSources := map[string]bool{"ffxf_databases": true, "ffxf_database_options": true, "ffxf_database_credentials": true, "ffxf_database_backups": true, "ffxf_database_metrics": true, "ffxf_database_queries": true}
	for _, constructor := range dataSources {
		ds := constructor()
		var metadata datasource.MetadataResponse
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		if !wantDataSources[metadata.TypeName] {
			t.Errorf("unexpected data source %q", metadata.TypeName)
		}
		delete(wantDataSources, metadata.TypeName)
		var schemaResponse datasource.SchemaResponse
		ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
		for name, attribute := range schemaResponse.Schema.Attributes {
			if attribute.GetDescription() == "" {
				t.Errorf("%s.%s has no description", metadata.TypeName, name)
			}
		}
		var configure datasource.ConfigureResponse
		ds.(datasource.DataSourceWithConfigure).Configure(context.Background(), datasource.ConfigureRequest{ProviderData: apiClient}, &configure)
		if configure.Diagnostics.HasError() {
			t.Errorf("%s Configure: %v", metadata.TypeName, configure.Diagnostics)
		}
		if configurable, ok := ds.(datasource.DataSourceWithConfigValidators); ok {
			_ = configurable.ConfigValidators(context.Background())
		}
	}
	for name := range wantDataSources {
		t.Errorf("missing data source %q", name)
	}
}

func TestDatabaseResourceAPIErrorDiagnostics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	r := &databaseResource{client: testAPIClient(server)}
	plan := newDatabasePlan(t, databaseResourceModel{Name: types.StringValue("orders"), VPC: types.Int64Value(8)})
	create := resource.CreateResponse{State: newDatabaseState(t, databaseResourceModel{})}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &create)
	if !create.Diagnostics.HasError() {
		t.Fatal("create API failure should produce diagnostics")
	}
	state := newDatabaseState(t, databaseResourceModel{ID: types.Int64Value(8), Name: types.StringValue("orders"), VPC: types.Int64Value(4)})
	read := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &read)
	if !read.Diagnostics.HasError() {
		t.Fatal("read API failure should produce diagnostics")
	}
	deleted := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleted)
	if !deleted.Diagnostics.HasError() {
		t.Fatal("delete API failure should produce diagnostics")
	}
}

func TestDatabaseResponseMappingReportsMalformedNestedJSON(t *testing.T) {
	model := databaseResourceModel{}
	diagnostics := diag.Diagnostics{}
	databaseSetResourceModel(context.Background(), &model, client.Database{
		ID: 1, Name: "orders", Status: "active", Maintenance: json.RawMessage(`invalid`),
		Billing: json.RawMessage(`invalid`), Disk: json.RawMessage(`invalid`),
		Connection: json.RawMessage(`invalid`), PublicAccess: json.RawMessage(`invalid`),
		DatabaseJSON: json.RawMessage(`invalid`),
	}, &diagnostics)
	if !diagnostics.HasError() {
		t.Fatal("malformed nested JSON should produce diagnostics")
	}
	if !databaseJSONValue(json.RawMessage(`invalid`), &diagnostics).IsNull() {
		t.Fatal("invalid JSON should not be exposed as a string")
	}
}

func TestDatabaseValidatorsRejectBadRestoreAndNumericValues(t *testing.T) {
	var response resource.SchemaResponse
	r := &databaseResource{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &response)
	objectType := response.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["name"] = tftypes.NewValue(objectType.AttributeTypes["name"], "orders")
	values["vpc"] = tftypes.NewValue(objectType.AttributeTypes["vpc"], int64(8))
	values["size_gb"] = tftypes.NewValue(objectType.AttributeTypes["size_gb"], int64(0))
	values["maintenance_day"] = tftypes.NewValue(objectType.AttributeTypes["maintenance_day"], int64(7))
	values["maintenance_hour"] = tftypes.NewValue(objectType.AttributeTypes["maintenance_hour"], int64(24))
	values["disk_cap_gb"] = tftypes.NewValue(objectType.AttributeTypes["disk_cap_gb"], int64(0))
	config := tfsdk.Config{Schema: response.Schema, Raw: tftypes.NewValue(objectType, values)}
	validation := resource.ValidateConfigResponse{}
	validator := databaseConfigValidator{kind: "database"}
	validator.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: config}, &validation)
	if !validation.Diagnostics.HasError() {
		t.Fatal("invalid numeric database settings should fail validation")
	}
	if validator.Description(context.Background()) == "" || validator.MarkdownDescription(context.Background()) == "" {
		t.Fatal("validator descriptions should not be empty")
	}

	var restoreSchema resource.SchemaResponse
	restore := &databaseDerivedResource{kind: "restore"}
	restore.Schema(context.Background(), resource.SchemaRequest{}, &restoreSchema)
	restoreType := restoreSchema.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	restoreValues := map[string]tftypes.Value{}
	for name, typ := range restoreType.AttributeTypes {
		restoreValues[name] = tftypes.NewValue(typ, nil)
	}
	restoreValues["source_database_id"] = tftypes.NewValue(restoreType.AttributeTypes["source_database_id"], int64(1))
	restoreValues["name"] = tftypes.NewValue(restoreType.AttributeTypes["name"], "restore")
	restoreValues["at"] = tftypes.NewValue(restoreType.AttributeTypes["at"], "not-a-date")
	restoreValidation := resource.ValidateConfigResponse{}
	databaseConfigValidator{kind: "restore"}.ValidateResource(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: restoreSchema.Schema, Raw: tftypes.NewValue(restoreType, restoreValues)}}, &restoreValidation)
	if !restoreValidation.Diagnostics.HasError() {
		t.Fatal("invalid restore timestamp should fail validation")
	}
}

func TestDatabaseResourceOptionalCreateReadAndDelete(t *testing.T) {
	oldInterval, oldTimeout := databasePollInterval, databasePollTimeout
	databasePollInterval, databasePollTimeout = time.Millisecond, time.Second
	defer func() { databasePollInterval, databasePollTimeout = oldInterval, oldTimeout }()
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/databases" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			if body["engine"] != "mysql" || body["version"] != "8.4" || body["size_gb"] != float64(4) || body["billing"] != "monthly" {
				t.Errorf("create body = %#v", body)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":33,"name":"orders","status":"active","engine":"mysql","version":"8.4","vpc":8,"size_gb":4,"created_at":"2026-01-01T00:00:00Z"}}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/databases/33" {
			if deleted {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"data":{"id":33,"name":"orders","status":"active","engine":"mysql","version":"8.4","vpc":8,"size_gb":4,"created_at":"2026-01-01T00:00:00Z"}}`))
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/databases/33" {
			deleted = true
			w.WriteHeader(http.StatusAccepted)
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	r := &databaseResource{client: testAPIClient(server)}
	plan := newDatabasePlan(t, databaseResourceModel{Name: types.StringValue("orders"), VPC: types.Int64Value(8), Engine: types.StringValue("mysql"), Version: types.StringValue("8.4"), SizeGB: types.Int64Value(4), Billing: types.StringValue("monthly")})
	create := resource.CreateResponse{State: newDatabaseState(t, databaseResourceModel{})}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &create)
	if create.Diagnostics.HasError() {
		t.Fatalf("Create: %v", create.Diagnostics)
	}
	read := resource.ReadResponse{State: create.State}
	r.Read(context.Background(), resource.ReadRequest{State: create.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatalf("Read: %v", read.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: read.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleteResponse.Diagnostics)
	}
}

func TestDatabasePublicAccessUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/databases/9/public-access" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["enabled"] != false || body["sources"] == nil {
			t.Errorf("public access body = %#v, error %v", body, err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	r := &databaseChildResource{client: testAPIClient(server), kind: "public_access"}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	stateValues := map[string]tftypes.Value{"id": tftypes.NewValue(objectType.AttributeTypes["id"], "9"), "database_id": tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9)), "enabled": tftypes.NewValue(objectType.AttributeTypes["enabled"], true), "sources": tftypes.NewValue(objectType.AttributeTypes["sources"], []tftypes.Value{tftypes.NewValue(tftypes.String, "192.0.2.1")})}
	planValues := map[string]tftypes.Value{"id": tftypes.NewValue(objectType.AttributeTypes["id"], "9"), "database_id": tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9)), "enabled": tftypes.NewValue(objectType.AttributeTypes["enabled"], false), "sources": tftypes.NewValue(objectType.AttributeTypes["sources"], []tftypes.Value{tftypes.NewValue(tftypes.String, "198.51.100.2")})}
	state := tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, stateValues)}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, planValues)}
	response := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("public access Update: %v", response.Diagnostics)
	}
}

func TestDatabaseCredentialPasswordRejectsMissingPassword(t *testing.T) {
	if _, err := databaseCredentialPassword(map[string]json.RawMessage{"data": json.RawMessage(`{}`)}); err == nil {
		t.Fatal("missing password should be an error")
	}
	if _, err := databaseCredentialPassword(map[string]json.RawMessage{"data": json.RawMessage(`not-json`)}); err == nil {
		t.Fatal("malformed credentials should be an error")
	}
}

func TestDatabaseResourceCreateKeepsStateWhenReadinessFails(t *testing.T) {
	oldInterval, oldTimeout := databasePollInterval, databasePollTimeout
	databasePollInterval, databasePollTimeout = time.Millisecond, time.Second
	defer func() { databasePollInterval, databasePollTimeout = oldInterval, oldTimeout }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":12,"name":"orders","status":"pending","engine":"postgresql","version":"18","vpc":8}}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	resourceImpl := &databaseResource{client: testAPIClient(server)}
	plan := newDatabasePlan(t, databaseResourceModel{Name: types.StringValue("orders"), VPC: types.Int64Value(8), Engine: types.StringNull(), Version: types.StringNull(), SizeGB: types.Int64Null(), Billing: types.StringNull(), MaintenanceDay: types.Int64Null(), MaintenanceHour: types.Int64Null(), DiskCapGB: types.Int64Null(), EvictionPolicy: types.StringNull()})
	response := resource.CreateResponse{State: newDatabaseState(t, databaseResourceModel{})}
	resourceImpl.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("Create should report the readiness failure")
	}
	var got databaseResourceModel
	_ = response.State.Get(context.Background(), &got)
	if got.ID.ValueInt64() != 12 {
		t.Fatalf("created database must stay in state so it can be destroyed, got %#v", got)
	}
}

func TestDatabaseDeleteWaitsUntilDatabaseIsGone(t *testing.T) {
	oldInterval, oldTimeout := databasePollInterval, databasePollTimeout
	databasePollInterval, databasePollTimeout = time.Millisecond, time.Second
	defer func() { databasePollInterval, databasePollTimeout = oldInterval, oldTimeout }()
	gets := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			gets++
			if gets < 3 {
				_, _ = w.Write([]byte(`{"data":{"id":12,"status":"deleting"}}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	r := &databaseResource{client: testAPIClient(server)}
	state := newDatabaseState(t, databaseResourceModel{ID: types.Int64Value(12), Name: types.StringValue("orders"), VPC: types.Int64Value(8)})
	response := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
	if response.Diagnostics.HasError() || gets < 3 {
		t.Fatalf("Delete must poll until 404, gets=%d diagnostics=%v", gets, response.Diagnostics)
	}
}

func TestDatabaseUserCreateResolvesUnknownReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"username":"app","password":"secret"}}`))
	}))
	defer server.Close()
	r := &databaseChildResource{client: testAPIClient(server), kind: "user"}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(9))
	values["username"] = tftypes.NewValue(objectType.AttributeTypes["username"], "app")
	values["read_only"] = tftypes.NewValue(objectType.AttributeTypes["read_only"], tftypes.UnknownValue)
	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, values)}
	response := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Create: %v", response.Diagnostics)
	}
	var created types.Object
	response.Diagnostics.Append(response.State.Get(context.Background(), &created)...)
	readOnly := created.Attributes()["read_only"].(types.Bool)
	if response.Diagnostics.HasError() || readOnly.IsUnknown() || readOnly.IsNull() || readOnly.ValueBool() {
		t.Fatalf("read_only must resolve to a known false, got %v (%v)", readOnly, response.Diagnostics)
	}
}

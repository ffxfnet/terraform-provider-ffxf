package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDatabaseDataSourcesMetadataSchemaConfigureAndRead(t *testing.T) {
	tests := []struct {
		kind, name string
		status     int
		body       string
	}{
		{"s", "ffxf_databases", http.StatusOK, `{"data":[{"id":1,"name":"orders"}],"quota":4}`},
		{"_options", "ffxf_database_options", http.StatusOK, `{"data":{"engines":[{"engine":"postgresql","versions":["18"]}]}}`},
		{"_credentials", "ffxf_database_credentials", http.StatusOK, `{"data":{"users":[{"username":"app","password":"secret"}],"connection_uri":"postgres://secret","public_connection_uri":null}}`},
		{"_backups", "ffxf_database_backups", http.StatusOK, `{"data":[{"label":"manual-1"}],"restore_window":{"from":"a","until":"b"}}`},
		{"_metrics", "ffxf_database_metrics", http.StatusOK, `{"data":{"range":"1h","points":[]}}`},
		{"_queries", "ffxf_database_queries", http.StatusOK, `{"data":[{"query":"select 1"}]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing authorization")
				}
				if tc.kind == "s" && r.URL.Path != "/databases" {
					t.Errorf("path %q", r.URL.Path)
				}
				if tc.kind == "_options" && r.URL.Path != "/databases/options" {
					t.Errorf("path %q", r.URL.Path)
				}
				if tc.kind != "s" && tc.kind != "_options" && r.URL.Path != "/databases/8/"+tc.kind[1:] {
					t.Errorf("path = %q", r.URL.Path)
				}
				if tc.kind == "_metrics" && r.URL.Query().Get("range") != "1h" {
					t.Errorf("range query = %q", r.URL.RawQuery)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			ds := &databaseDataSource{kind: tc.kind, client: testAPIClient(server)}
			var metadata datasource.MetadataResponse
			ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
			if metadata.TypeName != tc.name {
				t.Fatalf("metadata = %q", metadata.TypeName)
			}
			var schemaResponse datasource.SchemaResponse
			ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
			for name, attr := range schemaResponse.Schema.Attributes {
				if attr.GetDescription() == "" {
					t.Errorf("attribute %q has no description", name)
				}
			}
			var configure datasource.ConfigureResponse
			ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &configure)
			if configure.Diagnostics.HasError() {
				t.Fatalf("Configure: %v", configure.Diagnostics)
			}
			request := datasource.ReadRequest{}
			if tc.kind != "s" {
				request.Config = newDatabaseDataSourceConfig(t, schemaResponse.Schema)
			}
			response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			ds.Read(context.Background(), request, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("Read: %v", response.Diagnostics)
			}
		})
	}
}

func TestDatabaseDataSourceReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	ds := &databaseDataSource{kind: "_options", client: testAPIClient(server)}
	var schemaResponse datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
	response := datasource.ReadResponse{}
	ds.Read(context.Background(), datasource.ReadRequest{}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("403 should produce a diagnostic")
	}
}

func newDatabaseDataSourceConfig(t *testing.T, dataSchema datasourceschema.Schema) tfsdk.Config {
	t.Helper()
	ctx := context.Background()
	objectType, ok := dataSchema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("data source schema type = %T", dataSchema.Type().TerraformType(ctx))
	}
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	if _, ok := values["database_id"]; ok {
		values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(8))
	}
	if _, ok := values["range"]; ok {
		values["range"] = tftypes.NewValue(objectType.AttributeTypes["range"], "1h")
	}
	return tfsdk.Config{Raw: tftypes.NewValue(objectType, values), Schema: dataSchema}
}

func TestDatabaseDataSourceConfigurationAndEndpointErrors(t *testing.T) {
	unconfigured := &databaseDataSource{kind: "_options"}
	response := datasource.ReadResponse{}
	unconfigured.Read(context.Background(), datasource.ReadRequest{}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("unconfigured data source should report an error")
	}

	wrongClient := &databaseDataSource{}
	var configure datasource.ConfigureResponse
	wrongClient.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "bad"}, &configure)
	if !configure.Diagnostics.HasError() {
		t.Fatal("unexpected provider data type should report an error")
	}

	for _, kind := range []string{"_credentials", "_backups", "_metrics", "_queries"} {
		t.Run(kind, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
			defer server.Close()
			ds := &databaseDataSource{kind: kind, client: testAPIClient(server)}
			var schemaResponse datasource.SchemaResponse
			ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
			request := datasource.ReadRequest{Config: newDatabaseDataSourceConfig(t, schemaResponse.Schema)}
			result := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			ds.Read(context.Background(), request, &result)
			if !result.Diagnostics.HasError() {
				t.Fatal("API error should produce diagnostics")
			}
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"data":"invalid"}`)) }))
	defer server.Close()
	ds := &databaseDataSource{kind: "_credentials", client: testAPIClient(server)}
	var schemaResponse datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
	request := datasource.ReadRequest{Config: newDatabaseDataSourceConfig(t, schemaResponse.Schema)}
	result := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	ds.Read(context.Background(), request, &result)
	if !result.Diagnostics.HasError() {
		t.Fatal("malformed credentials data should be diagnosed")
	}
}

func TestDatabaseDataSourceRawNullAndValidator(t *testing.T) {
	if !databaseRawValue(nil).IsNull() || !databaseRawValue([]byte("null")).IsNull() {
		t.Fatal("empty and null JSON should map to null Terraform strings")
	}
	if got := databaseRawValue([]byte(`{"ok":true}`)).ValueString(); got != `{"ok":true}` {
		t.Fatalf("raw value = %q", got)
	}

	var schemaResponse datasource.SchemaResponse
	(&databaseDataSource{kind: "_metrics"}).Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["database_id"] = tftypes.NewValue(objectType.AttributeTypes["database_id"], int64(4))
	values["range"] = tftypes.NewValue(objectType.AttributeTypes["range"], "30d")
	config := tfsdk.Config{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, values)}
	validation := datasource.ValidateConfigResponse{}
	databaseConfigValidator{kind: "metrics"}.ValidateDataSource(context.Background(), datasource.ValidateConfigRequest{Config: config}, &validation)
	if !validation.Diagnostics.HasError() {
		t.Fatal("invalid metrics range should fail validation")
	}
}

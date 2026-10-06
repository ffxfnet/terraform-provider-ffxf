package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

func TestProviderMetadata(t *testing.T) {
	provider := &FFXFProvider{version: "1.2.3"}
	var response frameworkprovider.MetadataResponse

	provider.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &response)

	if response.TypeName != "ffxf" {
		t.Errorf("Metadata() TypeName = %q, want %q", response.TypeName, "ffxf")
	}
	if response.Version != "1.2.3" {
		t.Errorf("Metadata() Version = %q, want %q", response.Version, "1.2.3")
	}
}

func TestProviderSchema(t *testing.T) {
	provider := &FFXFProvider{}
	var response frameworkprovider.SchemaResponse

	provider.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &response)

	for _, name := range []string{"endpoint", "token"} {
		if _, ok := response.Schema.Attributes[name]; !ok {
			t.Errorf("Schema() is missing the %q attribute", name)
		}
	}
}

func TestInstanceResourceMetadata(t *testing.T) {
	instance := NewInstanceResource()
	var response resource.MetadataResponse

	instance.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "ffxf"}, &response)

	if response.TypeName != "ffxf_instance" {
		t.Errorf("Metadata() TypeName = %q, want %q", response.TypeName, "ffxf_instance")
	}
}

func TestProviderConfigureUsesEnvironmentAndExplicitValues(t *testing.T) {
	t.Setenv("FFXF_TOKEN", "environment-token")
	t.Setenv("FFXF_ENDPOINT", "https://environment.example/v1")

	tests := []struct {
		name     string
		token    string
		endpoint string
		wantURL  string
		wantKey  string
	}{
		{
			name:    "environment defaults",
			wantURL: "https://environment.example/v1",
			wantKey: "environment-token",
		},
		{
			name:     "explicit values",
			token:    "configured-token",
			endpoint: "https://configured.example/v1",
			wantURL:  "https://configured.example/v1",
			wantKey:  "configured-token",
		},
		{
			name:    "default endpoint",
			token:   "configured-token",
			wantURL: "https://api.ffxf.net/v1",
			wantKey: "configured-token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "default endpoint" {
				t.Setenv("FFXF_ENDPOINT", "")
			}
			provider := &FFXFProvider{}
			var response frameworkprovider.ConfigureResponse
			provider.Configure(context.Background(), frameworkprovider.ConfigureRequest{
				Config: newProviderConfig(t, provider, test.endpoint, test.token),
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("Configure() diagnostics = %v", response.Diagnostics)
			}
			apiClient, ok := response.ResourceData.(*client.Client)
			if !ok {
				t.Fatalf("ResourceData = %T, want *client.Client", response.ResourceData)
			}
			if apiClient.BaseURL != test.wantURL || apiClient.Token != test.wantKey {
				t.Errorf("configured client = (%q, %q), want (%q, %q)", apiClient.BaseURL, apiClient.Token, test.wantURL, test.wantKey)
			}
			if response.DataSourceData != apiClient {
				t.Error("Configure() should provide the same client to data sources")
			}
		})
	}
}

func TestProviderConfigureRequiresToken(t *testing.T) {
	t.Setenv("FFXF_TOKEN", "")
	t.Setenv("FFXF_ENDPOINT", "")

	provider := &FFXFProvider{}
	var response frameworkprovider.ConfigureResponse
	provider.Configure(context.Background(), frameworkprovider.ConfigureRequest{
		Config: newProviderConfig(t, provider, "", ""),
	}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("Configure() should report an error when no token is configured")
	}
}

func TestProviderRegistersResourcesAndDataSources(t *testing.T) {
	provider := &FFXFProvider{}
	if resources := provider.Resources(context.Background()); len(resources) != 1 {
		t.Fatalf("Resources() returned %d entries, want 1", len(resources))
	}

	dataSources := provider.DataSources(context.Background())
	if len(dataSources) != 3 {
		t.Fatalf("DataSources() returned %d entries, want 3", len(dataSources))
	}
	wantNames := map[string]bool{"ffxf_region": false, "ffxf_plan": false, "ffxf_image": false}
	for _, newDataSource := range dataSources {
		dataSource := newDataSource()
		var metadata datasource.MetadataResponse
		dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		if _, ok := wantNames[metadata.TypeName]; !ok {
			t.Errorf("data source type name = %q, want one of ffxf_region, ffxf_plan, ffxf_image", metadata.TypeName)
		}
		wantNames[metadata.TypeName] = true

		var configureResponse datasource.ConfigureResponse
		apiClient := client.NewClient("https://example.test/v1", "token")
		configurableDataSource, ok := dataSource.(datasource.DataSourceWithConfigure)
		if !ok {
			t.Fatalf("%s does not implement DataSourceWithConfigure", metadata.TypeName)
		}
		configurableDataSource.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: apiClient}, &configureResponse)
		if configureResponse.Diagnostics.HasError() {
			t.Errorf("%s Configure() diagnostics = %v", metadata.TypeName, configureResponse.Diagnostics)
		}
	}
	for name, found := range wantNames {
		if !found {
			t.Errorf("DataSources() is missing %q", name)
		}
	}
}

func newProviderConfig(t *testing.T, provider *FFXFProvider, endpoint, token string) tfsdk.Config {
	t.Helper()
	ctx := context.Background()
	var schemaResponse frameworkprovider.SchemaResponse
	provider.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResponse)
	objectType, ok := schemaResponse.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("provider schema type = %T, want tftypes.Object", schemaResponse.Schema.Type().TerraformType(ctx))
	}
	values := map[string]tftypes.Value{
		"endpoint": tftypes.NewValue(objectType.AttributeTypes["endpoint"], nil),
		"token":    tftypes.NewValue(objectType.AttributeTypes["token"], nil),
	}
	if endpoint != "" {
		values["endpoint"] = tftypes.NewValue(objectType.AttributeTypes["endpoint"], endpoint)
	}
	if token != "" {
		values["token"] = tftypes.NewValue(objectType.AttributeTypes["token"], token)
	}
	return tfsdk.Config{Raw: tftypes.NewValue(objectType, values), Schema: schemaResponse.Schema}
}

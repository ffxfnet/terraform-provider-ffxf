package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

func TestCatalogDataSourcesReadCatalogValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/regions":
			_, _ = w.Write([]byte(`{"data":[{"slug":"montreal","name":"Montreal","country":"CA","status":"available","ipv4":true,"ipv6":true}]}`))
		case "/plans/nano":
			_, _ = w.Write([]byte(`{"data":{"slug":"nano","name":"Nano","description":"Small plan","vcpu":1,"memory_mb":2048,"disk_gb":20,"traffic_tb":null,"port_mbps":1000,"regions":["montreal"],"status":"available","prices":[{"currency":"CAD","hourly":"0.018","hourly_month_equivalent":"13.14","hourly_stopped":null,"monthly":"8.50","annual":"85.00","setup_fee":"0.00"}]}}`))
		case "/plans/pro":
			_, _ = w.Write([]byte(`{"data":{"slug":"pro","name":"Pro","description":"Large plan","vcpu":8,"memory_mb":32768,"disk_gb":500,"traffic_tb":10,"port_mbps":10000,"regions":["montreal"],"status":"available","prices":[{"currency":"CAD","hourly":"0.25","hourly_month_equivalent":"182.50","hourly_stopped":"0.06","monthly":"125.00","annual":"1250.00","setup_fee":"0.00"}]}}`))
		case "/images/debian-13":
			_, _ = w.Write([]byte(`{"data":{"slug":"debian-13","name":"Debian 13","family":"debian","category":"linux","version":"13","status":"available","regions":["montreal"],"min_disk_gb":null,"min_memory_mb":null,"default_user":"debian","supports_ssh_keys":true}}`))
		case "/images/windows-2022":
			_, _ = w.Write([]byte(`{"data":{"slug":"windows-2022","name":"Windows Server 2022","family":"windows","category":"windows","version":"2022","status":"available","regions":["montreal"],"min_disk_gb":32,"min_memory_mb":4096,"default_user":"Administrator","supports_ssh_keys":false}}`))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient := client.NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()

	var region RegionDataSourceModel
	readCatalogDataSource(t, &RegionDataSource{client: apiClient}, "montreal", &region)
	if region.Name.ValueString() != "Montreal" || region.Country.ValueString() != "CA" || !region.IPv6.ValueBool() {
		t.Errorf("region data = %#v, want Montreal with IPv6", region)
	}

	var plan PlanDataSourceModel
	readCatalogDataSource(t, &PlanDataSource{client: apiClient}, "nano", &plan)
	if plan.VCPU.ValueInt64() != 1 || !plan.TrafficTB.IsNull() || plan.Regions.IsNull() || plan.Prices.IsNull() {
		t.Errorf("plan data = %#v, want capacity, null unmetered traffic, regions, and prices", plan)
	}
	var meteredPlan PlanDataSourceModel
	readCatalogDataSource(t, &PlanDataSource{client: apiClient}, "pro", &meteredPlan)
	if meteredPlan.TrafficTB.ValueFloat64() != 10 || !meteredPlan.Prices.Elements()[0].(types.Object).Attributes()["hourly_stopped"].(types.String).Equal(types.StringValue("0.06")) {
		t.Errorf("metered plan data = %#v, want traffic and stopped-hourly price", meteredPlan)
	}

	var image ImageDataSourceModel
	readCatalogDataSource(t, &ImageDataSource{client: apiClient}, "debian-13", &image)
	if image.Name.ValueString() != "Debian 13" || !image.SupportsSSHKeys.ValueBool() || !image.MinDiskGB.IsNull() || !image.Version.Equal(types.StringValue("13")) {
		t.Errorf("image data = %#v, want Debian metadata and null disk floor", image)
	}
	var constrainedImage ImageDataSourceModel
	readCatalogDataSource(t, &ImageDataSource{client: apiClient}, "windows-2022", &constrainedImage)
	if constrainedImage.MinDiskGB.ValueInt64() != 32 || constrainedImage.MinMemoryMB.ValueInt64() != 4096 || constrainedImage.SupportsSSHKeys.ValueBool() {
		t.Errorf("constrained image data = %#v, want disk/memory floors and no SSH-key support", constrainedImage)
	}
}

func TestCatalogDataSourcesReportReadErrors(t *testing.T) {
	tests := []struct {
		name       string
		source     datasource.DataSource
		slug       string
		statusCode int
		body       string
	}{
		{
			name:   "region slug not found",
			source: &RegionDataSource{},
			slug:   "unknown",
			body:   `{"data":[]}`,
		},
		{
			name:       "region API error",
			source:     &RegionDataSource{},
			slug:       "montreal",
			statusCode: http.StatusServiceUnavailable,
		},
		{
			name:       "plan API error",
			source:     &PlanDataSource{},
			slug:       "nano",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "image API error",
			source:     &ImageDataSource{},
			slug:       "debian-13",
			statusCode: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.statusCode != 0 {
					w.WriteHeader(test.statusCode)
				}
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			apiClient := client.NewClient(server.URL, "test-token")
			apiClient.HTTPClient = server.Client()
			setCatalogDataSourceClient(test.source, apiClient)

			response := runCatalogDataSource(t, test.source, test.slug)
			if !response.Diagnostics.HasError() {
				t.Fatal("Read() should report a diagnostic for missing catalog values or API errors")
			}
		})
	}
}

func readCatalogDataSource(t *testing.T, source datasource.DataSource, slug string, target any) {
	t.Helper()
	response := runCatalogDataSource(t, source, slug)
	if response.Diagnostics.HasError() {
		t.Fatalf("read %T: %v", source, response.Diagnostics)
	}
	response.Diagnostics.Append(response.State.Get(context.Background(), target)...)
	if response.Diagnostics.HasError() {
		t.Fatalf("decode %T state: %v", source, response.Diagnostics)
	}
}

func runCatalogDataSource(t *testing.T, source datasource.DataSource, slug string) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	var schemaResponse datasource.SchemaResponse
	source.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("create datasource schema: %v", schemaResponse.Diagnostics)
	}

	objectType, ok := schemaResponse.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("datasource schema type = %T, want tftypes.Object", schemaResponse.Schema.Type().TerraformType(ctx))
	}
	configValues := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		if name == "slug" {
			configValues[name] = tftypes.NewValue(attributeType, slug)
		} else {
			configValues[name] = tftypes.NewValue(attributeType, nil)
		}
	}
	config := tfsdk.Config{
		Raw:    tftypes.NewValue(objectType, configValues),
		Schema: schemaResponse.Schema,
	}

	state := tfsdk.State{Schema: schemaResponse.Schema}
	response := datasource.ReadResponse{State: state}
	source.Read(ctx, datasource.ReadRequest{Config: config}, &response)
	return response
}

func setCatalogDataSourceClient(source datasource.DataSource, apiClient *client.Client) {
	switch dataSource := source.(type) {
	case *RegionDataSource:
		dataSource.client = apiClient
	case *PlanDataSource:
		dataSource.client = apiClient
	case *ImageDataSource:
		dataSource.client = apiClient
	}
}

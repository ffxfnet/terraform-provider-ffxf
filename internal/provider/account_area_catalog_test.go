package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAccountAreaCatalogDataSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/plans":
			if r.URL.Query().Get("region") != "montreal" {
				t.Errorf("query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"slug":"nano","name":"Nano","description":"small","vcpu":1,"memory_mb":1024,"disk_gb":10,"traffic_tb":null,"port_mbps":1000,"regions":["montreal"],"status":"available","prices":[{"currency":"CAD","hourly":"0.01","hourly_month_equivalent":"7.30","hourly_stopped":null,"monthly":"7.30","annual":"73","setup_fee":"0"}]},{"slug":"pro","traffic_tb":5,"prices":[{"hourly_stopped":"0.02"}]}]}`))
		case "/images":
			q := r.URL.Query()
			if q.Get("region") != "montreal" || q.Get("plan") != "nano" || q.Get("category") != "linux" {
				t.Errorf("query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"slug":"debian","name":"Debian","family":"debian","category":"linux","version":null,"status":"available","regions":["montreal"],"min_disk_gb":null,"min_memory_mb":1024,"default_user":"debian","supports_ssh_keys":true},{"slug":"ubuntu","version":"24.04","min_disk_gb":20,"min_memory_mb":2048}]}`))
		case "/vpcs":
			_, _ = w.Write([]byte(`{"data":[{"id":3,"name":"private","cidr":"10.0.0.0/24","region":"montreal","status":"active","gateway":"10.0.0.1","internet_gateway":true,"members":[{"vm":8,"hostname":"web","ip":"10.0.0.2","configuration":"automatic"}],"created_at":"2026-01-01T00:00:00Z"}],"quota":4}`))
		case "/ssh-keys":
			_, _ = w.Write([]byte(`{"data":[{"id":8,"name":"work","fingerprint":"SHA256:abc","public_key":"ssh-ed25519 public","created_at":"2026-01-01T00:00:00Z"}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	api := testAPIClient(server)
	plans := NewAccountAreaPlansDataSource().(*AccountAreaPlansDataSource)
	plans.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var planSchema datasource.SchemaResponse
	plans.Schema(ctx, datasource.SchemaRequest{}, &planSchema)
	planRead := datasource.ReadResponse{State: tfsdk.State{Schema: planSchema.Schema}}
	plans.Read(ctx, datasource.ReadRequest{Config: accountAreaTestConfig(t, planSchema.Schema, map[string]any{"region": "montreal"})}, &planRead)
	var planState AccountAreaPlansModel
	planRead.Diagnostics.Append(planRead.State.Get(ctx, &planState)...)
	if planRead.Diagnostics.HasError() || len(planState.Plans.Elements()) != 2 {
		t.Fatalf("plans = %#v; %v", planState, planRead.Diagnostics)
	}

	images := NewAccountAreaImagesDataSource().(*AccountAreaImagesDataSource)
	images.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var imageSchema datasource.SchemaResponse
	images.Schema(ctx, datasource.SchemaRequest{}, &imageSchema)
	imageRead := datasource.ReadResponse{State: tfsdk.State{Schema: imageSchema.Schema}}
	images.Read(ctx, datasource.ReadRequest{Config: accountAreaTestConfig(t, imageSchema.Schema, map[string]any{"region": "montreal", "plan": "nano", "category": "linux"})}, &imageRead)
	var imageState AccountAreaImagesModel
	imageRead.Diagnostics.Append(imageRead.State.Get(ctx, &imageState)...)
	if imageRead.Diagnostics.HasError() || len(imageState.Images.Elements()) != 2 {
		t.Fatalf("images = %#v; %v", imageState, imageRead.Diagnostics)
	}

	vpcs := NewAccountAreaVPCsDataSource().(*AccountAreaVPCsDataSource)
	vpcs.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var vpcSchema datasource.SchemaResponse
	vpcs.Schema(ctx, datasource.SchemaRequest{}, &vpcSchema)
	vpcRead := datasource.ReadResponse{State: tfsdk.State{Schema: vpcSchema.Schema}}
	vpcs.Read(ctx, datasource.ReadRequest{}, &vpcRead)
	var vpcState AccountAreaVPCsModel
	vpcRead.Diagnostics.Append(vpcRead.State.Get(ctx, &vpcState)...)
	if vpcRead.Diagnostics.HasError() || vpcState.Quota.ValueInt64() != 4 || len(vpcState.VPCs.Elements()) != 1 {
		t.Fatalf("VPCs = %#v; %v", vpcState, vpcRead.Diagnostics)
	}

	keys := NewAccountAreaSSHKeysDataSource().(*AccountAreaSSHKeysDataSource)
	keys.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var keySchema datasource.SchemaResponse
	keys.Schema(ctx, datasource.SchemaRequest{}, &keySchema)
	keyRead := datasource.ReadResponse{State: tfsdk.State{Schema: keySchema.Schema}}
	keys.Read(ctx, datasource.ReadRequest{}, &keyRead)
	var keyState AccountAreaSSHKeysModel
	keyRead.Diagnostics.Append(keyRead.State.Get(ctx, &keyState)...)
	if keyRead.Diagnostics.HasError() || len(keyState.SSHKeys.Elements()) != 1 {
		t.Fatalf("SSH keys = %#v; %v", keyState, keyRead.Diagnostics)
	}
}

func TestAccountAreaImageErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		category string
	}{{"category", http.StatusOK, "invalid"}, {"api", http.StatusServiceUnavailable, "linux"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status) }))
			defer server.Close()
			ds := NewAccountAreaImagesDataSource().(*AccountAreaImagesDataSource)
			ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
			var sch datasource.SchemaResponse
			ds.Schema(context.Background(), datasource.SchemaRequest{}, &sch)
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
			ds.Read(context.Background(), datasource.ReadRequest{Config: accountAreaTestConfig(t, sch.Schema, map[string]any{"category": tc.category})}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("Read should report an error")
			}
		})
	}
}

func accountAreaTestConfig(t *testing.T, schema schema.Schema, inputs map[string]any) tfsdk.Config {
	t.Helper()
	ctx := context.Background()
	objectType := schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, typ := range objectType.AttributeTypes {
		value, ok := inputs[name]
		if ok {
			values[name] = tftypes.NewValue(typ, value)
		} else {
			values[name] = tftypes.NewValue(typ, nil)
		}
	}
	return tfsdk.Config{Raw: tftypes.NewValue(objectType, values), Schema: schema}
}

func TestAccountAreaCatalogAPIErrorBranches(t *testing.T) {
	for _, source := range []datasource.DataSource{NewAccountAreaPlansDataSource(), NewAccountAreaImagesDataSource(), NewAccountAreaVPCsDataSource(), NewAccountAreaSSHKeysDataSource()} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
		ds := source
		if configurable, ok := ds.(datasource.DataSourceWithConfigure); ok {
			configurable.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
		}
		var sch datasource.SchemaResponse
		ds.Schema(context.Background(), datasource.SchemaRequest{}, &sch)
		resp := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
		request := datasource.ReadRequest{Config: accountAreaTestConfig(t, sch.Schema, nil)}
		ds.Read(context.Background(), request, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%T should surface an API error", ds)
		}
		server.Close()
	}
}

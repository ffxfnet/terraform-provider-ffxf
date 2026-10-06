package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const poolJSON = `{"data":{"id":9,"name":"frontend","protocol":"http","algorithm":"roundrobin","health_type":"http","health_path":"/health","health_interval":5,"send_proxy":false,"targets":[{"id":10,"vm":7,"hostname":"web-01","address":"10.0.0.7","port":3000,"weight":1,"drain":false,"health":"up","sessions":2,"response_ms":3}]}}`

const listenerJSON = `{"data":{"id":11,"port":443,"protocol":"https","default_pool":9,"redirect_https":true,"sources":["203.0.113.0/24"],"rules":[{"hostname":"app.example.test","path_prefix":"/api","pool":9}]}}`

func TestLoadBalancerPoolResourceLifecycle(t *testing.T) {
	lbJSON := `{"data":{"id":8,"name":"edge","status":"active","vpc":12,"pools":[` + poolJSON[8:len(poolJSON)-1] + `],"listeners":[]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /load-balancers/8/pools":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(poolJSON))
		case "GET /load-balancers/8":
			_, _ = w.Write([]byte(lbJSON))
		case "PATCH /load-balancers/8/pools/9", "PUT /load-balancers/8/pools/9/targets":
			_, _ = w.Write([]byte(poolJSON))
		case "DELETE /load-balancers/8/pools/9":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	pool := NewLoadBalancerPoolResource().(*LoadBalancerPoolResource)
	pool.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	var schemaResponse resource.SchemaResponse
	pool.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	var metadata resource.MetadataResponse
	pool.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_load_balancer_pool" {
		t.Fatalf("Metadata() TypeName = %q", metadata.TypeName)
	}
	planned := LoadBalancerPoolResourceModel{
		ID: types.Int64Null(), LoadBalancerID: types.Int64Value(8), Name: types.StringValue("frontend"),
		Protocol: types.StringValue("http"), Algorithm: types.StringValue("roundrobin"), HealthType: types.StringValue("http"),
		HealthPath: types.StringValue("/health"), HealthInterval: types.Int64Value(5), SendProxy: types.BoolValue(false), Targets: poolTargetList(t),
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("set pool plan: %v", diags)
	}
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	pool.Create(ctx, resource.CreateRequest{Plan: plan}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", createResponse.Diagnostics)
	}
	var created LoadBalancerPoolResourceModel
	createResponse.Diagnostics.Append(createResponse.State.Get(ctx, &created)...)
	if created.ID.ValueInt64() != 9 || len(created.Targets.Elements()) != 1 {
		t.Fatalf("Create() state = %#v, diagnostics=%v", created, createResponse.Diagnostics)
	}
	readResponse := resource.ReadResponse{State: createResponse.State}
	pool.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", readResponse.Diagnostics)
	}
	updateResponse := resource.UpdateResponse{State: readResponse.State}
	pool.Update(ctx, resource.UpdateRequest{Plan: plan, State: readResponse.State}, &updateResponse)
	if updateResponse.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics = %v", updateResponse.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	pool.Delete(ctx, resource.DeleteRequest{State: updateResponse.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete() diagnostics = %v", deleteResponse.Diagnostics)
	}
}

func TestLoadBalancerListenerResourceLifecycle(t *testing.T) {
	lbJSON := `{"data":{"id":8,"name":"edge","status":"active","vpc":12,"pools":[],"listeners":[` + listenerJSON[8:len(listenerJSON)-1] + `]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /load-balancers/8/listeners":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(listenerJSON))
		case "GET /load-balancers/8":
			_, _ = w.Write([]byte(lbJSON))
		case "PATCH /load-balancers/8/listeners/11", "PUT /load-balancers/8/listeners/11/rules":
			_, _ = w.Write([]byte(listenerJSON))
		case "DELETE /load-balancers/8/listeners/11":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	listener := NewLoadBalancerListenerResource().(*LoadBalancerListenerResource)
	listener.Configure(ctx, resource.ConfigureRequest{ProviderData: testAPIClient(server)}, &resource.ConfigureResponse{})
	var schemaResponse resource.SchemaResponse
	listener.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	var metadata resource.MetadataResponse
	listener.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
	if metadata.TypeName != "ffxf_load_balancer_listener" {
		t.Fatalf("Metadata() TypeName = %q", metadata.TypeName)
	}
	planned := LoadBalancerListenerResourceModel{
		ID: types.Int64Null(), LoadBalancerID: types.Int64Value(8), Port: types.Int64Value(443), Protocol: types.StringValue("https"),
		DefaultPool: types.Int64Value(9), RedirectHTTPS: types.BoolValue(true), Sources: sourceList(t), Rules: listenerRuleList(t),
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("set listener plan: %v", diags)
	}
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	listener.Create(ctx, resource.CreateRequest{Plan: plan}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatalf("Create() diagnostics = %v", createResponse.Diagnostics)
	}
	var created LoadBalancerListenerResourceModel
	createResponse.Diagnostics.Append(createResponse.State.Get(ctx, &created)...)
	if created.ID.ValueInt64() != 11 || len(created.Rules.Elements()) != 1 {
		t.Fatalf("Create() state = %#v", created)
	}
	readResponse := resource.ReadResponse{State: createResponse.State}
	listener.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", readResponse.Diagnostics)
	}
	updateResponse := resource.UpdateResponse{State: readResponse.State}
	listener.Update(ctx, resource.UpdateRequest{Plan: plan, State: readResponse.State}, &updateResponse)
	if updateResponse.Diagnostics.HasError() {
		t.Fatalf("Update() diagnostics = %v", updateResponse.Diagnostics)
	}
	deleteResponse := resource.DeleteResponse{}
	listener.Delete(ctx, resource.DeleteRequest{State: updateResponse.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() {
		t.Fatalf("Delete() diagnostics = %v", deleteResponse.Diagnostics)
	}
}

func TestNetworkInventoryDataSourcesRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/firewalls":
			_, _ = w.Write([]byte(`{"data":[{"id":2,"name":"web","created_at":"2026-10-06T12:00:00Z"}],"quota":10}`))
		case "/load-balancers":
			_, _ = w.Write([]byte(`{"data":[{"id":8,"name":"edge","status":"active","vpc":12,"ipv4":"192.0.2.8","ipv6":null,"dns_name":"edge.example.test","config_applied":true}],"quota":3}`))
		case "/load-balancers/8/metrics":
			_, _ = w.Write([]byte(`{"data":{"range":"24h","step_minutes":15,"points":[{"at":"2026-10-06T12:00:00Z","requests":4,"http_2xx":3,"http_3xx":0,"http_4xx":0,"http_5xx":1,"connections":2,"bytes_in":100,"bytes_out":200,"sessions_max":5}],"totals":{"requests":4,"errors_pct":25,"bytes":300,"sessions_max":5}}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()

	firewalls := NewFirewallsDataSource().(*FirewallsDataSource)
	firewalls.Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
	var firewallSchema datasource.SchemaResponse
	firewalls.Schema(ctx, datasource.SchemaRequest{}, &firewallSchema)
	var firewallMetadata datasource.MetadataResponse
	firewalls.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &firewallMetadata)
	if firewallMetadata.TypeName != "ffxf_firewalls" {
		t.Fatalf("Metadata() TypeName = %q", firewallMetadata.TypeName)
	}
	firewallResponse := datasource.ReadResponse{State: tfsdk.State{Schema: firewallSchema.Schema}}
	firewalls.Read(ctx, datasource.ReadRequest{}, &firewallResponse)
	if firewallResponse.Diagnostics.HasError() {
		t.Fatalf("Firewalls Read() diagnostics = %v", firewallResponse.Diagnostics)
	}
	var firewallData FirewallsDataSourceModel
	firewallResponse.Diagnostics.Append(firewallResponse.State.Get(ctx, &firewallData)...)
	if firewallData.Quota.ValueInt64() != 10 || len(firewallData.Firewalls.Elements()) != 1 {
		t.Fatalf("firewalls datasource state = %#v", firewallData)
	}

	balancers := NewLoadBalancersDataSource().(*LoadBalancersDataSource)
	balancers.Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
	var balancerSchema datasource.SchemaResponse
	balancers.Schema(ctx, datasource.SchemaRequest{}, &balancerSchema)
	var balancerMetadata datasource.MetadataResponse
	balancers.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &balancerMetadata)
	if balancerMetadata.TypeName != "ffxf_load_balancers" {
		t.Fatalf("Metadata() TypeName = %q", balancerMetadata.TypeName)
	}
	balancerResponse := datasource.ReadResponse{State: tfsdk.State{Schema: balancerSchema.Schema}}
	balancers.Read(ctx, datasource.ReadRequest{}, &balancerResponse)
	if balancerResponse.Diagnostics.HasError() {
		t.Fatalf("LoadBalancers Read() diagnostics = %v", balancerResponse.Diagnostics)
	}
	var balancerData LoadBalancersDataSourceModel
	balancerResponse.Diagnostics.Append(balancerResponse.State.Get(ctx, &balancerData)...)
	if balancerData.Quota.ValueInt64() != 3 || len(balancerData.LoadBalancers.Elements()) != 1 {
		t.Fatalf("load balancers datasource state = %#v", balancerData)
	}

	metrics := NewLoadBalancerMetricsDataSource().(*LoadBalancerMetricsDataSource)
	metrics.Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
	var metricsSchema datasource.SchemaResponse
	metrics.Schema(ctx, datasource.SchemaRequest{}, &metricsSchema)
	var metricsMetadata datasource.MetadataResponse
	metrics.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metricsMetadata)
	if metricsMetadata.TypeName != "ffxf_load_balancer_metrics" {
		t.Fatalf("Metadata() TypeName = %q", metricsMetadata.TypeName)
	}
	objectType, ok := metricsSchema.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("metrics schema type is not an object")
	}
	configValues := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		configValues[name] = tftypes.NewValue(attributeType, nil)
	}
	configValues["load_balancer_id"] = tftypes.NewValue(objectType.AttributeTypes["load_balancer_id"], int64(8))
	configValues["range"] = tftypes.NewValue(objectType.AttributeTypes["range"], "24h")
	config := tfsdk.Config{Raw: tftypes.NewValue(objectType, configValues), Schema: metricsSchema.Schema}
	metricsResponse := datasource.ReadResponse{State: tfsdk.State{Schema: metricsSchema.Schema}}
	metrics.Read(ctx, datasource.ReadRequest{Config: config}, &metricsResponse)
	if metricsResponse.Diagnostics.HasError() {
		t.Fatalf("Metrics Read() diagnostics = %v", metricsResponse.Diagnostics)
	}
	var metricsData LoadBalancerMetricsDataSourceModel
	metricsResponse.Diagnostics.Append(metricsResponse.State.Get(ctx, &metricsData)...)
	if metricsData.Requests.ValueInt64() != 4 || metricsData.Points.IsNull() {
		t.Fatalf("metrics datasource state = %#v", metricsData)
	}
}

func poolTargetList(t *testing.T) types.List {
	t.Helper()
	attrTypes := map[string]attr.Type{
		"id": types.Int64Type, "vm": types.Int64Type, "hostname": types.StringType, "address": types.StringType,
		"port": types.Int64Type, "weight": types.Int64Type, "drain": types.BoolType, "health": types.StringType,
		"sessions": types.Int64Type, "response_ms": types.Int64Type,
	}
	values := []LoadBalancerTargetModel{{
		ID: types.Int64Unknown(), VM: types.Int64Value(7), Hostname: types.StringUnknown(), Address: types.StringUnknown(), Port: types.Int64Value(3000),
		Weight: types.Int64Null(), Drain: types.BoolNull(), Health: types.StringUnknown(), Sessions: types.Int64Unknown(), ResponseMS: types.Int64Unknown(),
	}}
	list, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: attrTypes}, values)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

func sourceList(t *testing.T) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"203.0.113.0/24"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

func listenerRuleList(t *testing.T) types.List {
	t.Helper()
	attrTypes := map[string]attr.Type{"hostname": types.StringType, "path_prefix": types.StringType, "pool": types.Int64Type}
	values := []LoadBalancerRuleModel{{Hostname: types.StringValue("app.example.test"), PathPrefix: types.StringValue("/api"), Pool: types.Int64Value(9)}}
	list, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: attrTypes}, values)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

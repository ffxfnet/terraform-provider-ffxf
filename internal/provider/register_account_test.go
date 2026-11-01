package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAccountAreaRegistrationConstructors(t *testing.T) {
	ctx := context.Background()
	wantResources := map[string]bool{"ffxf_ssh_key": false, "ffxf_vpc_member": false, "ffxf_account_budget": false}
	for _, newResource := range accountAreaResources() {
		r := newResource()
		var metadata resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		if _, ok := wantResources[metadata.TypeName]; !ok {
			t.Errorf("unexpected resource %q", metadata.TypeName)
			continue
		}
		wantResources[metadata.TypeName] = true
		var sch resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sch)
		if sch.Diagnostics.HasError() {
			t.Errorf("%s schema: %v", metadata.TypeName, sch.Diagnostics)
		}
		if configurable, ok := r.(resource.ResourceWithConfigure); ok {
			configurable.Configure(ctx, resource.ConfigureRequest{}, &resource.ConfigureResponse{})
		}
	}
	for name, found := range wantResources {
		if !found {
			t.Errorf("missing resource %s", name)
		}
	}
	wantSources := map[string]bool{"ffxf_plans": false, "ffxf_images": false, "ffxf_vpcs": false, "ffxf_ssh_keys": false, "ffxf_account": false, "ffxf_usage": false, "ffxf_runners": false, "ffxf_runner_jobs": false}
	for _, newSource := range accountAreaDataSources() {
		d := newSource()
		var metadata datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
		if _, ok := wantSources[metadata.TypeName]; !ok {
			t.Errorf("unexpected data source %q", metadata.TypeName)
			continue
		}
		wantSources[metadata.TypeName] = true
		var sch datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &sch)
		if sch.Diagnostics.HasError() {
			t.Errorf("%s schema: %v", metadata.TypeName, sch.Diagnostics)
		}
		if configurable, ok := d.(datasource.DataSourceWithConfigure); ok {
			configurable.Configure(ctx, datasource.ConfigureRequest{}, &datasource.ConfigureResponse{})
		}
	}
	for name, found := range wantSources {
		if !found {
			t.Errorf("missing data source %s", name)
		}
	}
}

package provider

import (
	"context"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &FirewallsDataSource{}
var _ datasource.DataSource = &LoadBalancersDataSource{}

type FirewallsDataSource struct{ client *client.Client }
type LoadBalancersDataSource struct{ client *client.Client }

type FirewallsDataSourceModel struct {
	Quota     types.Int64 `tfsdk:"quota"`
	Firewalls types.List  `tfsdk:"firewalls"`
}

type FirewallInventoryModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type LoadBalancersDataSourceModel struct {
	Quota         types.Int64 `tfsdk:"quota"`
	LoadBalancers types.List  `tfsdk:"load_balancers"`
}

type LoadBalancerInventoryModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Status        types.String `tfsdk:"status"`
	VPC           types.Int64  `tfsdk:"vpc"`
	IPv4          types.String `tfsdk:"ipv4"`
	IPv6          types.String `tfsdk:"ipv6"`
	DNSName       types.String `tfsdk:"dns_name"`
	ConfigApplied types.Bool   `tfsdk:"config_applied"`
}

func NewFirewallsDataSource() datasource.DataSource     { return &FirewallsDataSource{} }
func NewLoadBalancersDataSource() datasource.DataSource { return &LoadBalancersDataSource{} }

func (d *FirewallsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewalls"
}

func (d *FirewallsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the account's firewalls and the account firewall quota.",
		Attributes: map[string]schema.Attribute{
			"quota": schema.Int64Attribute{Computed: true},
			"firewalls": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id": schema.Int64Attribute{Computed: true}, "name": schema.StringAttribute{Computed: true}, "created_at": schema.StringAttribute{Computed: true},
			}}},
		},
	}
}

func (d *FirewallsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}

func (d *FirewallsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListFirewalls(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF firewalls", err.Error())
		return
	}
	data := FirewallsDataSourceModel{Quota: types.Int64Value(int64(result.Quota))}
	firewalls := make([]FirewallInventoryModel, 0, len(result.Data))
	for _, firewall := range result.Data {
		firewalls = append(firewalls, FirewallInventoryModel{ID: types.Int64Value(firewall.ID), Name: types.StringValue(firewall.Name), CreatedAt: types.StringValue(firewall.CreatedAt)})
	}
	attrTypes := map[string]attr.Type{"id": types.Int64Type, "name": types.StringType, "created_at": types.StringType}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, firewalls)
	resp.Diagnostics.Append(diags...)
	data.Firewalls = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *LoadBalancersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancers"
}

func (d *LoadBalancersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the account's load balancers, public addresses, configuration status, and account quota.",
		Attributes: map[string]schema.Attribute{
			"quota": schema.Int64Attribute{Computed: true},
			"load_balancers": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id": schema.Int64Attribute{Computed: true}, "name": schema.StringAttribute{Computed: true},
				"status": schema.StringAttribute{Computed: true}, "vpc": schema.Int64Attribute{Computed: true},
				"ipv4": schema.StringAttribute{Computed: true}, "ipv6": schema.StringAttribute{Computed: true},
				"dns_name": schema.StringAttribute{Computed: true}, "config_applied": schema.BoolAttribute{Computed: true},
			}}},
		},
	}
}

func (d *LoadBalancersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}

func (d *LoadBalancersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListLoadBalancers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF load balancers", err.Error())
		return
	}
	data := LoadBalancersDataSourceModel{Quota: types.Int64Value(int64(result.Quota))}
	loadBalancers := make([]LoadBalancerInventoryModel, 0, len(result.Data))
	for _, lb := range result.Data {
		loadBalancers = append(loadBalancers, LoadBalancerInventoryModel{
			ID: types.Int64Value(lb.ID), Name: types.StringValue(lb.Name), Status: types.StringValue(lb.Status), VPC: types.Int64Value(lb.VPC),
			IPv4: nullableString(lb.IPv4), IPv6: nullableString(lb.IPv6), DNSName: nullableString(lb.DNSName), ConfigApplied: types.BoolValue(lb.ConfigApplied),
		})
	}
	attrTypes := map[string]attr.Type{
		"id": types.Int64Type, "name": types.StringType, "status": types.StringType, "vpc": types.Int64Type,
		"ipv4": types.StringType, "ipv6": types.StringType, "dns_name": types.StringType, "config_applied": types.BoolType,
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, loadBalancers)
	resp.Diagnostics.Append(diags...)
	data.LoadBalancers = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

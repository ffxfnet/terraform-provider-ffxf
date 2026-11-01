package provider

import (
	"context"
	"fmt"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*AccountAreaPlansDataSource)(nil)
var _ datasource.DataSource = (*AccountAreaImagesDataSource)(nil)
var _ datasource.DataSource = (*AccountAreaVPCsDataSource)(nil)
var _ datasource.DataSource = (*AccountAreaSSHKeysDataSource)(nil)

type AccountAreaPlansDataSource struct{ client *client.Client }
type AccountAreaImagesDataSource struct{ client *client.Client }
type AccountAreaVPCsDataSource struct{ client *client.Client }
type AccountAreaSSHKeysDataSource struct{ client *client.Client }

func NewAccountAreaPlansDataSource() datasource.DataSource   { return &AccountAreaPlansDataSource{} }
func NewAccountAreaImagesDataSource() datasource.DataSource  { return &AccountAreaImagesDataSource{} }
func NewAccountAreaVPCsDataSource() datasource.DataSource    { return &AccountAreaVPCsDataSource{} }
func NewAccountAreaSSHKeysDataSource() datasource.DataSource { return &AccountAreaSSHKeysDataSource{} }

type AccountAreaPlansModel struct {
	Region types.String `tfsdk:"region"`
	Plans  types.List   `tfsdk:"plans"`
}
type AccountAreaPlanModel struct {
	Slug        types.String  `tfsdk:"slug"`
	Name        types.String  `tfsdk:"name"`
	Description types.String  `tfsdk:"description"`
	VCPU        types.Int64   `tfsdk:"vcpu"`
	MemoryMB    types.Int64   `tfsdk:"memory_mb"`
	DiskGB      types.Int64   `tfsdk:"disk_gb"`
	TrafficTB   types.Float64 `tfsdk:"traffic_tb"`
	PortMbps    types.Int64   `tfsdk:"port_mbps"`
	Regions     types.List    `tfsdk:"regions"`
	Status      types.String  `tfsdk:"status"`
	Prices      types.List    `tfsdk:"prices"`
}
type AccountAreaImagesModel struct {
	Region   types.String `tfsdk:"region"`
	Plan     types.String `tfsdk:"plan"`
	Category types.String `tfsdk:"category"`
	Images   types.List   `tfsdk:"images"`
}
type AccountAreaImageModel struct {
	Slug            types.String `tfsdk:"slug"`
	Name            types.String `tfsdk:"name"`
	Family          types.String `tfsdk:"family"`
	Category        types.String `tfsdk:"category"`
	Version         types.String `tfsdk:"version"`
	Status          types.String `tfsdk:"status"`
	Regions         types.List   `tfsdk:"regions"`
	MinDiskGB       types.Int64  `tfsdk:"min_disk_gb"`
	MinMemoryMB     types.Int64  `tfsdk:"min_memory_mb"`
	DefaultUser     types.String `tfsdk:"default_user"`
	SupportsSSHKeys types.Bool   `tfsdk:"supports_ssh_keys"`
}
type AccountAreaVPCsModel struct {
	Quota types.Int64 `tfsdk:"quota"`
	VPCs  types.List  `tfsdk:"vpcs"`
}
type AccountAreaVPCModel struct {
	ID              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	CIDR            types.String `tfsdk:"cidr"`
	Region          types.String `tfsdk:"region"`
	Status          types.String `tfsdk:"status"`
	Gateway         types.String `tfsdk:"gateway"`
	InternetGateway types.Bool   `tfsdk:"internet_gateway"`
	Members         types.List   `tfsdk:"members"`
	CreatedAt       types.String `tfsdk:"created_at"`
}
type AccountAreaVPCMemberModel struct {
	VM            types.Int64  `tfsdk:"vm"`
	Hostname      types.String `tfsdk:"hostname"`
	IP            types.String `tfsdk:"ip"`
	Configuration types.String `tfsdk:"configuration"`
}
type AccountAreaSSHKeysModel struct {
	SSHKeys types.List `tfsdk:"ssh_keys"`
}
type AccountAreaSSHKeyModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	PublicKey   types.String `tfsdk:"public_key"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (d *AccountAreaPlansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plans"
}
func (d *AccountAreaPlansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaPlansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists FFXF Cloud compute plans, optionally filtered by region.", Attributes: map[string]schema.Attribute{
		"region": schema.StringAttribute{Optional: true, Description: "Keep only plans offered in this region."},
		"plans":  schema.ListNestedAttribute{Computed: true, Description: "Plans available to the account.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaPlanAttributes()}},
	}}
}
func (d *AccountAreaPlansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountAreaPlansModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.AccountAreaListPlans(ctx, data.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF plans", err.Error())
		return
	}
	values := make([]AccountAreaPlanModel, 0, len(result.Data))
	for _, plan := range result.Data {
		values = append(values, accountAreaPlanModel(ctx, plan, &resp.Diagnostics))
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: accountAreaPlanAttrTypes()}, values)
	resp.Diagnostics.Append(diags...)
	data.Plans = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *AccountAreaImagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}
func (d *AccountAreaImagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaImagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists FFXF Cloud images, optionally filtered by region, plan, or category.", Attributes: map[string]schema.Attribute{
		"region": schema.StringAttribute{Optional: true, Description: "Keep only images offered in this region."}, "plan": schema.StringAttribute{Optional: true, Description: "Keep only images compatible with this compute plan."},
		"category": schema.StringAttribute{Optional: true, Description: "Keep only images in linux, windows, bsd, app, or other category.", Validators: []validator.String{accountAreaEnumValidator{allowed: []string{"linux", "windows", "bsd", "app", "other"}}}}, "images": schema.ListNestedAttribute{Computed: true, Description: "Images matching the filters.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaImageAttributes()}},
	}}
}
func (d *AccountAreaImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountAreaImagesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	filters := map[string]string{"region": data.Region.ValueString(), "plan": data.Plan.ValueString(), "category": data.Category.ValueString()}
	if category := filters["category"]; category != "" && category != "linux" && category != "windows" && category != "bsd" && category != "app" && category != "other" {
		resp.Diagnostics.AddError("Invalid image category", fmt.Sprintf("Category %q must be linux, windows, bsd, app, or other.", category))
		return
	}
	result, err := d.client.AccountAreaListImages(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF images", err.Error())
		return
	}
	values := make([]AccountAreaImageModel, 0, len(result.Data))
	for _, image := range result.Data {
		values = append(values, accountAreaImageModel(ctx, image, &resp.Diagnostics))
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: accountAreaImageAttrTypes()}, values)
	resp.Diagnostics.Append(diags...)
	data.Images = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *AccountAreaVPCsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpcs"
}
func (d *AccountAreaVPCsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaVPCsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists private networks and the account quota.", Attributes: map[string]schema.Attribute{"quota": schema.Int64Attribute{Computed: true, Description: "Maximum number of private networks for this account."}, "vpcs": schema.ListNestedAttribute{Computed: true, Description: "Private networks in this account.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaVPCAttributes()}}}}
}
func (d *AccountAreaVPCsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.AccountAreaListVPCs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF VPCs", err.Error())
		return
	}
	data := AccountAreaVPCsModel{Quota: types.Int64Value(int64(result.Quota))}
	values := make([]AccountAreaVPCModel, 0, len(result.Data))
	for _, vpc := range result.Data {
		values = append(values, accountAreaVPCModel(ctx, vpc, &resp.Diagnostics))
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: accountAreaVPCAttrTypes()}, values)
	resp.Diagnostics.Append(diags...)
	data.VPCs = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (d *AccountAreaSSHKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}
func (d *AccountAreaSSHKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaSSHKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists SSH public keys registered to the account.", Attributes: map[string]schema.Attribute{"ssh_keys": schema.ListNestedAttribute{Computed: true, Description: "SSH public keys registered to the account.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaSSHKeyAttributes()}}}}
}
func (d *AccountAreaSSHKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.AccountAreaListSSHKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF SSH keys", err.Error())
		return
	}
	values := make([]AccountAreaSSHKeyModel, 0, len(result.Data))
	for _, key := range result.Data {
		values = append(values, accountAreaSSHKeyModel(key))
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: accountAreaSSHKeyAttrTypes()}, values)
	resp.Diagnostics.Append(diags...)
	data := AccountAreaSSHKeysModel{SSHKeys: list}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func accountAreaPlanAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"slug": schema.StringAttribute{Computed: true, Description: "Stable plan slug."}, "name": schema.StringAttribute{Computed: true, Description: "Display name."}, "description": schema.StringAttribute{Computed: true, Description: "Intended workload."}, "vcpu": schema.Int64Attribute{Computed: true, Description: "Virtual CPU count."}, "memory_mb": schema.Int64Attribute{Computed: true, Description: "Memory in MiB."}, "disk_gb": schema.Int64Attribute{Computed: true, Description: "Disk size in GiB."}, "traffic_tb": schema.Float64Attribute{Computed: true, Description: "Included traffic in TiB, when metered."}, "port_mbps": schema.Int64Attribute{Computed: true, Description: "Network port speed in Mbps."}, "regions": schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Available region slugs."}, "status": schema.StringAttribute{Computed: true, Description: "Plan availability status."}, "prices": schema.ListNestedAttribute{Computed: true, Description: "Prices by currency.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaPriceAttributes()}},
	}
}
func accountAreaPriceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"currency": schema.StringAttribute{Computed: true, Description: "Currency code."}, "hourly": schema.StringAttribute{Computed: true, Description: "Hourly price as a decimal string."}, "hourly_month_equivalent": schema.StringAttribute{Computed: true, Description: "Monthly equivalent of hourly billing."}, "hourly_stopped": schema.StringAttribute{Computed: true, Description: "Hourly stopped-machine price, when offered."}, "monthly": schema.StringAttribute{Computed: true, Description: "Monthly price."}, "annual": schema.StringAttribute{Computed: true, Description: "Annual price."}, "setup_fee": schema.StringAttribute{Computed: true, Description: "One-time setup fee."}}
}
func accountAreaPlanAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"slug": types.StringType, "name": types.StringType, "description": types.StringType, "vcpu": types.Int64Type, "memory_mb": types.Int64Type, "disk_gb": types.Int64Type, "traffic_tb": types.Float64Type, "port_mbps": types.Int64Type, "regions": types.ListType{ElemType: types.StringType}, "status": types.StringType, "prices": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaPriceAttrTypes()}}}
}
func accountAreaPriceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"currency": types.StringType, "hourly": types.StringType, "hourly_month_equivalent": types.StringType, "hourly_stopped": types.StringType, "monthly": types.StringType, "annual": types.StringType, "setup_fee": types.StringType}
}
func accountAreaPlanModel(ctx context.Context, p client.Plan, diags *diag.Diagnostics) AccountAreaPlanModel {
	regions, d := types.ListValueFrom(ctx, types.StringType, p.Regions)
	diags.Append(d...)
	prices := make([]attr.Value, 0, len(p.Prices))
	for _, price := range p.Prices {
		stopped := types.StringNull()
		if price.HourlyStopped != nil {
			stopped = types.StringValue(*price.HourlyStopped)
		}
		obj, od := types.ObjectValue(accountAreaPriceAttrTypes(), map[string]attr.Value{"currency": types.StringValue(price.Currency), "hourly": types.StringValue(price.Hourly), "hourly_month_equivalent": types.StringValue(price.HourlyMonthEquivalent), "hourly_stopped": stopped, "monthly": types.StringValue(price.Monthly), "annual": types.StringValue(price.Annual), "setup_fee": types.StringValue(price.SetupFee)})
		diags.Append(od...)
		prices = append(prices, obj)
	}
	priceList, pd := types.ListValue(types.ObjectType{AttrTypes: accountAreaPriceAttrTypes()}, prices)
	diags.Append(pd...)
	traffic := types.Float64Null()
	if p.TrafficTB != nil {
		traffic = types.Float64Value(*p.TrafficTB)
	}
	return AccountAreaPlanModel{Slug: types.StringValue(p.Slug), Name: types.StringValue(p.Name), Description: types.StringValue(p.Description), VCPU: types.Int64Value(int64(p.VCPU)), MemoryMB: types.Int64Value(int64(p.MemoryMB)), DiskGB: types.Int64Value(int64(p.DiskGB)), TrafficTB: traffic, PortMbps: types.Int64Value(int64(p.PortMbps)), Regions: regions, Status: types.StringValue(p.Status), Prices: priceList}
}
func accountAreaImageAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"slug": schema.StringAttribute{Computed: true, Description: "Stable image slug."}, "name": schema.StringAttribute{Computed: true, Description: "Display name."}, "family": schema.StringAttribute{Computed: true, Description: "Operating system or application family."}, "category": schema.StringAttribute{Computed: true, Description: "Image category."}, "version": schema.StringAttribute{Computed: true, Description: "Image version, when applicable."}, "status": schema.StringAttribute{Computed: true, Description: "Availability status."}, "regions": schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Available region slugs."}, "min_disk_gb": schema.Int64Attribute{Computed: true, Description: "Minimum disk size in GiB, when specified."}, "min_memory_mb": schema.Int64Attribute{Computed: true, Description: "Minimum memory in MiB, when specified."}, "default_user": schema.StringAttribute{Computed: true, Description: "Default operating-system user."}, "supports_ssh_keys": schema.BoolAttribute{Computed: true, Description: "Whether SSH key provisioning is supported."}}
}
func accountAreaImageAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"slug": types.StringType, "name": types.StringType, "family": types.StringType, "category": types.StringType, "version": types.StringType, "status": types.StringType, "regions": types.ListType{ElemType: types.StringType}, "min_disk_gb": types.Int64Type, "min_memory_mb": types.Int64Type, "default_user": types.StringType, "supports_ssh_keys": types.BoolType}
}
func accountAreaImageModel(ctx context.Context, i client.Image, diags *diag.Diagnostics) AccountAreaImageModel {
	regions, d := types.ListValueFrom(ctx, types.StringType, i.Regions)
	diags.Append(d...)
	version := types.StringNull()
	if i.Version != nil {
		version = types.StringValue(*i.Version)
	}
	disk := types.Int64Null()
	if i.MinDiskGB != nil {
		disk = types.Int64Value(int64(*i.MinDiskGB))
	}
	memory := types.Int64Null()
	if i.MinMemoryMB != nil {
		memory = types.Int64Value(int64(*i.MinMemoryMB))
	}
	return AccountAreaImageModel{Slug: types.StringValue(i.Slug), Name: types.StringValue(i.Name), Family: types.StringValue(i.Family), Category: types.StringValue(i.Category), Version: version, Status: types.StringValue(i.Status), Regions: regions, MinDiskGB: disk, MinMemoryMB: memory, DefaultUser: types.StringValue(i.DefaultUser), SupportsSSHKeys: types.BoolValue(i.SupportsSSHKeys)}
}
func accountAreaVPCAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"id": schema.Int64Attribute{Computed: true, Description: "Numeric VPC identifier."}, "name": schema.StringAttribute{Computed: true, Description: "Network name."}, "cidr": schema.StringAttribute{Computed: true, Description: "Private IPv4 range."}, "region": schema.StringAttribute{Computed: true, Description: "Network region."}, "status": schema.StringAttribute{Computed: true, Description: "Network status."}, "gateway": schema.StringAttribute{Computed: true, Description: "Network gateway address."}, "internet_gateway": schema.BoolAttribute{Computed: true, Description: "Whether internet egress is enabled."}, "members": schema.ListNestedAttribute{Computed: true, Description: "Virtual machines connected to this network.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaVPCMemberAttributes()}}, "created_at": schema.StringAttribute{Computed: true, Description: "UTC creation time."}}
}
func accountAreaVPCMemberAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"vm": schema.Int64Attribute{Computed: true, Description: "Virtual machine identifier."}, "hostname": schema.StringAttribute{Computed: true, Description: "Virtual machine hostname."}, "ip": schema.StringAttribute{Computed: true, Description: "Private IP address."}, "configuration": schema.StringAttribute{Computed: true, Description: "Network configuration mode."}}
}
func accountAreaVPCAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"id": types.Int64Type, "name": types.StringType, "cidr": types.StringType, "region": types.StringType, "status": types.StringType, "gateway": types.StringType, "internet_gateway": types.BoolType, "members": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaVPCMemberAttrTypes()}}, "created_at": types.StringType}
}
func accountAreaVPCMemberAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"vm": types.Int64Type, "hostname": types.StringType, "ip": types.StringType, "configuration": types.StringType}
}
func accountAreaVPCModel(ctx context.Context, v client.AccountAreaVPC, diags *diag.Diagnostics) AccountAreaVPCModel {
	members := make([]AccountAreaVPCMemberModel, 0, len(v.Members))
	for _, m := range v.Members {
		members = append(members, AccountAreaVPCMemberModel{VM: types.Int64Value(int64(m.VM)), Hostname: types.StringValue(m.Hostname), IP: types.StringValue(m.IP), Configuration: types.StringValue(m.Configuration)})
	}
	list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: accountAreaVPCMemberAttrTypes()}, members)
	diags.Append(d...)
	return AccountAreaVPCModel{ID: types.Int64Value(int64(v.ID)), Name: types.StringValue(v.Name), CIDR: types.StringValue(v.CIDR), Region: types.StringValue(v.Region), Status: types.StringValue(v.Status), Gateway: types.StringValue(v.Gateway), InternetGateway: types.BoolValue(v.InternetGateway), Members: list, CreatedAt: types.StringValue(v.CreatedAt)}
}
func accountAreaSSHKeyAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"id": schema.Int64Attribute{Computed: true, Description: "Key identifier."}, "name": schema.StringAttribute{Computed: true, Description: "Key name."}, "fingerprint": schema.StringAttribute{Computed: true, Description: "Public-key fingerprint."}, "public_key": schema.StringAttribute{Computed: true, Description: "SSH public key."}, "created_at": schema.StringAttribute{Computed: true, Description: "UTC creation time."}}
}
func accountAreaSSHKeyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"id": types.Int64Type, "name": types.StringType, "fingerprint": types.StringType, "public_key": types.StringType, "created_at": types.StringType}
}
func accountAreaSSHKeyModel(k client.AccountAreaSSHKey) AccountAreaSSHKeyModel {
	return AccountAreaSSHKeyModel{ID: types.Int64Value(int64(k.ID)), Name: types.StringValue(k.Name), Fingerprint: types.StringValue(k.Fingerprint), PublicKey: types.StringValue(k.PublicKey), CreatedAt: types.StringValue(k.CreatedAt)}
}

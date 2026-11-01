package provider

import (
	"context"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VMAreaVMsDataSource{}
var _ datasource.DataSource = &VMAreaVMNetworkDataSource{}
var _ datasource.DataSource = &VMAreaVMMetricsDataSource{}
var _ datasource.DataSource = &VMAreaVMBackupsDataSource{}
var _ datasource.DataSource = &VMAreaVMActionsDataSource{}

type VMAreaVMsDataSource struct{ client *client.Client }
type VMAreaVMNetworkDataSource struct{ client *client.Client }
type VMAreaVMMetricsDataSource struct{ client *client.Client }
type VMAreaVMBackupsDataSource struct{ client *client.Client }
type VMAreaVMActionsDataSource struct{ client *client.Client }

type VMAreaVMsModel struct {
	Status  types.String `tfsdk:"status"`
	Region  types.String `tfsdk:"region"`
	Billing types.String `tfsdk:"billing"`
	VMs     types.List   `tfsdk:"vms"`
}
type VMAreaVMItemModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Hostname    types.String `tfsdk:"hostname"`
	Status      types.String `tfsdk:"status"`
	Plan        types.String `tfsdk:"plan"`
	Region      types.String `tfsdk:"region"`
	Image       types.String `tfsdk:"image"`
	IPv4        types.String `tfsdk:"ipv4"`
	IPv6        types.String `tfsdk:"ipv6"`
	BillingMode types.String `tfsdk:"billing_mode"`
}
type VMAreaVMNetworkModel struct {
	VMID       types.Int64  `tfsdk:"vm_id"`
	IPv4       types.List   `tfsdk:"ipv4"`
	IPv6       types.List   `tfsdk:"ipv6"`
	IPv6Prefix types.String `tfsdk:"ipv6_prefix"`
	PortMbps   types.Int64  `tfsdk:"port_mbps"`
	Private    types.List   `tfsdk:"private"`
}
type VMAreaPrivateNetworkModel struct {
	VPC           types.Int64  `tfsdk:"vpc"`
	Name          types.String `tfsdk:"name"`
	Address       types.String `tfsdk:"address"`
	PrefixLength  types.Int64  `tfsdk:"prefix_length"`
	Configuration types.String `tfsdk:"configuration"`
}
type VMAreaAddressModel struct {
	Address      types.String `tfsdk:"address"`
	Version      types.Int64  `tfsdk:"version"`
	Primary      types.Bool   `tfsdk:"primary"`
	Gateway      types.String `tfsdk:"gateway"`
	PrefixLength types.Int64  `tfsdk:"prefix_length"`
	RDNS         types.String `tfsdk:"rdns"`
}
type VMAreaMetricsModel struct {
	VMID        types.Int64  `tfsdk:"vm_id"`
	Timeframe   types.String `tfsdk:"timeframe"`
	StepSeconds types.Int64  `tfsdk:"step_seconds"`
	Series      types.List   `tfsdk:"series"`
}
type VMAreaMetricPointModel struct {
	Time           types.String  `tfsdk:"time"`
	CPU            types.Float64 `tfsdk:"cpu"`
	MemoryBytes    types.Float64 `tfsdk:"memory_bytes"`
	DiskReadBytes  types.Float64 `tfsdk:"disk_read_bytes"`
	DiskWriteBytes types.Float64 `tfsdk:"disk_write_bytes"`
	NetInBytes     types.Float64 `tfsdk:"net_in_bytes"`
	NetOutBytes    types.Float64 `tfsdk:"net_out_bytes"`
}
type VMAreaBackupsModel struct {
	VMID             types.Int64 `tfsdk:"vm_id"`
	AutomaticEnabled types.Bool  `tfsdk:"automatic_enabled"`
	Retention        types.Int64 `tfsdk:"retention"`
	Backups          types.List  `tfsdk:"backups"`
}
type VMAreaBackupModel struct {
	ID         types.String `tfsdk:"id"`
	Type       types.String `tfsdk:"type"`
	CreatedAt  types.String `tfsdk:"created_at"`
	SizeBytes  types.Int64  `tfsdk:"size_bytes"`
	Restorable types.Bool   `tfsdk:"restorable"`
}
type VMAreaActionsModel struct {
	VMID    types.Int64  `tfsdk:"vm_id"`
	Status  types.String `tfsdk:"status"`
	Actions types.List   `tfsdk:"actions"`
}
type VMAreaActionModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Type       types.String `tfsdk:"type"`
	Status     types.String `tfsdk:"status"`
	VMID       types.Int64  `tfsdk:"vm_id"`
	Source     types.String `tfsdk:"source"`
	CreatedAt  types.String `tfsdk:"created_at"`
	StartedAt  types.String `tfsdk:"started_at"`
	FinishedAt types.String `tfsdk:"finished_at"`
	Error      types.String `tfsdk:"error"`
}

func NewVMAreaVMsDataSource() datasource.DataSource       { return &VMAreaVMsDataSource{} }
func NewVMAreaVMNetworkDataSource() datasource.DataSource { return &VMAreaVMNetworkDataSource{} }
func NewVMAreaVMMetricsDataSource() datasource.DataSource { return &VMAreaVMMetricsDataSource{} }
func NewVMAreaVMBackupsDataSource() datasource.DataSource { return &VMAreaVMBackupsDataSource{} }
func NewVMAreaVMActionsDataSource() datasource.DataSource { return &VMAreaVMActionsDataSource{} }

func (d *VMAreaVMsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vms"
}
func (d *VMAreaVMsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists FFXF Cloud virtual machines, optionally filtered by status, region, and billing mode.", Attributes: map[string]schema.Attribute{
		"status": schema.StringAttribute{Optional: true, Description: "Filter by VM lifecycle status.", Validators: []validator.String{VMAreaStringOneOfValidator{Values: []string{"pending_payment", "provisioning", "running", "stopped", "installing", "suspended", "deleting", "deleted", "unknown", "error"}}}}, "region": schema.StringAttribute{Optional: true, Description: "Filter by region slug."}, "billing": schema.StringAttribute{Optional: true, Description: "Filter by billing mode, hourly or monthly.", Validators: []validator.String{VMAreaStringOneOfValidator{Values: []string{"hourly", "monthly"}}}},
		"vms": schema.ListNestedAttribute{Computed: true, Description: "Virtual machines matching the filters.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true, Description: "Stable numeric VM ID."}, "hostname": schema.StringAttribute{Computed: true, Description: "VM hostname."}, "status": schema.StringAttribute{Computed: true, Description: "Current VM status."}, "plan": schema.StringAttribute{Computed: true, Description: "Plan slug."}, "region": schema.StringAttribute{Computed: true, Description: "Region slug."}, "image": schema.StringAttribute{Computed: true, Description: "Installed image slug."}, "ipv4": schema.StringAttribute{Computed: true, Description: "Public IPv4 address, if assigned."}, "ipv6": schema.StringAttribute{Computed: true, Description: "Public IPv6 address, if assigned."}, "billing_mode": schema.StringAttribute{Computed: true, Description: "Hourly or monthly billing mode."},
		}}},
	}}
}
func (d *VMAreaVMsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *VMAreaVMsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMAreaVMsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vms, err := d.client.VMAreaListVMs(ctx, data.Status.ValueString(), data.Region.ValueString(), data.Billing.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF VMs", err.Error())
		return
	}
	items := make([]VMAreaVMItemModel, 0, len(vms))
	for _, vm := range vms {
		items = append(items, VMAreaVMItemModel{ID: types.Int64Value(int64(vm.ID)), Hostname: types.StringValue(vm.Hostname), Status: types.StringValue(vm.Status), Plan: types.StringValue(vm.Plan), Region: types.StringValue(vm.Region), Image: types.StringValue(vm.Image), IPv4: nullableString(vm.IPv4), IPv6: nullableString(vm.VMAreaIPv6), BillingMode: types.StringValue(vm.Billing.Mode)})
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.Int64Type, "hostname": types.StringType, "status": types.StringType, "plan": types.StringType, "region": types.StringType, "image": types.StringType, "ipv4": types.StringType, "ipv6": types.StringType, "billing_mode": types.StringType}}, items)
	resp.Diagnostics.Append(diags...)
	data.VMs = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *VMAreaVMNetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_network"
}
func (d *VMAreaVMNetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Reads network interfaces and public addresses assigned to a virtual machine.", Attributes: map[string]schema.Attribute{
		"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine."}, "ipv6_prefix": schema.StringAttribute{Computed: true, Description: "Delegated IPv6 prefix, when available."}, "port_mbps": schema.Int64Attribute{Computed: true, Description: "Network port speed in Mbps."},
		"ipv4": vmAreaAddressListSchema(), "ipv6": vmAreaAddressListSchema(),
		"private": schema.ListNestedAttribute{Computed: true, Description: "Private networks (VPCs) attached to the VM.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"vpc": schema.Int64Attribute{Computed: true, Description: "VPC numeric ID."}, "name": schema.StringAttribute{Computed: true, Description: "VPC name."}, "address": schema.StringAttribute{Computed: true, Description: "Private IP address."}, "prefix_length": schema.Int64Attribute{Computed: true, Description: "Private address prefix length."}, "configuration": schema.StringAttribute{Computed: true, Description: "Address configuration mode."}}}},
	}}
}
func (d *VMAreaVMNetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *VMAreaVMNetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMAreaVMNetworkModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	res, err := d.client.VMAreaGetNetwork(ctx, data.VMID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF VM network", err.Error())
		return
	}
	data.IPv4 = vmAreaAddressList(ctx, res.Data.IPv4, resp)
	data.IPv6 = vmAreaAddressList(ctx, res.Data.IPv6, resp)
	data.IPv6Prefix = nullableString(res.Data.IPv6Prefix)
	data.PortMbps = types.Int64Value(int64(res.Data.PortMbps))
	private := make([]VMAreaPrivateNetworkModel, 0, len(res.Data.Private))
	for _, network := range res.Data.Private {
		private = append(private, VMAreaPrivateNetworkModel{VPC: types.Int64Value(network.VPC), Name: types.StringValue(network.Name), Address: types.StringValue(network.Address), PrefixLength: types.Int64Value(int64(network.PrefixLength)), Configuration: types.StringValue(network.Configuration)})
	}
	privateList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"vpc": types.Int64Type, "name": types.StringType, "address": types.StringType, "prefix_length": types.Int64Type, "configuration": types.StringType}}, private)
	resp.Diagnostics.Append(diags...)
	data.Private = privateList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func vmAreaAddressListSchema() schema.Attribute {
	return schema.ListNestedAttribute{Computed: true, Description: "IP addresses assigned to the VM.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"address": schema.StringAttribute{Computed: true, Description: "IP address."}, "version": schema.Int64Attribute{Computed: true, Description: "IP version."}, "primary": schema.BoolAttribute{Computed: true, Description: "Whether this is the primary address."}, "gateway": schema.StringAttribute{Computed: true, Description: "Address gateway, when provided."}, "prefix_length": schema.Int64Attribute{Computed: true, Description: "Address prefix length, when provided."}, "rdns": schema.StringAttribute{Computed: true, Description: "Reverse-DNS hostname, when configured."}}}}
}
func vmAreaAddressList(ctx context.Context, addresses []client.VMAreaAddress, resp *datasource.ReadResponse) types.List {
	items := make([]VMAreaAddressModel, 0, len(addresses))
	for _, a := range addresses {
		items = append(items, VMAreaAddressModel{Address: types.StringValue(a.Address), Version: types.Int64Value(int64(a.Version)), Primary: types.BoolValue(a.Primary), Gateway: vmAreaNullableString(a.Gateway), PrefixLength: vmAreaNullableInt(a.PrefixLength), RDNS: vmAreaNullableString(a.RDNS)})
	}
	v, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"address": types.StringType, "version": types.Int64Type, "primary": types.BoolType, "gateway": types.StringType, "prefix_length": types.Int64Type, "rdns": types.StringType}}, items)
	resp.Diagnostics.Append(d...)
	return v
}
func vmAreaNullableString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
func vmAreaNullableInt(i *int) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*i))
}

func (d *VMAreaVMMetricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_metrics"
}
func (d *VMAreaVMMetricsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Reads time-series CPU, memory, disk, and network metrics for a virtual machine.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine."}, "timeframe": schema.StringAttribute{Optional: true, Description: "Metrics range: hour, day, week, month, or year. Defaults to hour.", Validators: []validator.String{VMAreaStringOneOfValidator{Values: []string{"hour", "day", "week", "month", "year"}}}}, "step_seconds": schema.Int64Attribute{Computed: true, Description: "Time interval represented by each metric point."}, "series": schema.ListNestedAttribute{Computed: true, Description: "Time-series samples.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"time": schema.StringAttribute{Computed: true, Description: "Sample timestamp."}, "cpu": schema.Float64Attribute{Computed: true, Description: "Share of allotted vCPU, from 0 to 1."}, "memory_bytes": schema.Float64Attribute{Computed: true, Description: "Memory usage in bytes."}, "disk_read_bytes": schema.Float64Attribute{Computed: true, Description: "Disk read bytes."}, "disk_write_bytes": schema.Float64Attribute{Computed: true, Description: "Disk write bytes."}, "net_in_bytes": schema.Float64Attribute{Computed: true, Description: "Inbound network bytes."}, "net_out_bytes": schema.Float64Attribute{Computed: true, Description: "Outbound network bytes."}}}}}}
}
func (d *VMAreaVMMetricsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *VMAreaVMMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMAreaMetricsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeframe := data.Timeframe.ValueString()
	if timeframe == "" {
		timeframe = "hour"
	}
	m, err := d.client.VMAreaGetMetrics(ctx, data.VMID.ValueInt64(), timeframe)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF VM metrics", err.Error())
		return
	}
	data.Timeframe = types.StringValue(m.Data.Timeframe)
	data.StepSeconds = types.Int64Value(int64(m.Data.StepSeconds))
	items := make([]VMAreaMetricPointModel, 0, len(m.Data.Series))
	for _, p := range m.Data.Series {
		items = append(items, VMAreaMetricPointModel{Time: types.StringValue(p.Time), CPU: vmAreaNullableFloat(p.CPU), MemoryBytes: vmAreaNullableFloat(p.MemoryBytes), DiskReadBytes: vmAreaNullableFloat(p.DiskReadBytes), DiskWriteBytes: vmAreaNullableFloat(p.DiskWriteBytes), NetInBytes: vmAreaNullableFloat(p.NetInBytes), NetOutBytes: vmAreaNullableFloat(p.NetOutBytes)})
	}
	data.Series, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"time": types.StringType, "cpu": types.Float64Type, "memory_bytes": types.Float64Type, "disk_read_bytes": types.Float64Type, "disk_write_bytes": types.Float64Type, "net_in_bytes": types.Float64Type, "net_out_bytes": types.Float64Type}}, items)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func vmAreaNullableFloat(v *float64) types.Float64 {
	if v == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*v)
}

func (d *VMAreaVMBackupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_backups"
}
func (d *VMAreaVMBackupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists backups and automatic-backup settings for a virtual machine.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine."}, "automatic_enabled": schema.BoolAttribute{Computed: true, Description: "Whether automatic backups are enabled."}, "retention": schema.Int64Attribute{Computed: true, Description: "Number of retained automatic backups."}, "backups": schema.ListNestedAttribute{Computed: true, Description: "Available backups.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, Description: "Opaque backup identifier."}, "type": schema.StringAttribute{Computed: true, Description: "Backup type, automatic or manual."}, "created_at": schema.StringAttribute{Computed: true, Description: "Backup creation timestamp."}, "size_bytes": schema.Int64Attribute{Computed: true, Description: "Backup size in bytes, when known."}, "restorable": schema.BoolAttribute{Computed: true, Description: "Whether this backup can be restored."}}}}}}
}
func (d *VMAreaVMBackupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *VMAreaVMBackupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMAreaBackupsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := d.client.VMAreaGetBackups(ctx, data.VMID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF VM backups", err.Error())
		return
	}
	data.AutomaticEnabled = types.BoolValue(b.Meta.AutomaticEnabled)
	data.Retention = types.Int64Value(int64(b.Meta.Retention))
	items := make([]VMAreaBackupModel, 0, len(b.Data))
	for _, v := range b.Data {
		items = append(items, VMAreaBackupModel{ID: types.StringValue(v.ID), Type: types.StringValue(v.Type), CreatedAt: types.StringValue(v.CreatedAt), SizeBytes: vmAreaNullableInt64(v.SizeBytes), Restorable: types.BoolValue(v.Restorable)})
	}
	data.Backups, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.StringType, "type": types.StringType, "created_at": types.StringType, "size_bytes": types.Int64Type, "restorable": types.BoolType}}, items)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func vmAreaNullableInt64(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func (d *VMAreaVMActionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_actions"
}
func (d *VMAreaVMActionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists account-wide actions or actions for one virtual machine, with optional account-wide status filtering.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Optional: true, Description: "Limit results to a virtual machine. Omit to list account actions."}, "status": schema.StringAttribute{Optional: true, Description: "Filter account-wide actions by status when vm_id is omitted."}, "actions": schema.ListNestedAttribute{Computed: true, Description: "Matching VM actions.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"id": schema.Int64Attribute{Computed: true, Description: "Action ID."}, "type": schema.StringAttribute{Computed: true, Description: "Action type."}, "status": schema.StringAttribute{Computed: true, Description: "Action status."}, "vm_id": schema.Int64Attribute{Computed: true, Description: "VM ID associated with the action, when known."}, "source": schema.StringAttribute{Computed: true, Description: "Action origin, API or console."}, "created_at": schema.StringAttribute{Computed: true, Description: "Action creation timestamp."}, "started_at": schema.StringAttribute{Computed: true, Description: "Action start timestamp, when available."}, "finished_at": schema.StringAttribute{Computed: true, Description: "Action completion timestamp, when available."}, "error": schema.StringAttribute{Computed: true, Description: "Action failure message, when available."}}}}}}
}
func (d *VMAreaVMActionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *VMAreaVMActionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VMAreaActionsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var vmID *int64
	if !data.VMID.IsNull() && !data.VMID.IsUnknown() {
		if !data.Status.IsNull() && !data.Status.IsUnknown() && data.Status.ValueString() != "" {
			resp.Diagnostics.AddError("Invalid VM actions filters", "status can only filter account-wide actions when vm_id is omitted.")
			return
		}
		v := data.VMID.ValueInt64()
		vmID = &v
	}
	actions, err := d.client.VMAreaListActions(ctx, vmID, data.Status.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF VM actions", err.Error())
		return
	}
	items := make([]VMAreaActionModel, 0, len(actions))
	for _, a := range actions {
		vm := types.Int64Null()
		if a.VMID != nil {
			vm = types.Int64Value(*a.VMID)
		}
		e := types.StringNull()
		if a.Error != nil {
			e = types.StringValue(a.Error.Message)
		}
		items = append(items, VMAreaActionModel{ID: types.Int64Value(int64(a.ID)), Type: types.StringValue(a.Type), Status: types.StringValue(a.Status), VMID: vm, Source: types.StringValue(a.Source), CreatedAt: types.StringValue(a.CreatedAt), StartedAt: vmAreaNullableString(a.StartedAt), FinishedAt: vmAreaNullableString(a.FinishedAt), Error: e})
	}
	data.Actions, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.Int64Type, "type": types.StringType, "status": types.StringType, "vm_id": types.Int64Type, "source": types.StringType, "created_at": types.StringType, "started_at": types.StringType, "finished_at": types.StringType, "error": types.StringType}}, items)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

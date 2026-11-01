package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VMAreaIPv6AddressResource{}
var _ resource.ResourceWithImportState = &VMAreaIPv6AddressResource{}
var _ resource.Resource = &VMAreaRDNSResource{}
var _ resource.Resource = &VMAreaPowerStateResource{}
var _ resource.Resource = &VMAreaReinstallResource{}
var _ resource.Resource = &VMAreaBackupRestoreResource{}

type VMAreaIPv6AddressResource struct{ client *client.Client }
type VMAreaRDNSResource struct{ client *client.Client }
type VMAreaPowerStateResource struct{ client *client.Client }
type VMAreaReinstallResource struct{ client *client.Client }
type VMAreaBackupRestoreResource struct{ client *client.Client }
type VMAreaIPv6AddressModel struct {
	ID      types.String `tfsdk:"id"`
	VMID    types.Int64  `tfsdk:"vm_id"`
	Address types.String `tfsdk:"address"`
}
type VMAreaRDNSModel struct {
	VMID     types.Int64  `tfsdk:"vm_id"`
	Address  types.String `tfsdk:"address"`
	Hostname types.String `tfsdk:"hostname"`
}
type VMAreaPowerStateModel struct {
	VMID          types.Int64  `tfsdk:"vm_id"`
	State         types.String `tfsdk:"state"`
	Method        types.String `tfsdk:"method"`
	RebootTrigger types.String `tfsdk:"reboot_trigger"`
}
type VMAreaReinstallModel struct {
	VMID    types.Int64  `tfsdk:"vm_id"`
	Image   types.String `tfsdk:"image"`
	SSHKeys types.Set    `tfsdk:"ssh_keys"`
}
type VMAreaBackupRestoreModel struct {
	VMID   types.Int64  `tfsdk:"vm_id"`
	Backup types.String `tfsdk:"backup"`
}

func NewVMAreaIPv6AddressResource() resource.Resource   { return &VMAreaIPv6AddressResource{} }
func NewVMAreaRDNSResource() resource.Resource          { return &VMAreaRDNSResource{} }
func NewVMAreaPowerStateResource() resource.Resource    { return &VMAreaPowerStateResource{} }
func NewVMAreaReinstallResource() resource.Resource     { return &VMAreaReinstallResource{} }
func NewVMAreaBackupRestoreResource() resource.Resource { return &VMAreaBackupRestoreResource{} }

func (r *VMAreaIPv6AddressResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_ipv6_address"
}
func (r *VMAreaIPv6AddressResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Assigns and manages an additional IPv6 address on a virtual machine.", Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, Description: "Import identifier in vm_id/address form."}, "vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "address": schema.StringAttribute{Required: true, Description: "IPv6 address inside the VM's assigned prefix.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}}}
}
func (r *VMAreaIPv6AddressResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *VMAreaIPv6AddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m VMAreaIPv6AddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.VMAreaAddIPv6(ctx, m.VMID.ValueInt64(), m.Address.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to assign FFXF VM IPv6 address", err.Error())
		return
	}
	m.ID = types.StringValue(fmt.Sprintf("%d/%s", m.VMID.ValueInt64(), m.Address.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaIPv6AddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m VMAreaIPv6AddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	n, err := r.client.VMAreaGetNetwork(ctx, m.VMID.ValueInt64())
	if err != nil {
		if err == client.ErrNetworkResourceNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read FFXF VM IPv6 address", err.Error())
		return
	}
	for _, a := range n.Data.IPv6 {
		if a.Address == m.Address.ValueString() {
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *VMAreaIPv6AddressResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported IPv6 address update", "Changing the VM or IPv6 address requires replacement.")
}
func (r *VMAreaIPv6AddressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m VMAreaIPv6AddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.VMAreaDeleteIPv6(ctx, m.VMID.ValueInt64(), m.Address.ValueString())
	if err != nil && err != client.ErrNetworkResourceNotFound {
		resp.Diagnostics.AddError("Unable to remove FFXF VM IPv6 address", err.Error())
	}
}
func (r *VMAreaIPv6AddressResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid IPv6 address import ID", "Expected an import ID in vm_id/address form.")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id < 1 {
		resp.Diagnostics.AddError("Invalid IPv6 address import ID", "The vm_id portion must be a positive integer.")
		return
	}
	m := VMAreaIPv6AddressModel{ID: types.StringValue(req.ID), VMID: types.Int64Value(id), Address: types.StringValue(parts[1])}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *VMAreaRDNSResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_rdns"
}
func (r *VMAreaRDNSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Configures reverse DNS for an IP address assigned to a virtual machine. Destroying the resource clears the custom record and restores the default.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "address": schema.StringAttribute{Required: true, Description: "IP address assigned to the VM.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "hostname": schema.StringAttribute{Required: true, Description: "Reverse-DNS hostname to associate with the address."}}}
}
func (r *VMAreaRDNSResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *VMAreaRDNSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m VMAreaRDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	h := m.Hostname.ValueString()
	if _, err := r.client.VMAreaUpdateRDNS(ctx, m.VMID.ValueInt64(), m.Address.ValueString(), &h); err != nil {
		resp.Diagnostics.AddError("Unable to configure FFXF VM reverse DNS", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaRDNSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m VMAreaRDNSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	n, err := r.client.VMAreaGetNetwork(ctx, m.VMID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF VM reverse DNS", err.Error())
		return
	}
	for _, a := range append(n.Data.IPv4, n.Data.IPv6...) {
		if a.Address == m.Address.ValueString() {
			m.Hostname = vmAreaNullableString(a.RDNS)
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *VMAreaRDNSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m VMAreaRDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	h := m.Hostname.ValueString()
	if _, err := r.client.VMAreaUpdateRDNS(ctx, m.VMID.ValueInt64(), m.Address.ValueString(), &h); err != nil {
		resp.Diagnostics.AddError("Unable to update FFXF VM reverse DNS", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaRDNSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m VMAreaRDNSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.VMAreaUpdateRDNS(ctx, m.VMID.ValueInt64(), m.Address.ValueString(), nil); err != nil {
		resp.Diagnostics.AddError("Unable to restore default FFXF VM reverse DNS", err.Error())
	}
}

func (r *VMAreaPowerStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_power_state"
}
func (r *VMAreaPowerStateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manages the desired running or stopped state of a virtual machine. Removing this resource does not change VM power state.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "state": schema.StringAttribute{Required: true, Description: "Desired VM state: running or stopped.", Validators: []validator.String{VMAreaStringOneOfValidator{Values: []string{"running", "stopped"}}}}, "method": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("graceful"), Description: "Power operation method: graceful (default) or forced. Graceful shutdown uses shutdown; forced stop uses stop.", Validators: []validator.String{VMAreaStringOneOfValidator{Values: []string{"graceful", "forced"}}}}, "reboot_trigger": schema.StringAttribute{Optional: true, Description: "Changing this value triggers a reboot action."}}}
}
func (r *VMAreaPowerStateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *VMAreaPowerStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m VMAreaPowerStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := vmAreaApplyPower(ctx, r.client, &m, false); err != nil {
		resp.Diagnostics.AddError("Unable to set FFXF VM power state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaPowerStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m VMAreaPowerStateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vm, err := r.client.GetVM(ctx, m.VMID.ValueInt64())
	if err == client.ErrVMNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF VM power state", err.Error())
		return
	}
	if vm.Data.Status == "running" || vm.Data.Status == "stopped" {
		m.State = types.StringValue(vm.Data.Status)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaPowerStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p, s VMAreaPowerStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	reboot := !p.RebootTrigger.Equal(s.RebootTrigger)
	if err := vmAreaApplyPower(ctx, r.client, &p, reboot); err != nil {
		resp.Diagnostics.AddError("Unable to update FFXF VM power state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *VMAreaPowerStateResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}
func vmAreaApplyPower(ctx context.Context, c *client.Client, m *VMAreaPowerStateModel, reboot bool) error {
	action := ""
	if reboot {
		action = "reboot"
	} else if m.State.ValueString() == "running" {
		action = "start"
	} else if m.Method.ValueString() == "forced" {
		action = "stop"
	} else {
		action = "shutdown"
	}
	res, err := c.VMAreaPowerAction(ctx, m.VMID.ValueInt64(), action)
	if err != nil {
		return err
	}
	if id := res.Data.Action.ID; id != 0 {
		return c.WaitForAction(ctx, id)
	}
	return nil
}

func (r *VMAreaReinstallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_reinstall"
}
func (r *VMAreaReinstallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Reinstalls a virtual machine with a new image. WARNING: reinstalling wipes the VM disk and replaces all its data.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "image": schema.StringAttribute{Required: true, Description: "Image slug to install; the existing disk is wiped.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "ssh_keys": schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: "Account SSH key fingerprints or IDs to install.", PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()}}}}
}
func (r *VMAreaReinstallResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *VMAreaReinstallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m VMAreaReinstallModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var keys []string
	if !m.SSHKeys.IsNull() {
		resp.Diagnostics.Append(m.SSHKeys.ElementsAs(ctx, &keys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	action, err := r.client.VMAreaReinstall(ctx, m.VMID.ValueInt64(), m.Image.ValueString(), keys)
	if err != nil {
		resp.Diagnostics.AddError("Unable to reinstall FFXF VM", err.Error())
		return
	}
	if action.Data.Action.ID != 0 {
		if err := r.client.WaitForAction(ctx, action.Data.Action.ID); err != nil {
			resp.Diagnostics.AddError("Unable to wait for FFXF VM reinstall", err.Error())
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaReinstallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m VMAreaReinstallModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}
func (r *VMAreaReinstallResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported VM reinstall update", "Changing reinstall inputs requires replacement.")
}
func (r *VMAreaReinstallResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *VMAreaBackupRestoreResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_backup_restore"
}
func (r *VMAreaBackupRestoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Restores a virtual machine from a backup. WARNING: restoring overwrites the VM disk and replaces its current data.", Attributes: map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Required: true, Description: "Numeric ID of the virtual machine.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "backup": schema.StringAttribute{Required: true, Description: "Opaque backup identifier to restore; the current VM disk is overwritten.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}}}
}
func (r *VMAreaBackupRestoreResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *VMAreaBackupRestoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m VMAreaBackupRestoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	action, err := r.client.VMAreaRestoreBackup(ctx, m.VMID.ValueInt64(), m.Backup.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to restore FFXF VM backup", err.Error())
		return
	}
	if action.Data.Action.ID != 0 {
		if err := r.client.WaitForAction(ctx, action.Data.Action.ID); err != nil {
			resp.Diagnostics.AddError("Unable to wait for FFXF VM backup restore", err.Error())
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *VMAreaBackupRestoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m VMAreaBackupRestoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}
func (r *VMAreaBackupRestoreResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported VM backup restore update", "Changing restore inputs requires replacement.")
}
func (r *VMAreaBackupRestoreResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

type VMAreaStringOneOfValidator struct{ Values []string }

func (v VMAreaStringOneOfValidator) Description(context.Context) string {
	return "value must be one of " + strings.Join(v.Values, ", ")
}
func (v VMAreaStringOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v VMAreaStringOneOfValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, value := range v.Values {
		if req.ConfigValue.ValueString() == value {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", v.Description(context.Background()))
}

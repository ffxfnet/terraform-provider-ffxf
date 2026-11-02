package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*AccountAreaSSHKeyResource)(nil)

type AccountAreaSSHKeyResource struct{ client *client.Client }
type AccountAreaVPCMemberResource struct{ client *client.Client }

func NewAccountAreaSSHKeyResource() resource.Resource    { return &AccountAreaSSHKeyResource{} }
func NewAccountAreaVPCMemberResource() resource.Resource { return &AccountAreaVPCMemberResource{} }

type AccountAreaSSHKeyResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	CreatedAt   types.String `tfsdk:"created_at"`
}
type AccountAreaVPCMemberResourceModel struct {
	ID            types.String `tfsdk:"id"`
	VPCID         types.Int64  `tfsdk:"vpc_id"`
	VMID          types.Int64  `tfsdk:"vm_id"`
	Hostname      types.String `tfsdk:"hostname"`
	IP            types.String `tfsdk:"ip"`
	Configuration types.String `tfsdk:"configuration"`
}

func (r *AccountAreaSSHKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}
func (r *AccountAreaSSHKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *AccountAreaSSHKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Registers and manages an SSH public key. Key name and public key changes replace the registered key.", Attributes: map[string]schema.Attribute{
		"id": schema.Int64Attribute{Computed: true, Description: "Numeric SSH-key identifier."}, "name": schema.StringAttribute{Required: true, Description: "Display name for this SSH key.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "public_key": schema.StringAttribute{Required: true, Description: "SSH public-key material. This is a public key, not a private secret.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "fingerprint": schema.StringAttribute{Computed: true, Description: "Fingerprint assigned to the key."}, "created_at": schema.StringAttribute{Computed: true, Description: "UTC time when the key was registered."},
	}}
}
func (r *AccountAreaSSHKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountAreaSSHKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key, err := r.client.AccountAreaCreateSSHKey(ctx, plan.Name.ValueString(), plan.PublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create FFXF SSH key", err.Error())
		return
	}
	accountAreaSetSSHKeyResource(&plan, key.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *AccountAreaSSHKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccountAreaSSHKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	keys, err := r.client.AccountAreaListSSHKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF SSH key", err.Error())
		return
	}
	for _, key := range keys.Data {
		if int64(key.ID) == state.ID.ValueInt64() {
			accountAreaSetSSHKeyResource(&state, key)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *AccountAreaSSHKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported SSH key update", "SSH key changes require replacement.")
}
func (r *AccountAreaSSHKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountAreaSSHKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.AccountAreaDeleteSSHKey(ctx, state.ID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Unable to delete FFXF SSH key", err.Error())
	}
}
func accountAreaSetSSHKeyResource(state *AccountAreaSSHKeyResourceModel, key client.AccountAreaSSHKey) {
	state.ID = types.Int64Value(int64(key.ID))
	state.Name = types.StringValue(key.Name)
	state.PublicKey = types.StringValue(key.PublicKey)
	state.Fingerprint = types.StringValue(key.Fingerprint)
	state.CreatedAt = types.StringValue(key.CreatedAt)
}

func (r *AccountAreaVPCMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_member"
}
func (r *AccountAreaVPCMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *AccountAreaVPCMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Connects a virtual machine to a private network. Changing vpc_id moves the member to the target network in place.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true, Description: "Stable identifier composed of the VPC and VM identifiers."}, "vpc_id": schema.Int64Attribute{Required: true, Description: "Private network identifier."}, "vm_id": schema.Int64Attribute{Required: true, Description: "Virtual machine identifier.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "hostname": schema.StringAttribute{Computed: true, Description: "Virtual machine hostname."}, "ip": schema.StringAttribute{Computed: true, Description: "Private IP address assigned by the network."}, "configuration": schema.StringAttribute{Computed: true, Description: "Network configuration mode, automatic or manual."},
	}}
}
func (r *AccountAreaVPCMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountAreaVPCMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	member, err := r.client.AccountAreaCreateVPCMember(ctx, plan.VPCID.ValueInt64(), plan.VMID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to add FFXF VPC member", err.Error())
		return
	}
	accountAreaSetVPCMemberResource(&plan, member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *AccountAreaVPCMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccountAreaVPCMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	vpc, err := r.client.AccountAreaGetVPC(ctx, state.VPCID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF VPC member", err.Error())
		return
	}
	for _, member := range vpc.Data.Members {
		if int64(member.VM) == state.VMID.ValueInt64() {
			accountAreaSetVPCMemberResource(&state, &member)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *AccountAreaVPCMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccountAreaVPCMemberResourceModel
	var state AccountAreaVPCMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	member, err := r.client.AccountAreaMoveVPCMember(ctx, state.VPCID.ValueInt64(), state.VMID.ValueInt64(), plan.VPCID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to move FFXF VPC member", err.Error())
		return
	}
	accountAreaSetVPCMemberResource(&plan, member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *AccountAreaVPCMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountAreaVPCMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.AccountAreaDeleteVPCMember(ctx, state.VPCID.ValueInt64(), state.VMID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Unable to remove FFXF VPC member", err.Error())
	}
}
func accountAreaSetVPCMemberResource(state *AccountAreaVPCMemberResourceModel, member *client.AccountAreaVPCMember) {
	state.ID = types.StringValue(strconv.FormatInt(state.VPCID.ValueInt64(), 10) + "-" + strconv.FormatInt(state.VMID.ValueInt64(), 10))
	state.Hostname = types.StringValue(member.Hostname)
	state.IP = types.StringValue(member.IP)
	state.Configuration = types.StringValue(member.Configuration)
}

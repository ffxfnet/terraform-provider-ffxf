package provider

import (
	"context"
	"errors"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FirewallMemberResource{}

type FirewallMemberResource struct{ client *client.Client }

type FirewallMemberResourceModel struct {
	FirewallID types.Int64  `tfsdk:"firewall_id"`
	VMID       types.Int64  `tfsdk:"vm_id"`
	Hostname   types.String `tfsdk:"hostname"`
	Status     types.String `tfsdk:"status"`
}

func NewFirewallMemberResource() resource.Resource { return &FirewallMemberResource{} }

func (r *FirewallMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_member"
}

func (r *FirewallMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches one firewall to one virtual machine. Up to five firewalls can be attached to a machine.",
		Attributes: map[string]schema.Attribute{
			"firewall_id": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"vm_id":       schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"hostname":    schema.StringAttribute{Computed: true},
			"status":      schema.StringAttribute{Computed: true, Description: "Application status: applied, pending, or error."},
		},
	}
}

func (r *FirewallMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}

func (r *FirewallMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	member, err := r.client.AttachFirewall(ctx, plan.FirewallID.ValueInt64(), plan.VMID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error attaching FFXF firewall", err.Error())
		return
	}
	setFirewallMemberModel(&plan, *member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	firewall, err := r.client.GetFirewall(ctx, state.FirewallID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF firewall attachment", err.Error())
		return
	}
	for _, member := range firewall.Data.Members {
		if member.VM == state.VMID.ValueInt64() {
			setFirewallMemberModel(&state, member)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *FirewallMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported firewall attachment update", "Changing the firewall or VM requires replacement.")
}

func (r *FirewallMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DetachFirewall(ctx, state.FirewallID.ValueInt64(), state.VMID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error detaching FFXF firewall", err.Error())
	}
}

func setFirewallMemberModel(state *FirewallMemberResourceModel, member client.FirewallMember) {
	state.Hostname = types.StringValue(member.Hostname)
	state.Status = types.StringValue(member.Status)
}

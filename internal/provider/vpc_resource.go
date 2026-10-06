package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

var _ resource.Resource = &VPCResource{}

type VPCResource struct {
	client *client.Client
}

func NewVPCResource() resource.Resource {
	return &VPCResource{}
}

type VPCResourceModel struct {
	ID              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	CIDR            types.String `tfsdk:"cidr"`
	Region          types.String `tfsdk:"region"`
	Status          types.String `tfsdk:"status"`
	Gateway         types.String `tfsdk:"gateway"`
	InternetGateway types.Bool   `tfsdk:"internet_gateway"`
	CreatedAt       types.String `tfsdk:"created_at"`
}

func (r *VPCResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (r *VPCResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a private IPv4 network in FFXF Cloud. The network is ready when creation completes. When the network quota is reached, the API error includes details.increase_url for the console quota request flow. Changing the name, CIDR, or region replaces the network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Numeric ID assigned to the private network by FFXF Cloud.",
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Name of the private network.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cidr": schema.StringAttribute{
				Required:      true,
				Description:   "Private IPv4 range from /16 to /29, within 10.0.0.0/8, 172.16.0.0/12, or 192.168.0.0/16.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region": schema.StringAttribute{
				Required:      true,
				Description:   "Region where the private network will run.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current network status, such as active.",
			},
			"gateway": schema.StringAttribute{
				Computed:    true,
				Description: "Reserved first address of the network range.",
			},
			"internet_gateway": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the network gateway provides Internet access to machines without a public address.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "UTC timestamp when the network was created.",
			},
		},
	}
}

func (r *VPCResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *VPCResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPCResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpc, err := r.client.CreateVPC(ctx, client.CreateVPCRequest{
		Name:   plan.Name.ValueString(),
		CIDR:   plan.CIDR.ValueString(),
		Region: plan.Region.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF VPC", err.Error())
		return
	}

	setVPCModelFromResponse(&plan, vpc.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VPCResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPCResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpc, err := r.client.GetVPC(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrVPCNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF VPC", err.Error())
		return
	}

	setVPCModelFromResponse(&state, vpc.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VPCResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported VPC update", "FFXF Cloud does not support updating private network attributes. Changes to name, CIDR, or region require replacement.")
}

func (r *VPCResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPCResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteVPC(ctx, state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting FFXF VPC", err.Error())
	}
}

func setVPCModelFromResponse(state *VPCResourceModel, vpc client.VPC) {
	state.ID = types.Int64Value(int64(vpc.ID))
	state.Name = types.StringValue(vpc.Name)
	state.CIDR = types.StringValue(vpc.CIDR)
	state.Region = types.StringValue(vpc.Region)
	state.Status = types.StringValue(vpc.Status)
	state.Gateway = types.StringValue(vpc.Gateway)
	state.InternetGateway = types.BoolValue(vpc.InternetGateway)
	state.CreatedAt = types.StringValue(vpc.CreatedAt)
}

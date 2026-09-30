package provider

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

var _ resource.Resource = &InstanceResource{}

type InstanceResource struct {
	client *client.Client
}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	Plan     types.String `tfsdk:"plan"`
	Region   types.String `tfsdk:"region"`
	Image    types.String `tfsdk:"image"`
	Billing  types.String `tfsdk:"billing"`
	IPv4     types.String `tfsdk:"ipv4"`
}

func (r *InstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an FFXF Cloud virtual machine.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"plan": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"image": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"billing": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ipv4": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *InstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	billingMode := "hourly"
	if !plan.Billing.IsNull() {
		billingMode = plan.Billing.ValueString()
	}

	// 1. Appel HTTP de création
	idempotencyKey := uuid.New().String()
	res, err := r.client.CreateVM(ctx, client.CreateVmRequest{
		Plan:     plan.Plan.ValueString(),
		Region:   plan.Region.ValueString(),
		Image:    plan.Image.ValueString(),
		Hostname: plan.Hostname.ValueString(),
		Billing:  billingMode,
	}, idempotencyKey)

	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF VM", err.Error())
		return
	}

	// 2. Polling de l'action de déploiement
	if res.Data.Action.ID != 0 {
		err = r.client.WaitForAction(ctx, res.Data.Action.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error deploying VM", err.Error())
			return
		}
	}

	// 3. Mise à jour du state OpenTofu
	plan.ID = types.Int64Value(int64(res.Data.VM.ID))
	plan.IPv4 = types.StringValue(res.Data.VM.IPv4)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.client.GetVM(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrVMNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF VM", err.Error())
		return
	}

	setInstanceModelFromVM(&state, vm.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.client.UpdateVM(ctx, state.ID.ValueInt64(), plan.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF VM", err.Error())
		return
	}

	setInstanceModelFromVM(&state, vm.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func setInstanceModelFromVM(state *InstanceResourceModel, vm client.VM) {
	state.ID = types.Int64Value(int64(vm.ID))
	state.Hostname = types.StringValue(vm.Hostname)
	state.Plan = types.StringValue(vm.Plan)
	state.Region = types.StringValue(vm.Region)
	state.Image = types.StringValue(vm.Image)
	if vm.IPv4 == nil {
		state.IPv4 = types.StringNull()
	} else {
		state.IPv4 = types.StringValue(*vm.IPv4)
	}
	if !state.Billing.IsNull() && vm.Billing.Mode != "" {
		state.Billing = types.StringValue(vm.Billing.Mode)
	}
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteVM(ctx, int(state.ID.ValueInt64()), state.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting VM", err.Error())
	}
}

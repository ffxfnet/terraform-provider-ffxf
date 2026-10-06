package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LoadBalancerPoolResource{}

type LoadBalancerPoolResource struct{ client *client.Client }

type LoadBalancerPoolResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	LoadBalancerID types.Int64  `tfsdk:"load_balancer_id"`
	Name           types.String `tfsdk:"name"`
	Protocol       types.String `tfsdk:"protocol"`
	Algorithm      types.String `tfsdk:"algorithm"`
	HealthType     types.String `tfsdk:"health_type"`
	HealthPath     types.String `tfsdk:"health_path"`
	HealthInterval types.Int64  `tfsdk:"health_interval"`
	SendProxy      types.Bool   `tfsdk:"send_proxy"`
	Targets        types.List   `tfsdk:"targets"`
}

type LoadBalancerTargetModel struct {
	ID         types.Int64  `tfsdk:"id"`
	VM         types.Int64  `tfsdk:"vm"`
	Hostname   types.String `tfsdk:"hostname"`
	Address    types.String `tfsdk:"address"`
	Port       types.Int64  `tfsdk:"port"`
	Weight     types.Int64  `tfsdk:"weight"`
	Drain      types.Bool   `tfsdk:"drain"`
	Health     types.String `tfsdk:"health"`
	Sessions   types.Int64  `tfsdk:"sessions"`
	ResponseMS types.Int64  `tfsdk:"response_ms"`
}

func NewLoadBalancerPoolResource() resource.Resource { return &LoadBalancerPoolResource{} }

func (r *LoadBalancerPoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer_pool"
}

func (r *LoadBalancerPoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a load-balancer backend pool and manages its complete set of VM targets.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.Int64Attribute{Computed: true},
			"load_balancer_id": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"name":             schema.StringAttribute{Required: true},
			"protocol":         schema.StringAttribute{Optional: true, Computed: true, Description: "http or tcp."},
			"algorithm":        schema.StringAttribute{Optional: true, Computed: true, Description: "roundrobin or leastconn."},
			"health_type":      schema.StringAttribute{Optional: true, Computed: true, Description: "tcp or http health checks."},
			"health_path":      schema.StringAttribute{Optional: true, Computed: true},
			"health_interval":  schema.Int64Attribute{Optional: true, Computed: true, Description: "Health-check interval, 2 to 60 seconds."},
			"send_proxy":       schema.BoolAttribute{Optional: true, Computed: true, Description: "Send PROXY protocol v2 to targets (TCP pools only)."},
			"targets": schema.ListNestedAttribute{
				Optional: true, Computed: true,
				Description: "Complete target set. Updating this list replaces all targets in the pool; each VM must belong to the load balancer's VPC.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":          schema.Int64Attribute{Computed: true},
					"vm":          schema.Int64Attribute{Required: true},
					"hostname":    schema.StringAttribute{Computed: true},
					"address":     schema.StringAttribute{Computed: true},
					"port":        schema.Int64Attribute{Required: true},
					"weight":      schema.Int64Attribute{Optional: true, Computed: true},
					"drain":       schema.BoolAttribute{Optional: true, Computed: true},
					"health":      schema.StringAttribute{Computed: true},
					"sessions":    schema.Int64Attribute{Computed: true},
					"response_ms": schema.Int64Attribute{Computed: true},
				}},
			},
		},
	}
}

func (r *LoadBalancerPoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}

func (r *LoadBalancerPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LoadBalancerPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pool, err := r.client.CreateLoadBalancerPool(ctx, plan.LoadBalancerID.ValueInt64(), poolInputMap(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF load-balancer pool", err.Error())
		return
	}
	if !plan.Targets.IsNull() && !plan.Targets.IsUnknown() {
		targets, ok := targetsFromModel(ctx, plan.Targets, &resp.Diagnostics)
		if !ok {
			return
		}
		pool, err = r.client.ReplaceLoadBalancerTargets(ctx, plan.LoadBalancerID.ValueInt64(), pool.ID, targets)
		if err != nil {
			resp.Diagnostics.AddError("Error setting FFXF load-balancer targets", err.Error())
			return
		}
	}
	setPoolModel(ctx, &plan, *pool, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoadBalancerPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LoadBalancerPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lb, err := r.client.GetLoadBalancer(ctx, state.LoadBalancerID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF load-balancer pool", err.Error())
		return
	}
	for _, pool := range lb.Data.Pools {
		if pool.ID == state.ID.ValueInt64() {
			setPoolModel(ctx, &state, pool, &resp.Diagnostics)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *LoadBalancerPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LoadBalancerPoolResourceModel
	var state LoadBalancerPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pool, err := r.client.UpdateLoadBalancerPool(ctx, state.LoadBalancerID.ValueInt64(), state.ID.ValueInt64(), poolInputMap(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF load-balancer pool", err.Error())
		return
	}
	if !plan.Targets.IsNull() && !plan.Targets.IsUnknown() {
		targets, ok := targetsFromModel(ctx, plan.Targets, &resp.Diagnostics)
		if !ok {
			return
		}
		pool, err = r.client.ReplaceLoadBalancerTargets(ctx, state.LoadBalancerID.ValueInt64(), state.ID.ValueInt64(), targets)
		if err != nil {
			resp.Diagnostics.AddError("Error replacing FFXF load-balancer targets", err.Error())
			return
		}
	}
	setPoolModel(ctx, &plan, *pool, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoadBalancerPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LoadBalancerPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteLoadBalancerPool(ctx, state.LoadBalancerID.ValueInt64(), state.ID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF load-balancer pool", err.Error())
	}
}

func poolInputMap(model LoadBalancerPoolResourceModel) map[string]any {
	input := map[string]any{"name": model.Name.ValueString()}
	if !model.Protocol.IsNull() && !model.Protocol.IsUnknown() {
		input["protocol"] = model.Protocol.ValueString()
	}
	if !model.Algorithm.IsNull() && !model.Algorithm.IsUnknown() {
		input["algorithm"] = model.Algorithm.ValueString()
	}
	if !model.HealthType.IsNull() && !model.HealthType.IsUnknown() {
		input["health_type"] = model.HealthType.ValueString()
	}
	if !model.HealthPath.IsUnknown() {
		if model.HealthPath.IsNull() {
			input["health_path"] = nil
		} else {
			input["health_path"] = model.HealthPath.ValueString()
		}
	}
	if !model.HealthInterval.IsNull() && !model.HealthInterval.IsUnknown() {
		input["health_interval"] = model.HealthInterval.ValueInt64()
	}
	if !model.SendProxy.IsNull() && !model.SendProxy.IsUnknown() {
		input["send_proxy"] = model.SendProxy.ValueBool()
	}
	return input
}

func targetsFromModel(ctx context.Context, values types.List, diagnostics *diag.Diagnostics) ([]client.LoadBalancerTarget, bool) {
	var models []LoadBalancerTargetModel
	diagnostics.Append(values.ElementsAs(ctx, &models, false)...)
	if diagnostics.HasError() {
		return nil, false
	}
	targets := make([]client.LoadBalancerTarget, 0, len(models))
	for _, model := range models {
		weight := int64(1)
		if !model.Weight.IsNull() && !model.Weight.IsUnknown() {
			weight = model.Weight.ValueInt64()
		}
		drain := false
		if !model.Drain.IsNull() && !model.Drain.IsUnknown() {
			drain = model.Drain.ValueBool()
		}
		targets = append(targets, client.LoadBalancerTarget{VM: model.VM.ValueInt64(), Port: int(model.Port.ValueInt64()), Weight: int(weight), Drain: drain})
	}
	return targets, true
}

func setPoolModel(ctx context.Context, state *LoadBalancerPoolResourceModel, pool client.LoadBalancerPool, diagnostics *diag.Diagnostics) {
	state.ID = types.Int64Value(pool.ID)
	state.Name = types.StringValue(pool.Name)
	state.Protocol = types.StringValue(pool.Protocol)
	state.Algorithm = types.StringValue(pool.Algorithm)
	state.HealthType = types.StringValue(pool.HealthType)
	state.HealthPath = nullableString(pool.HealthPath)
	state.HealthInterval = types.Int64Value(int64(pool.HealthInterval))
	state.SendProxy = types.BoolValue(pool.SendProxy)
	models := make([]LoadBalancerTargetModel, 0, len(pool.Targets))
	for _, target := range pool.Targets {
		responseMS := types.Int64Null()
		if target.ResponseMS != nil {
			responseMS = types.Int64Value(int64(*target.ResponseMS))
		}
		models = append(models, LoadBalancerTargetModel{
			ID: types.Int64Value(target.ID), VM: types.Int64Value(target.VM), Hostname: types.StringValue(target.Hostname), Address: types.StringValue(target.Address),
			Port: types.Int64Value(int64(target.Port)), Weight: types.Int64Value(int64(target.Weight)), Drain: types.BoolValue(target.Drain), Health: types.StringValue(target.Health),
			Sessions: types.Int64Value(target.Sessions), ResponseMS: responseMS,
		})
	}
	attrTypes := map[string]attr.Type{"id": types.Int64Type, "vm": types.Int64Type, "hostname": types.StringType, "address": types.StringType, "port": types.Int64Type, "weight": types.Int64Type, "drain": types.BoolType, "health": types.StringType, "sessions": types.Int64Type, "response_ms": types.Int64Type}
	targets, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, models)
	diagnostics.Append(diags...)
	state.Targets = targets
}

var _ = strconv.FormatInt

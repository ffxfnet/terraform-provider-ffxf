package provider

import (
	"context"
	"errors"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LoadBalancerListenerResource{}

type LoadBalancerListenerResource struct{ client *client.Client }

type LoadBalancerListenerResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	LoadBalancerID types.Int64  `tfsdk:"load_balancer_id"`
	Port           types.Int64  `tfsdk:"port"`
	Protocol       types.String `tfsdk:"protocol"`
	DefaultPool    types.Int64  `tfsdk:"default_pool"`
	RedirectHTTPS  types.Bool   `tfsdk:"redirect_https"`
	Sources        types.List   `tfsdk:"sources"`
	Rules          types.List   `tfsdk:"rules"`
}

type LoadBalancerRuleModel struct {
	Hostname   types.String `tfsdk:"hostname"`
	PathPrefix types.String `tfsdk:"path_prefix"`
	Pool       types.Int64  `tfsdk:"pool"`
}

func NewLoadBalancerListenerResource() resource.Resource { return &LoadBalancerListenerResource{} }

func (r *LoadBalancerListenerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer_listener"
}

func (r *LoadBalancerListenerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Opens a load-balancer listener and manages its HTTP(S) routing rules. TCP listeners do not support rules.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.Int64Attribute{Computed: true},
			"load_balancer_id": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"port":             schema.Int64Attribute{Required: true},
			"protocol":         schema.StringAttribute{Required: true, Description: "http, https, or tcp."},
			"default_pool":     schema.Int64Attribute{Optional: true, Computed: true, Description: "Default pool ID, or null to return 503 for unmatched HTTP requests."},
			"redirect_https":   schema.BoolAttribute{Optional: true, Computed: true, Description: "For HTTPS listeners, redirect HTTP requests from port 80."},
			"sources":          schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Allowed IPv4/IPv6 addresses or CIDRs; empty or null permits everyone."},
			"rules": schema.ListNestedAttribute{
				Optional: true, Computed: true,
				Description: "Complete ordered routing rule set. Each rule needs a hostname, a path prefix, or both.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"hostname":    schema.StringAttribute{Optional: true, Computed: true},
					"path_prefix": schema.StringAttribute{Optional: true, Computed: true},
					"pool":        schema.Int64Attribute{Required: true},
				}},
			},
		},
	}
}

func (r *LoadBalancerListenerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}

func (r *LoadBalancerListenerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LoadBalancerListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, ok := listenerInputMap(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	listener, err := r.client.CreateLoadBalancerListener(ctx, plan.LoadBalancerID.ValueInt64(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF load-balancer listener", err.Error())
		return
	}
	if !plan.Rules.IsNull() && !plan.Rules.IsUnknown() {
		rules, ok := loadBalancerRulesFromModel(ctx, plan.Rules, &resp.Diagnostics)
		if !ok {
			return
		}
		listener, err = r.client.ReplaceLoadBalancerRules(ctx, plan.LoadBalancerID.ValueInt64(), listener.ID, rules)
		if err != nil {
			resp.Diagnostics.AddError("Error setting FFXF load-balancer rules", err.Error())
			return
		}
	}
	setListenerModel(ctx, &plan, *listener, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoadBalancerListenerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LoadBalancerListenerResourceModel
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
		resp.Diagnostics.AddError("Error reading FFXF load-balancer listener", err.Error())
		return
	}
	for _, listener := range lb.Data.Listeners {
		if listener.ID == state.ID.ValueInt64() {
			setListenerModel(ctx, &state, listener, &resp.Diagnostics)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *LoadBalancerListenerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LoadBalancerListenerResourceModel
	var state LoadBalancerListenerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, ok := listenerInputMap(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	listener, err := r.client.UpdateLoadBalancerListener(ctx, state.LoadBalancerID.ValueInt64(), state.ID.ValueInt64(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF load-balancer listener", err.Error())
		return
	}
	if !plan.Rules.IsNull() && !plan.Rules.IsUnknown() {
		rules, ok := loadBalancerRulesFromModel(ctx, plan.Rules, &resp.Diagnostics)
		if !ok {
			return
		}
		listener, err = r.client.ReplaceLoadBalancerRules(ctx, state.LoadBalancerID.ValueInt64(), state.ID.ValueInt64(), rules)
		if err != nil {
			resp.Diagnostics.AddError("Error replacing FFXF load-balancer rules", err.Error())
			return
		}
	}
	setListenerModel(ctx, &plan, *listener, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoadBalancerListenerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LoadBalancerListenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteLoadBalancerListener(ctx, state.LoadBalancerID.ValueInt64(), state.ID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF load-balancer listener", err.Error())
	}
}

func listenerInputMap(ctx context.Context, model LoadBalancerListenerResourceModel, diagnostics *diag.Diagnostics) (map[string]any, bool) {
	input := map[string]any{"port": model.Port.ValueInt64(), "protocol": model.Protocol.ValueString()}
	if !model.DefaultPool.IsUnknown() {
		if model.DefaultPool.IsNull() {
			input["default_pool"] = nil
		} else {
			input["default_pool"] = model.DefaultPool.ValueInt64()
		}
	}
	if !model.RedirectHTTPS.IsNull() && !model.RedirectHTTPS.IsUnknown() {
		input["redirect_https"] = model.RedirectHTTPS.ValueBool()
	}
	if !model.Sources.IsUnknown() {
		if model.Sources.IsNull() {
			input["sources"] = nil
		} else {
			var sources []string
			diagnostics.Append(model.Sources.ElementsAs(ctx, &sources, false)...)
			input["sources"] = sources
		}
	}
	return input, !diagnostics.HasError()
}

func loadBalancerRulesFromModel(ctx context.Context, values types.List, diagnostics *diag.Diagnostics) ([]client.LoadBalancerRule, bool) {
	var models []LoadBalancerRuleModel
	diagnostics.Append(values.ElementsAs(ctx, &models, false)...)
	if diagnostics.HasError() {
		return nil, false
	}
	rules := make([]client.LoadBalancerRule, 0, len(models))
	for _, model := range models {
		rule := client.LoadBalancerRule{Pool: model.Pool.ValueInt64()}
		if !model.Hostname.IsNull() && !model.Hostname.IsUnknown() {
			value := model.Hostname.ValueString()
			rule.Hostname = &value
		}
		if !model.PathPrefix.IsNull() && !model.PathPrefix.IsUnknown() {
			value := model.PathPrefix.ValueString()
			rule.PathPrefix = &value
		}
		rules = append(rules, rule)
	}
	return rules, true
}

func setListenerModel(ctx context.Context, state *LoadBalancerListenerResourceModel, listener client.LoadBalancerListener, diagnostics *diag.Diagnostics) {
	state.ID = types.Int64Value(listener.ID)
	state.Port = types.Int64Value(int64(listener.Port))
	state.Protocol = types.StringValue(listener.Protocol)
	if listener.DefaultPool == nil {
		state.DefaultPool = types.Int64Null()
	} else {
		state.DefaultPool = types.Int64Value(*listener.DefaultPool)
	}
	state.RedirectHTTPS = types.BoolValue(listener.RedirectHTTPS)
	sources, diags := types.ListValueFrom(ctx, types.StringType, listener.Sources)
	diagnostics.Append(diags...)
	state.Sources = sources
	models := make([]LoadBalancerRuleModel, 0, len(listener.Rules))
	for _, rule := range listener.Rules {
		models = append(models, LoadBalancerRuleModel{Hostname: nullableString(rule.Hostname), PathPrefix: nullableString(rule.PathPrefix), Pool: types.Int64Value(rule.Pool)})
	}
	attrTypes := map[string]attr.Type{"hostname": types.StringType, "path_prefix": types.StringType, "pool": types.Int64Type}
	rules, ruleDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, models)
	diagnostics.Append(ruleDiags...)
	state.Rules = rules
}

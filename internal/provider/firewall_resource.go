package provider

import (
	"context"
	"errors"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FirewallResource{}

type FirewallResource struct{ client *client.Client }

type FirewallResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Rules     types.List   `tfsdk:"rules"`
	Members   types.List   `tfsdk:"members"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type FirewallRuleModel struct {
	Direction    types.String `tfsdk:"direction"`
	Protocol     types.String `tfsdk:"protocol"`
	Ports        types.String `tfsdk:"ports"`
	Sources      types.List   `tfsdk:"sources"`
	Destinations types.List   `tfsdk:"destinations"`
	Description  types.String `tfsdk:"description"`
}

type FirewallMemberModel struct {
	VM       types.Int64  `tfsdk:"vm"`
	Hostname types.String `tfsdk:"hostname"`
	Status   types.String `tfsdk:"status"`
}

func NewFirewallResource() resource.Resource { return &FirewallResource{} }

func (r *FirewallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall"
}

func (r *FirewallResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages reusable inbound and outbound firewall rules. If rules are omitted, FFXF opens SSH and ping by default; set rules = [] for no rules. Attach the firewall to machines with ffxf_firewall_member.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.Int64Attribute{Computed: true, Description: "Firewall ID assigned by FFXF Cloud."},
			"name":       schema.StringAttribute{Required: true, Description: "Firewall name, up to 64 characters."},
			"created_at": schema.StringAttribute{Computed: true},
			"rules": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Complete ordered set of rules. Omitting sources or destinations allows any address. A rule with direction out restricts outbound traffic on attached machines.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"direction":    schema.StringAttribute{Optional: true, Computed: true, Description: "in (default) or out."},
					"protocol":     schema.StringAttribute{Required: true, Description: "tcp, udp, icmp, or any."},
					"ports":        schema.StringAttribute{Optional: true, Description: "TCP/UDP ports, comma-separated list, or range."},
					"sources":      schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Inbound source addresses or CIDRs."},
					"destinations": schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Outbound destination addresses or CIDRs."},
					"description":  schema.StringAttribute{Optional: true, Description: "Optional rule label, up to 64 characters."},
				}},
			},
			"members": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Machines attached to this firewall and whether its current rules have been applied.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"vm":       schema.Int64Attribute{Computed: true},
					"hostname": schema.StringAttribute{Computed: true},
					"status":   schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (r *FirewallResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}

func (r *FirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, ok := firewallRulesFromModel(ctx, plan.Rules, &resp.Diagnostics)
	if !ok {
		return
	}
	created, err := r.client.CreateFirewall(ctx, plan.Name.ValueString(), rules)
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF firewall", err.Error())
		return
	}
	setFirewallModel(ctx, &plan, created.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	firewall, err := r.client.GetFirewall(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF firewall", err.Error())
		return
	}
	setFirewallModel(ctx, &state, firewall.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallResourceModel
	var state FirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, ok := firewallRulesFromModel(ctx, plan.Rules, &resp.Diagnostics)
	if !ok {
		return
	}
	updated, err := r.client.UpdateFirewall(ctx, state.ID.ValueInt64(), plan.Name.ValueString(), rules)
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF firewall", err.Error())
		return
	}
	setFirewallModel(ctx, &plan, updated.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteFirewall(ctx, state.ID.ValueInt64()); err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF firewall", err.Error())
	}
}

func firewallRulesFromModel(ctx context.Context, values types.List, diagnostics *diag.Diagnostics) ([]client.FirewallRule, bool) {
	if values.IsNull() || values.IsUnknown() {
		return nil, true
	}
	var models []FirewallRuleModel
	diagnostics.Append(values.ElementsAs(ctx, &models, false)...)
	if diagnostics.HasError() {
		return nil, false
	}
	rules := make([]client.FirewallRule, 0, len(models))
	for _, model := range models {
		direction := model.Direction.ValueString()
		if direction == "" {
			direction = "in"
		}
		rule := client.FirewallRule{Direction: direction, Protocol: model.Protocol.ValueString()}
		if !model.Ports.IsNull() && !model.Ports.IsUnknown() {
			value := model.Ports.ValueString()
			rule.Ports = &value
		}
		if !model.Description.IsNull() && !model.Description.IsUnknown() {
			value := model.Description.ValueString()
			rule.Description = &value
		}
		if !model.Sources.IsNull() && !model.Sources.IsUnknown() {
			diagnostics.Append(model.Sources.ElementsAs(ctx, &rule.Sources, false)...)
		}
		if !model.Destinations.IsNull() && !model.Destinations.IsUnknown() {
			diagnostics.Append(model.Destinations.ElementsAs(ctx, &rule.Destinations, false)...)
		}
		rules = append(rules, rule)
	}
	return rules, !diagnostics.HasError()
}

func setFirewallModel(ctx context.Context, state *FirewallResourceModel, firewall client.Firewall, diagnostics *diag.Diagnostics) {
	state.ID = types.Int64Value(firewall.ID)
	state.Name = types.StringValue(firewall.Name)
	state.CreatedAt = types.StringValue(firewall.CreatedAt)
	models := make([]FirewallRuleModel, 0, len(firewall.Rules))
	for _, rule := range firewall.Rules {
		model := FirewallRuleModel{Direction: types.StringValue(rule.Direction), Protocol: types.StringValue(rule.Protocol), Ports: types.StringNull(), Description: types.StringNull()}
		if rule.Ports != nil {
			model.Ports = types.StringValue(*rule.Ports)
		}
		if rule.Description != nil {
			model.Description = types.StringValue(*rule.Description)
		}
		model.Sources = firewallStringList(ctx, rule.Sources, diagnostics)
		model.Destinations = firewallStringList(ctx, rule.Destinations, diagnostics)
		models = append(models, model)
	}
	attrTypes := map[string]attr.Type{
		"direction": types.StringType, "protocol": types.StringType, "ports": types.StringType,
		"sources": types.ListType{ElemType: types.StringType}, "destinations": types.ListType{ElemType: types.StringType}, "description": types.StringType,
	}
	rules, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: attrTypes}, models)
	diagnostics.Append(diags...)
	state.Rules = rules
	members := make([]FirewallMemberModel, 0, len(firewall.Members))
	for _, member := range firewall.Members {
		members = append(members, FirewallMemberModel{VM: types.Int64Value(member.VM), Hostname: types.StringValue(member.Hostname), Status: types.StringValue(member.Status)})
	}
	memberTypes := map[string]attr.Type{"vm": types.Int64Type, "hostname": types.StringType, "status": types.StringType}
	memberList, memberDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: memberTypes}, members)
	diagnostics.Append(memberDiags...)
	state.Members = memberList
}

func firewallStringList(ctx context.Context, values []string, diagnostics *diag.Diagnostics) types.List {
	list, diags := types.ListValueFrom(ctx, types.StringType, values)
	diagnostics.Append(diags...)
	return list
}

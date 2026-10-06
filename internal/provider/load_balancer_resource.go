package provider

import (
	"context"
	"errors"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LoadBalancerResource{}

type LoadBalancerResource struct{ client *client.Client }

type LoadBalancerResourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	VPC           types.Int64  `tfsdk:"vpc"`
	Status        types.String `tfsdk:"status"`
	IPv4          types.String `tfsdk:"ipv4"`
	IPv6          types.String `tfsdk:"ipv6"`
	DNSName       types.String `tfsdk:"dns_name"`
	ConfigApplied types.Bool   `tfsdk:"config_applied"`
	LastSeenAt    types.String `tfsdk:"last_seen_at"`
	LastError     types.String `tfsdk:"last_error"`
	CreatedAt     types.String `tfsdk:"created_at"`
	Certificates  types.List   `tfsdk:"certificates"`
	Billing       types.Object `tfsdk:"billing"`
}

type LoadBalancerCertificateModel struct {
	Hostname  types.String `tfsdk:"hostname"`
	Status    types.String `tfsdk:"status"`
	ExpiresAt types.String `tfsdk:"expires_at"`
	Error     types.String `tfsdk:"error"`
}

type LoadBalancerBillingModel struct {
	Mode     types.String `tfsdk:"mode"`
	Rate     types.String `tfsdk:"rate"`
	Currency types.String `tfsdk:"currency"`
}

func NewLoadBalancerResource() resource.Resource { return &LoadBalancerResource{} }

func (r *LoadBalancerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer"
}

func (r *LoadBalancerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a metered FFXF Cloud load balancer in a private network. Configure pools and listeners with their dedicated resources.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.Int64Attribute{Computed: true},
			"name":           schema.StringAttribute{Required: true, Description: "Load balancer name, up to 64 characters."},
			"vpc":            schema.Int64Attribute{Required: true, Description: "Private network ID whose machines may be targets.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"status":         schema.StringAttribute{Computed: true},
			"ipv4":           schema.StringAttribute{Computed: true},
			"ipv6":           schema.StringAttribute{Computed: true},
			"dns_name":       schema.StringAttribute{Computed: true},
			"config_applied": schema.BoolAttribute{Computed: true},
			"last_seen_at":   schema.StringAttribute{Computed: true},
			"last_error":     schema.StringAttribute{Computed: true},
			"created_at":     schema.StringAttribute{Computed: true},
			"certificates": schema.ListNestedAttribute{Computed: true, Description: "Managed HTTPS certificates and their current issuance status.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"hostname": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true},
				"expires_at": schema.StringAttribute{Computed: true}, "error": schema.StringAttribute{Computed: true},
			}}},
			"billing": schema.SingleNestedAttribute{Computed: true, Attributes: map[string]schema.Attribute{
				"mode": schema.StringAttribute{Computed: true}, "rate": schema.StringAttribute{Computed: true}, "currency": schema.StringAttribute{Computed: true},
			}},
		},
	}
}

func (r *LoadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}

func (r *LoadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LoadBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lb, err := r.client.CreateLoadBalancer(ctx, plan.Name.ValueString(), plan.VPC.ValueInt64(), uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF load balancer", err.Error())
		return
	}
	setLoadBalancerModel(ctx, &plan, lb.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LoadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lb, err := r.client.GetLoadBalancer(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF load balancer", err.Error())
		return
	}
	setLoadBalancerModel(ctx, &state, lb.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LoadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LoadBalancerResourceModel
	var state LoadBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lb, err := r.client.RenameLoadBalancer(ctx, state.ID.ValueInt64(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error renaming FFXF load balancer", err.Error())
		return
	}
	setLoadBalancerModel(ctx, &plan, lb.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LoadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteLoadBalancer(ctx, state.ID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF load balancer", err.Error())
	}
}

func setLoadBalancerModel(ctx context.Context, state *LoadBalancerResourceModel, lb client.LoadBalancer, diagnostics *diag.Diagnostics) {
	state.ID = types.Int64Value(lb.ID)
	state.Name = types.StringValue(lb.Name)
	state.VPC = types.Int64Value(lb.VPC)
	state.Status = types.StringValue(lb.Status)
	state.IPv4 = nullableString(lb.IPv4)
	state.IPv6 = nullableString(lb.IPv6)
	state.DNSName = nullableString(lb.DNSName)
	state.ConfigApplied = types.BoolValue(lb.ConfigApplied)
	state.LastSeenAt = nullableString(lb.LastSeenAt)
	state.LastError = nullableString(lb.LastError)
	state.CreatedAt = types.StringValue(lb.CreatedAt)
	certificates := make([]LoadBalancerCertificateModel, 0, len(lb.Certificates))
	for _, certificate := range lb.Certificates {
		certificates = append(certificates, LoadBalancerCertificateModel{
			Hostname: types.StringValue(certificate.Hostname), Status: types.StringValue(certificate.Status),
			ExpiresAt: nullableString(certificate.ExpiresAt), Error: nullableString(certificate.Error),
		})
	}
	certificateTypes := map[string]attr.Type{"hostname": types.StringType, "status": types.StringType, "expires_at": types.StringType, "error": types.StringType}
	certificateList, certificateDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: certificateTypes}, certificates)
	diagnostics.Append(certificateDiags...)
	state.Certificates = certificateList
	billingTypes := map[string]attr.Type{"mode": types.StringType, "rate": types.StringType, "currency": types.StringType}
	if lb.Billing == nil {
		state.Billing = types.ObjectNull(billingTypes)
	} else {
		billing, billingDiags := types.ObjectValue(billingTypes, map[string]attr.Value{
			"mode": types.StringValue(lb.Billing.Mode), "rate": types.StringValue(lb.Billing.Rate), "currency": types.StringValue(lb.Billing.Currency),
		})
		diagnostics.Append(billingDiags...)
		state.Billing = billing
	}
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

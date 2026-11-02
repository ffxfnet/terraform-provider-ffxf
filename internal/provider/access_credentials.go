package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func accessStringList(ctx context.Context, list types.List) ([]string, error) {
	values := []string{}
	if list.IsNull() || list.IsUnknown() {
		return values, nil
	}
	if diags := list.ElementsAs(ctx, &values, false); diags.HasError() {
		return nil, fmt.Errorf("%v", diags)
	}
	return values, nil
}

func accessRoleList(ctx context.Context, list types.List) ([]any, error) {
	names, err := accessStringList(ctx, list)
	if err != nil {
		return nil, err
	}
	roles := make([]any, 0, len(names))
	for _, name := range names {
		roles = append(roles, accessRoleValue(name))
	}
	return roles, nil
}

type accessAPIKeyModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Role           types.String `tfsdk:"role"`
	ExchangeOnly   types.Bool   `tfsdk:"exchange_only"`
	AssumableRoles types.List   `tfsdk:"assumable_roles"`
	IPAddresses    types.List   `tfsdk:"ip_addresses"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	RoleID         types.Int64  `tfsdk:"role_id"`
	Prefix         types.String `tfsdk:"prefix"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Token          types.String `tfsdk:"token"`
	LastUsedAt     types.String `tfsdk:"last_used_at"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

type AccessAPIKeyResource struct{ client *client.Client }

func NewAccessAPIKeyResource() resource.Resource { return &AccessAPIKeyResource{} }

func (r *AccessAPIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *AccessAPIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureAccessClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *AccessAPIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Creates an API key that acts through a role. The API has no update endpoint, so every change replaces the key. The secret is only returned at creation and is kept in the Terraform state; protect your state accordingly.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.Int64Attribute{Computed: true, Description: "API key identifier.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"name":            schema.StringAttribute{Required: true, Description: "Key name, up to 64 characters.", PlanModifiers: replaceString},
			"role":            schema.StringAttribute{Optional: true, Description: "Id or name of the role the key has the rights of. Required unless exchange_only is true.", PlanModifiers: replaceString},
			"exchange_only":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "When true the key can only be exchanged for a short-lived token through POST /auth/token and is refused everywhere else.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"assumable_roles": schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Ids or names of the roles obtainable through token exchange. Required when exchange_only is true.", PlanModifiers: []planmodifier.List{listRequiresReplace()}},
			"ip_addresses":    schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Restrict the key to these IPv4 or IPv6 addresses, at most 50.", PlanModifiers: []planmodifier.List{listRequiresReplace()}},
			"expires_at":      schema.StringAttribute{Optional: true, Description: "Expiry date in YYYY-MM-DD format; it must be in the future.", PlanModifiers: replaceString},
			"role_id":         schema.Int64Attribute{Computed: true, Description: "Identifier of the role the key holds, null for exchange-only keys.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"prefix":          schema.StringAttribute{Computed: true, Description: "Non-secret key prefix shown in the console.", PlanModifiers: replaceUnknownString()},
			"enabled":         schema.BoolAttribute{Computed: true, Description: "Whether the key is enabled.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"token":           schema.StringAttribute{Computed: true, Sensitive: true, Description: "The secret key, only available from creation. It is empty after an import.", PlanModifiers: replaceUnknownString()},
			"last_used_at":    schema.StringAttribute{Computed: true, Description: "Timestamp of the last request made with the key."},
			"created_at":      schema.StringAttribute{Computed: true, Description: "Creation timestamp.", PlanModifiers: replaceUnknownString()},
		},
	}
}

func accessAPIKeyApply(state *accessAPIKeyModel, key client.AccessAPIKey) {
	state.ID = types.Int64Value(key.ID)
	state.RoleID = accessRoleID(key.Role)
	state.ExchangeOnly = types.BoolValue(key.ExchangeOnly)
	state.Prefix = accessOptionalString(types.StringNull(), key.Prefix)
	state.Enabled = types.BoolValue(key.Enabled)
	state.ExpiresAt = accessDate(state.ExpiresAt, key.ExpiresAt)
	state.LastUsedAt = accessOptionalString(types.StringNull(), key.LastUsedAt)
	state.CreatedAt = accessOptionalString(types.StringNull(), key.CreatedAt)
}

func (r *AccessAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessAPIKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"name": plan.Name.ValueString(), "exchange_only": plan.ExchangeOnly.ValueBool()}
	if !plan.Role.IsNull() {
		body["role"] = accessRoleValue(plan.Role.ValueString())
	}
	roles, err := accessRoleList(ctx, plan.AssumableRoles)
	if err == nil && !plan.AssumableRoles.IsNull() {
		body["assumable_roles"] = roles
	}
	addresses, addressErr := accessStringList(ctx, plan.IPAddresses)
	if err == nil {
		err = addressErr
	}
	if err != nil {
		resp.Diagnostics.AddError("Invalid FFXF API key", err.Error())
		return
	}
	if !plan.IPAddresses.IsNull() {
		body["ip_addresses"] = addresses
	}
	if !plan.ExpiresAt.IsNull() {
		body["expires_at"] = plan.ExpiresAt.ValueString()
	}
	key, err := r.client.AccessCreateAPIKey(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF API key", err.Error())
		return
	}
	accessAPIKeyApply(&plan, key)
	plan.Token = types.StringValue(key.Token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessAPIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key, err := r.client.AccessGetAPIKey(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF API key", err.Error())
		return
	}
	state.Name = types.StringValue(key.Name)
	if state.Role.IsNull() && key.Role != nil {
		state.Role = types.StringValue(key.Role.Name)
	}
	if state.AssumableRoles.IsNull() && len(key.AssumableRoles) > 0 {
		names := make([]attr.Value, 0, len(key.AssumableRoles))
		for _, role := range key.AssumableRoles {
			names = append(names, types.StringValue(role.Name))
		}
		state.AssumableRoles = types.ListValueMust(types.StringType, names)
	}
	if state.IPAddresses.IsNull() && len(key.IPAddresses) > 0 {
		addresses := make([]attr.Value, 0, len(key.IPAddresses))
		for _, address := range key.IPAddresses {
			addresses = append(addresses, types.StringValue(address))
		}
		state.IPAddresses = types.ListValueMust(types.StringType, addresses)
	}
	if state.Token.IsNull() {
		state.Token = types.StringValue("")
	}
	accessAPIKeyApply(&state, key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessAPIKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported API key update", "API key changes require replacement.")
}

func (r *AccessAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessAPIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AccessDeleteAPIKey(ctx, state.ID.ValueInt64()); err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF API key", err.Error())
	}
}

func (r *AccessAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	accessImportID(ctx, req, resp)
}

type accessClaimConditionModel struct {
	Claim types.String `tfsdk:"claim"`
	Value types.String `tfsdk:"value"`
}

type accessOIDCBindingModel struct {
	ID              types.Int64  `tfsdk:"id"`
	ProviderID      types.Int64  `tfsdk:"provider_id"`
	Name            types.String `tfsdk:"name"`
	ClaimConditions types.List   `tfsdk:"claim_conditions"`
	Role            types.String `tfsdk:"role"`
	ExchangeOnly    types.Bool   `tfsdk:"exchange_only"`
	AssumableRoles  types.List   `tfsdk:"assumable_roles"`
	ExpiresAt       types.String `tfsdk:"expires_at"`
	RoleID          types.Int64  `tfsdk:"role_id"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	CreatedAt       types.String `tfsdk:"created_at"`
}

var accessClaimConditionType = types.ObjectType{AttrTypes: map[string]attr.Type{"claim": types.StringType, "value": types.StringType}}

type AccessOIDCBindingResource struct{ client *client.Client }

func NewAccessOIDCBindingResource() resource.Resource { return &AccessOIDCBindingResource{} }

func (r *AccessOIDCBindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_binding"
}

func (r *AccessOIDCBindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureAccessClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *AccessOIDCBindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Binds an OIDC identity (for example a GitHub Actions workflow) to a role so it can call the API without a stored secret. A binding's claim conditions must include a `sub` condition that names the owner, and a credential cannot modify its own binding.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.Int64Attribute{Computed: true, Description: "Binding identifier.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"provider_id": schema.Int64Attribute{Required: true, Description: "OIDC provider identifier, from the ffxf_oidc_providers data source."},
			"name":        schema.StringAttribute{Required: true, Description: "Binding name, up to 80 characters."},
			"claim_conditions": schema.ListNestedAttribute{
				Required:    true,
				Description: "Claim conditions the token must satisfy; at least one must be a `sub` condition naming the owner. Values may use `*` and `?` wildcards after the owner.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"claim": schema.StringAttribute{Required: true, Description: "Claim name, for example sub or repository."},
					"value": schema.StringAttribute{Required: true, Description: "Expected claim value or pattern."},
				}},
			},
			"role":            schema.StringAttribute{Optional: true, Description: "Id or name of the role the binding has the rights of. Required unless exchange_only is true."},
			"exchange_only":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "When true the binding can only be exchanged for a short-lived token through POST /auth/token."},
			"assumable_roles": schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Ids or names of the roles obtainable through token exchange. Required when exchange_only is true."},
			"expires_at":      schema.StringAttribute{Optional: true, Description: "Expiry date in YYYY-MM-DD format; it must be in the future."},
			"role_id":         schema.Int64Attribute{Computed: true, Description: "Identifier of the role the binding holds, null for exchange-only bindings."},
			"enabled":         schema.BoolAttribute{Computed: true, Description: "Whether the binding is enabled.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"created_at":      schema.StringAttribute{Computed: true, Description: "Creation timestamp.", PlanModifiers: replaceUnknownString()},
		},
	}
}

func accessBindingBody(ctx context.Context, plan accessOIDCBindingModel) (map[string]any, error) {
	var conditions []accessClaimConditionModel
	if diags := plan.ClaimConditions.ElementsAs(ctx, &conditions, false); diags.HasError() {
		return nil, fmt.Errorf("invalid claim_conditions: %v", diags)
	}
	claims := make([]map[string]string, 0, len(conditions))
	for _, condition := range conditions {
		claims = append(claims, map[string]string{"claim": condition.Claim.ValueString(), "value": condition.Value.ValueString()})
	}
	body := map[string]any{
		"provider_id":      plan.ProviderID.ValueInt64(),
		"name":             plan.Name.ValueString(),
		"claim_conditions": claims,
		"exchange_only":    plan.ExchangeOnly.ValueBool(),
	}
	if !plan.Role.IsNull() {
		body["role"] = accessRoleValue(plan.Role.ValueString())
	}
	if !plan.AssumableRoles.IsNull() {
		roles, err := accessRoleList(ctx, plan.AssumableRoles)
		if err != nil {
			return nil, err
		}
		body["assumable_roles"] = roles
	}
	if !plan.ExpiresAt.IsNull() {
		body["expires_at"] = plan.ExpiresAt.ValueString()
	}
	return body, nil
}

func accessBindingApply(ctx context.Context, state *accessOIDCBindingModel, binding client.AccessOIDCBinding) error {
	conditions := make([]attr.Value, 0, len(binding.ClaimConditions))
	for _, condition := range binding.ClaimConditions {
		conditions = append(conditions, types.ObjectValueMust(accessClaimConditionType.AttrTypes, map[string]attr.Value{
			"claim": types.StringValue(condition.Claim), "value": types.StringValue(condition.Value),
		}))
	}
	state.ID = types.Int64Value(binding.ID)
	state.ProviderID = types.Int64Value(binding.ProviderID)
	state.Name = types.StringValue(binding.Name)
	state.ClaimConditions = types.ListValueMust(accessClaimConditionType, conditions)
	state.RoleID = accessRoleID(binding.Role)
	state.ExchangeOnly = types.BoolValue(binding.ExchangeOnly)
	state.Enabled = types.BoolValue(binding.Enabled)
	state.ExpiresAt = accessDate(state.ExpiresAt, binding.ExpiresAt)
	state.CreatedAt = accessOptionalString(types.StringNull(), binding.CreatedAt)
	if state.Role.IsNull() && binding.Role != nil {
		state.Role = types.StringValue(binding.Role.Name)
	}
	if state.AssumableRoles.IsNull() && len(binding.AssumableRoles) > 0 {
		names := make([]attr.Value, 0, len(binding.AssumableRoles))
		for _, role := range binding.AssumableRoles {
			names = append(names, types.StringValue(role.Name))
		}
		state.AssumableRoles = types.ListValueMust(types.StringType, names)
	}
	return nil
}

func (r *AccessOIDCBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessOIDCBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := accessBindingBody(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid FFXF OIDC binding", err.Error())
		return
	}
	binding, err := r.client.AccessCreateOIDCBinding(ctx, body, uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF OIDC binding", err.Error())
		return
	}
	if err := accessBindingApply(ctx, &plan, binding); err != nil {
		resp.Diagnostics.AddError("Invalid FFXF OIDC binding response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessOIDCBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessOIDCBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	binding, err := r.client.AccessGetOIDCBinding(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF OIDC binding", err.Error())
		return
	}
	if err := accessBindingApply(ctx, &state, binding); err != nil {
		resp.Diagnostics.AddError("Invalid FFXF OIDC binding response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessOIDCBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state accessOIDCBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := accessBindingBody(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid FFXF OIDC binding", err.Error())
		return
	}
	binding, err := r.client.AccessUpdateOIDCBinding(ctx, state.ID.ValueInt64(), body, uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF OIDC binding", err.Error())
		return
	}
	plan.ID = state.ID
	if err := accessBindingApply(ctx, &plan, binding); err != nil {
		resp.Diagnostics.AddError("Invalid FFXF OIDC binding response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessOIDCBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessOIDCBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AccessDeleteOIDCBinding(ctx, state.ID.ValueInt64()); err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF OIDC binding", err.Error())
	}
}

func (r *AccessOIDCBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	accessImportID(ctx, req, resp)
}

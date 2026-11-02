package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type accessPolicyModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Document           types.String `tfsdk:"document"`
	NormalizedDocument types.String `tfsdk:"normalized_document"`
	CreatedAt          types.String `tfsdk:"created_at"`
}

type AccessPolicyResource struct{ client *client.Client }

func NewAccessPolicyResource() resource.Resource { return &AccessPolicyResource{} }

func (r *AccessPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *AccessPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureAccessClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *AccessPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an access policy: a JSON document of allow/deny statements that a role can carry. A credential cannot modify a policy attached to a role it holds or can assume, and temporary access tokens cannot manage policies.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.Int64Attribute{Computed: true, Description: "Policy identifier.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"name":                schema.StringAttribute{Required: true, Description: "Policy name, unique per account, up to 80 characters."},
			"description":         schema.StringAttribute{Optional: true, Description: "Free-form description, up to 255 characters."},
			"document":            schema.StringAttribute{Required: true, Description: "Policy document as JSON text, for example from jsonencode(). Semantically equal JSON does not produce a diff."},
			"normalized_document": schema.StringAttribute{Computed: true, Description: "The document as stored by the API, used to detect changes made outside Terraform."},
			"created_at":          schema.StringAttribute{Computed: true, Description: "Creation timestamp.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func accessPolicyBody(plan accessPolicyModel) map[string]any {
	body := map[string]any{"name": plan.Name.ValueString(), "document": plan.Document.ValueString()}
	if !plan.Description.IsNull() {
		body["description"] = plan.Description.ValueString()
	}
	return body
}

// accessPolicyApply records the API result without changing the configured document text.
func accessPolicyApply(state *accessPolicyModel, policy client.AccessPolicy) {
	state.ID = types.Int64Value(policy.ID)
	state.Name = types.StringValue(policy.Name)
	state.Description = accessOptionalString(state.Description, policy.Description)
	state.NormalizedDocument = types.StringValue(accessCompactJSON(policy.Document))
	state.CreatedAt = accessOptionalString(types.StringNull(), policy.CreatedAt)
}

func (r *AccessPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policy, err := r.client.AccessCreatePolicy(ctx, accessPolicyBody(plan), uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF policy", err.Error())
		return
	}
	accessPolicyApply(&plan, policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policy, err := r.client.AccessGetPolicy(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF policy", err.Error())
		return
	}
	remote := accessCompactJSON(policy.Document)
	if state.NormalizedDocument.IsNull() || !accessJSONEqual(state.NormalizedDocument.ValueString(), remote) {
		state.Document = types.StringValue(remote)
	}
	accessPolicyApply(&state, policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state accessPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policy, err := r.client.AccessUpdatePolicy(ctx, state.ID.ValueInt64(), accessPolicyBody(plan), uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF policy", err.Error())
		return
	}
	plan.ID = state.ID
	accessPolicyApply(&plan, policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AccessDeletePolicy(ctx, state.ID.ValueInt64()); err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF policy", err.Error())
	}
}

func (r *AccessPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	accessImportID(ctx, req, resp)
}

func accessImportID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import identifier", fmt.Sprintf("Expected a numeric identifier, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func configureAccessClient(providerData any, target **client.Client, diagnostics interface {
	AddError(string, string)
}) {
	if providerData == nil {
		return
	}
	apiClient, ok := providerData.(*client.Client)
	if !ok {
		diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, received %T", providerData))
		return
	}
	*target = apiClient
}

type accessRoleModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PolicyIDs   types.Set    `tfsdk:"policy_ids"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

type AccessRoleResource struct{ client *client.Client }

func NewAccessRoleResource() resource.Resource { return &AccessRoleResource{} }

func (r *AccessRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *AccessRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureAccessClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *AccessRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a role: a named set of policies that API keys and OIDC bindings act through. A credential cannot modify a role it holds or can assume, and a role still used by a key or binding cannot be deleted or emptied.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.Int64Attribute{Computed: true, Description: "Role identifier.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"name":        schema.StringAttribute{Required: true, Description: "Role name, unique per account, up to 80 characters."},
			"description": schema.StringAttribute{Optional: true, Description: "Free-form description, up to 255 characters."},
			"policy_ids":  schema.SetAttribute{Required: true, ElementType: types.Int64Type, Description: "Identifiers of the policies the role carries, yours or provided ones, at most 10."},
			"created_at":  schema.StringAttribute{Computed: true, Description: "Creation timestamp.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func accessRoleBody(ctx context.Context, plan accessRoleModel) (map[string]any, error) {
	var ids []int64
	if diags := plan.PolicyIDs.ElementsAs(ctx, &ids, false); diags.HasError() {
		return nil, fmt.Errorf("invalid policy_ids: %v", diags)
	}
	if ids == nil {
		ids = []int64{}
	}
	body := map[string]any{"name": plan.Name.ValueString(), "policy_ids": ids}
	if !plan.Description.IsNull() {
		body["description"] = plan.Description.ValueString()
	}
	return body, nil
}

func accessRoleApply(ctx context.Context, state *accessRoleModel, role client.AccessRole) error {
	ids := role.PolicyIDs
	if ids == nil {
		ids = []int64{}
	}
	set, diags := types.SetValueFrom(ctx, types.Int64Type, ids)
	if diags.HasError() {
		return fmt.Errorf("invalid policy_ids: %v", diags)
	}
	state.ID = types.Int64Value(role.ID)
	state.Name = types.StringValue(role.Name)
	state.Description = accessOptionalString(state.Description, role.Description)
	state.PolicyIDs = set
	state.CreatedAt = accessOptionalString(types.StringNull(), role.CreatedAt)
	return nil
}

func (r *AccessRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := accessRoleBody(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid FFXF role", err.Error())
		return
	}
	role, err := r.client.AccessCreateRole(ctx, body, uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF role", err.Error())
		return
	}
	if err := accessRoleApply(ctx, &plan, role); err != nil {
		resp.Diagnostics.AddError("Invalid FFXF role response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	role, err := r.client.AccessGetRole(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF role", err.Error())
		return
	}
	if err := accessRoleApply(ctx, &state, role); err != nil {
		resp.Diagnostics.AddError("Invalid FFXF role response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state accessRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := accessRoleBody(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid FFXF role", err.Error())
		return
	}
	role, err := r.client.AccessUpdateRole(ctx, state.ID.ValueInt64(), body, uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating FFXF role", err.Error())
		return
	}
	plan.ID = state.ID
	if err := accessRoleApply(ctx, &plan, role); err != nil {
		resp.Diagnostics.AddError("Invalid FFXF role response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AccessDeleteRole(ctx, state.ID.ValueInt64()); err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF role", err.Error())
	}
}

func (r *AccessRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	accessImportID(ctx, req, resp)
}

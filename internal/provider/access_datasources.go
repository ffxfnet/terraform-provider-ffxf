package provider

import (
	"context"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type accessDataSource struct{ client *client.Client }

func (d *accessDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	configureAccessClient(req.ProviderData, &d.client, &resp.Diagnostics)
}

func accessListAttribute(description string, attributes map[string]schema.Attribute) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{Computed: true, Description: description, NestedObject: schema.NestedAttributeObject{Attributes: attributes}}
}

type accessPolicyItem struct {
	ID          int64   `tfsdk:"id"`
	Name        string  `tfsdk:"name"`
	Description *string `tfsdk:"description"`
	Provided    bool    `tfsdk:"provided"`
	Document    string  `tfsdk:"document"`
	RolesCount  *int64  `tfsdk:"roles_count"`
	CreatedAt   *string `tfsdk:"created_at"`
}

type accessPoliciesModel struct {
	Policies []accessPolicyItem `tfsdk:"policies"`
}

type AccessPoliciesDataSource struct{ accessDataSource }

func NewAccessPoliciesDataSource() datasource.DataSource { return &AccessPoliciesDataSource{} }

func (d *AccessPoliciesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policies"
}

func (d *AccessPoliciesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists the policies available to the account, including the read-only policies provided by FFXF.", Attributes: map[string]schema.Attribute{
		"policies": accessListAttribute("Available policies.", map[string]schema.Attribute{
			"id":          schema.Int64Attribute{Computed: true, Description: "Policy identifier."},
			"name":        schema.StringAttribute{Computed: true, Description: "Policy name."},
			"description": schema.StringAttribute{Computed: true, Description: "Policy description."},
			"provided":    schema.BoolAttribute{Computed: true, Description: "True for read-only policies provided by FFXF."},
			"document":    schema.StringAttribute{Computed: true, Description: "Policy document as JSON text."},
			"roles_count": schema.Int64Attribute{Computed: true, Description: "Number of roles the policy is attached to."},
			"created_at":  schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
		}),
	}}
}

func (d *AccessPoliciesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	policies, err := d.client.AccessListPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF policies", err.Error())
		return
	}
	state := accessPoliciesModel{Policies: make([]accessPolicyItem, 0, len(policies))}
	for _, policy := range policies {
		state.Policies = append(state.Policies, accessPolicyItem{ID: policy.ID, Name: policy.Name, Description: policy.Description, Provided: policy.Provided, Document: accessCompactJSON(policy.Document), RolesCount: policy.RolesCount, CreatedAt: policy.CreatedAt})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type accessRoleItem struct {
	ID          int64   `tfsdk:"id"`
	Name        string  `tfsdk:"name"`
	Description *string `tfsdk:"description"`
	PolicyIDs   []int64 `tfsdk:"policy_ids"`
	CreatedAt   *string `tfsdk:"created_at"`
}

type accessRolesModel struct {
	Roles []accessRoleItem `tfsdk:"roles"`
}

type AccessRolesDataSource struct{ accessDataSource }

func NewAccessRolesDataSource() datasource.DataSource { return &AccessRolesDataSource{} }

func (d *AccessRolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *AccessRolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists the roles of the account.", Attributes: map[string]schema.Attribute{
		"roles": accessListAttribute("Roles of the account.", map[string]schema.Attribute{
			"id":          schema.Int64Attribute{Computed: true, Description: "Role identifier."},
			"name":        schema.StringAttribute{Computed: true, Description: "Role name."},
			"description": schema.StringAttribute{Computed: true, Description: "Role description."},
			"policy_ids":  schema.ListAttribute{Computed: true, ElementType: types.Int64Type, Description: "Identifiers of the policies the role carries."},
			"created_at":  schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
		}),
	}}
}

func (d *AccessRolesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	roles, err := d.client.AccessListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF roles", err.Error())
		return
	}
	state := accessRolesModel{Roles: make([]accessRoleItem, 0, len(roles))}
	for _, role := range roles {
		ids := role.PolicyIDs
		if ids == nil {
			ids = []int64{}
		}
		state.Roles = append(state.Roles, accessRoleItem{ID: role.ID, Name: role.Name, Description: role.Description, PolicyIDs: ids, CreatedAt: role.CreatedAt})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type accessOIDCProviderItem struct {
	ID       int64  `tfsdk:"id"`
	Name     string `tfsdk:"name"`
	Issuer   string `tfsdk:"issuer"`
	Audience string `tfsdk:"audience"`
	Enabled  bool   `tfsdk:"enabled"`
}

type accessOIDCProvidersModel struct {
	Providers []accessOIDCProviderItem `tfsdk:"providers"`
}

type AccessOIDCProvidersDataSource struct{ accessDataSource }

func NewAccessOIDCProvidersDataSource() datasource.DataSource {
	return &AccessOIDCProvidersDataSource{}
}

func (d *AccessOIDCProvidersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_providers"
}

func (d *AccessOIDCProvidersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists the OIDC providers configured on the account. Providers are managed in the console; use their id with ffxf_oidc_binding.", Attributes: map[string]schema.Attribute{
		"providers": accessListAttribute("OIDC providers.", map[string]schema.Attribute{
			"id":       schema.Int64Attribute{Computed: true, Description: "Provider identifier."},
			"name":     schema.StringAttribute{Computed: true, Description: "Provider name."},
			"issuer":   schema.StringAttribute{Computed: true, Description: "Issuer URL."},
			"audience": schema.StringAttribute{Computed: true, Description: "Expected token audience."},
			"enabled":  schema.BoolAttribute{Computed: true, Description: "Whether the provider is enabled."},
		}),
	}}
}

func (d *AccessOIDCProvidersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	providers, err := d.client.AccessListOIDCProviders(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF OIDC providers", err.Error())
		return
	}
	state := accessOIDCProvidersModel{Providers: make([]accessOIDCProviderItem, 0, len(providers))}
	for _, provider := range providers {
		state.Providers = append(state.Providers, accessOIDCProviderItem(provider))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type accessAPIKeyItem struct {
	ID             int64    `tfsdk:"id"`
	Name           string   `tfsdk:"name"`
	Prefix         *string  `tfsdk:"prefix"`
	Enabled        bool     `tfsdk:"enabled"`
	Role           *string  `tfsdk:"role"`
	ExchangeOnly   bool     `tfsdk:"exchange_only"`
	AssumableRoles []string `tfsdk:"assumable_roles"`
	IPAddresses    []string `tfsdk:"ip_addresses"`
	ExpiresAt      *string  `tfsdk:"expires_at"`
	LastUsedAt     *string  `tfsdk:"last_used_at"`
	CreatedAt      *string  `tfsdk:"created_at"`
}

type accessAPIKeysModel struct {
	APIKeys []accessAPIKeyItem `tfsdk:"api_keys"`
}

type AccessAPIKeysDataSource struct{ accessDataSource }

func NewAccessAPIKeysDataSource() datasource.DataSource { return &AccessAPIKeysDataSource{} }

func (d *AccessAPIKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *AccessAPIKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists the API keys of the account. Secrets are never returned.", Attributes: map[string]schema.Attribute{
		"api_keys": accessListAttribute("API keys.", map[string]schema.Attribute{
			"id":              schema.Int64Attribute{Computed: true, Description: "API key identifier."},
			"name":            schema.StringAttribute{Computed: true, Description: "Key name."},
			"prefix":          schema.StringAttribute{Computed: true, Description: "Non-secret key prefix."},
			"enabled":         schema.BoolAttribute{Computed: true, Description: "Whether the key is enabled."},
			"role":            schema.StringAttribute{Computed: true, Description: "Name of the role the key holds."},
			"exchange_only":   schema.BoolAttribute{Computed: true, Description: "Whether the key can only be exchanged for a temporary token."},
			"assumable_roles": schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Names of the roles obtainable through token exchange."},
			"ip_addresses":    schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Addresses the key is restricted to."},
			"expires_at":      schema.StringAttribute{Computed: true, Description: "Expiry timestamp."},
			"last_used_at":    schema.StringAttribute{Computed: true, Description: "Timestamp of the last use."},
			"created_at":      schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
		}),
	}}
}

func (d *AccessAPIKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.AccessListAPIKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF API keys", err.Error())
		return
	}
	state := accessAPIKeysModel{APIKeys: make([]accessAPIKeyItem, 0, len(keys))}
	for _, key := range keys {
		item := accessAPIKeyItem{ID: key.ID, Name: key.Name, Prefix: key.Prefix, Enabled: key.Enabled, ExchangeOnly: key.ExchangeOnly, ExpiresAt: key.ExpiresAt, LastUsedAt: key.LastUsedAt, CreatedAt: key.CreatedAt, AssumableRoles: []string{}, IPAddresses: key.IPAddresses}
		if item.IPAddresses == nil {
			item.IPAddresses = []string{}
		}
		if key.Role != nil {
			item.Role = &key.Role.Name
		}
		for _, role := range key.AssumableRoles {
			item.AssumableRoles = append(item.AssumableRoles, role.Name)
		}
		state.APIKeys = append(state.APIKeys, item)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

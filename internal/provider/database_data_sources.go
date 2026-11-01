package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type databaseDataSource struct {
	client *client.Client
	kind   string
}
type databaseListDataSourceModel struct {
	DatabasesJSON types.String `tfsdk:"databases_json"`
	Quota         types.Int64  `tfsdk:"quota"`
}
type databaseOptionsDataSourceModel struct {
	OptionsJSON types.String `tfsdk:"options_json"`
}
type databaseCredentialsDataSourceModel struct {
	DatabaseID          types.Int64  `tfsdk:"database_id"`
	CredentialsJSON     types.String `tfsdk:"credentials_json"`
	UsersJSON           types.String `tfsdk:"users_json"`
	ConnectionURI       types.String `tfsdk:"connection_uri"`
	PublicConnectionURI types.String `tfsdk:"public_connection_uri"`
}
type databaseBackupsDataSourceModel struct {
	DatabaseID        types.Int64  `tfsdk:"database_id"`
	BackupsJSON       types.String `tfsdk:"backups_json"`
	RestoreWindowJSON types.String `tfsdk:"restore_window_json"`
}
type databaseMetricsDataSourceModel struct {
	DatabaseID  types.Int64  `tfsdk:"database_id"`
	Range       types.String `tfsdk:"range"`
	MetricsJSON types.String `tfsdk:"metrics_json"`
}
type databaseQueriesDataSourceModel struct {
	DatabaseID  types.Int64  `tfsdk:"database_id"`
	QueriesJSON types.String `tfsdk:"queries_json"`
}

func databaseDataSourceConstructor(kind string) func() datasource.DataSource {
	return func() datasource.DataSource { return &databaseDataSource{kind: kind} }
}
func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database" + d.kind
}
func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{}
	switch d.kind {
	case "s":
		attributes["databases_json"] = schema.StringAttribute{Computed: true, Description: "JSON array of databases returned by the API; use jsondecode to access its fields."}
		attributes["quota"] = schema.Int64Attribute{Computed: true, Description: "Maximum number of databases available to this account."}
	case "_options":
		attributes["options_json"] = schema.StringAttribute{Computed: true, Description: "Database engines, versions, extensions, sizes, and pricing options as JSON."}
	case "_credentials":
		databaseIDAttr(attributes, "ID of the database whose credentials to retrieve.")
		attributes["credentials_json"] = schema.StringAttribute{Computed: true, Sensitive: true, Description: "Credentials response as JSON, including passwords and connection URIs."}
		attributes["users_json"] = schema.StringAttribute{Computed: true, Sensitive: true, Description: "Database users and their passwords as JSON."}
		attributes["connection_uri"] = schema.StringAttribute{Computed: true, Sensitive: true, Description: "Private connection URI containing credentials."}
		attributes["public_connection_uri"] = schema.StringAttribute{Computed: true, Sensitive: true, Description: "Public connection URI containing credentials, when public access is enabled."}
	case "_backups":
		databaseIDAttr(attributes, "ID of the database to query.")
		attributes["backups_json"] = schema.StringAttribute{Computed: true, Description: "Backups available for point-in-time restore as JSON."}
		attributes["restore_window_json"] = schema.StringAttribute{Computed: true, Description: "Earliest and latest restorable timestamps as JSON."}
	case "_metrics":
		databaseIDAttr(attributes, "ID of the database to query.")
		attributes["range"] = schema.StringAttribute{Optional: true, Description: "Metrics time range: 1h, 24h (default), or 7d."}
		attributes["metrics_json"] = schema.StringAttribute{Computed: true, Description: "Database metrics and time-series points as JSON."}
	case "_queries":
		databaseIDAttr(attributes, "ID of the database to query.")
		attributes["queries_json"] = schema.StringAttribute{Computed: true, Description: "Recent database query statistics as JSON."}
	}
	resp.Schema = schema.Schema{Description: "Reads FFXF Cloud database API data. Database features are in beta; accounts without access receive HTTP 403.", Attributes: attributes}
}
func databaseIDAttr(attributes map[string]schema.Attribute, description string) {
	attributes["database_id"] = schema.Int64Attribute{Required: true, Description: description}
}
func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, received %T", req.ProviderData))
		return
	}
	d.client = apiClient
}
func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured database data source", "The FFXF provider client is not configured.")
		return
	}
	switch d.kind {
	case "s":
		result, err := d.client.DatabaseList(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list FFXF databases", err.Error())
			return
		}
		rawDatabases := make([]json.RawMessage, 0, len(result.Data))
		for _, database := range result.Data {
			if len(database.DatabaseJSON) == 0 {
				resp.Diagnostics.AddError("Unable to encode FFXF database list", "The database API returned an empty database object.")
				return
			}
			rawDatabases = append(rawDatabases, database.DatabaseJSON)
		}
		encoded, err := json.Marshal(rawDatabases)
		if err != nil {
			resp.Diagnostics.AddError("Unable to encode FFXF database list", err.Error())
			return
		}
		model := databaseListDataSourceModel{DatabasesJSON: types.StringValue(string(encoded)), Quota: types.Int64Value(int64(result.Quota))}
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	case "_options":
		var model databaseOptionsDataSourceModel
		var result map[string]json.RawMessage
		if err := d.client.DatabaseOptions(ctx, &result); err != nil {
			resp.Diagnostics.AddError("Unable to read FFXF database options", err.Error())
			return
		}
		model.OptionsJSON = databaseRawValue(result["data"])
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	case "_credentials":
		var model databaseCredentialsDataSourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var result map[string]json.RawMessage
		if err := d.client.DatabaseEndpoint(ctx, model.DatabaseID.ValueInt64(), "credentials", nil, &result); err != nil {
			resp.Diagnostics.AddError("Unable to read FFXF database credentials", err.Error())
			return
		}
		model.CredentialsJSON = databaseRawValue(result["data"])
		var credentials struct {
			Users               json.RawMessage `json:"users"`
			ConnectionURI       *string         `json:"connection_uri"`
			PublicConnectionURI *string         `json:"public_connection_uri"`
		}
		if err := json.Unmarshal(result["data"], &credentials); err != nil {
			resp.Diagnostics.AddError("Unable to decode FFXF database credentials", err.Error())
			return
		}
		model.UsersJSON = databaseRawValue(credentials.Users)
		model.ConnectionURI = nullableString(credentials.ConnectionURI)
		model.PublicConnectionURI = nullableString(credentials.PublicConnectionURI)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	case "_backups":
		var model databaseBackupsDataSourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var result map[string]json.RawMessage
		if err := d.client.DatabaseEndpoint(ctx, model.DatabaseID.ValueInt64(), "backups", nil, &result); err != nil {
			resp.Diagnostics.AddError("Unable to read FFXF database backups", err.Error())
			return
		}
		model.BackupsJSON = databaseRawValue(result["data"])
		model.RestoreWindowJSON = databaseRawValue(result["restore_window"])
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	case "_metrics":
		var model databaseMetricsDataSourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
		if resp.Diagnostics.HasError() {
			return
		}
		query := url.Values{}
		if !model.Range.IsNull() && !model.Range.IsUnknown() && model.Range.ValueString() != "" {
			query.Set("range", model.Range.ValueString())
		}
		var result map[string]json.RawMessage
		if err := d.client.DatabaseEndpoint(ctx, model.DatabaseID.ValueInt64(), "metrics", query, &result); err != nil {
			resp.Diagnostics.AddError("Unable to read FFXF database metrics", err.Error())
			return
		}
		model.MetricsJSON = databaseRawValue(result["data"])
		if model.Range.IsNull() || model.Range.ValueString() == "" {
			model.Range = types.StringValue("24h")
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	case "_queries":
		var model databaseQueriesDataSourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var result map[string]json.RawMessage
		if err := d.client.DatabaseEndpoint(ctx, model.DatabaseID.ValueInt64(), "queries", nil, &result); err != nil {
			resp.Diagnostics.AddError("Unable to read FFXF database queries", err.Error())
			return
		}
		model.QueriesJSON = databaseRawValue(result["data"])
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	default:
		resp.Diagnostics.AddError("Unsupported database data source", fmt.Sprintf("Unknown database data source kind %q", d.kind))
	}
}
func databaseRawValue(raw json.RawMessage) types.String {
	if len(raw) == 0 || string(raw) == "null" {
		return types.StringNull()
	}
	return types.StringValue(string(raw))
}

var _ datasource.DataSource = (*databaseDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*databaseDataSource)(nil)

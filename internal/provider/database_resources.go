package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var databasePollInterval = 5 * time.Second
var databasePollTimeout = 10 * time.Minute

type databaseResource struct{ client *client.Client }
type databaseResourceModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	VPC                  types.Int64  `tfsdk:"vpc"`
	Engine               types.String `tfsdk:"engine"`
	Version              types.String `tfsdk:"version"`
	SizeGB               types.Int64  `tfsdk:"size_gb"`
	Billing              types.String `tfsdk:"billing"`
	MaintenanceDay       types.Int64  `tfsdk:"maintenance_day"`
	MaintenanceHour      types.Int64  `tfsdk:"maintenance_hour"`
	DiskCapGB            types.Int64  `tfsdk:"disk_cap_gb"`
	EvictionPolicy       types.String `tfsdk:"eviction_policy"`
	Status               types.String `tfsdk:"status"`
	EngineVersion        types.String `tfsdk:"engine_version"`
	Region               types.String `tfsdk:"region"`
	ConnectionJSON       types.String `tfsdk:"connection_json"`
	PublicAccessJSON     types.String `tfsdk:"public_access_json"`
	DiskJSON             types.String `tfsdk:"disk_json"`
	UsersJSON            types.String `tfsdk:"users_json"`
	LogicalDatabasesJSON types.String `tfsdk:"logical_databases_json"`
	CreatedAt            types.String `tfsdk:"created_at"`
	DatabaseJSON         types.String `tfsdk:"database_json"`
}

type databaseDerivedResource struct {
	client *client.Client
	kind   string
}

func databaseResourceConstructor() resource.Resource { return &databaseResource{} }
func databaseForkResourceConstructor() resource.Resource {
	return &databaseDerivedResource{kind: "fork"}
}
func databaseRestoreResourceConstructor() resource.Resource {
	return &databaseDerivedResource{kind: "restore"}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}
func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceVPC := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{Description: "Creates and manages a FFXF Cloud database. Databases are in beta; an account without database access receives HTTP 403.", Attributes: map[string]schema.Attribute{
		"id":                     schema.Int64Attribute{Computed: true, Description: "Database ID."},
		"name":                   schema.StringAttribute{Required: true, Description: "Database name, up to 64 characters."},
		"vpc":                    schema.Int64Attribute{Required: true, Description: "Private network ID where the database receives its address.", PlanModifiers: replaceVPC},
		"engine":                 schema.StringAttribute{Optional: true, Computed: true, Description: "Database engine: postgresql, mysql (Percona Server), mariadb, or valkey."},
		"version":                schema.StringAttribute{Optional: true, Computed: true, Description: "Major engine version. Defaults to the latest version available in database options."},
		"size_gb":                schema.Int64Attribute{Optional: true, Computed: true, Description: "Database memory size in GB. Resizing is in place and supports increases only."},
		"billing":                schema.StringAttribute{Optional: true, Computed: true, Description: "Billing period: hourly, monthly, or yearly."},
		"maintenance_day":        schema.Int64Attribute{Optional: true, Computed: true, Description: "Weekly maintenance day, where 0 is Sunday."},
		"maintenance_hour":       schema.Int64Attribute{Optional: true, Computed: true, Description: "Weekly maintenance hour in the database's timezone."},
		"disk_cap_gb":            schema.Int64Attribute{Optional: true, Computed: true, Description: "Maximum disk usage in GB, from 1 to 1000."},
		"eviction_policy":        schema.StringAttribute{Optional: true, Computed: true, Description: "Valkey eviction policy; not used by other engines."},
		"status":                 schema.StringAttribute{Computed: true, Description: "Current lifecycle state, including pending, active, maintenance, error, and deleting."},
		"engine_version":         schema.StringAttribute{Computed: true, Description: "Exact engine version currently running."},
		"region":                 schema.StringAttribute{Computed: true, Description: "Database region."},
		"connection_json":        schema.StringAttribute{Computed: true, Description: "Private connection endpoint settings as JSON. Passwords are available from the credentials data source."},
		"public_access_json":     schema.StringAttribute{Computed: true, Description: "Public-access settings as JSON."},
		"disk_json":              schema.StringAttribute{Computed: true, Description: "Disk capacity and usage details as JSON."},
		"users_json":             schema.StringAttribute{Computed: true, Description: "Database user access settings as JSON."},
		"logical_databases_json": schema.StringAttribute{Computed: true, Description: "Logical databases and extensions as JSON."},
		"created_at":             schema.StringAttribute{Computed: true, Description: "Database creation timestamp."},
		"database_json":          schema.StringAttribute{Computed: true, Description: "Complete database API object as JSON, including fields not separately modeled by this resource."},
	}}
}
func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, received %T", req.ProviderData))
		return
	}
	r.client = apiClient
}
func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured database resource", "The FFXF provider client is not configured.")
		return
	}
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"name": plan.Name.ValueString(), "vpc": plan.VPC.ValueInt64()}
	if !plan.Engine.IsNull() && !plan.Engine.IsUnknown() {
		body["engine"] = plan.Engine.ValueString()
	}
	if !plan.Version.IsNull() && !plan.Version.IsUnknown() {
		body["version"] = plan.Version.ValueString()
	}
	if !plan.SizeGB.IsNull() && !plan.SizeGB.IsUnknown() {
		body["size_gb"] = plan.SizeGB.ValueInt64()
	}
	if !plan.Billing.IsNull() && !plan.Billing.IsUnknown() {
		body["billing"] = plan.Billing.ValueString()
	}
	database, err := r.client.DatabaseCreate(ctx, body, uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF database", err.Error())
		return
	}
	ready, err := databaseWaitReady(ctx, r.client, database.Data)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for FFXF database readiness", err.Error())
		return
	}
	databaseSetResourceModel(ctx, &plan, ready, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	database, err := r.client.DatabaseGet(ctx, state.ID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF database", err.Error())
		return
	}
	databaseSetResourceModel(ctx, &state, database.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}
	for _, field := range []struct {
		name           string
		next, previous types.Int64
	}{{"maintenance_day", plan.MaintenanceDay, state.MaintenanceDay}, {"maintenance_hour", plan.MaintenanceHour, state.MaintenanceHour}, {"disk_cap_gb", plan.DiskCapGB, state.DiskCapGB}} {
		if !field.next.IsNull() && !field.next.IsUnknown() && (field.previous.IsNull() || field.next.ValueInt64() != field.previous.ValueInt64()) {
			body[field.name] = field.next.ValueInt64()
		}
	}
	if !plan.EvictionPolicy.IsNull() && !plan.EvictionPolicy.IsUnknown() && !plan.EvictionPolicy.Equal(state.EvictionPolicy) {
		body["eviction_policy"] = plan.EvictionPolicy.ValueString()
	}
	if len(body) > 0 {
		if _, err := r.client.DatabasePatch(ctx, state.ID.ValueInt64(), body); err != nil {
			resp.Diagnostics.AddError("Error updating FFXF database", err.Error())
			return
		}
	}
	if !plan.SizeGB.IsNull() && !plan.SizeGB.IsUnknown() && (state.SizeGB.IsNull() || plan.SizeGB.ValueInt64() != state.SizeGB.ValueInt64()) {
		if _, err := r.client.DatabaseResize(ctx, state.ID.ValueInt64(), plan.SizeGB.ValueInt64(), uuid.NewString()); err != nil {
			resp.Diagnostics.AddError("Error resizing FFXF database", err.Error())
			return
		}
		ready, err := databasePoll(ctx, r.client, state.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for FFXF database resize", err.Error())
			return
		}
		databaseSetResourceModel(ctx, &plan, ready, &resp.Diagnostics)
	} else {
		current, err := r.client.DatabaseGet(ctx, state.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Error reading updated FFXF database", err.Error())
			return
		}
		databaseSetResourceModel(ctx, &plan, current.Data, &resp.Diagnostics)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DatabaseDelete(ctx, state.ID.ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF database", err.Error())
	}
}

func databaseWaitReady(ctx context.Context, apiClient *client.Client, initial client.Database) (client.Database, error) {
	if initial.Status == "active" {
		return initial, nil
	}
	return databasePoll(ctx, apiClient, initial.ID)
}
func databasePoll(ctx context.Context, apiClient *client.Client, id int64) (client.Database, error) {
	pollCtx, cancel := context.WithTimeout(ctx, databasePollTimeout)
	defer cancel()
	ticker := time.NewTicker(databasePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-pollCtx.Done():
			return client.Database{}, fmt.Errorf("database %d did not become active before timeout: %w", id, pollCtx.Err())
		case <-ticker.C:
			database, err := apiClient.DatabaseGet(pollCtx, id)
			if err != nil {
				return client.Database{}, err
			}
			switch database.Data.Status {
			case "active":
				return database.Data, nil
			case "error":
				return client.Database{}, fmt.Errorf("database %d entered error state", id)
			case "pending", "restoring", "maintenance":
			default:
				return client.Database{}, fmt.Errorf("database %d entered unexpected state %q", id, database.Data.Status)
			}
		}
	}
}
func databaseSetResourceModel(ctx context.Context, model *databaseResourceModel, value client.Database, diagnostics *diag.Diagnostics) {
	model.ID = types.Int64Value(value.ID)
	model.Name = types.StringValue(value.Name)
	model.Status = types.StringValue(value.Status)
	model.Engine = types.StringValue(value.Engine)
	model.Version = types.StringValue(value.Version)
	model.VPC = types.Int64Value(value.VPC)
	if value.SizeGB == nil {
		model.SizeGB = types.Int64Null()
	} else {
		model.SizeGB = types.Int64Value(*value.SizeGB)
	}
	model.EngineVersion = nullableString(value.EngineVersion)
	model.Region = nullableString(value.Region)
	model.CreatedAt = types.StringValue(value.CreatedAt)
	model.DatabaseJSON = databaseJSONValue(value.DatabaseJSON, diagnostics)
	model.EvictionPolicy = nullableString(value.EvictionPolicy)
	model.ConnectionJSON = databaseJSONValue(value.Connection, diagnostics)
	model.PublicAccessJSON = databaseJSONValue(value.PublicAccess, diagnostics)
	model.DiskJSON = databaseJSONValue(value.Disk, diagnostics)
	users, err := json.Marshal(value.Users)
	if err != nil {
		diagnostics.AddError("Unable to encode database users", err.Error())
	} else {
		model.UsersJSON = types.StringValue(string(users))
	}
	databases, err := json.Marshal(value.Databases)
	if err != nil {
		diagnostics.AddError("Unable to encode logical databases", err.Error())
	} else {
		model.LogicalDatabasesJSON = types.StringValue(string(databases))
	}
	var maintenance struct {
		Day  int64 `json:"day"`
		Hour int64 `json:"hour"`
	}
	if len(value.Maintenance) > 0 && string(value.Maintenance) != "null" {
		if err := json.Unmarshal(value.Maintenance, &maintenance); err != nil {
			diagnostics.AddError("Unable to decode database maintenance", err.Error())
		} else {
			model.MaintenanceDay = types.Int64Value(maintenance.Day)
			model.MaintenanceHour = types.Int64Value(maintenance.Hour)
		}
	}
	if len(value.Billing) > 0 && string(value.Billing) != "null" {
		var billing struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(value.Billing, &billing); err != nil {
			diagnostics.AddError("Unable to decode database billing", err.Error())
		} else if billing.Mode != "" {
			model.Billing = types.StringValue(billing.Mode)
		}
	}
	var disk struct {
		CapGB int64 `json:"cap_gb"`
	}
	if len(value.Disk) > 0 && string(value.Disk) != "null" {
		if err := json.Unmarshal(value.Disk, &disk); err != nil {
			diagnostics.AddError("Unable to decode database disk", err.Error())
		} else {
			model.DiskCapGB = types.Int64Value(disk.CapGB)
		}
	}
}
func databaseJSONValue(raw json.RawMessage, diagnostics *diag.Diagnostics) types.String {
	if len(raw) == 0 || string(raw) == "null" {
		return types.StringNull()
	}
	if !json.Valid(raw) {
		diagnostics.AddError("Invalid database JSON", "The database API returned malformed JSON.")
		return types.StringNull()
	}
	return types.StringValue(string(raw))
}

func (r *databaseDerivedResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_" + r.kind
}
func (r *databaseDerivedResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	attributes := map[string]schema.Attribute{
		"id":                 schema.Int64Attribute{Computed: true, Description: "ID of the newly created database."},
		"source_database_id": schema.Int64Attribute{Required: true, Description: "Source database ID.", PlanModifiers: replaceInt},
		"name":               schema.StringAttribute{Required: true, Description: "Name for the new database.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"vpc":                schema.Int64Attribute{Optional: true, Computed: true, Description: "Private network ID. Defaults to the source database network.", PlanModifiers: replaceInt},
		"size_gb":            schema.Int64Attribute{Optional: true, Computed: true, Description: "Memory size in GB. Defaults to the source database size.", PlanModifiers: replaceInt},
		"status":             schema.StringAttribute{Computed: true, Description: "Current lifecycle state of the created database."},
		"engine":             schema.StringAttribute{Computed: true, Description: "Database engine inherited from the source."},
		"version":            schema.StringAttribute{Computed: true, Description: "Database major version inherited from the source."},
		"region":             schema.StringAttribute{Computed: true, Description: "Region of the created database."},
		"database_json":      schema.StringAttribute{Computed: true, Description: "Complete resulting database API object as JSON."},
	}
	if r.kind == "restore" {
		attributes["at"] = schema.StringAttribute{Required: true, Description: "Point in time to restore in RFC3339 format.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
	}
	resp.Schema = schema.Schema{Description: fmt.Sprintf("Creates a new database from a source using the FFXF %s API. Database features are in beta; an account without database access receives HTTP 403.", r.kind), Attributes: attributes}
}
func (r *databaseDerivedResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, received %T", req.ProviderData))
		return
	}
	r.client = apiClient
}
func (r *databaseDerivedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := plan.Attributes()
	sourceID := attrs["source_database_id"].(types.Int64).ValueInt64()
	body := map[string]any{"name": attrs["name"].(types.String).ValueString()}
	if value, ok := attrs["vpc"].(types.Int64); ok && !value.IsNull() && !value.IsUnknown() {
		body["vpc"] = value.ValueInt64()
	}
	if value, ok := attrs["size_gb"].(types.Int64); ok && !value.IsNull() && !value.IsUnknown() {
		body["size_gb"] = value.ValueInt64()
	}
	if r.kind == "restore" {
		body["at"] = attrs["at"].(types.String).ValueString()
	}
	operation := map[string]string{"fork": "forks", "restore": "restore"}[r.kind]
	created, err := r.client.DatabaseDerivedCreate(ctx, sourceID, operation, body, uuid.NewString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF database "+r.kind, err.Error())
		return
	}
	ready, err := databaseWaitReady(ctx, r.client, created.Data)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for FFXF database readiness", err.Error())
		return
	}
	databaseDerivedSetValues(attrs, ready, &resp.Diagnostics)
	databaseDerivedSetState(ctx, &resp.Diagnostics, &resp.State, plan, attrs)
}
func (r *databaseDerivedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := state.Attributes()
	database, err := r.client.DatabaseGet(ctx, attrs["id"].(types.Int64).ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF derived database", err.Error())
		return
	}
	databaseDerivedSetValues(attrs, database.Data, &resp.Diagnostics)
	databaseDerivedSetState(ctx, &resp.Diagnostics, &resp.State, state, attrs)
}
func (r *databaseDerivedResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *databaseDerivedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DatabaseDelete(ctx, state.Attributes()["id"].(types.Int64).ValueInt64())
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF derived database", err.Error())
	}
}
func databaseDerivedSetValues(attributes map[string]attr.Value, value client.Database, diagnostics *diag.Diagnostics) {
	attributes["id"] = types.Int64Value(value.ID)
	attributes["status"] = types.StringValue(value.Status)
	attributes["engine"] = types.StringValue(value.Engine)
	attributes["version"] = types.StringValue(value.Version)
	attributes["vpc"] = types.Int64Value(value.VPC)
	if value.SizeGB == nil {
		attributes["size_gb"] = types.Int64Null()
	} else {
		attributes["size_gb"] = types.Int64Value(*value.SizeGB)
	}
	attributes["region"] = nullableString(value.Region)
	attributes["database_json"] = databaseJSONValue(value.DatabaseJSON, diagnostics)
}
func databaseDerivedSetState(ctx context.Context, diagnostics *diag.Diagnostics, state *tfsdk.State, current types.Object, attributes map[string]attr.Value) {
	value, valueDiags := types.ObjectValue(current.AttributeTypes(ctx), attributes)
	diagnostics.Append(valueDiags...)
	if diagnostics.HasError() {
		return
	}
	diagnostics.Append(state.Set(ctx, &value)...)
}

type databaseChildResource struct {
	client *client.Client
	kind   string
}
type databaseDetachResource struct{ client *client.Client }
type databaseDetachResourceModel struct {
	ID         types.String `tfsdk:"id"`
	DatabaseID types.Int64  `tfsdk:"database_id"`
}

func databaseUserResourceConstructor() resource.Resource { return &databaseChildResource{kind: "user"} }
func databaseLogicalDatabaseResourceConstructor() resource.Resource {
	return &databaseChildResource{kind: "logical_database"}
}
func databaseExtensionResourceConstructor() resource.Resource {
	return &databaseChildResource{kind: "extension"}
}
func databasePublicAccessResourceConstructor() resource.Resource {
	return &databaseChildResource{kind: "public_access"}
}
func databaseBackupResourceConstructor() resource.Resource {
	return &databaseChildResource{kind: "backup"}
}
func databaseDetachResourceConstructor() resource.Resource { return &databaseDetachResource{} }

func (r *databaseChildResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_" + r.kind
}
func (r *databaseChildResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	replaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	attributes := map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, Description: "Terraform identifier for this managed database object."}, "database_id": schema.Int64Attribute{Required: true, Description: "Parent database ID.", PlanModifiers: replaceInt}}
	switch r.kind {
	case "user":
		attributes["username"] = schema.StringAttribute{Required: true, Description: "Username to create or manage.", PlanModifiers: replaceString}
		attributes["databases"] = schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Logical databases this user can access; null grants the API default."}
		attributes["read_only"] = schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the user is restricted to read-only access."}
		attributes["password_rotation_trigger"] = schema.StringAttribute{Optional: true, Description: "Change this value to request a password reset."}
		attributes["password"] = schema.StringAttribute{Computed: true, Sensitive: true, Description: "Password returned by create or password reset. The API does not return it on subsequent reads."}
	case "logical_database":
		attributes["name"] = schema.StringAttribute{Required: true, Description: "Logical database name, up to 63 characters.", PlanModifiers: replaceString}
	case "extension":
		attributes["name"] = schema.StringAttribute{Required: true, Description: "Logical database name on which to enable the extension.", PlanModifiers: replaceString}
		attributes["extension"] = schema.StringAttribute{Required: true, Description: "Extension name supported by the database engine.", PlanModifiers: replaceString}
	case "public_access":
		attributes["enabled"] = schema.BoolAttribute{Required: true, Description: "Whether the database accepts public connections."}
		attributes["sources"] = schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: "Allowed IPv4 or IPv6 addresses/CIDRs. An empty list allows all sources."}
	case "backup":
	}
	description := fmt.Sprintf("Manages the %s feature of an FFXF Cloud database. Database features are in beta; accounts without access receive HTTP 403.", r.kind)
	if r.kind == "backup" {
		description += " Creating a manual backup is append-only; the API has no delete endpoint, so Terraform deletion only removes the resource from state."
	}
	resp.Schema = schema.Schema{Description: description, Attributes: attributes}
}
func (r *databaseChildResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, received %T", req.ProviderData))
		return
	}
	r.client = apiClient
}
func (r *databaseChildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := plan.Attributes()
	id := attrs["database_id"].(types.Int64).ValueInt64()
	var err error
	switch r.kind {
	case "user":
		username := attrs["username"].(types.String).ValueString()
		body := map[string]any{"username": username}
		if value, ok := attrs["databases"].(types.List); ok && !value.IsNull() && !value.IsUnknown() {
			values, diags := databaseStringListFromValue(ctx, value)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["databases"] = values
		}
		if value, ok := attrs["read_only"].(types.Bool); ok && !value.IsNull() && !value.IsUnknown() {
			body["read_only"] = value.ValueBool()
		}
		var result map[string]json.RawMessage
		err = r.client.DatabaseCall(ctx, http.MethodPost, r.client.DatabasePath(id, "users"), body, &result, []int{http.StatusCreated}, nil)
		if err == nil {
			password, passwordErr := databaseCredentialPassword(result)
			if passwordErr != nil {
				resp.Diagnostics.AddError("Invalid FFXF database user response", passwordErr.Error())
				return
			}
			attrs["password"] = password
		}
		attrs["id"] = types.StringValue(fmt.Sprintf("%d/%s", id, username))
	case "logical_database":
		name := attrs["name"].(types.String).ValueString()
		err = r.client.DatabaseCall(ctx, http.MethodPost, r.client.DatabasePath(id, "databases"), map[string]string{"name": name}, nil, []int{http.StatusCreated}, nil)
		attrs["id"] = types.StringValue(fmt.Sprintf("%d/%s", id, name))
	case "extension":
		name := attrs["name"].(types.String).ValueString()
		extension := attrs["extension"].(types.String).ValueString()
		err = r.client.DatabaseCall(ctx, http.MethodPost, r.client.DatabaseLogicalPath(id, name, "extensions"), map[string]string{"extension": extension}, nil, []int{http.StatusOK}, nil)
		attrs["id"] = types.StringValue(fmt.Sprintf("%d/%s/%s", id, name, extension))
	case "public_access":
		body := map[string]any{"enabled": attrs["enabled"].(types.Bool).ValueBool()}
		if value, ok := attrs["sources"].(types.List); ok && !value.IsNull() && !value.IsUnknown() {
			values, diags := databaseStringListFromValue(ctx, value)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["sources"] = values
		}
		err = r.client.DatabaseCall(ctx, http.MethodPut, r.client.DatabasePath(id, "public-access"), body, nil, []int{http.StatusOK}, nil)
		attrs["id"] = types.StringValue(fmt.Sprint(id))
	case "backup":
		err = r.client.DatabaseCall(ctx, http.MethodPost, r.client.DatabasePath(id, "backups"), nil, nil, []int{http.StatusAccepted}, nil)
		attrs["id"] = types.StringValue(fmt.Sprint(id))
	default:
		err = fmt.Errorf("unsupported database resource %q", r.kind)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating FFXF database "+r.kind, err.Error())
		return
	}
	databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, plan, attrs)
}
func (r *databaseChildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := state.Attributes()
	database, err := r.client.DatabaseGet(ctx, attrs["database_id"].(types.Int64).ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading FFXF database "+r.kind, err.Error())
		return
	}
	switch r.kind {
	case "user":
		username := attrs["username"].(types.String).ValueString()
		for _, user := range database.Data.Users {
			if user.Username == username {
				attrs["read_only"] = types.BoolValue(user.ReadOnly)
				if user.Databases == nil {
					attrs["databases"] = types.ListNull(types.StringType)
				} else {
					list, diags := types.ListValueFrom(ctx, types.StringType, user.Databases)
					resp.Diagnostics.Append(diags...)
					attrs["databases"] = list
				}
				databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, state, attrs)
				return
			}
		}
		resp.State.RemoveResource(ctx)
	case "logical_database":
		name := attrs["name"].(types.String).ValueString()
		for _, value := range database.Data.Databases {
			if value.Name == name {
				databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, state, attrs)
				return
			}
		}
		resp.State.RemoveResource(ctx)
	case "extension":
		name, extension := attrs["name"].(types.String).ValueString(), attrs["extension"].(types.String).ValueString()
		for _, value := range database.Data.Databases {
			if value.Name == name {
				for _, item := range value.Extensions {
					if item == extension {
						databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, state, attrs)
						return
					}
				}
			}
		}
		resp.State.RemoveResource(ctx)
	case "public_access":
		var access struct {
			Enabled bool     `json:"enabled"`
			Sources []string `json:"sources"`
		}
		if err := json.Unmarshal(database.Data.PublicAccess, &access); err != nil {
			resp.Diagnostics.AddError("Unable to decode database public access", err.Error())
			return
		}
		attrs["enabled"] = types.BoolValue(access.Enabled)
		list, diags := types.ListValueFrom(ctx, types.StringType, access.Sources)
		resp.Diagnostics.Append(diags...)
		attrs["sources"] = list
		databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, state, attrs)
	case "backup":
		databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, state, attrs)
	}
}
func (r *databaseChildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	planAttrs, stateAttrs := plan.Attributes(), state.Attributes()
	id := stateAttrs["database_id"].(types.Int64).ValueInt64()
	switch r.kind {
	case "user":
		username := stateAttrs["username"].(types.String).ValueString()
		nextDB, priorDB := planAttrs["databases"].(types.List), stateAttrs["databases"].(types.List)
		nextReadOnly, priorReadOnly := planAttrs["read_only"].(types.Bool), stateAttrs["read_only"].(types.Bool)
		if !nextDB.Equal(priorDB) || !nextReadOnly.Equal(priorReadOnly) {
			body := map[string]any{}
			if !nextDB.IsNull() && !nextDB.IsUnknown() {
				values, diags := databaseStringListFromValue(ctx, nextDB)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				body["databases"] = values
			}
			if !nextReadOnly.IsNull() && !nextReadOnly.IsUnknown() {
				body["read_only"] = nextReadOnly.ValueBool()
			}
			if err := r.client.DatabaseCall(ctx, http.MethodPatch, r.client.DatabaseUserPath(id, username, ""), body, nil, []int{http.StatusOK}, nil); err != nil {
				resp.Diagnostics.AddError("Error updating FFXF database user", err.Error())
				return
			}
		}
		if !planAttrs["password_rotation_trigger"].Equal(stateAttrs["password_rotation_trigger"]) {
			var result map[string]json.RawMessage
			err := r.client.DatabaseCall(ctx, http.MethodPost, r.client.DatabaseUserPath(id, username, "reset-password"), nil, &result, []int{http.StatusOK}, nil)
			if err != nil {
				resp.Diagnostics.AddError("Error rotating FFXF database user password", err.Error())
				return
			}
			password, passwordErr := databaseCredentialPassword(result)
			if passwordErr != nil {
				resp.Diagnostics.AddError("Invalid FFXF password reset response", passwordErr.Error())
				return
			}
			planAttrs["password"] = password
		} else {
			planAttrs["password"] = stateAttrs["password"]
		}
	case "public_access":
		body := map[string]any{"enabled": planAttrs["enabled"].(types.Bool).ValueBool()}
		if value := planAttrs["sources"].(types.List); !value.IsNull() && !value.IsUnknown() {
			values, diags := databaseStringListFromValue(ctx, value)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["sources"] = values
		}
		if err := r.client.DatabaseCall(ctx, http.MethodPut, r.client.DatabasePath(id, "public-access"), body, nil, []int{http.StatusOK}, nil); err != nil {
			resp.Diagnostics.AddError("Error updating FFXF database public access", err.Error())
			return
		}
	}
	databaseChildSetState(ctx, &resp.Diagnostics, &resp.State, plan, planAttrs)
}
func (r *databaseChildResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs := state.Attributes()
	id := attrs["database_id"].(types.Int64).ValueInt64()
	var err error
	switch r.kind {
	case "user":
		err = r.client.DatabaseCall(ctx, http.MethodDelete, r.client.DatabaseUserPath(id, attrs["username"].(types.String).ValueString(), ""), nil, nil, []int{http.StatusNoContent}, nil)
	case "logical_database":
		err = r.client.DatabaseCall(ctx, http.MethodDelete, r.client.DatabaseLogicalPath(id, attrs["name"].(types.String).ValueString(), ""), nil, nil, []int{http.StatusNoContent}, nil)
	case "extension":
		err = r.client.DatabaseCall(ctx, http.MethodDelete, r.client.DatabaseLogicalPath(id, attrs["name"].(types.String).ValueString(), "extensions")+"/"+url.PathEscape(attrs["extension"].(types.String).ValueString()), nil, nil, []int{http.StatusOK}, nil)
	case "public_access":
		err = r.client.DatabaseCall(ctx, http.MethodPut, r.client.DatabasePath(id, "public-access"), map[string]any{"enabled": false, "sources": []string{}}, nil, []int{http.StatusOK}, nil)
	case "backup":
		return
	}
	if err != nil && !errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.Diagnostics.AddError("Error deleting FFXF database "+r.kind, err.Error())
	}
}
func databaseStringListFromValue(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	var values []string
	diagnostics := value.ElementsAs(ctx, &values, false)
	return values, diagnostics
}
func databaseCredentialPassword(response map[string]json.RawMessage) (types.String, error) {
	var data struct {
		Password *string `json:"password"`
	}
	if err := json.Unmarshal(response["data"], &data); err != nil {
		return types.StringNull(), err
	}
	if data.Password == nil {
		return types.StringNull(), fmt.Errorf("API response omitted the password")
	}
	return types.StringValue(*data.Password), nil
}
func databaseChildSetState(ctx context.Context, diagnostics *diag.Diagnostics, state *tfsdk.State, current types.Object, attributes map[string]attr.Value) {
	value, valueDiags := types.ObjectValue(current.AttributeTypes(ctx), attributes)
	diagnostics.Append(valueDiags...)
	if diagnostics.HasError() {
		return
	}
	diagnostics.Append(state.Set(ctx, &value)...)
}

func (r *databaseDetachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_detach"
}
func (r *databaseDetachResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Detaches a fork from its parent as a one-shot operation. The database is unavailable for about 30 seconds; backups begin afterward. This operation cannot be reversed. Database features are in beta; accounts without access receive HTTP 403.", Attributes: map[string]schema.Attribute{
		"id":          schema.StringAttribute{Computed: true, Description: "Database ID for the detach operation."},
		"database_id": schema.Int64Attribute{Required: true, Description: "ID of the linked fork to detach.", PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
	}}
}
func (r *databaseDetachResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, received %T", req.ProviderData))
		return
	}
	r.client = apiClient
}
func (r *databaseDetachResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseDetachResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DatabaseCall(ctx, http.MethodPost, r.client.DatabasePath(plan.DatabaseID.ValueInt64(), "detach"), nil, nil, []int{http.StatusAccepted}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error detaching FFXF database fork", err.Error())
		return
	}
	plan.ID = types.StringValue(fmt.Sprint(plan.DatabaseID.ValueInt64()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *databaseDetachResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseDetachResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DatabaseGet(ctx, state.DatabaseID.ValueInt64())
	if errors.Is(err, client.ErrNetworkResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading detached FFXF database", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (*databaseDetachResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (*databaseDetachResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type databaseConfigValidator struct{ kind string }

func (databaseConfigValidator) Description(context.Context) string {
	return "Validates database engine, billing, maintenance, size, metrics range, and restore timestamp values."
}
func (databaseConfigValidator) MarkdownDescription(ctx context.Context) string {
	return (databaseConfigValidator{}).Description(ctx)
}
func (v databaseConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config types.Object
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.IsNull() || config.IsUnknown() {
		return
	}
	attrs := config.Attributes()
	switch v.kind {
	case "database":
		databaseValidateChoice(&resp.Diagnostics, attrs, "engine", []string{"postgresql", "mysql", "mariadb", "valkey"})
		databaseValidateChoice(&resp.Diagnostics, attrs, "billing", []string{"hourly", "monthly", "yearly"})
		databaseValidatePositive(&resp.Diagnostics, attrs, "size_gb")
		databaseValidateInt(&resp.Diagnostics, attrs, "maintenance_day", 0, 6)
		databaseValidateInt(&resp.Diagnostics, attrs, "maintenance_hour", 0, 23)
		databaseValidateInt(&resp.Diagnostics, attrs, "disk_cap_gb", 1, 1000)
		databaseValidateChoice(&resp.Diagnostics, attrs, "eviction_policy", []string{"noeviction", "allkeys-lru", "allkeys-lfu", "allkeys-random", "volatile-lru", "volatile-lfu", "volatile-random", "volatile-ttl"})
	case "restore":
		if value, ok := attrs["at"].(types.String); ok && !value.IsNull() && !value.IsUnknown() {
			if _, err := time.Parse(time.RFC3339, value.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("at"), "Invalid restore timestamp", fmt.Sprintf("The restore timestamp must be RFC3339 date-time: %v", err))
			}
		}
	}
}
func (v databaseConfigValidator) ValidateDataSource(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	if v.kind != "metrics" {
		return
	}
	var config types.Object
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.IsNull() || config.IsUnknown() {
		return
	}
	databaseValidateChoice(&resp.Diagnostics, config.Attributes(), "range", []string{"1h", "24h", "7d"})
}
func databaseValidateChoice(diagnostics *diag.Diagnostics, attributes map[string]attr.Value, name string, allowed []string) {
	value, ok := attributes[name].(types.String)
	if !ok || value.IsNull() || value.IsUnknown() {
		return
	}
	for _, option := range allowed {
		if value.ValueString() == option {
			return
		}
	}
	detail := "Choose one of: " + strings.Join(allowed, ", ") + "."
	diagnostics.AddAttributeError(path.Root(name), "Invalid "+name, detail)
}
func databaseValidatePositive(diagnostics *diag.Diagnostics, attributes map[string]attr.Value, name string) {
	value, ok := attributes[name].(types.Int64)
	if !ok || value.IsNull() || value.IsUnknown() {
		return
	}
	if value.ValueInt64() < 1 {
		diagnostics.AddAttributeError(path.Root(name), "Invalid "+name, "The value must be greater than zero.")
	}
}

func databaseValidateInt(diagnostics *diag.Diagnostics, attributes map[string]attr.Value, name string, min, max int64) {
	value, ok := attributes[name].(types.Int64)
	if !ok || value.IsNull() || value.IsUnknown() {
		return
	}
	if value.ValueInt64() < min || value.ValueInt64() > max {
		diagnostics.AddAttributeError(path.Root(name), "Invalid "+name, fmt.Sprintf("The value must be between %d and %d.", min, max))
	}
}

func (r *databaseResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{databaseConfigValidator{kind: "database"}}
}
func (r *databaseDerivedResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	if r.kind == "restore" {
		return []resource.ConfigValidator{databaseConfigValidator{kind: "restore"}}
	}
	return nil
}
func (d *databaseDataSource) ConfigValidators(context.Context) []datasource.ConfigValidator {
	if d.kind == "_metrics" {
		return []datasource.ConfigValidator{databaseConfigValidator{kind: "metrics"}}
	}
	return nil
}

var _ resource.ConfigValidator = databaseConfigValidator{}
var _ datasource.ConfigValidator = databaseConfigValidator{}

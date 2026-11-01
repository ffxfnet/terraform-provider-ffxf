package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type accountAreaEnumValidator struct{ allowed []string }

func (v accountAreaEnumValidator) Description(context.Context) string {
	return "Value must match one of: " + strings.Join(v.allowed, ", ") + "."
}
func (v accountAreaEnumValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v accountAreaEnumValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, value := range v.allowed {
		if req.ConfigValue.ValueString() == value {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", fmt.Sprintf("Value %q must be one of: %s.", req.ConfigValue.ValueString(), strings.Join(v.allowed, ", ")))
}

type accountAreaInt64RangeValidator struct{ minimum, maximum int64 }

func (v accountAreaInt64RangeValidator) Description(context.Context) string {
	return fmt.Sprintf("Value must be between %d and %d.", v.minimum, v.maximum)
}
func (v accountAreaInt64RangeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v accountAreaInt64RangeValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueInt64()
	if value < v.minimum || value > v.maximum {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", fmt.Sprintf("Value %d must be between %d and %d.", value, v.minimum, v.maximum))
	}
}

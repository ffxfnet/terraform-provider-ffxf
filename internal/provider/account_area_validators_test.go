package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountAreaEnumValidator(t *testing.T) {
	v := accountAreaEnumValidator{allowed: []string{"hour", "vm"}}
	for _, tc := range []struct {
		value   types.String
		invalid bool
	}{{types.StringValue("hour"), false}, {types.StringValue("invalid"), true}, {types.StringNull(), false}, {types.StringUnknown(), false}} {
		response := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: tc.value}, &response)
		if response.Diagnostics.HasError() != tc.invalid {
			t.Fatalf("value %s diagnostics = %v", tc.value.String(), response.Diagnostics)
		}
	}
	if v.Description(context.Background()) == "" || v.MarkdownDescription(context.Background()) == "" {
		t.Fatal("validator descriptions must be populated")
	}
}

func TestAccountAreaInt64RangeValidator(t *testing.T) {
	v := accountAreaInt64RangeValidator{minimum: 1, maximum: 100}
	if v.Description(context.Background()) == "" || v.MarkdownDescription(context.Background()) == "" {
		t.Fatal("range validator descriptions must be populated")
	}
	for _, tc := range []struct {
		value   types.Int64
		invalid bool
	}{{types.Int64Value(1), false}, {types.Int64Value(101), true}, {types.Int64Null(), false}, {types.Int64Unknown(), false}} {
		response := validator.Int64Response{}
		v.ValidateInt64(context.Background(), validator.Int64Request{ConfigValue: tc.value}, &response)
		if response.Diagnostics.HasError() != tc.invalid {
			t.Fatalf("value %s diagnostics = %v", tc.value.String(), response.Diagnostics)
		}
	}
}

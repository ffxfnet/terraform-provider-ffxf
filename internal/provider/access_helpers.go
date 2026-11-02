package provider

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// accessJSONEqual compares two JSON texts ignoring whitespace and key order.
func accessJSONEqual(a, b string) bool {
	var left, right any
	if json.Unmarshal([]byte(a), &left) != nil || json.Unmarshal([]byte(b), &right) != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}

func accessCompactJSON(raw json.RawMessage) string {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return string(raw)
	}
	return buffer.String()
}

// accessOptionalString keeps a null configuration null when the API reports an empty value.
func accessOptionalString(prior types.String, api *string) types.String {
	if api == nil || (*api == "" && prior.IsNull()) {
		return types.StringNull()
	}
	return types.StringValue(*api)
}

func accessRoleID(ref *client.AccessRoleRef) types.Int64 {
	if ref == nil {
		return types.Int64Null()
	}
	return types.Int64Value(ref.ID)
}

func accessDate(prior types.String, api *string) types.String {
	if api == nil {
		return types.StringNull()
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	if len(*api) >= 10 {
		return types.StringValue((*api)[:10])
	}
	return types.StringValue(*api)
}

// accessRoleValue converts a configured role (id or name) into the JSON value the API accepts.
func accessRoleValue(role string) any {
	if id, err := strconv.ParseInt(role, 10, 64); err == nil {
		return id
	}
	return role
}

func replaceUnknownString() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
}

func listRequiresReplace() planmodifier.List { return listplanmodifier.RequiresReplace() }

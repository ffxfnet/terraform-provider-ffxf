package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func accessAreaResources() []func() resource.Resource {
	return []func() resource.Resource{NewAccessPolicyResource, NewAccessRoleResource, NewAccessAPIKeyResource, NewAccessOIDCBindingResource}
}

func accessAreaDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{NewAccessPoliciesDataSource, NewAccessRolesDataSource, NewAccessOIDCProvidersDataSource, NewAccessAPIKeysDataSource}
}

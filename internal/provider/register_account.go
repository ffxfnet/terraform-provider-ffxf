package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func accountAreaResources() []func() resource.Resource {
	return []func() resource.Resource{NewAccountAreaSSHKeyResource, NewAccountAreaVPCMemberResource, NewAccountAreaBudgetResource}
}

func accountAreaDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{NewAccountAreaPlansDataSource, NewAccountAreaImagesDataSource, NewAccountAreaVPCsDataSource, NewAccountAreaSSHKeysDataSource, NewAccountAreaAccountDataSource, NewAccountAreaUsageDataSource, NewAccountAreaRunnersDataSource, NewAccountAreaRunnerJobsDataSource}
}

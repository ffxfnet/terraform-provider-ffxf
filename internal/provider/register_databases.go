package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func databaseAreaResources() []func() resource.Resource {
	return []func() resource.Resource{
		databaseResourceConstructor,
		databaseUserResourceConstructor,
		databaseLogicalDatabaseResourceConstructor,
		databaseExtensionResourceConstructor,
		databasePublicAccessResourceConstructor,
		databaseBackupResourceConstructor,
		databaseForkResourceConstructor,
		databaseRestoreResourceConstructor,
		databaseDetachResourceConstructor,
	}
}

func databaseAreaDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		databaseDataSourceConstructor("s"),
		databaseDataSourceConstructor("_options"),
		databaseDataSourceConstructor("_credentials"),
		databaseDataSourceConstructor("_backups"),
		databaseDataSourceConstructor("_metrics"),
		databaseDataSourceConstructor("_queries"),
	}
}

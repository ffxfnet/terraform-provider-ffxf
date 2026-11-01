package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// vmAreaResources registers supported VM lifecycle resources. Console sessions are single-use side effects, and backup deletion has no matching create lifecycle.
func vmAreaResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewVMAreaIPv6AddressResource,
		NewVMAreaRDNSResource,
		NewVMAreaPowerStateResource,
		NewVMAreaReinstallResource,
		NewVMAreaBackupRestoreResource,
	}
}

func vmAreaDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVMAreaVMsDataSource,
		NewVMAreaVMNetworkDataSource,
		NewVMAreaVMMetricsDataSource,
		NewVMAreaVMBackupsDataSource,
		NewVMAreaVMActionsDataSource,
	}
}

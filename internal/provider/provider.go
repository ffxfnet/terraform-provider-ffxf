package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

var _ provider.Provider = &FFXFProvider{}

type FFXFProvider struct {
	version string
}

type FFXFProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &FFXFProvider{
			version: version,
		}
	}
}

func (p *FFXFProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ffxf"
	resp.Version = p.version
}

func (p *FFXFProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Configure access to the FFXF Cloud API. Supply an API token directly or through the FFXF_TOKEN environment variable.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "FFXF Cloud API base URL. Defaults to https://api.ffxf.net/v1. Can also be set with FFXF_ENDPOINT.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "FFXF Cloud API token. If omitted, the provider reads FFXF_TOKEN from the environment.",
			},
		},
	}
}

func (p *FFXFProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data FFXFProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := data.Token.ValueString()
	if token == "" {
		token = os.Getenv("FFXF_TOKEN")
	}

	// 2. Validation si aucun token n'est fourni
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The API Token should be provided either by the parameter 'token' in the provider block, or via the environment variable FFXF_TOKEN.",
		)
		return
	}

	endpoint := data.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("FFXF_ENDPOINT")
	}

	apiClient := client.NewClient(endpoint, token)
	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *FFXFProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewVPCResource,
	}
}

func (p *FFXFProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRegionDataSource,
		NewPlanDataSource,
		NewImageDataSource,
	}
}

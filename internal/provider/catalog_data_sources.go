package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

var _ datasource.DataSource = &RegionDataSource{}
var _ datasource.DataSource = &PlanDataSource{}
var _ datasource.DataSource = &ImageDataSource{}

type RegionDataSource struct {
	client *client.Client
}

type RegionDataSourceModel struct {
	Slug    types.String `tfsdk:"slug"`
	Name    types.String `tfsdk:"name"`
	Country types.String `tfsdk:"country"`
	Status  types.String `tfsdk:"status"`
	IPv4    types.Bool   `tfsdk:"ipv4"`
	IPv6    types.Bool   `tfsdk:"ipv6"`
}

func NewRegionDataSource() datasource.DataSource {
	return &RegionDataSource{}
}

func (d *RegionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_region"
}

func (d *RegionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an FFXF Cloud region by slug.",
		Attributes: map[string]schema.Attribute{
			"slug":    schema.StringAttribute{Required: true, Description: "Stable region slug, such as montreal."},
			"name":    schema.StringAttribute{Computed: true, Description: "Display name of the region."},
			"country": schema.StringAttribute{Computed: true, Description: "Two-letter country code."},
			"status":  schema.StringAttribute{Computed: true, Description: "Availability status: available, limited, or unavailable."},
			"ipv4":    schema.BoolAttribute{Computed: true, Description: "Whether a public IPv4 address is currently available."},
			"ipv6":    schema.BoolAttribute{Computed: true, Description: "Whether IPv6 is available."},
		},
	}
}

func (d *RegionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.Client)
}

func (d *RegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RegionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	regions, err := d.client.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF region", err.Error())
		return
	}

	for _, region := range regions.Data {
		if region.Slug != data.Slug.ValueString() {
			continue
		}
		data.Name = types.StringValue(region.Name)
		data.Country = types.StringValue(region.Country)
		data.Status = types.StringValue(region.Status)
		data.IPv4 = types.BoolValue(region.IPv4)
		data.IPv6 = types.BoolValue(region.IPv6)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError("FFXF region not found", fmt.Sprintf("No region exists with slug %q.", data.Slug.ValueString()))
}

type PlanDataSource struct {
	client *client.Client
}

type PlanDataSourceModel struct {
	Slug        types.String  `tfsdk:"slug"`
	Name        types.String  `tfsdk:"name"`
	Description types.String  `tfsdk:"description"`
	VCPU        types.Int64   `tfsdk:"vcpu"`
	MemoryMB    types.Int64   `tfsdk:"memory_mb"`
	DiskGB      types.Int64   `tfsdk:"disk_gb"`
	TrafficTB   types.Float64 `tfsdk:"traffic_tb"`
	PortMbps    types.Int64   `tfsdk:"port_mbps"`
	Regions     types.List    `tfsdk:"regions"`
	Status      types.String  `tfsdk:"status"`
	Prices      types.List    `tfsdk:"prices"`
}

func NewPlanDataSource() datasource.DataSource {
	return &PlanDataSource{}
}

func (d *PlanDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plan"
}

func (d *PlanDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an FFXF Cloud compute plan by slug.",
		Attributes: map[string]schema.Attribute{
			"slug":        schema.StringAttribute{Required: true, Description: "Stable plan slug, such as nano."},
			"name":        schema.StringAttribute{Computed: true, Description: "Display name of the plan."},
			"description": schema.StringAttribute{Computed: true, Description: "Intended workload for the plan."},
			"vcpu":        schema.Int64Attribute{Computed: true, Description: "Number of virtual CPUs."},
			"memory_mb":   schema.Int64Attribute{Computed: true, Description: "Memory in MiB."},
			"disk_gb":     schema.Int64Attribute{Computed: true, Description: "Disk capacity in GiB."},
			"traffic_tb":  schema.Float64Attribute{Computed: true, Description: "Included traffic in TiB, or null when unmetered."},
			"port_mbps":   schema.Int64Attribute{Computed: true, Description: "Network port speed in Mbps."},
			"regions":     schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Region slugs where the plan is offered."},
			"status":      schema.StringAttribute{Computed: true, Description: "Availability status of the plan."},
			"prices": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Prices by currency. Monetary amounts are decimal strings to avoid rounding errors.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"currency":                schema.StringAttribute{Computed: true},
					"hourly":                  schema.StringAttribute{Computed: true},
					"hourly_month_equivalent": schema.StringAttribute{Computed: true},
					"hourly_stopped":          schema.StringAttribute{Computed: true},
					"monthly":                 schema.StringAttribute{Computed: true},
					"annual":                  schema.StringAttribute{Computed: true},
					"setup_fee":               schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *PlanDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.Client)
}

func (d *PlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PlanDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan, err := d.client.GetPlan(ctx, data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF plan", err.Error())
		return
	}

	data.Name = types.StringValue(plan.Data.Name)
	data.Description = types.StringValue(plan.Data.Description)
	data.VCPU = types.Int64Value(int64(plan.Data.VCPU))
	data.MemoryMB = types.Int64Value(int64(plan.Data.MemoryMB))
	data.DiskGB = types.Int64Value(int64(plan.Data.DiskGB))
	if plan.Data.TrafficTB == nil {
		data.TrafficTB = types.Float64Null()
	} else {
		data.TrafficTB = types.Float64Value(*plan.Data.TrafficTB)
	}
	data.PortMbps = types.Int64Value(int64(plan.Data.PortMbps))
	regions, diags := types.ListValueFrom(ctx, types.StringType, plan.Data.Regions)
	resp.Diagnostics.Append(diags...)
	data.Regions = regions
	data.Status = types.StringValue(plan.Data.Status)
	priceAttributeTypes := map[string]attr.Type{
		"currency":                types.StringType,
		"hourly":                  types.StringType,
		"hourly_month_equivalent": types.StringType,
		"hourly_stopped":          types.StringType,
		"monthly":                 types.StringType,
		"annual":                  types.StringType,
		"setup_fee":               types.StringType,
	}
	priceValues := make([]attr.Value, 0, len(plan.Data.Prices))
	for _, price := range plan.Data.Prices {
		hourlyStopped := types.StringNull()
		if price.HourlyStopped != nil {
			hourlyStopped = types.StringValue(*price.HourlyStopped)
		}
		priceValue, priceDiags := types.ObjectValue(priceAttributeTypes, map[string]attr.Value{
			"currency":                types.StringValue(price.Currency),
			"hourly":                  types.StringValue(price.Hourly),
			"hourly_month_equivalent": types.StringValue(price.HourlyMonthEquivalent),
			"hourly_stopped":          hourlyStopped,
			"monthly":                 types.StringValue(price.Monthly),
			"annual":                  types.StringValue(price.Annual),
			"setup_fee":               types.StringValue(price.SetupFee),
		})
		resp.Diagnostics.Append(priceDiags...)
		priceValues = append(priceValues, priceValue)
	}
	prices, diags := types.ListValue(types.ObjectType{AttrTypes: priceAttributeTypes}, priceValues)
	resp.Diagnostics.Append(diags...)
	data.Prices = prices
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type ImageDataSource struct {
	client *client.Client
}

type ImageDataSourceModel struct {
	Slug            types.String `tfsdk:"slug"`
	Name            types.String `tfsdk:"name"`
	Family          types.String `tfsdk:"family"`
	Category        types.String `tfsdk:"category"`
	Version         types.String `tfsdk:"version"`
	Status          types.String `tfsdk:"status"`
	Regions         types.List   `tfsdk:"regions"`
	MinDiskGB       types.Int64  `tfsdk:"min_disk_gb"`
	MinMemoryMB     types.Int64  `tfsdk:"min_memory_mb"`
	DefaultUser     types.String `tfsdk:"default_user"`
	SupportsSSHKeys types.Bool   `tfsdk:"supports_ssh_keys"`
}

func NewImageDataSource() datasource.DataSource {
	return &ImageDataSource{}
}

func (d *ImageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *ImageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an FFXF Cloud operating system or application image by slug.",
		Attributes: map[string]schema.Attribute{
			"slug":              schema.StringAttribute{Required: true, Description: "Stable image slug, such as debian-13."},
			"name":              schema.StringAttribute{Computed: true, Description: "Display name of the image."},
			"family":            schema.StringAttribute{Computed: true, Description: "Operating system or application family."},
			"category":          schema.StringAttribute{Computed: true, Description: "Image category: linux, windows, bsd, app, or other."},
			"version":           schema.StringAttribute{Computed: true, Description: "Image version, when applicable."},
			"status":            schema.StringAttribute{Computed: true, Description: "Availability status of the image."},
			"regions":           schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Region slugs where the image is offered."},
			"min_disk_gb":       schema.Int64Attribute{Computed: true, Description: "Minimum compatible disk size in GiB, when specified."},
			"min_memory_mb":     schema.Int64Attribute{Computed: true, Description: "Minimum compatible memory in MiB, when specified."},
			"default_user":      schema.StringAttribute{Computed: true, Description: "Default account name on the image."},
			"supports_ssh_keys": schema.BoolAttribute{Computed: true, Description: "Whether SSH key provisioning is supported."},
		},
	}
}

func (d *ImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*client.Client)
}

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := d.client.GetImage(ctx, data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF image", err.Error())
		return
	}

	data.Name = types.StringValue(image.Data.Name)
	data.Family = types.StringValue(image.Data.Family)
	data.Category = types.StringValue(image.Data.Category)
	if image.Data.Version == nil {
		data.Version = types.StringNull()
	} else {
		data.Version = types.StringValue(*image.Data.Version)
	}
	data.Status = types.StringValue(image.Data.Status)
	regions, diags := types.ListValueFrom(ctx, types.StringType, image.Data.Regions)
	resp.Diagnostics.Append(diags...)
	data.Regions = regions
	if image.Data.MinDiskGB == nil {
		data.MinDiskGB = types.Int64Null()
	} else {
		data.MinDiskGB = types.Int64Value(int64(*image.Data.MinDiskGB))
	}
	if image.Data.MinMemoryMB == nil {
		data.MinMemoryMB = types.Int64Null()
	} else {
		data.MinMemoryMB = types.Int64Value(int64(*image.Data.MinMemoryMB))
	}
	data.DefaultUser = types.StringValue(image.Data.DefaultUser)
	data.SupportsSSHKeys = types.BoolValue(image.Data.SupportsSSHKeys)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

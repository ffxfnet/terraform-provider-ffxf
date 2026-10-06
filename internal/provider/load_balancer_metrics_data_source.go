package provider

import (
	"context"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &LoadBalancerMetricsDataSource{}

type LoadBalancerMetricsDataSource struct{ client *client.Client }

type LoadBalancerMetricsDataSourceModel struct {
	LoadBalancerID types.Int64   `tfsdk:"load_balancer_id"`
	Range          types.String  `tfsdk:"range"`
	StepMinutes    types.Int64   `tfsdk:"step_minutes"`
	Points         types.List    `tfsdk:"points"`
	Requests       types.Int64   `tfsdk:"requests"`
	ErrorsPct      types.Float64 `tfsdk:"errors_pct"`
	Bytes          types.Int64   `tfsdk:"bytes"`
	SessionsMax    types.Int64   `tfsdk:"sessions_max"`
}

type LoadBalancerMetricPointModel struct {
	At          types.String `tfsdk:"at"`
	Requests    types.Int64  `tfsdk:"requests"`
	HTTP2xx     types.Int64  `tfsdk:"http_2xx"`
	HTTP3xx     types.Int64  `tfsdk:"http_3xx"`
	HTTP4xx     types.Int64  `tfsdk:"http_4xx"`
	HTTP5xx     types.Int64  `tfsdk:"http_5xx"`
	Connections types.Int64  `tfsdk:"connections"`
	BytesIn     types.Int64  `tfsdk:"bytes_in"`
	BytesOut    types.Int64  `tfsdk:"bytes_out"`
	SessionsMax types.Int64  `tfsdk:"sessions_max"`
}

func NewLoadBalancerMetricsDataSource() datasource.DataSource {
	return &LoadBalancerMetricsDataSource{}
}

func (d *LoadBalancerMetricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer_metrics"
}

func (d *LoadBalancerMetricsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads load-balancer request, traffic, connection, and session metrics for the selected time range.",
		Attributes: map[string]schema.Attribute{
			"load_balancer_id": schema.Int64Attribute{Required: true},
			"range":            schema.StringAttribute{Optional: true, Description: "1h, 24h, or 7d. Defaults to 1h."},
			"step_minutes":     schema.Int64Attribute{Computed: true},
			"points": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"at": schema.StringAttribute{Computed: true}, "requests": schema.Int64Attribute{Computed: true},
				"http_2xx": schema.Int64Attribute{Computed: true}, "http_3xx": schema.Int64Attribute{Computed: true},
				"http_4xx": schema.Int64Attribute{Computed: true}, "http_5xx": schema.Int64Attribute{Computed: true},
				"connections": schema.Int64Attribute{Computed: true}, "bytes_in": schema.Int64Attribute{Computed: true},
				"bytes_out": schema.Int64Attribute{Computed: true}, "sessions_max": schema.Int64Attribute{Computed: true},
			}}},
			"requests":     schema.Int64Attribute{Computed: true},
			"errors_pct":   schema.Float64Attribute{Computed: true},
			"bytes":        schema.Int64Attribute{Computed: true},
			"sessions_max": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *LoadBalancerMetricsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}

func (d *LoadBalancerMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LoadBalancerMetricsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	metrics, err := d.client.GetLoadBalancerMetrics(ctx, data.LoadBalancerID.ValueInt64(), data.Range.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF load-balancer metrics", err.Error())
		return
	}
	data.Range = types.StringValue(metrics.Data.Range)
	data.StepMinutes = types.Int64Value(int64(metrics.Data.StepMinutes))
	data.Requests = types.Int64Value(metrics.Data.Totals.Requests)
	data.ErrorsPct = types.Float64Value(metrics.Data.Totals.ErrorsPct)
	data.Bytes = types.Int64Value(metrics.Data.Totals.Bytes)
	data.SessionsMax = types.Int64Value(metrics.Data.Totals.SessionsMax)
	points := make([]LoadBalancerMetricPointModel, 0, len(metrics.Data.Points))
	for _, point := range metrics.Data.Points {
		points = append(points, LoadBalancerMetricPointModel{
			At: types.StringValue(point.At), Requests: types.Int64Value(point.Requests), HTTP2xx: types.Int64Value(point.HTTP2xx),
			HTTP3xx: types.Int64Value(point.HTTP3xx), HTTP4xx: types.Int64Value(point.HTTP4xx), HTTP5xx: types.Int64Value(point.HTTP5xx),
			Connections: types.Int64Value(point.Connections), BytesIn: types.Int64Value(point.BytesIn), BytesOut: types.Int64Value(point.BytesOut),
			SessionsMax: types.Int64Value(point.SessionsMax),
		})
	}
	pointTypes := map[string]attr.Type{
		"at": types.StringType, "requests": types.Int64Type, "http_2xx": types.Int64Type, "http_3xx": types.Int64Type,
		"http_4xx": types.Int64Type, "http_5xx": types.Int64Type, "connections": types.Int64Type,
		"bytes_in": types.Int64Type, "bytes_out": types.Int64Type, "sessions_max": types.Int64Type,
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: pointTypes}, points)
	resp.Diagnostics.Append(diags...)
	data.Points = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

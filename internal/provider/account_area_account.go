package provider

import (
	"context"
	"fmt"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*AccountAreaAccountDataSource)(nil)
var _ datasource.DataSource = (*AccountAreaUsageDataSource)(nil)
var _ resource.Resource = (*AccountAreaBudgetResource)(nil)

type AccountAreaAccountDataSource struct{ client *client.Client }
type AccountAreaUsageDataSource struct{ client *client.Client }
type AccountAreaBudgetResource struct{ client *client.Client }

func NewAccountAreaAccountDataSource() datasource.DataSource { return &AccountAreaAccountDataSource{} }
func NewAccountAreaUsageDataSource() datasource.DataSource   { return &AccountAreaUsageDataSource{} }
func NewAccountAreaBudgetResource() resource.Resource        { return &AccountAreaBudgetResource{} }

type AccountAreaAccountModel struct {
	Account types.Object `tfsdk:"account"`
}
type AccountAreaUsageModel struct {
	From     types.String `tfsdk:"from"`
	To       types.String `tfsdk:"to"`
	VMID     types.Int64  `tfsdk:"vm_id"`
	GroupBy  types.String `tfsdk:"group_by"`
	Records  types.List   `tfsdk:"records"`
	Total    types.String `tfsdk:"total"`
	Currency types.String `tfsdk:"currency"`
}
type AccountAreaBudgetModel struct {
	ID          types.String  `tfsdk:"id"`
	Monthly     types.String  `tfsdk:"monthly"`
	MonthToDate types.String  `tfsdk:"month_to_date"`
	Percent     types.Float64 `tfsdk:"percent"`
	CapPercent  types.Int64   `tfsdk:"cap_percent"`
}

func (d *AccountAreaAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}
func (d *AccountAreaAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Reads billing balance, usage, budget, quota, and auto-top-up details for the account.", Attributes: map[string]schema.Attribute{"account": schema.SingleNestedAttribute{Computed: true, Description: "Current account billing and quota information.", Attributes: accountAreaAccountAttributes()}}}
}
func (d *AccountAreaAccountDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.AccountAreaGetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF account", err.Error())
		return
	}
	v := result.Data
	credit := accountAreaObject(ctx, accountAreaCreditTypes(), map[string]attr.Value{"balance": types.StringValue(string(v.Credit.Balance)), "paid": types.StringValue(string(v.Credit.Paid)), "gifted": types.StringValue(string(v.Credit.Gifted)), "gifted_expires_at": accountAreaNullableString(v.Credit.GiftedExpiresAt)}, &resp.Diagnostics)
	runway := types.Float64Null()
	if v.Hourly.RunwayHours != nil {
		runway = types.Float64Value(*v.Hourly.RunwayHours)
	}
	hourly := accountAreaObject(ctx, accountAreaHourlyTypes(), map[string]attr.Value{"burn_rate": types.StringValue(string(v.Hourly.BurnRate)), "runway_hours": runway, "active_machines": types.Int64Value(int64(v.Hourly.ActiveMachines))}, &resp.Diagnostics)
	monthly := accountAreaNullableMoney(v.Budget.Monthly)
	percent := types.Float64Null()
	if v.Budget.Percent != nil {
		percent = types.Float64Value(*v.Budget.Percent)
	}
	budget := accountAreaObject(ctx, accountAreaBudgetTypes(), map[string]attr.Value{"monthly": monthly, "month_to_date": types.StringValue(string(v.Budget.MonthToDate)), "percent": percent, "cap_percent": types.Int64Value(int64(v.Budget.CapPercent))}, &resp.Diagnostics)
	quota := accountAreaObject(ctx, accountAreaQuotaTypes(), map[string]attr.Value{"limit": types.Int64Value(int64(v.Quota.Limit)), "used": types.Int64Value(int64(v.Quota.Used)), "remaining": types.Int64Value(int64(v.Quota.Remaining))}, &resp.Diagnostics)
	auto := accountAreaObject(ctx, accountAreaTopupTypes(), map[string]attr.Value{"enabled": types.BoolValue(v.AutoTopup.Enabled), "threshold": accountAreaNullableMoney(v.AutoTopup.Threshold), "amount": accountAreaNullableMoney(v.AutoTopup.Amount)}, &resp.Diagnostics)
	account := accountAreaObject(ctx, accountAreaAccountTypes(), map[string]attr.Value{"currency": types.StringValue(v.Currency), "billing_mode": types.StringValue(v.BillingMode), "credit": credit, "hourly": hourly, "budget": budget, "quota": quota, "auto_topup": auto}, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &AccountAreaAccountModel{Account: account})...)
}

func (d *AccountAreaUsageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_usage"
}
func (d *AccountAreaUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists paginated account usage records and their total.", Attributes: map[string]schema.Attribute{
		"from": schema.StringAttribute{Optional: true, Description: "Inclusive beginning of the usage interval as an RFC3339 timestamp."}, "to": schema.StringAttribute{Optional: true, Description: "Exclusive end of the usage interval as an RFC3339 timestamp."}, "vm_id": schema.Int64Attribute{Optional: true, Description: "Limit usage to this virtual machine identifier."}, "group_by": schema.StringAttribute{Optional: true, Description: "Group records by hour or virtual machine; defaults to hour.", Validators: []validator.String{accountAreaEnumValidator{allowed: []string{"hour", "vm"}}}},
		"records": schema.ListNestedAttribute{Computed: true, Description: "Usage records across all API pages.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaUsageRecordAttributes()}}, "total": schema.StringAttribute{Computed: true, Description: "Total amount as a decimal string."}, "currency": schema.StringAttribute{Computed: true, Description: "Currency for the total."},
	}}
}
func (d *AccountAreaUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountAreaUsageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group := data.GroupBy.ValueString()
	if group != "" && group != "hour" && group != "vm" {
		resp.Diagnostics.AddError("Invalid usage group_by", fmt.Sprintf("group_by %q must be hour or vm.", group))
		return
	}
	var vmID *int64
	if !data.VMID.IsNull() && !data.VMID.IsUnknown() {
		v := data.VMID.ValueInt64()
		vmID = &v
	}
	result, err := d.client.AccountAreaListUsage(ctx, client.AccountAreaUsageQuery{From: data.From.ValueString(), To: data.To.ValueString(), VMID: vmID, GroupBy: group})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF usage", err.Error())
		return
	}
	values := make([]AccountAreaUsageRecordModel, 0, len(result.Data))
	for _, record := range result.Data {
		values = append(values, AccountAreaUsageRecordModel{VMID: accountAreaNullableInt(record.VMID), Hostname: accountAreaNullableString(record.Hostname), Kind: types.StringValue(record.Kind), PeriodStart: types.StringValue(record.PeriodStart), Hours: types.Int64Value(int64(record.Hours)), Rate: types.StringValue(string(record.Rate)), RateUnit: types.StringValue(record.RateUnit), Amount: types.StringValue(string(record.Amount)), Currency: types.StringValue(record.Currency)})
	}
	records, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: accountAreaUsageRecordTypes()}, values)
	resp.Diagnostics.Append(diags...)
	data.Records = records
	data.Total = types.StringValue(string(result.Total))
	data.Currency = types.StringValue(result.Currency)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type AccountAreaUsageRecordModel struct {
	VMID        types.Int64  `tfsdk:"vm_id"`
	Hostname    types.String `tfsdk:"hostname"`
	Kind        types.String `tfsdk:"kind"`
	PeriodStart types.String `tfsdk:"period_start"`
	Hours       types.Int64  `tfsdk:"hours"`
	Rate        types.String `tfsdk:"rate"`
	RateUnit    types.String `tfsdk:"rate_unit"`
	Amount      types.String `tfsdk:"amount"`
	Currency    types.String `tfsdk:"currency"`
}

func (r *AccountAreaBudgetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_budget"
}
func (r *AccountAreaBudgetResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*client.Client)
	}
}
func (r *AccountAreaBudgetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema.Schema{Description: "Manages the account monthly spending budget. Requires the billing.write scope; deleting this resource removes the budget.", Attributes: map[string]resourceSchema.Attribute{
		"id": resourceSchema.StringAttribute{Computed: true, Description: "Stable account budget identifier."}, "monthly": resourceSchema.StringAttribute{Optional: true, Description: "Monthly spending cap in the account currency, as a decimal string. Leave null to remove the budget."}, "month_to_date": resourceSchema.StringAttribute{Computed: true, Description: "Amount spent this month."}, "percent": resourceSchema.Float64Attribute{Computed: true, Description: "Percentage of the monthly budget consumed, when a budget is set."}, "cap_percent": resourceSchema.Int64Attribute{Computed: true, Description: "Budget share at which machines are stopped."},
	}}
}
func (r *AccountAreaBudgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountAreaBudgetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var monthly *string
	if !plan.Monthly.IsNull() && !plan.Monthly.IsUnknown() {
		v := plan.Monthly.ValueString()
		monthly = &v
	}
	budget, err := r.client.AccountAreaSetBudget(ctx, monthly)
	if err != nil {
		resp.Diagnostics.AddError("Unable to set FFXF account budget", err.Error())
		return
	}
	account, err := r.client.AccountAreaGetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF account budget", err.Error())
		return
	}
	accountAreaSetBudgetModel(&plan, *budget, account.Data.Budget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *AccountAreaBudgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccountAreaBudgetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	account, err := r.client.AccountAreaGetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF account budget", err.Error())
		return
	}
	if account.Data.Budget.Monthly == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	accountAreaSetBudgetModel(&state, account.Data.Budget, account.Data.Budget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *AccountAreaBudgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccountAreaBudgetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var monthly *string
	if !plan.Monthly.IsNull() && !plan.Monthly.IsUnknown() {
		v := plan.Monthly.ValueString()
		monthly = &v
	}
	budget, err := r.client.AccountAreaSetBudget(ctx, monthly)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update FFXF account budget", err.Error())
		return
	}
	account, err := r.client.AccountAreaGetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF account budget", err.Error())
		return
	}
	accountAreaSetBudgetModel(&plan, *budget, account.Data.Budget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *AccountAreaBudgetResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if _, err := r.client.AccountAreaSetBudget(ctx, nil); err != nil {
		resp.Diagnostics.AddError("Unable to remove FFXF account budget", err.Error())
	}
}
func accountAreaSetBudgetModel(model *AccountAreaBudgetModel, b client.AccountAreaBudget, current client.AccountAreaBudget) {
	model.ID = types.StringValue("account-budget")
	model.Monthly = accountAreaNullableMoney(current.Monthly)
	model.MonthToDate = types.StringValue(string(current.MonthToDate))
	if current.Percent == nil {
		model.Percent = types.Float64Null()
	} else {
		model.Percent = types.Float64Value(*current.Percent)
	}
	model.CapPercent = types.Int64Value(int64(current.CapPercent))
	if b.Monthly != nil && current.Monthly == nil {
		model.Monthly = types.StringValue(string(*b.Monthly))
	}
}

func accountAreaNullableString(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}
func accountAreaNullableInt(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}
func accountAreaNullableMoney(v *client.AccountAreaMoney) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*v))
}
func accountAreaObject(ctx context.Context, typesMap map[string]attr.Type, values map[string]attr.Value, diags *diag.Diagnostics) types.Object {
	result, d := types.ObjectValue(typesMap, values)
	diags.Append(d...)
	return result
}
func accountAreaUsageRecordTypes() map[string]attr.Type {
	return map[string]attr.Type{"vm_id": types.Int64Type, "hostname": types.StringType, "kind": types.StringType, "period_start": types.StringType, "hours": types.Int64Type, "rate": types.StringType, "rate_unit": types.StringType, "amount": types.StringType, "currency": types.StringType}
}
func accountAreaUsageRecordAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"vm_id": schema.Int64Attribute{Computed: true, Description: "VM identifier, or null for non-VM usage."}, "hostname": schema.StringAttribute{Computed: true, Description: "VM hostname, when applicable."}, "kind": schema.StringAttribute{Computed: true, Description: "Usage kind, such as vm or github_runners."}, "period_start": schema.StringAttribute{Computed: true, Description: "Start of this billing period."}, "hours": schema.Int64Attribute{Computed: true, Description: "Billable hours."}, "rate": schema.StringAttribute{Computed: true, Description: "Rate as a decimal string."}, "rate_unit": schema.StringAttribute{Computed: true, Description: "Rate unit, hour or minute."}, "amount": schema.StringAttribute{Computed: true, Description: "Amount as a decimal string."}, "currency": schema.StringAttribute{Computed: true, Description: "Currency code."}}
}

func accountAreaAccountAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"currency":     schema.StringAttribute{Computed: true, Description: "Account currency code."},
		"billing_mode": schema.StringAttribute{Computed: true, Description: "Billing mode, prepaid or postpaid."},
		"credit":       schema.SingleNestedAttribute{Computed: true, Description: "Available account credit.", Attributes: map[string]schema.Attribute{"balance": schema.StringAttribute{Computed: true, Description: "Total spendable credit."}, "paid": schema.StringAttribute{Computed: true, Description: "Purchased credit."}, "gifted": schema.StringAttribute{Computed: true, Description: "Granted credit."}, "gifted_expires_at": schema.StringAttribute{Computed: true, Description: "Earliest remaining gifted-credit expiry."}}},
		"hourly":       schema.SingleNestedAttribute{Computed: true, Description: "Current hourly usage and runway.", Attributes: map[string]schema.Attribute{"burn_rate": schema.StringAttribute{Computed: true, Description: "Sum of current hourly machine rates."}, "runway_hours": schema.Float64Attribute{Computed: true, Description: "Estimated hours remaining at the current burn rate."}, "active_machines": schema.Int64Attribute{Computed: true, Description: "Number of active machines."}}},
		"budget":       schema.SingleNestedAttribute{Computed: true, Description: "Monthly account budget and usage.", Attributes: map[string]schema.Attribute{"monthly": schema.StringAttribute{Computed: true, Description: "Monthly budget, or null when unset."}, "month_to_date": schema.StringAttribute{Computed: true, Description: "Amount spent this month."}, "percent": schema.Float64Attribute{Computed: true, Description: "Percentage of budget consumed."}, "cap_percent": schema.Int64Attribute{Computed: true, Description: "Budget usage percentage at which machines are stopped."}}},
		"quota":        schema.SingleNestedAttribute{Computed: true, Description: "Machine quota usage.", Attributes: map[string]schema.Attribute{"limit": schema.Int64Attribute{Computed: true, Description: "Maximum machines."}, "used": schema.Int64Attribute{Computed: true, Description: "Machines currently used."}, "remaining": schema.Int64Attribute{Computed: true, Description: "Remaining machine quota."}}},
		"auto_topup":   schema.SingleNestedAttribute{Computed: true, Description: "Automatic credit top-up configuration.", Attributes: map[string]schema.Attribute{"enabled": schema.BoolAttribute{Computed: true, Description: "Whether automatic top-up is enabled."}, "threshold": schema.StringAttribute{Computed: true, Description: "Credit balance that triggers a top-up, or null."}, "amount": schema.StringAttribute{Computed: true, Description: "Amount added by each top-up, or null."}}},
	}
}
func accountAreaCreditTypes() map[string]attr.Type {
	return map[string]attr.Type{"balance": types.StringType, "paid": types.StringType, "gifted": types.StringType, "gifted_expires_at": types.StringType}
}
func accountAreaHourlyTypes() map[string]attr.Type {
	return map[string]attr.Type{"burn_rate": types.StringType, "runway_hours": types.Float64Type, "active_machines": types.Int64Type}
}
func accountAreaBudgetTypes() map[string]attr.Type {
	return map[string]attr.Type{"monthly": types.StringType, "month_to_date": types.StringType, "percent": types.Float64Type, "cap_percent": types.Int64Type}
}
func accountAreaQuotaTypes() map[string]attr.Type {
	return map[string]attr.Type{"limit": types.Int64Type, "used": types.Int64Type, "remaining": types.Int64Type}
}
func accountAreaTopupTypes() map[string]attr.Type {
	return map[string]attr.Type{"enabled": types.BoolType, "threshold": types.StringType, "amount": types.StringType}
}
func accountAreaAccountTypes() map[string]attr.Type {
	return map[string]attr.Type{"currency": types.StringType, "billing_mode": types.StringType, "credit": types.ObjectType{AttrTypes: accountAreaCreditTypes()}, "hourly": types.ObjectType{AttrTypes: accountAreaHourlyTypes()}, "budget": types.ObjectType{AttrTypes: accountAreaBudgetTypes()}, "quota": types.ObjectType{AttrTypes: accountAreaQuotaTypes()}, "auto_topup": types.ObjectType{AttrTypes: accountAreaTopupTypes()}}
}

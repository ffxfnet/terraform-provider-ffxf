package provider

import (
	"context"
	"fmt"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*AccountAreaRunnersDataSource)(nil)
var _ datasource.DataSource = (*AccountAreaRunnerJobsDataSource)(nil)

type AccountAreaRunnersDataSource struct{ client *client.Client }
type AccountAreaRunnerJobsDataSource struct{ client *client.Client }

func NewAccountAreaRunnersDataSource() datasource.DataSource { return &AccountAreaRunnersDataSource{} }
func NewAccountAreaRunnerJobsDataSource() datasource.DataSource {
	return &AccountAreaRunnerJobsDataSource{}
}

type AccountAreaRunnersModel struct {
	Runners types.Object `tfsdk:"runners"`
}
type AccountAreaRunnerJobsModel struct {
	Repository types.String `tfsdk:"repository"`
	Status     types.String `tfsdk:"status"`
	Limit      types.Int64  `tfsdk:"limit"`
	Jobs       types.List   `tfsdk:"jobs"`
}

func (d *AccountAreaRunnersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runners"
}
func (d *AccountAreaRunnersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaRunnersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Reads GitHub Actions runner capacity, installations, usage, and cache information.", Attributes: map[string]schema.Attribute{"runners": schema.SingleNestedAttribute{Computed: true, Description: "Current runner configuration and usage details.", Attributes: accountAreaRunnerAttributes()}}}
}
func (d *AccountAreaRunnersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.AccountAreaGetRunners(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read FFXF runners", err.Error())
		return
	}
	runners := accountAreaRunnersValue(ctx, result.Data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &AccountAreaRunnersModel{Runners: runners})...)
}

func (d *AccountAreaRunnerJobsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_jobs"
}
func (d *AccountAreaRunnerJobsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*client.Client)
	}
}
func (d *AccountAreaRunnerJobsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Lists GitHub Actions runner jobs across all available API pages.", Attributes: map[string]schema.Attribute{"repository": schema.StringAttribute{Optional: true, Description: "Filter by GitHub owner/repository."}, "status": schema.StringAttribute{Optional: true, Description: "Filter by queued, running, succeeded, failed, cancelled, or refused status."}, "limit": schema.Int64Attribute{Optional: true, Description: "Page size from 1 to 100; defaults to the API's default.", Validators: []validator.Int64{accountAreaInt64RangeValidator{minimum: 1, maximum: 100}}}, "jobs": schema.ListNestedAttribute{Computed: true, Description: "Matching runner jobs.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaRunnerJobAttributes()}}}}
}
func (d *AccountAreaRunnerJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountAreaRunnerJobsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status := data.Status.ValueString()
	if status != "" && !accountAreaRunnerStatusValid(status) {
		resp.Diagnostics.AddError("Invalid runner job status", fmt.Sprintf("status %q is not a supported runner job status.", status))
		return
	}
	limit := 0
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
		limit = int(data.Limit.ValueInt64())
		if limit < 1 || limit > 100 {
			resp.Diagnostics.AddError("Invalid runner job limit", "limit must be between 1 and 100.")
			return
		}
	}
	result, err := d.client.AccountAreaListRunnerJobs(ctx, data.Repository.ValueString(), status, limit)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list FFXF runner jobs", err.Error())
		return
	}
	values := make([]attr.Value, 0, len(result.Data))
	for _, job := range result.Data {
		obj, diags := types.ObjectValue(accountAreaRunnerJobTypes(), accountAreaRunnerJobValueMap(job))
		resp.Diagnostics.Append(diags...)
		values = append(values, obj)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: accountAreaRunnerJobTypes()}, values)
	resp.Diagnostics.Append(diags...)
	data.Jobs = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func accountAreaRunnerStatusValid(v string) bool {
	switch v {
	case "queued", "running", "succeeded", "failed", "cancelled", "refused":
		return true
	}
	return false
}

func accountAreaRunnerAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"sizes":         schema.ListNestedAttribute{Computed: true, Description: "Runner sizes and prices.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"label": schema.StringAttribute{Computed: true, Description: "Runner label used in GitHub workflows."}, "vcpu": schema.Int64Attribute{Computed: true, Description: "Virtual CPU count."}, "memory_gb": schema.Int64Attribute{Computed: true, Description: "Memory in GiB."}, "image": schema.StringAttribute{Computed: true, Description: "Runner image slug."}, "price_per_minute": schema.StringAttribute{Computed: true, Description: "Price per minute as a decimal string."}, "currency": schema.StringAttribute{Computed: true, Description: "Currency code."}}}},
		"concurrency":   schema.SingleNestedAttribute{Computed: true, Description: "Runner concurrency limits and current usage.", Attributes: map[string]schema.Attribute{"max_vcpu": schema.Int64Attribute{Computed: true, Description: "Maximum concurrent vCPU capacity."}, "running_vcpu": schema.Int64Attribute{Computed: true, Description: "Currently running vCPU capacity."}}},
		"installations": schema.ListNestedAttribute{Computed: true, Description: "Linked GitHub installations.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"id": schema.Int64Attribute{Computed: true, Description: "GitHub installation identifier."}, "account": schema.StringAttribute{Computed: true, Description: "GitHub organization or user login."}, "type": schema.StringAttribute{Computed: true, Description: "Installation owner type."}, "repository_selection": schema.StringAttribute{Computed: true, Description: "Repository selection mode, or null."}, "status": schema.StringAttribute{Computed: true, Description: "Installation status."}, "linked_at": schema.StringAttribute{Computed: true, Description: "When the installation was linked, if available."}}}},
		"month":         schema.SingleNestedAttribute{Computed: true, Description: "Current monthly runner usage.", Attributes: map[string]schema.Attribute{"period_start": schema.StringAttribute{Computed: true, Description: "Start of the current billing period."}, "minutes": schema.Int64Attribute{Computed: true, Description: "Runner minutes used."}, "free_minutes_used": schema.Int64Attribute{Computed: true, Description: "Free minutes used in 2-vCPU minutes."}, "free_minutes_allowance": schema.Int64Attribute{Computed: true, Description: "Free minutes allowance."}, "amount": schema.StringAttribute{Computed: true, Description: "Runner charges as a decimal string."}, "cache_amount": schema.StringAttribute{Computed: true, Description: "Actions cache charges as a decimal string."}, "currency": schema.StringAttribute{Computed: true, Description: "Currency code."}, "repositories": schema.ListNestedAttribute{Computed: true, Description: "Usage by repository.", NestedObject: schema.NestedAttributeObject{Attributes: accountAreaRunnerRepositoryAttributes()}}}},
		"cache":         schema.SingleNestedAttribute{Computed: true, Description: "Actions cache usage and pricing.", Attributes: map[string]schema.Attribute{"price_per_gb_month": schema.StringAttribute{Computed: true, Description: "Price per GiB-month as a decimal string."}, "currency": schema.StringAttribute{Computed: true, Description: "Currency code."}, "bytes": schema.Int64Attribute{Computed: true, Description: "Current actions cache bytes."}, "repository_limit_gb_max": schema.Int64Attribute{Computed: true, Description: "Maximum per-repository cache size in GiB."}, "retention_days": schema.Int64Attribute{Computed: true, Description: "Cache retention period in days."}, "repositories": schema.ListNestedAttribute{Computed: true, Description: "Actions cache usage by repository.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"repository": schema.StringAttribute{Computed: true, Description: "Repository name, or null."}, "repository_id": schema.Int64Attribute{Computed: true, Description: "GitHub repository identifier."}, "bytes": schema.Int64Attribute{Computed: true, Description: "Cache bytes."}, "entries": schema.Int64Attribute{Computed: true, Description: "Cache entry count."}, "last_used_at": schema.StringAttribute{Computed: true, Description: "Last cache use time, if available."}, "emptying": schema.BoolAttribute{Computed: true, Description: "Whether cache deletion is pending."}}}}}},
		"docker_cache":  schema.SingleNestedAttribute{Computed: true, Description: "Docker cache usage and policy.", Attributes: map[string]schema.Attribute{"repository_size_gb": schema.Int64Attribute{Computed: true, Description: "Docker cache size per repository in GiB."}, "retention_days": schema.Int64Attribute{Computed: true, Description: "Docker cache retention period in days."}, "repositories": schema.ListNestedAttribute{Computed: true, Description: "Docker cache status by repository.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"repository": schema.StringAttribute{Computed: true, Description: "Repository name, or null."}, "repository_id": schema.Int64Attribute{Computed: true, Description: "GitHub repository identifier."}, "enabled": schema.BoolAttribute{Computed: true, Description: "Whether Docker cache is enabled."}, "bytes": schema.Int64Attribute{Computed: true, Description: "Docker cache bytes."}, "last_success_at": schema.StringAttribute{Computed: true, Description: "Last successful cache update."}, "last_used_at": schema.StringAttribute{Computed: true, Description: "Last cache use time."}, "emptying": schema.BoolAttribute{Computed: true, Description: "Whether cache deletion is pending."}}}}}},
	}
}
func accountAreaRunnerRepositoryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"repository": schema.StringAttribute{Computed: true, Description: "Repository name, or null."}, "vcpu": schema.Int64Attribute{Computed: true, Description: "Virtual CPU count."}, "jobs": schema.Int64Attribute{Computed: true, Description: "Job count."}, "minutes": schema.Int64Attribute{Computed: true, Description: "Minutes used."}, "amount": schema.StringAttribute{Computed: true, Description: "Amount charged."}}
}

func accountAreaRunnerJobAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{"id": schema.Int64Attribute{Computed: true, Description: "Provider job identifier."}, "github_job_id": schema.Int64Attribute{Computed: true, Description: "GitHub job identifier."}, "run_id": schema.Int64Attribute{Computed: true, Description: "GitHub workflow run identifier."}, "run_attempt": schema.Int64Attribute{Computed: true, Description: "Workflow run attempt number."}, "repository": schema.StringAttribute{Computed: true, Description: "GitHub owner/repository."}, "workflow": schema.StringAttribute{Computed: true, Description: "Workflow name, when available."}, "job": schema.StringAttribute{Computed: true, Description: "Job name, when available."}, "label": schema.StringAttribute{Computed: true, Description: "Requested runner label."}, "vcpu": schema.Int64Attribute{Computed: true, Description: "Runner virtual CPU count."}, "status": schema.StringAttribute{Computed: true, Description: "Job status."}, "waiting_reason": schema.StringAttribute{Computed: true, Description: "Reason the job is queued, if applicable."}, "refusal": schema.StringAttribute{Computed: true, Description: "Reason the job was refused, if applicable."}, "conclusion": schema.StringAttribute{Computed: true, Description: "GitHub job conclusion, when available."}, "stopped_for_credit": schema.BoolAttribute{Computed: true, Description: "Whether the job stopped because of credit."}, "queued_at": schema.StringAttribute{Computed: true, Description: "Queue timestamp."}, "started_at": schema.StringAttribute{Computed: true, Description: "Start timestamp."}, "completed_at": schema.StringAttribute{Computed: true, Description: "Completion timestamp."}, "minutes": schema.Int64Attribute{Computed: true, Description: "Billable minutes, when available."}, "free_minutes": schema.Int64Attribute{Computed: true, Description: "Free minutes applied."}, "amount": schema.StringAttribute{Computed: true, Description: "Charged amount as a decimal string."}, "currency": schema.StringAttribute{Computed: true, Description: "Currency code, when available."}, "url": schema.StringAttribute{Computed: true, Description: "GitHub job URL."}}
}
func accountAreaRunnerJobTypes() map[string]attr.Type {
	return map[string]attr.Type{"id": types.Int64Type, "github_job_id": types.Int64Type, "run_id": types.Int64Type, "run_attempt": types.Int64Type, "repository": types.StringType, "workflow": types.StringType, "job": types.StringType, "label": types.StringType, "vcpu": types.Int64Type, "status": types.StringType, "waiting_reason": types.StringType, "refusal": types.StringType, "conclusion": types.StringType, "stopped_for_credit": types.BoolType, "queued_at": types.StringType, "started_at": types.StringType, "completed_at": types.StringType, "minutes": types.Int64Type, "free_minutes": types.Int64Type, "amount": types.StringType, "currency": types.StringType, "url": types.StringType}
}
func accountAreaRunnerJobValueMap(j client.AccountAreaRunnerJob) map[string]attr.Value {
	return map[string]attr.Value{"id": types.Int64Value(int64(j.ID)), "github_job_id": types.Int64Value(j.GitHubJobID), "run_id": types.Int64Value(j.RunID), "run_attempt": types.Int64Value(int64(j.RunAttempt)), "repository": types.StringValue(j.Repository), "workflow": accountAreaNullableString(j.Workflow), "job": accountAreaNullableString(j.Job), "label": types.StringValue(j.Label), "vcpu": types.Int64Value(int64(j.VCPU)), "status": types.StringValue(j.Status), "waiting_reason": accountAreaNullableString(j.WaitingReason), "refusal": accountAreaNullableString(j.Refusal), "conclusion": accountAreaNullableString(j.Conclusion), "stopped_for_credit": types.BoolValue(j.StoppedForCredit), "queued_at": accountAreaNullableString(j.QueuedAt), "started_at": accountAreaNullableString(j.StartedAt), "completed_at": accountAreaNullableString(j.CompletedAt), "minutes": accountAreaNullableInt(j.Minutes), "free_minutes": types.Int64Value(int64(j.FreeMinutes)), "amount": types.StringValue(string(j.Amount)), "currency": accountAreaNullableString(j.Currency), "url": types.StringValue(j.URL)}
}

func accountAreaRunnerSizeTypes() map[string]attr.Type {
	return map[string]attr.Type{"label": types.StringType, "vcpu": types.Int64Type, "memory_gb": types.Int64Type, "image": types.StringType, "price_per_minute": types.StringType, "currency": types.StringType}
}
func accountAreaRunnerInstallationTypes() map[string]attr.Type {
	return map[string]attr.Type{"id": types.Int64Type, "account": types.StringType, "type": types.StringType, "repository_selection": types.StringType, "status": types.StringType, "linked_at": types.StringType}
}
func accountAreaRunnerRepositoryTypes() map[string]attr.Type {
	return map[string]attr.Type{"repository": types.StringType, "vcpu": types.Int64Type, "jobs": types.Int64Type, "minutes": types.Int64Type, "amount": types.StringType}
}
func accountAreaCacheRepositoryTypes() map[string]attr.Type {
	return map[string]attr.Type{"repository": types.StringType, "repository_id": types.Int64Type, "bytes": types.Int64Type, "entries": types.Int64Type, "last_used_at": types.StringType, "emptying": types.BoolType}
}
func accountAreaDockerRepositoryTypes() map[string]attr.Type {
	return map[string]attr.Type{"repository": types.StringType, "repository_id": types.Int64Type, "enabled": types.BoolType, "bytes": types.Int64Type, "last_success_at": types.StringType, "last_used_at": types.StringType, "emptying": types.BoolType}
}
func accountAreaRunnerConcurrencyTypes() map[string]attr.Type {
	return map[string]attr.Type{"max_vcpu": types.Int64Type, "running_vcpu": types.Int64Type}
}
func accountAreaRunnerMonthTypes() map[string]attr.Type {
	return map[string]attr.Type{"period_start": types.StringType, "minutes": types.Int64Type, "free_minutes_used": types.Int64Type, "free_minutes_allowance": types.Int64Type, "amount": types.StringType, "cache_amount": types.StringType, "currency": types.StringType, "repositories": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaRunnerRepositoryTypes()}}}
}
func accountAreaRunnerCacheTypes() map[string]attr.Type {
	return map[string]attr.Type{"price_per_gb_month": types.StringType, "currency": types.StringType, "bytes": types.Int64Type, "repository_limit_gb_max": types.Int64Type, "retention_days": types.Int64Type, "repositories": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaCacheRepositoryTypes()}}}
}
func accountAreaDockerCacheTypes() map[string]attr.Type {
	return map[string]attr.Type{"repository_size_gb": types.Int64Type, "retention_days": types.Int64Type, "repositories": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaDockerRepositoryTypes()}}}
}
func accountAreaRunnersTypes() map[string]attr.Type {
	return map[string]attr.Type{"sizes": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaRunnerSizeTypes()}}, "concurrency": types.ObjectType{AttrTypes: accountAreaRunnerConcurrencyTypes()}, "installations": types.ListType{ElemType: types.ObjectType{AttrTypes: accountAreaRunnerInstallationTypes()}}, "month": types.ObjectType{AttrTypes: accountAreaRunnerMonthTypes()}, "cache": types.ObjectType{AttrTypes: accountAreaRunnerCacheTypes()}, "docker_cache": types.ObjectType{AttrTypes: accountAreaDockerCacheTypes()}}
}
func accountAreaBuildObjectList(ctx context.Context, typesMap map[string]attr.Type, values []map[string]attr.Value, diags *diag.Diagnostics) types.List {
	objects := make([]attr.Value, 0, len(values))
	for _, values := range values {
		obj, d := types.ObjectValue(typesMap, values)
		diags.Append(d...)
		objects = append(objects, obj)
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: typesMap}, objects)
	diags.Append(d...)
	return list
}
func accountAreaRunnersValue(ctx context.Context, r client.AccountAreaRunners, diags *diag.Diagnostics) types.Object {
	sizes := make([]map[string]attr.Value, 0, len(r.Sizes))
	for _, v := range r.Sizes {
		sizes = append(sizes, map[string]attr.Value{"label": types.StringValue(v.Label), "vcpu": types.Int64Value(int64(v.VCPU)), "memory_gb": types.Int64Value(int64(v.MemoryGB)), "image": types.StringValue(v.Image), "price_per_minute": types.StringValue(string(v.PricePerMinute)), "currency": types.StringValue(v.Currency)})
	}
	installations := make([]map[string]attr.Value, 0, len(r.Installations))
	for _, v := range r.Installations {
		installations = append(installations, map[string]attr.Value{"id": types.Int64Value(int64(v.ID)), "account": types.StringValue(v.Account), "type": types.StringValue(v.Type), "repository_selection": accountAreaNullableString(v.RepositorySelection), "status": types.StringValue(v.Status), "linked_at": accountAreaNullableString(v.LinkedAt)})
	}
	repositories := make([]map[string]attr.Value, 0, len(r.Month.Repositories))
	for _, v := range r.Month.Repositories {
		repositories = append(repositories, map[string]attr.Value{"repository": accountAreaNullableString(v.Repository), "vcpu": types.Int64Value(int64(v.VCPU)), "jobs": types.Int64Value(int64(v.Jobs)), "minutes": types.Int64Value(int64(v.Minutes)), "amount": types.StringValue(string(v.Amount))})
	}
	month := accountAreaObject(ctx, accountAreaRunnerMonthTypes(), map[string]attr.Value{"period_start": types.StringValue(r.Month.PeriodStart), "minutes": types.Int64Value(int64(r.Month.Minutes)), "free_minutes_used": types.Int64Value(int64(r.Month.FreeMinutesUsed)), "free_minutes_allowance": types.Int64Value(int64(r.Month.FreeMinutesAllowance)), "amount": types.StringValue(string(r.Month.Amount)), "cache_amount": types.StringValue(string(r.Month.CacheAmount)), "currency": types.StringValue(r.Month.Currency), "repositories": accountAreaBuildObjectList(ctx, accountAreaRunnerRepositoryTypes(), repositories, diags)}, diags)
	cacheRepos := make([]map[string]attr.Value, 0, len(r.Cache.Repositories))
	for _, v := range r.Cache.Repositories {
		cacheRepos = append(cacheRepos, map[string]attr.Value{"repository": accountAreaNullableString(v.Repository), "repository_id": types.Int64Value(int64(v.RepositoryID)), "bytes": types.Int64Value(v.Bytes), "entries": types.Int64Value(int64(v.Entries)), "last_used_at": accountAreaNullableString(v.LastUsedAt), "emptying": types.BoolValue(v.Emptying)})
	}
	cache := accountAreaObject(ctx, accountAreaRunnerCacheTypes(), map[string]attr.Value{"price_per_gb_month": types.StringValue(string(r.Cache.PricePerGBMonth)), "currency": types.StringValue(r.Cache.Currency), "bytes": types.Int64Value(r.Cache.Bytes), "repository_limit_gb_max": types.Int64Value(int64(r.Cache.RepositoryLimitGBMax)), "retention_days": types.Int64Value(int64(r.Cache.RetentionDays)), "repositories": accountAreaBuildObjectList(ctx, accountAreaCacheRepositoryTypes(), cacheRepos, diags)}, diags)
	dockerRepos := make([]map[string]attr.Value, 0, len(r.DockerCache.Repositories))
	for _, v := range r.DockerCache.Repositories {
		dockerRepos = append(dockerRepos, map[string]attr.Value{"repository": accountAreaNullableString(v.Repository), "repository_id": types.Int64Value(int64(v.RepositoryID)), "enabled": types.BoolValue(v.Enabled), "bytes": types.Int64Value(v.Bytes), "last_success_at": accountAreaNullableString(v.LastSuccessAt), "last_used_at": accountAreaNullableString(v.LastUsedAt), "emptying": types.BoolValue(v.Emptying)})
	}
	docker := accountAreaObject(ctx, accountAreaDockerCacheTypes(), map[string]attr.Value{"repository_size_gb": types.Int64Value(int64(r.DockerCache.RepositorySizeGB)), "retention_days": types.Int64Value(int64(r.DockerCache.RetentionDays)), "repositories": accountAreaBuildObjectList(ctx, accountAreaDockerRepositoryTypes(), dockerRepos, diags)}, diags)
	return accountAreaObject(ctx, accountAreaRunnersTypes(), map[string]attr.Value{"sizes": accountAreaBuildObjectList(ctx, accountAreaRunnerSizeTypes(), sizes, diags), "concurrency": accountAreaObject(ctx, accountAreaRunnerConcurrencyTypes(), map[string]attr.Value{"max_vcpu": types.Int64Value(int64(r.Concurrency.MaxVCPU)), "running_vcpu": types.Int64Value(int64(r.Concurrency.RunningVCPU))}, diags), "installations": accountAreaBuildObjectList(ctx, accountAreaRunnerInstallationTypes(), installations, diags), "month": month, "cache": cache, "docker_cache": docker}, diags)
}

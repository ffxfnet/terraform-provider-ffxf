package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountAreaRunnerDataSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/runners":
			_, _ = w.Write([]byte(`{"data":{"sizes":[{"label":"ffxf-2vcpu-ubuntu-2404","vcpu":2,"memory_gb":4,"image":"ubuntu-2404","price_per_minute":"0.01","currency":"CAD"}],"concurrency":{"max_vcpu":8,"running_vcpu":2},"installations":[{"id":1,"account":"org","type":"organization","repository_selection":"selected","status":"active","linked_at":"2026-01-01T00:00:00Z"}],"month":{"period_start":"2026-01-01T00:00:00Z","minutes":10,"free_minutes_used":5,"free_minutes_allowance":30,"amount":"0.10","cache_amount":"0","currency":"CAD","repositories":[{"repository":"owner/repo","vcpu":2,"jobs":1,"minutes":10,"amount":"0.10"}]},"cache":{"price_per_gb_month":"1.00","currency":"CAD","bytes":100,"repository_limit_gb_max":10,"retention_days":7,"repositories":[{"repository":"owner/repo","repository_id":10,"bytes":100,"entries":1,"last_used_at":"2026-01-02T00:00:00Z","emptying":false}]},"docker_cache":{"repository_size_gb":10,"retention_days":30,"repositories":[{"repository":"owner/repo","repository_id":10,"enabled":true,"bytes":200,"last_success_at":"2026-01-02T00:00:00Z","last_used_at":"2026-01-02T00:00:00Z","emptying":false}]}}}`))
		case "/runners/jobs":
			if r.URL.Query().Get("repository") != "owner/repo" || r.URL.Query().Get("status") != "queued" || r.URL.Query().Get("limit") != "5" {
				t.Errorf("jobs query = %s", r.URL.RawQuery)
			}
			if r.URL.Query().Get("cursor") == "" {
				_, _ = w.Write([]byte(`{"data":[{"id":1,"github_job_id":100,"run_id":200,"run_attempt":1,"repository":"owner/repo","workflow":null,"job":"test","label":"linux","vcpu":2,"status":"queued","waiting_reason":"credit","refusal":null,"conclusion":null,"stopped_for_credit":false,"queued_at":null,"started_at":null,"completed_at":null,"minutes":null,"free_minutes":0,"amount":"0","currency":"CAD","url":"https://github.test/job"}],"meta":{"next_cursor":"next","has_more":true}}`))
			} else {
				_, _ = w.Write([]byte(`{"data":[{"id":2,"repository":"owner/repo","status":"queued","url":"https://github.test/job2"}],"meta":{"next_cursor":null,"has_more":false}}`))
			}
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	api := testAPIClient(server)
	runners := NewAccountAreaRunnersDataSource().(*AccountAreaRunnersDataSource)
	runners.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var sch datasource.SchemaResponse
	runners.Schema(ctx, datasource.SchemaRequest{}, &sch)
	var meta datasource.MetadataResponse
	runners.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &meta)
	if meta.TypeName != "ffxf_runners" {
		t.Fatal(meta.TypeName)
	}
	rr := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
	runners.Read(ctx, datasource.ReadRequest{}, &rr)
	var state AccountAreaRunnersModel
	rr.Diagnostics.Append(rr.State.Get(ctx, &state)...)
	if rr.Diagnostics.HasError() {
		t.Fatal(rr.Diagnostics)
	}
	sizes := state.Runners.Attributes()["sizes"].(types.List)
	if len(sizes.Elements()) != 1 {
		t.Fatalf("runner sizes = %#v", sizes)
	}
	jobs := NewAccountAreaRunnerJobsDataSource().(*AccountAreaRunnerJobsDataSource)
	jobs.Configure(ctx, datasource.ConfigureRequest{ProviderData: api}, &datasource.ConfigureResponse{})
	var jobsSchema datasource.SchemaResponse
	jobs.Schema(ctx, datasource.SchemaRequest{}, &jobsSchema)
	var jobsMeta datasource.MetadataResponse
	jobs.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &jobsMeta)
	if jobsMeta.TypeName != "ffxf_runner_jobs" {
		t.Fatal(jobsMeta.TypeName)
	}
	config := accountAreaTestConfig(t, jobsSchema.Schema, map[string]any{"repository": "owner/repo", "status": "queued", "limit": int64(5)})
	jr := datasource.ReadResponse{State: tfsdk.State{Schema: jobsSchema.Schema}}
	jobs.Read(ctx, datasource.ReadRequest{Config: config}, &jr)
	var jobState AccountAreaRunnerJobsModel
	jr.Diagnostics.Append(jr.State.Get(ctx, &jobState)...)
	if jr.Diagnostics.HasError() || len(jobState.Jobs.Elements()) != 2 {
		t.Fatalf("jobs = %#v; %v", jobState, jr.Diagnostics)
	}
}

func TestAccountAreaRunnerJobsValidationAndAPIError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		inputs map[string]any
	}{{"status", http.StatusOK, map[string]any{"status": "unknown"}}, {"limit", http.StatusOK, map[string]any{"limit": int64(101)}}, {"api", http.StatusBadGateway, nil}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status) }))
			defer server.Close()
			ds := NewAccountAreaRunnerJobsDataSource().(*AccountAreaRunnerJobsDataSource)
			ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
			var sch datasource.SchemaResponse
			ds.Schema(context.Background(), datasource.SchemaRequest{}, &sch)
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
			ds.Read(context.Background(), datasource.ReadRequest{Config: accountAreaTestConfig(t, sch.Schema, tc.inputs)}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected diagnostic")
			}
		})
	}
}

func TestAccountAreaRunnersReadAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer server.Close()
	ds := NewAccountAreaRunnersDataSource().(*AccountAreaRunnersDataSource)
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
	var sch datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &sch)
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
	ds.Read(context.Background(), datasource.ReadRequest{}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("runner read should report an API error")
	}
}

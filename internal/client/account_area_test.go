package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAccountAreaClientOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer account-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		switch r.Method + " " + r.URL.Path {
		case "GET /plans":
			if r.URL.Query().Get("region") != "ca east" {
				t.Errorf("plan query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"slug":"nano","name":"Nano","prices":[]}]}`))
		case "GET /images":
			q := r.URL.Query()
			if q.Get("region") != "r1" || q.Get("plan") != "nano" || q.Get("category") != "linux" {
				t.Errorf("image query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"slug":"debian","name":"Debian","regions":[]}]}`))
		case "GET /vpcs":
			_, _ = w.Write([]byte(`{"data":[{"id":2,"members":[{"vm":4,"hostname":"web","ip":"10.0.0.4","configuration":"automatic"}]}],"quota":3}`))
		case "GET /vpcs/2":
			_, _ = w.Write([]byte(`{"data":{"id":2,"members":[{"vm":4,"hostname":"web","ip":"10.0.0.4","configuration":"automatic"}]}}`))
		case "POST /vpcs/2/members":
			if string(body) != `{"vm":4}` {
				t.Errorf("member body = %s", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"vm":4,"hostname":"web","ip":"10.0.0.4","configuration":"automatic"}}`))
		case "POST /vpcs/2/members/4/move":
			if string(body) != `{"to":3}` {
				t.Errorf("move body = %s", body)
			}
			_, _ = w.Write([]byte(`{"data":{"vm":4,"hostname":"web","ip":"10.0.1.4","configuration":"automatic"}}`))
		case "DELETE /vpcs/2/members/4":
			w.WriteHeader(http.StatusNoContent)
		case "GET /ssh-keys":
			_, _ = w.Write([]byte(`{"data":[{"id":7,"name":"work","fingerprint":"SHA256:x","public_key":"ssh-ed25519 abc","created_at":"now"}]}`))
		case "POST /ssh-keys":
			if !strings.Contains(string(body), `"name":"work"`) || !strings.Contains(string(body), `"public_key":"ssh-ed25519 abc"`) {
				t.Errorf("key body = %s", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":7,"fingerprint":"SHA256:x"}}`))
		case "DELETE /ssh-keys/7":
			w.WriteHeader(http.StatusNoContent)
		case "GET /account":
			_, _ = w.Write([]byte(`{"data":{"currency":"CAD","billing_mode":"prepaid","credit":{"balance":"10.50","paid":10.5,"gifted":"0","gifted_expires_at":null},"hourly":{"burn_rate":"0.1","runway_hours":100,"active_machines":1},"budget":{"monthly":"50.00","month_to_date":"5","percent":10,"cap_percent":100},"quota":{"limit":5,"used":1,"remaining":4},"auto_topup":{"enabled":false,"threshold":null,"amount":null}}}`))
		case "PUT /account/budget":
			if string(body) != `{"monthly":null}` {
				t.Errorf("budget body = %s", body)
			}
			_, _ = w.Write([]byte(`{"data":{"monthly":null,"month_to_date":"0","percent":null,"cap_percent":100}}`))
		case "GET /usage":
			q := r.URL.Query()
			if q.Get("from") != "2026-01-01" || q.Get("to") != "2026-02-01" || q.Get("vm_id") != "4" || q.Get("group_by") != "vm" || q.Get("limit") != "200" {
				t.Errorf("usage query = %s", r.URL.RawQuery)
			}
			if q.Get("cursor") == "" {
				_, _ = w.Write([]byte(`{"data":[{"vm_id":4,"kind":"vm","period_start":"start","hours":2,"rate":"0.1","rate_unit":"hour","amount":"0.2","currency":"CAD"}],"meta":{"next_cursor":"next","has_more":true,"total":"0.2","currency":"CAD"}}`))
			} else {
				_, _ = w.Write([]byte(`{"data":[{"vm_id":4,"kind":"vm","period_start":"later","hours":1,"rate":"0.1","rate_unit":"hour","amount":"0.1","currency":"CAD"}],"meta":{"next_cursor":null,"has_more":false,"total":"0.3","currency":"CAD"}}`))
			}
		case "GET /runners":
			_, _ = w.Write([]byte(`{"data":{"sizes":[{"label":"runner","vcpu":2,"memory_gb":4,"image":"ubuntu","price_per_minute":"0.01","currency":"CAD"}],"concurrency":{"max_vcpu":4,"running_vcpu":2},"installations":[],"month":{"period_start":"now","minutes":0,"free_minutes_used":0,"free_minutes_allowance":0,"amount":"0","cache_amount":"0","currency":"CAD","repositories":[]},"cache":{"price_per_gb_month":"1","currency":"CAD","bytes":0,"repository_limit_gb_max":10,"retention_days":7,"repositories":[]},"docker_cache":{"repository_size_gb":10,"retention_days":7,"repositories":[]}}}`))
		case "GET /runners/jobs":
			q := r.URL.Query()
			if q.Get("repository") != "owner/repo" || q.Get("status") != "queued" || q.Get("limit") != "25" {
				t.Errorf("jobs query = %s", r.URL.RawQuery)
			}
			if q.Get("cursor") == "" {
				_, _ = w.Write([]byte(`{"data":[{"id":1,"status":"queued","repository":"owner/repo","url":"https://github.test/job"}],"meta":{"next_cursor":"next","has_more":true}}`))
			} else {
				_, _ = w.Write([]byte(`{"data":[{"id":2,"status":"queued","repository":"owner/repo","url":"https://github.test/job2"}],"meta":{"next_cursor":null,"has_more":false}}`))
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	api := NewClient(server.URL, "account-token")
	api.HTTPClient = server.Client()
	ctx := context.Background()
	plans, err := api.AccountAreaListPlans(ctx, "ca east")
	if err != nil || len(plans.Data) != 1 || plans.Data[0].Slug != "nano" {
		t.Fatalf("plans = %#v, %v", plans, err)
	}
	images, err := api.AccountAreaListImages(ctx, map[string]string{"region": "r1", "plan": "nano", "category": "linux"})
	if err != nil || len(images.Data) != 1 {
		t.Fatalf("images = %#v, %v", images, err)
	}
	vpcs, err := api.AccountAreaListVPCs(ctx)
	if err != nil || vpcs.Quota != 3 || vpcs.Data[0].Members[0].VM != 4 {
		t.Fatalf("vpcs = %#v, %v", vpcs, err)
	}
	vpc, err := api.AccountAreaGetVPC(ctx, 2)
	if err != nil || len(vpc.Data.Members) != 1 {
		t.Fatalf("vpc = %#v, %v", vpc, err)
	}
	if _, err = api.AccountAreaCreateVPCMember(ctx, 2, 4); err != nil {
		t.Fatal(err)
	}
	if _, err = api.AccountAreaMoveVPCMember(ctx, 2, 4, 3); err != nil {
		t.Fatal(err)
	}
	if err = api.AccountAreaDeleteVPCMember(ctx, 2, 4); err != nil {
		t.Fatal(err)
	}
	keys, err := api.AccountAreaListSSHKeys(ctx)
	if err != nil || keys.Data[0].Fingerprint != "SHA256:x" {
		t.Fatalf("keys = %#v, %v", keys, err)
	}
	key, err := api.AccountAreaCreateSSHKey(ctx, "work", "ssh-ed25519 abc")
	if err != nil || key.Data.ID != 7 {
		t.Fatalf("key = %#v, %v", key, err)
	}
	if err = api.AccountAreaDeleteSSHKey(ctx, 7); err != nil {
		t.Fatal(err)
	}
	account, err := api.AccountAreaGetAccount(ctx)
	if err != nil || account.Data.Credit.Balance != "10.50" || account.Data.Credit.Paid != "10.5" {
		t.Fatalf("account = %#v, %v", account, err)
	}
	if _, err = api.AccountAreaSetBudget(ctx, nil); err != nil {
		t.Fatal(err)
	}
	vm := int64(4)
	usage, err := api.AccountAreaListUsage(ctx, AccountAreaUsageQuery{From: "2026-01-01", To: "2026-02-01", VMID: &vm, GroupBy: "vm"})
	if err != nil || len(usage.Data) != 2 || usage.Total != "0.3" {
		t.Fatalf("usage = %#v, %v", usage, err)
	}
	runners, err := api.AccountAreaGetRunners(ctx)
	if err != nil || runners.Data.Sizes[0].VCPU != 2 {
		t.Fatalf("runners = %#v, %v", runners, err)
	}
	jobs, err := api.AccountAreaListRunnerJobs(ctx, "owner/repo", "queued", 25)
	if err != nil || len(jobs.Data) != 2 {
		t.Fatalf("jobs = %#v, %v", jobs, err)
	}
}

func TestAccountAreaClientErrorBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ssh-keys" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	if _, err := c.AccountAreaListSSHKeys(context.Background()); !errors.Is(err, ErrNetworkResourceNotFound) {
		t.Fatalf("missing list error = %v", err)
	}
	if _, err := c.AccountAreaGetAccount(context.Background()); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("API error = %v", err)
	}
	var number AccountAreaMoney
	if err := json.Unmarshal([]byte(`12.50`), &number); err != nil || number != "12.50" {
		t.Fatalf("numeric money = %q, %v", number, err)
	}
	_ = url.Values{}
}

func TestAccountAreaQueryEncoding(t *testing.T) {
	got := accountAreaQueryPath("/plans", map[string]string{"region": "ca east", "empty": ""})
	u, err := url.Parse(got)
	if err != nil || u.Query().Get("region") != "ca east" || u.Query().Has("empty") {
		t.Fatalf("query path = %q, %v", got, err)
	}
}

func TestAccountAreaBudgetValueAndPaginatedErrors(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/account/budget":
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"monthly":"100.25"}` {
				t.Errorf("budget body = %s", body)
			}
			_, _ = w.Write([]byte(`{"data":{"monthly":"100.25"}}`))
		case "/usage":
			calls++
			w.WriteHeader(http.StatusBadGateway)
		case "/runners/jobs":
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			t.Errorf("unexpected %s", r.URL)
		}
	}))
	defer server.Close()
	api := NewClient(server.URL, "token")
	amount := "100.25"
	budget, err := api.AccountAreaSetBudget(context.Background(), &amount)
	if err != nil || budget.Monthly == nil || *budget.Monthly != AccountAreaMoney(amount) {
		t.Fatalf("budget = %#v, %v", budget, err)
	}
	if _, err := api.AccountAreaListUsage(context.Background(), AccountAreaUsageQuery{}); err == nil || calls != 1 {
		t.Fatalf("usage error = %v; calls=%d", err, calls)
	}
	if _, err := api.AccountAreaListRunnerJobs(context.Background(), "", "", 0); err == nil {
		t.Fatal("runner jobs should report the API error")
	}
	if got := accountAreaQueryPath("/plans", nil); got != "/plans" {
		t.Fatalf("empty query path = %q", got)
	}
	var invalid AccountAreaMoney
	if err := json.Unmarshal([]byte(`true`), &invalid); err == nil {
		t.Fatal("invalid money JSON should fail")
	}
}

func TestAccountAreaPaginationRejectsMissingNextCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/usage" {
			_, _ = w.Write([]byte(`{"data":[],"meta":{"has_more":true,"next_cursor":null,"total":"0","currency":"CAD"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[],"meta":{"has_more":true,"next_cursor":null}}`))
	}))
	defer server.Close()
	api := NewClient(server.URL, "token")
	api.HTTPClient = server.Client()
	if _, err := api.AccountAreaListUsage(context.Background(), AccountAreaUsageQuery{}); err == nil || !strings.Contains(err.Error(), "new cursor") {
		t.Fatalf("usage pagination error = %v", err)
	}
	if _, err := api.AccountAreaListRunnerJobs(context.Background(), "", "", 0); err == nil || !strings.Contains(err.Error(), "new cursor") {
		t.Fatalf("job pagination error = %v", err)
	}
}

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNewClientUsesDefaultEndpoint(t *testing.T) {
	client := NewClient("", "token")

	if client.BaseURL != "https://api.ffxf.net/v1" {
		t.Errorf("NewClient() BaseURL = %q, want %q", client.BaseURL, "https://api.ffxf.net/v1")
	}
	if client.Token != "token" {
		t.Errorf("NewClient() Token = %q, want %q", client.Token, "token")
	}
}

func TestCreateVMSendsRequestAndDecodesResponse(t *testing.T) {
	wantRequest := CreateVmRequest{
		Plan:     "nano",
		Region:   "montreal",
		Image:    "debian-13",
		Hostname: "web-01",
		Billing:  "hourly",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vms" {
			t.Errorf("request = %s %s, want POST /vms", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("Idempotency-Key"); got != "request-123" {
			t.Errorf("Idempotency-Key = %q, want %q", got, "request-123")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		var gotRequest CreateVmRequest
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decode request body: %v", err)
		} else if !reflect.DeepEqual(gotRequest, wantRequest) {
			t.Errorf("request body = %#v, want %#v", gotRequest, wantRequest)
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"vm":{"id":42,"hostname":"web-01","status":"deploying","ipv4":"192.0.2.4"},"action":{"id":7}}}`))
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	response, err := apiClient.CreateVM(context.Background(), wantRequest, "request-123")
	if err != nil {
		t.Fatalf("CreateVM() error = %v", err)
	}
	if response.Data.VM.ID != 42 || response.Data.VM.IPv4 != "192.0.2.4" {
		t.Errorf("CreateVM() VM = %#v, want ID 42 and IPv4 192.0.2.4", response.Data.VM)
	}
	if response.Data.Action.ID != 7 {
		t.Errorf("CreateVM() Action.ID = %d, want 7", response.Data.Action.ID)
	}
}

func TestCreateVMRejectsUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	_, err := apiClient.CreateVM(context.Background(), CreateVmRequest{}, "request-123")
	if err == nil || !strings.Contains(err.Error(), "API HTTP error 400 during creation") {
		t.Fatalf("CreateVM() error = %v, want HTTP 400 error", err)
	}
}

func TestGetVMSendsAuthenticatedRequestAndDecodesVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vms/42" {
			t.Errorf("request = %s %s, want GET /vms/42", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		_, _ = w.Write([]byte(`{"data":{"id":42,"hostname":"web-01","status":"running","plan":"nano","region":"montreal","image":"debian-13","ipv4":"192.0.2.4","billing":{"mode":"hourly"}}}`))
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	response, err := apiClient.GetVM(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetVM() error = %v", err)
	}
	if response.Data.ID != 42 || response.Data.Hostname != "web-01" || response.Data.Billing.Mode != "hourly" {
		t.Errorf("GetVM() data = %#v, want decoded VM 42", response.Data)
	}
	if response.Data.IPv4 == nil || *response.Data.IPv4 != "192.0.2.4" {
		t.Errorf("GetVM() IPv4 = %v, want 192.0.2.4", response.Data.IPv4)
	}
}

func TestGetVMReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	_, err := apiClient.GetVM(context.Background(), 42)
	if !errors.Is(err, ErrVMNotFound) {
		t.Fatalf("GetVM() error = %v, want ErrVMNotFound", err)
	}
}

func TestUpdateVMPatchesHostname(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/vms/42" {
			t.Errorf("request = %s %s, want PATCH /vms/42", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		} else if len(body) != 1 || body["hostname"] != "web-02" {
			t.Errorf("PATCH body = %#v, want hostname only", body)
		}
		_, _ = w.Write([]byte(`{"data":{"id":42,"hostname":"web-02","status":"running","plan":"nano","region":"montreal","image":"debian-13","ipv4":null,"billing":{"mode":"hourly"}}}`))
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	response, err := apiClient.UpdateVM(context.Background(), 42, "web-02")
	if err != nil {
		t.Fatalf("UpdateVM() error = %v", err)
	}
	if response.Data.Hostname != "web-02" || response.Data.IPv4 != nil {
		t.Errorf("UpdateVM() data = %#v, want hostname web-02 and null IPv4", response.Data)
	}
}

func TestDeleteVMSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/vms/42" {
			t.Errorf("request = %s %s, want DELETE /vms/42", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("confirm"); got != "web-01" {
			t.Errorf("confirm = %q, want %q", got, "web-01")
		}
		if got := r.URL.Query().Get("when"); got != "now" {
			t.Errorf("when = %q, want %q", got, "now")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	if err := apiClient.DeleteVM(context.Background(), 42, "web-01"); err != nil {
		t.Fatalf("DeleteVM() error = %v, want nil", err)
	}
}

func TestDeleteVMRejectsUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	err := apiClient.DeleteVM(context.Background(), 42, "web-01")
	if err == nil || !strings.Contains(err.Error(), "HTTP status code 404") {
		t.Fatalf("DeleteVM() error = %v, want HTTP 404 error", err)
	}
}

func TestWaitForActionReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := NewClient("http://127.0.0.1", "test-token").WaitForAction(ctx, 7)
	if err != context.Canceled {
		t.Fatalf("WaitForAction() error = %v, want %v", err, context.Canceled)
	}
}

func TestWaitForActionReturnsOnCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/actions/7" {
			t.Errorf("request = %s %s, want GET /actions/7", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		_, _ = w.Write([]byte(`{"data":{"id":7,"status":"completed"}}`))
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	if err := apiClient.WaitForAction(context.Background(), 7); err != nil {
		t.Fatalf("WaitForAction() error = %v, want nil", err)
	}
}

func TestWaitForActionReturnsActionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/actions/8" {
			t.Errorf("request = %s %s, want GET /actions/8", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"id":8,"status":"error","error":{"message":"provisioning failed"}}}`))
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	err := apiClient.WaitForAction(context.Background(), 8)
	if err == nil || !strings.Contains(err.Error(), "provisioning failed") {
		t.Fatalf("WaitForAction() error = %v, want provisioning failure", err)
	}
}

func TestCatalogLookupsDecodeResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}

		switch r.URL.Path {
		case "/regions":
			_, _ = w.Write([]byte(`{"data":[{"slug":"montreal","name":"Montreal","country":"CA","status":"available","ipv4":true,"ipv6":true}]}`))
		case "/plans/nano":
			_, _ = w.Write([]byte(`{"data":{"slug":"nano","name":"Nano","description":"Small plan","vcpu":1,"memory_mb":2048,"disk_gb":20,"traffic_tb":1,"port_mbps":1000,"regions":["montreal"],"status":"available","prices":[{"currency":"CAD","hourly":"0.018","hourly_month_equivalent":"13.14","hourly_stopped":null,"monthly":"8.50","annual":"85.00","setup_fee":"0.00"}]}}`))
		case "/images/debian-13":
			_, _ = w.Write([]byte(`{"data":{"slug":"debian-13","name":"Debian 13","family":"debian","category":"linux","version":"13","status":"available","regions":["montreal"],"min_disk_gb":null,"min_memory_mb":null,"default_user":"debian","supports_ssh_keys":true}}`))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()

	regions, err := apiClient.ListRegions(context.Background())
	if err != nil {
		t.Fatalf("ListRegions() error = %v", err)
	}
	if len(regions.Data) != 1 || regions.Data[0].Slug != "montreal" || !regions.Data[0].IPv6 {
		t.Errorf("ListRegions() data = %#v, want Montreal with IPv6", regions.Data)
	}

	plan, err := apiClient.GetPlan(context.Background(), "nano")
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if plan.Data.Slug != "nano" || plan.Data.VCPU != 1 || plan.Data.TrafficTB == nil || *plan.Data.TrafficTB != 1 {
		t.Errorf("GetPlan() data = %#v, want decoded nano plan", plan.Data)
	}
	if len(plan.Data.Prices) != 1 || plan.Data.Prices[0].Currency != "CAD" || plan.Data.Prices[0].Hourly != "0.018" || plan.Data.Prices[0].HourlyStopped != nil {
		t.Errorf("GetPlan() prices = %#v, want CAD rates with no stopped rate", plan.Data.Prices)
	}

	image, err := apiClient.GetImage(context.Background(), "debian-13")
	if err != nil {
		t.Fatalf("GetImage() error = %v", err)
	}
	if image.Data.Slug != "debian-13" || image.Data.Version == nil || *image.Data.Version != "13" || !image.Data.SupportsSSHKeys {
		t.Errorf("GetImage() data = %#v, want decoded Debian 13 image", image.Data)
	}
}

func TestCatalogLookupsRejectHTTPAndMalformedJSON(t *testing.T) {
	lookups := []struct {
		name string
		call func(context.Context, *Client) error
	}{
		{
			name: "regions",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.ListRegions(ctx)
				return err
			},
		},
		{
			name: "plan",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetPlan(ctx, "nano")
				return err
			},
		},
		{
			name: "image",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetImage(ctx, "debian-13")
				return err
			},
		},
	}

	for _, lookup := range lookups {
		for _, response := range []struct {
			name       string
			statusCode int
			body       string
		}{
			{name: "non-200 status", statusCode: http.StatusServiceUnavailable},
			{name: "malformed JSON", statusCode: http.StatusOK, body: "{"},
		} {
			t.Run(lookup.name+"/"+response.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(response.statusCode)
					_, _ = w.Write([]byte(response.body))
				}))
				defer server.Close()

				apiClient := NewClient(server.URL, "test-token")
				apiClient.HTTPClient = server.Client()
				if err := lookup.call(context.Background(), apiClient); err == nil {
					t.Fatal("catalog lookup should reject the API response")
				}
			})
		}
	}
}

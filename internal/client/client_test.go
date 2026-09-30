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

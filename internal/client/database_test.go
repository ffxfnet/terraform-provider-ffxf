package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDatabaseClientRequestsAndErrors(t *testing.T) {
	seenCreate := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer database-token" {
			t.Errorf("Authorization = %q", got)
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /databases":
			seenCreate = true
			if got := r.Header.Get("Idempotency-Key"); got != "request-key" {
				t.Errorf("Idempotency-Key = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request: %v", err)
			}
			if body["name"] != "orders" || body["vpc"] != float64(8) {
				t.Errorf("body = %#v", body)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":12,"name":"orders","status":"pending","vpc":8}}`))
		case "GET /databases/12":
			_, _ = w.Write([]byte(`{"data":{"id":12,"name":"orders","status":"active","engine":"postgresql","version":"18","vpc":8,"size_gb":2,"created_at":"2026-01-01T00:00:00Z"}}`))
		case "GET /databases/404":
			w.WriteHeader(http.StatusNotFound)
		case "GET /databases/403":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	apiClient := NewClient(server.URL, "database-token")
	created, err := apiClient.DatabaseCreate(context.Background(), map[string]any{"name": "orders", "vpc": int64(8)}, "request-key")
	if err != nil {
		t.Fatalf("DatabaseCreate() error = %v", err)
	}
	if created.Data.ID != 12 || created.Data.Status != "pending" || !seenCreate {
		t.Fatalf("DatabaseCreate() response = %#v", created.Data)
	}
	got, err := apiClient.DatabaseGet(context.Background(), 12)
	if err != nil {
		t.Fatalf("DatabaseGet() error = %v", err)
	}
	if got.Data.Status != "active" || got.Data.SizeGB == nil || *got.Data.SizeGB != 2 {
		t.Fatalf("DatabaseGet() response = %#v", got.Data)
	}
	if _, err := apiClient.DatabaseGet(context.Background(), 404); !errors.Is(err, ErrNetworkResourceNotFound) {
		t.Errorf("404 error = %v, want ErrNetworkResourceNotFound", err)
	}
	if _, err := apiClient.DatabaseGet(context.Background(), 403); err == nil {
		t.Error("403 request should return an API error")
	}
}

func TestDatabaseClientResizeAndDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing bearer token")
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /databases/23/resize":
			if r.Header.Get("Idempotency-Key") != "resize-key" {
				t.Error("missing idempotency key")
			}
			var body map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["size_gb"] != 4 {
				t.Errorf("resize body = %#v, error %v", body, err)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":23,"status":"maintenance"}}`))
		case "DELETE /databases/23":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	apiClient := NewClient(server.URL, "token")
	if _, err := apiClient.DatabaseResize(context.Background(), 23, 4, "resize-key"); err != nil {
		t.Fatalf("DatabaseResize() error = %v", err)
	}
	if err := apiClient.DatabaseDelete(context.Background(), 23); err != nil {
		t.Fatalf("DatabaseDelete() error = %v", err)
	}
}

func TestDatabaseClientAuxiliaryEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /databases":
			_, _ = w.Write([]byte(`{"data":[],"quota":3}`))
		case "PATCH /databases/3":
			_, _ = w.Write([]byte(`{"data":{"id":3,"name":"new"}}`))
		case "GET /databases/options":
			_, _ = w.Write([]byte(`{"data":{"engines":[]}}`))
		case "GET /databases/3/metrics":
			if r.URL.Query().Get("range") != "7d" {
				t.Errorf("query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":{"range":"7d"}}`))
		case "POST /databases/3/forks":
			if r.Header.Get("Idempotency-Key") != "fork-key" {
				t.Error("missing fork idempotency key")
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"id":4,"status":"active"}}`))
		case "GET /databases/3/users/app":
			_, _ = w.Write([]byte(`{"data":{"username":"app"}}`))
		case "GET /databases/3/databases/analytics":
			_, _ = w.Write([]byte(`{"data":{"name":"analytics"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	apiClient := NewClient(server.URL, "token")
	list, err := apiClient.DatabaseList(context.Background())
	if err != nil || list.Quota != 3 {
		t.Fatalf("DatabaseList() = %#v, %v", list, err)
	}
	patched, err := apiClient.DatabasePatch(context.Background(), 3, map[string]any{"name": "new"})
	if err != nil || patched.Data.Name != "new" {
		t.Fatalf("DatabasePatch() = %#v, %v", patched, err)
	}
	var options map[string]json.RawMessage
	if err := apiClient.DatabaseOptions(context.Background(), &options); err != nil {
		t.Fatalf("DatabaseOptions() error = %v", err)
	}
	var metrics map[string]json.RawMessage
	if err := apiClient.DatabaseEndpoint(context.Background(), 3, "metrics", url.Values{"range": []string{"7d"}}, &metrics); err != nil {
		t.Fatalf("DatabaseEndpoint() error = %v", err)
	}
	fork, err := apiClient.DatabaseDerivedCreate(context.Background(), 3, "forks", map[string]any{"name": "copy"}, "fork-key")
	if err != nil || fork.Data.ID != 4 {
		t.Fatalf("DatabaseDerivedCreate() = %#v, %v", fork, err)
	}
	if got := apiClient.DatabaseUserPath(3, "app", ""); got != "/databases/3/users/app" {
		t.Errorf("DatabaseUserPath() = %q", got)
	}
	if got := apiClient.DatabaseLogicalPath(3, "analytics", ""); got != "/databases/3/databases/analytics" {
		t.Errorf("DatabaseLogicalPath() = %q", got)
	}
	if got := apiClient.DatabasePath(3, ""); got != "/databases/3" {
		t.Errorf("DatabasePath() = %q", got)
	}
	var target map[string]json.RawMessage
	if err := apiClient.DatabaseCall(context.Background(), http.MethodGet, "/databases/3/users/app", nil, &target, []int{http.StatusOK}, nil); err != nil {
		t.Fatalf("DatabaseCall() error = %v", err)
	}
	if err := apiClient.DatabaseCall(context.Background(), http.MethodGet, "/databases/3/databases/analytics", nil, &target, []int{http.StatusOK}, nil); err != nil {
		t.Fatalf("DatabaseCall() error = %v", err)
	}
}

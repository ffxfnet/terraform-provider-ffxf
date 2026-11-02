package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessClientEndpoints(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch r.Method + " " + r.URL.Path {
		case "POST /policies", "POST /roles", "POST /oidc/bindings":
			if r.Header.Get("Idempotency-Key") != "key-1" {
				t.Errorf("%s: missing idempotency key", r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":5,"name":"created","provider_id":2,"policy_ids":[1,2],"document":{"version":"2026-10-01","statements":[]}}}`))
		case "POST /api-keys":
			if r.Header.Get("Idempotency-Key") != "" {
				t.Error("API key creation must not send an idempotency key")
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":9,"name":"ci","token":"ffxf_secret","enabled":true,"role":{"id":5,"name":"deploy"}}}`))
		case "PUT /policies/5", "PUT /roles/5", "PUT /oidc/bindings/5":
			_, _ = w.Write([]byte(`{"data":{"id":5,"name":"updated"}}`))
		case "GET /policies", "GET /roles", "GET /api-keys", "GET /oidc/providers":
			_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"one","issuer":"https://token.actions.githubusercontent.com"}]}`))
		case "GET /policies/5", "GET /roles/5", "GET /api-keys/9", "GET /oidc/bindings/5":
			_, _ = w.Write([]byte(`{"data":{"id":5,"name":"one","claim_conditions":[{"claim":"sub","value":"repo:acme/*"}]}}`))
		case "DELETE /policies/5", "DELETE /roles/5", "DELETE /api-keys/9", "DELETE /oidc/bindings/5":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	ctx := context.Background()
	body := map[string]any{"name": "x"}

	if policy, err := c.AccessCreatePolicy(ctx, body, "key-1"); err != nil || policy.ID != 5 || len(policy.Document) == 0 {
		t.Fatalf("AccessCreatePolicy() = %#v, %v", policy, err)
	}
	if role, err := c.AccessCreateRole(ctx, body, "key-1"); err != nil || len(role.PolicyIDs) != 2 {
		t.Fatalf("AccessCreateRole() = %#v, %v", role, err)
	}
	if binding, err := c.AccessCreateOIDCBinding(ctx, body, "key-1"); err != nil || binding.ProviderID != 2 {
		t.Fatalf("AccessCreateOIDCBinding() = %#v, %v", binding, err)
	}
	if key, err := c.AccessCreateAPIKey(ctx, body); err != nil || key.Token != "ffxf_secret" || key.Role == nil || key.Role.Name != "deploy" {
		t.Fatalf("AccessCreateAPIKey() = %#v, %v", key, err)
	}
	if _, err := c.AccessUpdatePolicy(ctx, 5, body, "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccessUpdateRole(ctx, 5, body, "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccessUpdateOIDCBinding(ctx, 5, body, "k"); err != nil {
		t.Fatal(err)
	}
	if list, err := c.AccessListPolicies(ctx); err != nil || len(list) != 1 {
		t.Fatalf("AccessListPolicies() = %v, %v", list, err)
	}
	if list, err := c.AccessListRoles(ctx); err != nil || len(list) != 1 {
		t.Fatalf("AccessListRoles() = %v, %v", list, err)
	}
	if list, err := c.AccessListAPIKeys(ctx); err != nil || len(list) != 1 {
		t.Fatalf("AccessListAPIKeys() = %v, %v", list, err)
	}
	if list, err := c.AccessListOIDCProviders(ctx); err != nil || len(list) != 1 || list[0].Issuer == "" {
		t.Fatalf("AccessListOIDCProviders() = %v, %v", list, err)
	}
	if _, err := c.AccessGetPolicy(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccessGetRole(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccessGetAPIKey(ctx, 9); err != nil {
		t.Fatal(err)
	}
	if binding, err := c.AccessGetOIDCBinding(ctx, 5); err != nil || len(binding.ClaimConditions) != 1 {
		t.Fatalf("AccessGetOIDCBinding() = %#v, %v", binding, err)
	}
	for name, call := range map[string]func() error{
		"policy":  func() error { return c.AccessDeletePolicy(ctx, 5) },
		"role":    func() error { return c.AccessDeleteRole(ctx, 5) },
		"api key": func() error { return c.AccessDeleteAPIKey(ctx, 9) },
		"binding": func() error { return c.AccessDeleteOIDCBinding(ctx, 5) },
	} {
		if err := call(); err != nil {
			t.Errorf("delete %s: %v", name, err)
		}
	}
	if len(seen) != 19 {
		t.Errorf("made %d requests, want 19: %v", len(seen), seen)
	}
}

func TestAccessClientReportsNotFoundAndAPIMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "action_conflict", "message": "This policy is attached to a role: detach it first."}})
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	if _, err := c.AccessGetRole(context.Background(), 1); err != ErrNetworkResourceNotFound {
		t.Fatalf("AccessGetRole() error = %v, want ErrNetworkResourceNotFound", err)
	}
	err := c.AccessDeletePolicy(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "detach it first") {
		t.Fatalf("AccessDeletePolicy() error = %v, want the API message", err)
	}
}

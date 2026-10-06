package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNetworkOperations(t *testing.T) {
	firewallBody := `{"data":{"id":4,"name":"edge","rules":[],"members":[{"vm":6,"hostname":"web-01","status":"applied"}],"created_at":"2026-10-06T12:00:00Z"}}`
	loadBalancerBody := `{"data":{"id":8,"name":"edge","status":"active","vpc":12,"ipv4":"192.0.2.8","ipv6":"2001:db8::8","dns_name":"lb.example.test","config_applied":true,"last_seen_at":"2026-10-06T12:00:00Z","last_error":null,"pools":[],"listeners":[],"certificates":[],"billing":{"mode":"hourly","rate":"0.010","currency":"CAD"},"created_at":"2026-10-06T12:00:00Z"}}`
	poolBody := `{"data":{"id":9,"name":"frontend","protocol":"http","algorithm":"roundrobin","health_type":"http","health_path":"/health","health_interval":5,"send_proxy":false,"targets":[{"id":10,"vm":6,"hostname":"web-01","address":"10.0.0.2","port":3000,"weight":1,"drain":false,"health":"up","sessions":2,"response_ms":3}]}}`
	listenerBody := `{"data":{"id":11,"port":443,"protocol":"https","default_pool":9,"redirect_https":true,"sources":[],"rules":[{"hostname":"app.example.test","path_prefix":"/api","pool":9}]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /firewalls":
			_, _ = w.Write([]byte(`{"data":[{"id":4,"name":"edge","rules":[],"members":[],"created_at":"2026-10-06T12:00:00Z"}],"quota":10}`))
		case "GET /firewalls/4", "PUT /firewalls/4":
			_, _ = w.Write([]byte(firewallBody))
		case "DELETE /firewalls/4", "DELETE /firewalls/4/members/6", "DELETE /load-balancers/8/pools/9", "DELETE /load-balancers/8/pools/9/targets/10", "DELETE /load-balancers/8/listeners/11":
			w.WriteHeader(http.StatusNoContent)
		case "POST /firewalls/4/members":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"vm":6,"hostname":"web-01","status":"applied"}}`))
		case "GET /load-balancers":
			_, _ = w.Write([]byte(`{"data":[{"id":8,"name":"edge","status":"active","vpc":12,"pools":[],"listeners":[]}],"quota":3}`))
		case "GET /load-balancers/8", "PATCH /load-balancers/8":
			_, _ = w.Write([]byte(loadBalancerBody))
		case "DELETE /load-balancers/8":
			w.WriteHeader(http.StatusAccepted)
		case "GET /load-balancers/8/metrics":
			_, _ = w.Write([]byte(`{"data":{"range":"1h","step_minutes":1,"points":[],"totals":{"requests":1,"errors_pct":0,"bytes":4,"sessions_max":2}}}`))
		case "POST /load-balancers/8/pools":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(poolBody))
		case "PATCH /load-balancers/8/pools/9", "PUT /load-balancers/8/pools/9/targets":
			_, _ = w.Write([]byte(poolBody))
		case "PATCH /load-balancers/8/pools/9/targets/10":
			_, _ = w.Write([]byte(`{"data":{"id":10,"vm":6,"port":3000,"weight":2,"drain":true,"health":"up","sessions":0}}`))
		case "POST /load-balancers/8/listeners":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(listenerBody))
		case "PATCH /load-balancers/8/listeners/11", "PUT /load-balancers/8/listeners/11/rules":
			_, _ = w.Write([]byte(listenerBody))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient := NewClient(server.URL, "test-token")
	apiClient.HTTPClient = server.Client()
	ctx := context.Background()

	firewalls, err := apiClient.ListFirewalls(ctx)
	if err != nil || firewalls.Quota != 10 || len(firewalls.Data) != 1 {
		t.Fatalf("ListFirewalls() = %#v, %v", firewalls, err)
	}
	firewall, err := apiClient.GetFirewall(ctx, 4)
	if err != nil || firewall.Data.Members[0].Status != "applied" {
		t.Fatalf("GetFirewall() = %#v, %v", firewall, err)
	}
	if _, err := apiClient.UpdateFirewall(ctx, 4, "edge", []FirewallRule{}); err != nil {
		t.Fatalf("UpdateFirewall() error = %v", err)
	}
	member, err := apiClient.AttachFirewall(ctx, 4, 6)
	if err != nil || member.VM != 6 {
		t.Fatalf("AttachFirewall() = %#v, %v", member, err)
	}
	if err := apiClient.DetachFirewall(ctx, 4, 6); err != nil {
		t.Fatalf("DetachFirewall() error = %v", err)
	}
	if err := apiClient.DeleteFirewall(ctx, 4); err != nil {
		t.Fatalf("DeleteFirewall() error = %v", err)
	}

	loadBalancers, err := apiClient.ListLoadBalancers(ctx)
	if err != nil || loadBalancers.Quota != 3 || len(loadBalancers.Data) != 1 {
		t.Fatalf("ListLoadBalancers() = %#v, %v", loadBalancers, err)
	}
	lb, err := apiClient.GetLoadBalancer(ctx, 8)
	if err != nil || lb.Data.Billing == nil || lb.Data.IPv4 == nil {
		t.Fatalf("GetLoadBalancer() = %#v, %v", lb, err)
	}
	if _, err := apiClient.RenameLoadBalancer(ctx, 8, "edge"); err != nil {
		t.Fatalf("RenameLoadBalancer() error = %v", err)
	}
	if err := apiClient.DeleteLoadBalancer(ctx, 8); err != nil {
		t.Fatalf("DeleteLoadBalancer() error = %v", err)
	}
	if _, err := apiClient.GetLoadBalancerMetrics(ctx, 8, ""); err != nil {
		t.Fatalf("GetLoadBalancerMetrics() error = %v", err)
	}
	pool, err := apiClient.CreateLoadBalancerPool(ctx, 8, map[string]any{"name": "frontend"})
	if err != nil || pool.ID != 9 {
		t.Fatalf("CreateLoadBalancerPool() = %#v, %v", pool, err)
	}
	if _, err := apiClient.UpdateLoadBalancerPool(ctx, 8, 9, map[string]any{"name": "frontend"}); err != nil {
		t.Fatalf("UpdateLoadBalancerPool() error = %v", err)
	}
	if _, err := apiClient.ReplaceLoadBalancerTargets(ctx, 8, 9, []LoadBalancerTarget{{VM: 6, Port: 3000, Weight: 1}}); err != nil {
		t.Fatalf("ReplaceLoadBalancerTargets() error = %v", err)
	}
	if _, err := apiClient.UpdateLoadBalancerTarget(ctx, 8, 9, 10, map[string]any{"drain": true}); err != nil {
		t.Fatalf("UpdateLoadBalancerTarget() error = %v", err)
	}
	if err := apiClient.DeleteLoadBalancerTarget(ctx, 8, 9, 10); err != nil {
		t.Fatalf("DeleteLoadBalancerTarget() error = %v", err)
	}
	if err := apiClient.DeleteLoadBalancerPool(ctx, 8, 9); err != nil {
		t.Fatalf("DeleteLoadBalancerPool() error = %v", err)
	}
	listener, err := apiClient.CreateLoadBalancerListener(ctx, 8, map[string]any{"port": 443, "protocol": "https"})
	if err != nil || listener.ID != 11 {
		t.Fatalf("CreateLoadBalancerListener() = %#v, %v", listener, err)
	}
	if _, err := apiClient.UpdateLoadBalancerListener(ctx, 8, 11, map[string]any{"port": 443}); err != nil {
		t.Fatalf("UpdateLoadBalancerListener() error = %v", err)
	}
	if _, err := apiClient.ReplaceLoadBalancerRules(ctx, 8, 11, []LoadBalancerRule{{Pool: 9}}); err != nil {
		t.Fatalf("ReplaceLoadBalancerRules() error = %v", err)
	}
	if err := apiClient.DeleteLoadBalancerListener(ctx, 8, 11); err != nil {
		t.Fatalf("DeleteLoadBalancerListener() error = %v", err)
	}
}

func TestNetworkRequestErrors(t *testing.T) {
	t.Run("not found sentinel", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()
		apiClient := NewClient(server.URL, "token")
		apiClient.HTTPClient = server.Client()
		err := apiClient.networkRequest(context.Background(), http.MethodGet, "/missing", nil, nil, http.StatusOK)
		if !errors.Is(err, ErrNetworkResourceNotFound) {
			t.Fatalf("networkRequest() error = %v, want not found sentinel", err)
		}
	})

	t.Run("unexpected status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
		defer server.Close()
		apiClient := NewClient(server.URL, "token")
		apiClient.HTTPClient = server.Client()
		err := apiClient.networkRequest(context.Background(), http.MethodGet, "/bad", nil, nil, http.StatusOK)
		if err == nil || !strings.Contains(err.Error(), "API HTTP error 400") {
			t.Fatalf("networkRequest() error = %v, want HTTP 400", err)
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) }))
		defer server.Close()
		apiClient := NewClient(server.URL, "token")
		apiClient.HTTPClient = server.Client()
		var result map[string]any
		if err := apiClient.networkRequest(context.Background(), http.MethodGet, "/bad-json", nil, &result, http.StatusOK); err == nil {
			t.Fatal("networkRequest() should reject malformed JSON")
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		apiClient := NewClient("http://example.test", "token")
		err := apiClient.networkRequest(context.Background(), http.MethodPost, "/bad-body", make(chan int), nil, http.StatusOK)
		if err == nil {
			t.Fatal("networkRequest() should reject an unsupported body")
		}
	})

	t.Run("invalid request URL", func(t *testing.T) {
		apiClient := NewClient("http://example.test", "token")
		err := apiClient.networkRequest(context.Background(), http.MethodGet, "%", nil, nil, http.StatusOK)
		if err == nil {
			t.Fatal("networkRequest() should reject an invalid URL")
		}
	})
}

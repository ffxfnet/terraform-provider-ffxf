package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestVMAreaListVMsPaginatesAndFilters(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/vms" || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("request = %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("status") != "running" || r.URL.Query().Get("region") != "montreal" || r.URL.Query().Get("billing") != "hourly" || r.URL.Query().Get("limit") != "100" {
			t.Errorf("query = %v", r.URL.Query())
		}
		if calls == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":1,"hostname":"one"}],"meta":{"next_cursor":"next","has_more":true}}`))
			return
		}
		if r.URL.Query().Get("cursor") != "next" {
			t.Errorf("cursor = %q", r.URL.Query().Get("cursor"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":2,"hostname":"two"}],"meta":{"next_cursor":null,"has_more":false}}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	got, err := c.VMAreaListVMs(context.Background(), "running", "montreal", "hourly")
	if err != nil || len(got) != 2 || got[1].ID != 2 || calls != 2 {
		t.Fatalf("ListVMs() = %#v, calls=%d, err=%v", got, calls, err)
	}
}

func TestVMAreaListVMsRejectsPageWithoutCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"meta":{"has_more":true,"next_cursor":null}}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	if _, err := c.VMAreaListVMs(context.Background(), "", "", ""); err == nil || !strings.Contains(err.Error(), "without a cursor") {
		t.Fatalf("error=%v", err)
	}
}

func TestVMAreaNetworkMetricsAndBackups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("missing auth")
		}
		switch r.URL.Path {
		case "/vms/12/network":
			_, _ = w.Write([]byte(`{"data":{"ipv6":[{"address":"2001:db8::1","version":6,"primary":true,"rdns":"host.example"}],"ipv4":[],"port_mbps":100}}`))
		case "/vms/12/metrics":
			if r.URL.Query().Get("timeframe") != "day" {
				t.Errorf("timeframe=%q", r.URL.Query().Get("timeframe"))
			}
			_, _ = w.Write([]byte(`{"data":{"timeframe":"day","step_seconds":60,"series":[{"time":"2026-01-01T00:00:00Z","cpu":0.5,"memory_bytes":null}]}}`))
		case "/vms/12/backups":
			_, _ = w.Write([]byte(`{"data":[{"id":"backup-1","type":"manual","restorable":true}],"meta":{"automatic_enabled":true,"retention":7}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	n, err := c.VMAreaGetNetwork(context.Background(), 12)
	if err != nil || n.Data.IPv6[0].Address != "2001:db8::1" {
		t.Fatalf("network=%#v err=%v", n, err)
	}
	m, err := c.VMAreaGetMetrics(context.Background(), 12, "day")
	if err != nil || m.Data.StepSeconds != 60 || *m.Data.Series[0].CPU != 0.5 {
		t.Fatalf("metrics=%#v err=%v", m, err)
	}
	b, err := c.VMAreaGetBackups(context.Background(), 12)
	if err != nil || b.Meta.Retention != 7 || b.Data[0].ID != "backup-1" {
		t.Fatalf("backups=%#v err=%v", b, err)
	}
}

func TestVMAreaActionsPaginationAndStatus(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/actions" || r.URL.Query().Get("status") != "error" {
			t.Errorf("request %s %v", r.URL.Path, r.URL.Query())
		}
		if calls == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":1,"status":"error"}],"meta":{"has_more":true,"next_cursor":"c"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":2,"status":"error"}],"meta":{"has_more":false,"next_cursor":null}}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	a, err := c.VMAreaListActions(context.Background(), nil, "error")
	if err != nil || len(a) != 2 || calls != 2 {
		t.Fatalf("actions=%#v calls=%d err=%v", a, calls, err)
	}
}

func TestVMAreaMutationRequestShapes(t *testing.T) {
	paths := map[string]bool{"/vms/9/network/ipv6": false, "/vms/9/network/ipv6/2001:db8::5": false, "/vms/9/network/rdns": false, "/vms/9/actions": false, "/vms/9/backups/opaque%2Fid/restore": false}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("authorization missing")
		}
		if _, ok := paths[r.URL.EscapedPath()]; !ok {
			t.Errorf("unexpected escaped path %s", r.URL.EscapedPath())
		}
		paths[r.URL.EscapedPath()] = true
		switch r.URL.Path {
		case "/vms/9/network/ipv6":
			var b map[string]string
			_ = json.NewDecoder(r.Body).Decode(&b)
			if !reflect.DeepEqual(b, map[string]string{"address": "2001:db8::5"}) {
				t.Errorf("body=%v", b)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"address":"2001:db8::5","version":6}}`))
		case "/vms/9/network/ipv6/2001:db8::5":
			w.WriteHeader(http.StatusNoContent)
		case "/vms/9/network/rdns":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["hostname"] != nil {
				t.Errorf("rdns body=%v", b)
			}
			_, _ = w.Write([]byte(`{"data":{"address":"192.0.2.1"}}`))
		case "/vms/9/actions":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"action":{"id":33,"status":"queued"}}}`))
		case "/vms/9/backups/opaque/id/restore":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"action":{"id":34,"status":"queued"}}}`))
		}
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	if _, err := c.VMAreaAddIPv6(context.Background(), 9, "2001:db8::5"); err != nil {
		t.Fatal(err)
	}
	if err := c.VMAreaDeleteIPv6(context.Background(), 9, "2001:db8::5"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.VMAreaUpdateRDNS(context.Background(), 9, "192.0.2.1", nil); err != nil {
		t.Fatal(err)
	}
	a, err := c.VMAreaPowerAction(context.Background(), 9, "reboot")
	if err != nil || a.Data.Action.ID != 33 {
		t.Fatalf("action=%#v err=%v", a, err)
	}
	b, err := c.VMAreaRestoreBackup(context.Background(), 9, "opaque/id")
	if err != nil || b.Data.Action.ID != 34 {
		t.Fatalf("restore=%#v err=%v", b, err)
	}
	for path, ok := range paths {
		if !ok {
			t.Errorf("did not issue request to %s", path)
		}
	}
}

func TestVMAreaDeleteVMEncodesConfirmAndReturnsAction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/vms/9" || r.URL.Query().Get("confirm") != "host & name" || r.URL.Query().Get("when") != "period_end" || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("request=%s %s query=%v", r.Method, r.URL.Path, r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"data":{"action":{"id":81}}}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	id, err := c.VMAreaDeleteVMWithAction(context.Background(), 9, "host & name", "period_end")
	if err != nil || id != 81 {
		t.Fatalf("id=%d err=%v", id, err)
	}
}

func TestVMAreaOperationsReportAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	if _, err := c.VMAreaGetNetwork(context.Background(), 1); err == nil {
		t.Fatal("expected network API error")
	}
	if _, err := c.VMAreaAddIPv6(context.Background(), 1, "2001:db8::1"); err == nil {
		t.Fatal("expected IPv6 API error")
	}
	if err := c.VMAreaDeleteIPv6(context.Background(), 1, "2001:db8::1"); err == nil {
		t.Fatal("expected delete API error")
	}
}

func TestVMAreaReinstallAndPerVMActions(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/vms/5/actions" || r.Method != http.MethodPost {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["type"] != "reinstall" || body["image"] != "ubuntu-24-04" {
			t.Errorf("reinstall body=%v", body)
		}
		keys, ok := body["ssh_keys"].([]any)
		if !ok || len(keys) != 1 || keys[0] != "key-id" {
			t.Errorf("ssh_keys=%v", body["ssh_keys"])
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"action":{"id":4,"status":"queued"}}}`))
	}))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	result, err := c.VMAreaReinstall(context.Background(), 5, "ubuntu-24-04", []string{"key-id"})
	if err != nil || result.Data.Action.ID != 4 {
		t.Fatalf("reinstall=%#v err=%v", result, err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}

	actionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vms/5/actions" || r.URL.Query().Get("status") != "" {
			t.Errorf("request=%s %v", r.URL.Path, r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"data":[{"id":5,"status":"completed"}],"meta":{"has_more":false}}`))
	}))
	defer actionServer.Close()
	c = NewClient(actionServer.URL, "token")
	c.HTTPClient = actionServer.Client()
	vmID := int64(5)
	actions, err := c.VMAreaListActions(context.Background(), &vmID, "")
	if err != nil || len(actions) != 1 || actions[0].ID != 5 {
		t.Fatalf("actions=%#v err=%v", actions, err)
	}
	if _, err := c.VMAreaListActions(context.Background(), &vmID, "error"); err == nil {
		t.Fatal("status filtering on per-VM actions should be rejected")
	}
}

func TestVMAreaDeleteVMRejectsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusConflict) }))
	defer server.Close()
	c := NewClient(server.URL, "token")
	c.HTTPClient = server.Client()
	if _, err := c.VMAreaDeleteVMWithAction(context.Background(), 3, "host", "now"); err == nil {
		t.Fatal("expected delete API error")
	}
}

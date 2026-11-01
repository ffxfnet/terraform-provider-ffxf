package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ffxfnet/terraform-provider-ffxf/internal/client"
)

func TestVMPowerStateSkipsActionWhenAlreadyInWantedState(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		model      VMAreaPowerStateModel
		wantAction string
	}{
		{"running VM is not started again", "running", VMAreaPowerStateModel{State: types.StringValue("running"), Method: types.StringValue("graceful")}, ""},
		{"stopped VM is not shut down again", "stopped", VMAreaPowerStateModel{State: types.StringValue("stopped"), Method: types.StringValue("graceful")}, ""},
		{"stopped VM is not stopped again", "stopped", VMAreaPowerStateModel{State: types.StringValue("stopped"), Method: types.StringValue("forced")}, ""},
		{"stopped VM is started", "stopped", VMAreaPowerStateModel{State: types.StringValue("running"), Method: types.StringValue("graceful")}, "start"},
		{"running VM is shut down", "running", VMAreaPowerStateModel{State: types.StringValue("stopped"), Method: types.StringValue("graceful")}, "shutdown"},
		{"reboot ignores the current state", "running", VMAreaPowerStateModel{State: types.StringValue("running"), Method: types.StringValue("graceful"), RebootTrigger: types.StringValue("x")}, "reboot"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var actions []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				switch r.Method + " " + r.URL.Path {
				case "GET /vms/7":
					_, _ = w.Write([]byte(`{"data":{"id":7,"status":"` + test.status + `"}}`))
				case "POST /vms/7/actions":
					actions = append(actions, "called")
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			model := test.model
			model.VMID = types.Int64Value(7)
			if err := vmAreaApplyPower(context.Background(), testAPIClient(server), &model, !model.RebootTrigger.IsNull() && model.RebootTrigger.ValueString() != ""); err != nil {
				t.Fatalf("vmAreaApplyPower() error = %v", err)
			}
			if got := len(actions) == 1; got != (test.wantAction != "") {
				t.Errorf("action called = %v, want %v", got, test.wantAction != "")
			}
		})
	}
}

func TestVMPowerStateStillActsWhenStatusLookupFails(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		called = true
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"action":{"id":0}}}`))
	}))
	defer server.Close()
	model := VMAreaPowerStateModel{VMID: types.Int64Value(7), State: types.StringValue("running"), Method: types.StringValue("graceful")}
	if err := vmAreaApplyPower(context.Background(), testAPIClient(server), &model, false); err != nil || !called {
		t.Fatalf("vmAreaApplyPower() = %v, called = %v, want the action to be attempted", err, called)
	}
}

func TestFirewallModelKeepsOmittedAnywhereListsNull(t *testing.T) {
	ctx := context.Background()
	stringList := func(values ...string) types.List {
		elements := make([]attr.Value, 0, len(values))
		for _, value := range values {
			elements = append(elements, types.StringValue(value))
		}
		list, _ := types.ListValue(types.StringType, elements)
		return list
	}
	ruleTypes := map[string]attr.Type{
		"direction": types.StringType, "protocol": types.StringType, "ports": types.StringType,
		"sources": types.ListType{ElemType: types.StringType}, "destinations": types.ListType{ElemType: types.StringType}, "description": types.StringType,
	}
	priorRules := func(sources types.List) types.List {
		list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ruleTypes}, []FirewallRuleModel{{
			Direction: types.StringValue("in"), Protocol: types.StringValue("icmp"), Ports: types.StringNull(),
			Sources: sources, Destinations: types.ListNull(types.StringType), Description: types.StringNull(),
		}})
		if diags.HasError() {
			t.Fatal(diags)
		}
		return list
	}
	anywhere := []string{"0.0.0.0/0", "::/0"}
	tests := []struct {
		name       string
		prior      types.List
		api        []string
		wantNull   bool
		wantLength int
	}{
		{"explicit empty sources stay empty", stringList(), anywhere, false, 0},
		{"null sources stay null", types.ListNull(types.StringType), anywhere, true, 0},
		{"null sources stay null for reversed defaults", types.ListNull(types.StringType), []string{"::/0", "0.0.0.0/0"}, true, 0},
		{"configured sources are kept", stringList("203.0.113.0/24"), []string{"203.0.113.0/24"}, false, 1},
		{"explicit sources that equal anywhere are kept", stringList("0.0.0.0/0", "::/0"), anywhere, false, 2},
		{"a single anywhere address is not the default", types.ListNull(types.StringType), []string{"0.0.0.0/0"}, false, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := FirewallResourceModel{Rules: priorRules(test.prior)}
			var diagnostics diag.Diagnostics
			setFirewallModel(ctx, &state, client.Firewall{ID: 1, Name: "web", Rules: []client.FirewallRule{{Direction: "in", Protocol: "icmp", Sources: test.api}}}, &diagnostics)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			var rules []FirewallRuleModel
			diagnostics.Append(state.Rules.ElementsAs(ctx, &rules, false)...)
			if diagnostics.HasError() || len(rules) != 1 {
				t.Fatalf("rules = %v, diagnostics = %v", rules, diagnostics)
			}
			if rules[0].Sources.IsNull() != test.wantNull {
				t.Errorf("sources null = %v, want %v", rules[0].Sources.IsNull(), test.wantNull)
			}
			if !test.wantNull && len(rules[0].Sources.Elements()) != test.wantLength {
				t.Errorf("sources length = %d, want %d", len(rules[0].Sources.Elements()), test.wantLength)
			}
			// The destinations were null in the plan and the API returned none, so they must stay null.
			if !rules[0].Destinations.IsNull() {
				t.Errorf("destinations = %v, want null", rules[0].Destinations)
			}
		})
	}
}

func TestFirewallReadWithoutPriorStateReturnsAPIValues(t *testing.T) {
	ctx := context.Background()
	state := FirewallResourceModel{Rules: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}})}
	var diagnostics diag.Diagnostics
	setFirewallModel(ctx, &state, client.Firewall{ID: 1, Name: "web", Rules: []client.FirewallRule{{Direction: "in", Protocol: "tcp", Sources: []string{"0.0.0.0/0", "::/0"}}}}, &diagnostics)
	var rules []FirewallRuleModel
	diagnostics.Append(state.Rules.ElementsAs(ctx, &rules, false)...)
	if diagnostics.HasError() || len(rules) != 1 || len(rules[0].Sources.Elements()) != 2 {
		t.Fatalf("imported rules = %v, diagnostics = %v", rules, diagnostics)
	}
}

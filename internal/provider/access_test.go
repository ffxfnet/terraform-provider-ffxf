package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func accessResourceSchema(t *testing.T, r resource.Resource) resource.SchemaResponse {
	t.Helper()
	var response resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("schema: %v", response.Diagnostics)
	}
	return response
}

func accessPlan(t *testing.T, r resource.Resource, model any) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: accessResourceSchema(t, r).Schema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return plan
}

func accessState(t *testing.T, r resource.Resource, model any) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: accessResourceSchema(t, r).Schema}
	if diags := state.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	return state
}

func accessStrings(values ...string) types.List {
	items := make([]attr.Value, 0, len(values))
	for _, value := range values {
		items = append(items, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, items)
}

func TestAccessPolicyResourceLifecycle(t *testing.T) {
	remote := `{"version":"2026-10-01","statements":[{"effect":"allow","actions":["vm:read"],"resources":["*"]}]}`
	var createdBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /policies":
			if err := json.NewDecoder(r.Body).Decode(&createdBody); err != nil {
				t.Errorf("decode body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":4,"name":"readers","description":null,"document":` + remote + `,"created_at":"2026-01-01T00:00:00Z"}}`))
		case "GET /policies/4":
			_, _ = w.Write([]byte(`{"data":{"id":4,"name":"readers","description":null,"document":` + remote + `}}`))
		case "PUT /policies/4":
			_, _ = w.Write([]byte(`{"data":{"id":4,"name":"readers-v2","description":"all","document":` + remote + `}}`))
		case "DELETE /policies/4":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &AccessPolicyResource{client: testAPIClient(server)}
	ctx := context.Background()

	document := `{ "statements": [ {"resources": ["*"], "actions": ["vm:read"], "effect": "allow"} ] }`
	plan := accessPlan(t, r, accessPolicyModel{ID: types.Int64Unknown(), Name: types.StringValue("readers"), Description: types.StringNull(), Document: types.StringValue(document), NormalizedDocument: types.StringUnknown(), CreatedAt: types.StringUnknown()})
	created := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	if createdBody["document"] != document || createdBody["name"] != "readers" {
		t.Fatalf("create body = %#v", createdBody)
	}
	var model accessPolicyModel
	created.State.Get(ctx, &model)
	if model.ID.ValueInt64() != 4 || model.Document.ValueString() != document || !accessJSONEqual(model.NormalizedDocument.ValueString(), remote) || !model.Description.IsNull() {
		t.Fatalf("created model = %#v", model)
	}

	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	read.State.Get(ctx, &model)
	if read.Diagnostics.HasError() || model.Document.ValueString() != document {
		t.Fatalf("an unchanged remote policy must keep the configured text, got %q (%v)", model.Document.ValueString(), read.Diagnostics)
	}

	drifted := model
	drifted.NormalizedDocument = types.StringValue(`{"version":"2026-10-01","statements":[]}`)
	driftState := accessState(t, r, drifted)
	driftRead := resource.ReadResponse{State: driftState}
	r.Read(ctx, resource.ReadRequest{State: driftState}, &driftRead)
	driftRead.State.Get(ctx, &model)
	if model.Document.ValueString() != accessCompactJSON(json.RawMessage(remote)) {
		t.Fatalf("a changed remote policy must replace the document, got %q", model.Document.ValueString())
	}

	next := model
	next.Name, next.Description = types.StringValue("readers-v2"), types.StringValue("all")
	updatePlan := accessPlan(t, r, next)
	updated := resource.UpdateResponse{State: created.State}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: created.State}, &updated)
	updated.State.Get(ctx, &model)
	if updated.Diagnostics.HasError() || model.Name.ValueString() != "readers-v2" || model.Description.ValueString() != "all" || model.ID.ValueInt64() != 4 {
		t.Fatalf("Update: %#v %v", model, updated.Diagnostics)
	}

	deleted := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: created.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleted.Diagnostics)
	}
}

func TestAccessResourcesRemoveMissingAndTolerateMissingDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer server.Close()
	api := testAPIClient(server)
	ctx := context.Background()
	for name, tc := range map[string]struct {
		resource resource.Resource
		model    any
	}{
		"policy":  {&AccessPolicyResource{client: api}, accessPolicyModel{ID: types.Int64Value(1), Name: types.StringValue("x"), Document: types.StringValue("{}"), Description: types.StringNull(), NormalizedDocument: types.StringNull(), CreatedAt: types.StringNull()}},
		"role":    {&AccessRoleResource{client: api}, accessRoleModel{ID: types.Int64Value(1), Name: types.StringValue("x"), Description: types.StringNull(), PolicyIDs: types.SetValueMust(types.Int64Type, nil), CreatedAt: types.StringNull()}},
		"api key": {&AccessAPIKeyResource{client: api}, accessAPIKeyModel{ID: types.Int64Value(1), Name: types.StringValue("x"), Role: types.StringNull(), ExchangeOnly: types.BoolValue(false), AssumableRoles: types.ListNull(types.StringType), IPAddresses: types.ListNull(types.StringType), ExpiresAt: types.StringNull(), RoleID: types.Int64Null(), Prefix: types.StringNull(), Enabled: types.BoolValue(true), Token: types.StringValue("t"), LastUsedAt: types.StringNull(), CreatedAt: types.StringNull()}},
		"binding": {&AccessOIDCBindingResource{client: api}, accessOIDCBindingModel{ID: types.Int64Value(1), ProviderID: types.Int64Value(1), Name: types.StringValue("x"), ClaimConditions: types.ListValueMust(accessClaimConditionType, nil), Role: types.StringNull(), ExchangeOnly: types.BoolValue(false), AssumableRoles: types.ListNull(types.StringType), ExpiresAt: types.StringNull(), RoleID: types.Int64Null(), Enabled: types.BoolValue(true), CreatedAt: types.StringNull()}},
	} {
		t.Run(name, func(t *testing.T) {
			state := accessState(t, tc.resource, tc.model)
			read := resource.ReadResponse{State: state}
			tc.resource.Read(ctx, resource.ReadRequest{State: state}, &read)
			if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
				t.Fatalf("Read must drop a missing object: %v", read.Diagnostics)
			}
			deleted := resource.DeleteResponse{}
			tc.resource.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
			if deleted.Diagnostics.HasError() {
				t.Fatalf("Delete must tolerate a missing object: %v", deleted.Diagnostics)
			}
		})
	}
}

func TestAccessRoleResourceLifecycle(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /roles":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":6,"name":"deploy","description":null,"policy_ids":[1,3],"created_at":"2026-01-01T00:00:00Z"}}`))
		case "GET /roles/6":
			_, _ = w.Write([]byte(`{"data":{"id":6,"name":"deploy","description":null,"policy_ids":[3,1,9]}}`))
		case "PUT /roles/6":
			_, _ = w.Write([]byte(`{"data":{"id":6,"name":"deploy","description":null,"policy_ids":[3]}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &AccessRoleResource{client: testAPIClient(server)}
	ctx := context.Background()
	ids := func(values ...int64) types.Set {
		set, _ := types.SetValueFrom(ctx, types.Int64Type, values)
		return set
	}
	plan := accessPlan(t, r, accessRoleModel{ID: types.Int64Unknown(), Name: types.StringValue("deploy"), Description: types.StringNull(), PolicyIDs: ids(1, 3), CreatedAt: types.StringUnknown()})
	created := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	var model accessRoleModel
	created.State.Get(ctx, &model)
	if model.ID.ValueInt64() != 6 || len(model.PolicyIDs.Elements()) != 2 || bodies[0]["name"] != "deploy" || len(bodies[0]["policy_ids"].([]any)) != 2 {
		t.Fatalf("created = %#v body = %#v", model, bodies[0])
	}
	if _, present := bodies[0]["description"]; present {
		t.Error("a null description must not be sent")
	}

	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	read.State.Get(ctx, &model)
	if read.Diagnostics.HasError() || len(model.PolicyIDs.Elements()) != 3 {
		t.Fatalf("Read must report remote policy drift: %v %v", model.PolicyIDs, read.Diagnostics)
	}

	next := model
	next.PolicyIDs = ids(3)
	updatePlan := accessPlan(t, r, next)
	updated := resource.UpdateResponse{State: read.State}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: read.State}, &updated)
	updated.State.Get(ctx, &model)
	if updated.Diagnostics.HasError() || len(model.PolicyIDs.Elements()) != 1 {
		t.Fatalf("Update: %v %v", model, updated.Diagnostics)
	}
}

func TestAccessAPIKeyResourceKeepsSecretAndReplaces(t *testing.T) {
	var createdBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api-keys":
			if r.Header.Get("Idempotency-Key") != "" {
				t.Error("key creation must not be replayable")
			}
			_ = json.NewDecoder(r.Body).Decode(&createdBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":9,"name":"ci","prefix":"ffxf_ab","enabled":true,"role":{"id":6,"name":"deploy"},"exchange_only":false,"assumable_roles":[],"ip_addresses":["203.0.113.4"],"expires_at":"2030-01-02T23:59:59Z","token":"ffxf_secret"}}`))
		case "GET /api-keys/9":
			_, _ = w.Write([]byte(`{"data":{"id":9,"name":"ci","prefix":"ffxf_ab","enabled":false,"role":{"id":6,"name":"deploy"},"exchange_only":false,"assumable_roles":[],"ip_addresses":["203.0.113.4"],"expires_at":"2030-01-02T23:59:59Z","last_used_at":"2026-02-01T00:00:00Z"}}`))
		case "DELETE /api-keys/9":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &AccessAPIKeyResource{client: testAPIClient(server)}
	ctx := context.Background()
	plan := accessPlan(t, r, accessAPIKeyModel{ID: types.Int64Unknown(), Name: types.StringValue("ci"), Role: types.StringValue("deploy"), ExchangeOnly: types.BoolValue(false), AssumableRoles: types.ListNull(types.StringType), IPAddresses: accessStrings("203.0.113.4"), ExpiresAt: types.StringValue("2030-01-02"), RoleID: types.Int64Unknown(), Prefix: types.StringUnknown(), Enabled: types.BoolUnknown(), Token: types.StringUnknown(), LastUsedAt: types.StringUnknown(), CreatedAt: types.StringUnknown()})
	created := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	var model accessAPIKeyModel
	created.State.Get(ctx, &model)
	if model.Token.ValueString() != "ffxf_secret" || model.RoleID.ValueInt64() != 6 || model.ExpiresAt.ValueString() != "2030-01-02" || model.Prefix.ValueString() != "ffxf_ab" {
		t.Fatalf("created = %#v", model)
	}
	if createdBody["role"] != "deploy" || createdBody["expires_at"] != "2030-01-02" || createdBody["exchange_only"] != false {
		t.Fatalf("create body = %#v", createdBody)
	}
	if _, present := createdBody["assumable_roles"]; present {
		t.Error("an unset assumable_roles must not be sent")
	}

	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	read.State.Get(ctx, &model)
	if read.Diagnostics.HasError() || model.Token.ValueString() != "ffxf_secret" || model.Enabled.ValueBool() || model.LastUsedAt.ValueString() == "" {
		t.Fatalf("Read must keep the secret and refresh computed values: %#v %v", model, read.Diagnostics)
	}

	imported := model
	imported.Token, imported.Role, imported.IPAddresses = types.StringNull(), types.StringNull(), types.ListNull(types.StringType)
	importState := accessState(t, r, imported)
	importRead := resource.ReadResponse{State: importState}
	r.Read(ctx, resource.ReadRequest{State: importState}, &importRead)
	importRead.State.Get(ctx, &model)
	if model.Token.IsNull() || model.Token.ValueString() != "" || model.Role.ValueString() != "deploy" || len(model.IPAddresses.Elements()) != 1 {
		t.Fatalf("an imported key must be fully populated without a secret: %#v", model)
	}

	updated := resource.UpdateResponse{}
	r.Update(ctx, resource.UpdateRequest{}, &updated)
	if !updated.Diagnostics.HasError() {
		t.Fatal("Update must be rejected because every change replaces the key")
	}
	deleted := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: created.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleted.Diagnostics)
	}
}

func TestAccessAPIKeyResourceExchangeOnlySendsAssumableRoles(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":10,"name":"ex","enabled":true,"role":null,"exchange_only":true,"assumable_roles":[{"id":6,"name":"deploy"}],"token":"ffxf_x"}}`))
	}))
	defer server.Close()
	r := &AccessAPIKeyResource{client: testAPIClient(server)}
	plan := accessPlan(t, r, accessAPIKeyModel{ID: types.Int64Unknown(), Name: types.StringValue("ex"), Role: types.StringNull(), ExchangeOnly: types.BoolValue(true), AssumableRoles: accessStrings("deploy", "7"), IPAddresses: types.ListNull(types.StringType), ExpiresAt: types.StringNull(), RoleID: types.Int64Unknown(), Prefix: types.StringUnknown(), Enabled: types.BoolUnknown(), Token: types.StringUnknown(), LastUsedAt: types.StringUnknown(), CreatedAt: types.StringUnknown()})
	created := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	roles := body["assumable_roles"].([]any)
	if len(roles) != 2 || roles[0] != "deploy" || roles[1] != float64(7) || body["exchange_only"] != true {
		t.Fatalf("body = %#v, numeric roles must be sent as ids and names as names", body)
	}
	var model accessAPIKeyModel
	created.State.Get(context.Background(), &model)
	if !model.RoleID.IsNull() || !model.ExchangeOnly.ValueBool() {
		t.Fatalf("exchange-only key = %#v", model)
	}
}

func TestAccessOIDCBindingResourceLifecycle(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("missing idempotency key")
			}
		}
		binding := `{"id":3,"provider_id":2,"name":"ci","claim_conditions":[{"claim":"sub","value":"repo:acme/app:*"}],"role":{"id":6,"name":"deploy"},"exchange_only":false,"assumable_roles":[],"enabled":true,"expires_at":null,"created_at":"2026-01-01T00:00:00Z"}`
		switch r.Method + " " + r.URL.Path {
		case "POST /oidc/bindings":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":` + binding + `}`))
		case "GET /oidc/bindings/3", "PUT /oidc/bindings/3":
			_, _ = w.Write([]byte(`{"data":` + binding + `}`))
		case "DELETE /oidc/bindings/3":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	r := &AccessOIDCBindingResource{client: testAPIClient(server)}
	ctx := context.Background()
	conditions := types.ListValueMust(accessClaimConditionType, []attr.Value{types.ObjectValueMust(accessClaimConditionType.AttrTypes, map[string]attr.Value{"claim": types.StringValue("sub"), "value": types.StringValue("repo:acme/app:*")})})
	plan := accessPlan(t, r, accessOIDCBindingModel{ID: types.Int64Unknown(), ProviderID: types.Int64Value(2), Name: types.StringValue("ci"), ClaimConditions: conditions, Role: types.StringValue("deploy"), ExchangeOnly: types.BoolValue(false), AssumableRoles: types.ListNull(types.StringType), ExpiresAt: types.StringNull(), RoleID: types.Int64Unknown(), Enabled: types.BoolUnknown(), CreatedAt: types.StringUnknown()})
	created := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	claims := bodies[0]["claim_conditions"].([]any)
	if bodies[0]["provider_id"] != float64(2) || bodies[0]["role"] != "deploy" || len(claims) != 1 || claims[0].(map[string]any)["claim"] != "sub" {
		t.Fatalf("create body = %#v", bodies[0])
	}
	var model accessOIDCBindingModel
	created.State.Get(ctx, &model)
	if model.ID.ValueInt64() != 3 || model.RoleID.ValueInt64() != 6 || !model.Enabled.ValueBool() || !model.ExpiresAt.IsNull() {
		t.Fatalf("created = %#v", model)
	}

	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatalf("Read: %v", read.Diagnostics)
	}

	updatePlan := accessPlan(t, r, model)
	updated := resource.UpdateResponse{State: read.State}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: read.State}, &updated)
	if updated.Diagnostics.HasError() || len(bodies) != 2 {
		t.Fatalf("Update: %v (%d writes)", updated.Diagnostics, len(bodies))
	}
	deleted := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: created.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleted.Diagnostics)
	}
}

func TestAccessResourcesSurfaceAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"forbidden","message":"A credential cannot modify a role it holds or can assume."}}`))
	}))
	defer server.Close()
	api := testAPIClient(server)
	ctx := context.Background()
	role := &AccessRoleResource{client: api}
	plan := accessPlan(t, role, accessRoleModel{ID: types.Int64Unknown(), Name: types.StringValue("x"), Description: types.StringNull(), PolicyIDs: types.SetValueMust(types.Int64Type, nil), CreatedAt: types.StringUnknown()})
	created := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	role.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if !created.Diagnostics.HasError() || created.Diagnostics[0].Detail() == "" {
		t.Fatalf("Create must report the API error: %v", created.Diagnostics)
	}
	state := accessState(t, role, accessRoleModel{ID: types.Int64Value(1), Name: types.StringValue("x"), Description: types.StringNull(), PolicyIDs: types.SetValueMust(types.Int64Type, nil), CreatedAt: types.StringNull()})
	deleted := resource.DeleteResponse{}
	role.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
	if !deleted.Diagnostics.HasError() {
		t.Fatal("Delete must report a forbidden response")
	}
}

func TestAccessDataSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/policies":
			_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"vms.read","description":"Read VMs","provided":true,"document":{"version":"2026-10-01","statements":[]},"roles_count":2,"created_at":"2026-01-01T00:00:00Z"},{"id":2,"name":"mine","description":null,"provided":false,"document":{"version":"2026-10-01","statements":[]},"roles_count":null,"created_at":null}]}`))
		case "/roles":
			_, _ = w.Write([]byte(`{"data":[{"id":6,"name":"deploy","description":null,"policy_ids":[1,2],"created_at":"2026-01-01T00:00:00Z"},{"id":7,"name":"empty","policy_ids":null}]}`))
		case "/oidc/providers":
			_, _ = w.Write([]byte(`{"data":[{"id":2,"name":"GitHub","issuer":"https://token.actions.githubusercontent.com","audience":"ffxf","enabled":true}]}`))
		case "/api-keys":
			_, _ = w.Write([]byte(`{"data":[{"id":9,"name":"ci","prefix":"ffxf_ab","enabled":true,"role":{"id":6,"name":"deploy"},"exchange_only":false,"assumable_roles":[{"id":7,"name":"empty"}],"ip_addresses":null,"expires_at":null},{"id":10,"name":"ex","enabled":true,"role":null,"exchange_only":true}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	for name, tc := range map[string]struct {
		source datasource.DataSource
		attr   string
		count  int
	}{
		"policies":       {NewAccessPoliciesDataSource(), "policies", 2},
		"roles":          {NewAccessRolesDataSource(), "roles", 2},
		"oidc providers": {NewAccessOIDCProvidersDataSource(), "providers", 1},
		"api keys":       {NewAccessAPIKeysDataSource(), "api_keys", 2},
	} {
		t.Run(name, func(t *testing.T) {
			var metadata datasource.MetadataResponse
			tc.source.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ffxf"}, &metadata)
			var sch datasource.SchemaResponse
			tc.source.Schema(ctx, datasource.SchemaRequest{}, &sch)
			if sch.Diagnostics.HasError() {
				t.Fatalf("%s schema: %v", metadata.TypeName, sch.Diagnostics)
			}
			for attrName, attribute := range sch.Schema.Attributes {
				if attribute.GetDescription() == "" {
					t.Errorf("attribute %q has no description", attrName)
				}
			}
			tc.source.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
			response := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
			tc.source.Read(ctx, datasource.ReadRequest{}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("Read: %v", response.Diagnostics)
			}
			var items types.List
			response.State.GetAttribute(ctx, pathRoot(tc.attr), &items)
			if len(items.Elements()) != tc.count {
				t.Fatalf("%s: %d items, want %d", tc.attr, len(items.Elements()), tc.count)
			}
		})
	}
}

func TestAccessDataSourcesReportAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	ctx := context.Background()
	for _, source := range []datasource.DataSource{NewAccessPoliciesDataSource(), NewAccessRolesDataSource(), NewAccessOIDCProvidersDataSource(), NewAccessAPIKeysDataSource()} {
		var sch datasource.SchemaResponse
		source.Schema(ctx, datasource.SchemaRequest{}, &sch)
		source.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: testAPIClient(server)}, &datasource.ConfigureResponse{})
		response := datasource.ReadResponse{State: tfsdk.State{Schema: sch.Schema}}
		source.Read(ctx, datasource.ReadRequest{}, &response)
		if !response.Diagnostics.HasError() {
			t.Errorf("%T must report a 403", source)
		}
	}
}

func TestAccessImportStateRejectsNonNumericIdentifier(t *testing.T) {
	r := &AccessRoleResource{}
	sch := accessResourceSchema(t, r)
	response := resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: tftypesNullObject(sch)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "deploy"}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("a name is not a valid import identifier")
	}
	good := resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: tftypesNullObject(sch)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "12"}, &good)
	var id types.Int64
	good.State.GetAttribute(context.Background(), pathRoot("id"), &id)
	if good.Diagnostics.HasError() || id.ValueInt64() != 12 {
		t.Fatalf("import = %v %v", id, good.Diagnostics)
	}
}

func TestAccessConfigureRejectsWrongProviderData(t *testing.T) {
	response := resource.ConfigureResponse{}
	(&AccessPolicyResource{}).Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong"}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("unexpected provider data must be rejected")
	}
	empty := resource.ConfigureResponse{}
	(&AccessPolicyResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &empty)
	if empty.Diagnostics.HasError() {
		t.Fatal("a nil provider data is fine before the provider is configured")
	}
}

func TestAccessHelpers(t *testing.T) {
	if !accessJSONEqual(`{"a":1,"b":[1,2]}`, `{ "b": [1, 2], "a": 1 }`) || accessJSONEqual(`{"a":1}`, `{"a":2}`) || accessJSONEqual(`nope`, `{}`) {
		t.Error("accessJSONEqual")
	}
	if accessCompactJSON(json.RawMessage(`{ "a" : 1 }`)) != `{"a":1}` || accessCompactJSON(json.RawMessage(`{`)) != `{` {
		t.Error("accessCompactJSON")
	}
	if accessRoleValue("12") != int64(12) || accessRoleValue("deploy") != "deploy" {
		t.Error("accessRoleValue")
	}
	prior := "2030-05-06"
	long := "2030-01-02T23:59:59Z"
	short := "2030"
	if got := accessDate(types.StringValue(prior), &long); got.ValueString() != prior {
		t.Errorf("a configured date must be kept, got %q", got.ValueString())
	}
	if got := accessDate(types.StringNull(), &long); got.ValueString() != "2030-01-02" {
		t.Errorf("an imported timestamp must be shortened to a date, got %q", got.ValueString())
	}
	if got := accessDate(types.StringNull(), &short); got.ValueString() != "2030" || !accessDate(types.StringValue(prior), nil).IsNull() {
		t.Error("accessDate edge cases")
	}
	empty := ""
	if !accessOptionalString(types.StringNull(), &empty).IsNull() || accessOptionalString(types.StringValue("x"), &empty).ValueString() != "" {
		t.Error("accessOptionalString")
	}
}

func pathRoot(name string) path.Path { return path.Root(name) }

func tftypesNullObject(response resource.SchemaResponse) tftypes.Value {
	return tftypes.NewValue(response.Schema.Type().TerraformType(context.Background()), nil)
}

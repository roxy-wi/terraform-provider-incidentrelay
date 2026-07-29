package incidentrelay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestSSOProviderResourceCRUDPreservesSecrets(t *testing.T) {
	state := map[string]interface{}{
		"id":                               301,
		"slug":                             "corporate-oidc",
		"label":                            "Corporate OIDC",
		"protocol":                         "oidc",
		"enabled":                          true,
		"subject_claim":                    "sub",
		"email_claim":                      "email",
		"username_claim":                   "preferred_username",
		"display_name_claim":               "name",
		"groups_claim":                     "groups",
		"phone_claim":                      "mobile",
		"allowed_domains":                  []interface{}{"example.com"},
		"auto_create_users":                true,
		"auto_link_by_email":               true,
		"require_verified_email":           true,
		"sync_group_memberships":           true,
		"remove_missing_group_memberships": false,
		"client_id":                        "incidentrelay",
		"has_client_secret":                true,
		"oidc_metadata_url":                "https://idp.example.com/.well-known/openid-configuration",
		"oidc_scope":                       "openid email profile",
		"has_saml_sp_private_key":          false,
		"extra_config":                     map[string]interface{}{},
	}

	var requests []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/admin/sso/providers":
			payload := decodeJSONBody(t, r)
			if got, want := payload["client_secret"], "oidc-client-secret"; got != want {
				t.Fatalf("create client_secret = %v, want %v", got, want)
			}
			if got, want := payload["extra_config"], map[string]interface{}{}; !reflect.DeepEqual(got, want) {
				t.Fatalf("create extra_config = %#v, want %#v", got, want)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/sso/providers":
			writeJSON(t, w, []map[string]interface{}{state})

		case r.Method == http.MethodPut && r.URL.Path == "/api/admin/sso/providers/301":
			payload := decodeJSONBody(t, r)
			if got, want := payload["client_secret"], "oidc-client-secret"; got != want {
				t.Fatalf("update client_secret = %v, want %v", got, want)
			}
			state["label"] = payload["label"]
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/admin/sso/providers/301":
			w.WriteHeader(http.StatusNoContent)

		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	resource := resourceSSOProvider()
	if !resource.Schema["client_secret"].Sensitive {
		t.Fatal("client_secret is not marked sensitive")
	}
	if !resource.Schema["saml_sp_private_key"].Sensitive {
		t.Fatal("saml_sp_private_key is not marked sensitive")
	}
	if got := resource.Schema["extra_config_json"].StateFunc(`{"saml_security":{"authnRequestsSigned":true}}`); got != `{"saml_security":{"authnRequestsSigned":true,"digestAlgorithm":"http://www.w3.org/2001/04/xmlenc#sha256","logoutRequestSigned":false,"logoutResponseSigned":false,"nameIdEncrypted":false,"requestedAuthnContext":false,"signMetadata":false,"signatureAlgorithm":"http://www.w3.org/2001/04/xmldsig-more#rsa-sha256","wantAssertionsEncrypted":false,"wantAssertionsSigned":false,"wantAttributeStatement":false,"wantMessagesSigned":false,"wantNameId":true,"wantNameIdEncrypted":false}}` {
		t.Fatalf("normalized extra_config_json = %v", got)
	}

	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"slug":              "corporate-oidc",
		"label":             "Corporate OIDC",
		"protocol":          "oidc",
		"enabled":           true,
		"allowed_domains":   []interface{}{"example.com"},
		"auto_create_users": true,
		"client_id":         "incidentrelay",
		"client_secret":     "oidc-client-secret",
		"oidc_metadata_url": "https://idp.example.com/.well-known/openid-configuration",
		"extra_config_json": `{}`,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Id(), "301"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("client_secret"), "oidc-client-secret"; got != want {
		t.Fatalf("client_secret after refresh = %v, want %v", got, want)
	}
	if got, want := data.Get("has_client_secret"), true; got != want {
		t.Fatalf("has_client_secret = %v, want %v", got, want)
	}

	if err := data.Set("label", "Corporate Login"); err != nil {
		t.Fatalf("set label: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("label"), "Corporate Login"; got != want {
		t.Fatalf("label after update = %v, want %v", got, want)
	}
	if got, want := data.Get("client_secret"), "oidc-client-secret"; got != want {
		t.Fatalf("client_secret after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/admin/sso/providers",
		"GET /api/admin/sso/providers",
		"PUT /api/admin/sso/providers/301",
		"GET /api/admin/sso/providers",
		"DELETE /api/admin/sso/providers/301",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestSSOGroupMappingResourceCRUD(t *testing.T) {
	state := map[string]interface{}{
		"id":             401,
		"provider_id":    301,
		"external_group": "incidentrelay-platform",
		"group_id":       11,
		"group_slug":     "infrastructure",
		"group_name":     "Infrastructure",
		"group_role":     "editor",
		"team_id":        22,
		"team_slug":      "platform",
		"team_name":      "Platform",
		"team_role":      "viewer",
		"active":         true,
		"priority":       100,
	}

	var requests []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/admin/sso/providers/301/mappings":
			payload := decodeJSONBody(t, r)
			if _, ok := payload["provider_id"]; ok {
				t.Fatalf("create payload unexpectedly contains provider_id: %#v", payload)
			}
			if _, ok := payload["team_role"]; ok {
				t.Fatalf("create payload unexpectedly contains defaulted team_role: %#v", payload)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/sso/providers/301/mappings":
			writeJSON(t, w, []map[string]interface{}{state})

		case r.Method == http.MethodPut && r.URL.Path == "/api/admin/sso/mappings/401":
			payload := decodeJSONBody(t, r)
			state["external_group"] = payload["external_group"]
			state["team_role"] = payload["team_role"]
			state["priority"] = payload["priority"]
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/admin/sso/mappings/401":
			w.WriteHeader(http.StatusNoContent)

		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	resource := resourceSSOGroupMapping()
	if !resource.Schema["provider_id"].ForceNew {
		t.Fatal("provider_id is not ForceNew")
	}
	if !resource.Schema["team_role"].Optional || !resource.Schema["team_role"].Computed {
		t.Fatal("team_role must be optional and computed")
	}

	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"provider_id":    301,
		"external_group": "incidentrelay-platform",
		"group_id":       11,
		"group_role":     "editor",
		"team_id":        22,
		"active":         true,
		"priority":       100,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Get("team_role"), "viewer"; got != want {
		t.Fatalf("team_role after create = %v, want %v", got, want)
	}
	if got, want := data.Get("group_slug"), "infrastructure"; got != want {
		t.Fatalf("group_slug = %v, want %v", got, want)
	}

	if err := data.Set("external_group", "incidentrelay-sre"); err != nil {
		t.Fatalf("set external_group: %v", err)
	}
	if err := data.Set("team_role", "manager"); err != nil {
		t.Fatalf("set team_role: %v", err)
	}
	if err := data.Set("priority", 50); err != nil {
		t.Fatalf("set priority: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("external_group"), "incidentrelay-sre"; got != want {
		t.Fatalf("external_group after update = %v, want %v", got, want)
	}
	if got, want := data.Get("team_role"), "manager"; got != want {
		t.Fatalf("team_role after update = %v, want %v", got, want)
	}
	if got, want := data.Get("priority"), 50; got != want {
		t.Fatalf("priority after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/admin/sso/providers/301/mappings",
		"GET /api/admin/sso/providers/301/mappings",
		"PUT /api/admin/sso/mappings/401",
		"GET /api/admin/sso/providers/301/mappings",
		"DELETE /api/admin/sso/mappings/401",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func decodeJSONBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return payload
}

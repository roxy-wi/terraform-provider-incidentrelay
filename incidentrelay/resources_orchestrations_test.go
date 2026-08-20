package incidentrelay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestEventOrchestrationResourcePublishesRulesAndManagesRuntime(t *testing.T) {
	state := map[string]interface{}{
		"id":                 901,
		"uid":                "4f1b5931-52a0-4a45-b120-f4efc2f8c3a1",
		"group_id":           10,
		"name":               "Production routing",
		"description":        "Managed by Terraform",
		"scope":              "global",
		"service_id":         nil,
		"enabled":            false,
		"mode":               "disabled",
		"compatibility_mode": "hybrid",
		"active_version_id":  1001,
		"created_at":         "2026-08-18T09:00:00Z",
		"updated_at":         "2026-08-18T09:00:00Z",
	}
	rules := []interface{}{
		map[string]interface{}{
			"name":            "Critical production",
			"enabled":         true,
			"condition_tree":  map[string]interface{}{"field": "event.severity", "operator": "equals", "value": "critical"},
			"actions":         []interface{}{map[string]interface{}{"type": "set_priority", "value": "P1"}},
			"processing_mode": "continue",
			"children":        []interface{}{},
		},
	}
	versionNumber := 1
	var requests []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/event-orchestrations":
			payload := decodeJSONBody(t, r)
			if got, want := payload["compatibility_mode"], "hybrid"; got != want {
				t.Fatalf("create compatibility_mode = %v, want %v", got, want)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodPost && r.URL.Path == "/api/event-orchestrations/901/draft":
			writeJSON(t, w, map[string]interface{}{"id": 1002, "status": "draft"})

		case r.Method == http.MethodPut && r.URL.Path == "/api/event-orchestrations/901/draft":
			payload := decodeJSONBody(t, r)
			if got, want := payload["comment"], "Terraform publication"; got != want {
				t.Fatalf("draft comment = %v, want %v", got, want)
			}
			parsedRules, ok := payload["rules"].([]interface{})
			if !ok {
				t.Fatalf("draft rules = %T, want array", payload["rules"])
			}
			rules = parsedRules
			writeJSON(t, w, map[string]interface{}{"id": 1002, "status": "draft"})

		case r.Method == http.MethodPost && r.URL.Path == "/api/event-orchestrations/901/publish":
			payload := decodeJSONBody(t, r)
			if got, want := payload["confirm_catch_all_drop"], false; got != want {
				t.Fatalf("publish confirmation = %v, want %v", got, want)
			}
			versionNumber++
			state["active_version_id"] = 1000 + versionNumber
			writeJSON(t, w, map[string]interface{}{"id": state["active_version_id"], "version_number": versionNumber, "status": "published"})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/event-orchestrations/901/runtime":
			payload := decodeJSONBody(t, r)
			state["mode"] = payload["mode"]
			state["compatibility_mode"] = payload["compatibility_mode"]
			state["enabled"] = payload["mode"] != "disabled"
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/event-orchestrations/901":
			response := cloneMap(state)
			response["active_version"] = map[string]interface{}{
				"id":             state["active_version_id"],
				"version_number": versionNumber,
				"status":         "published",
			}
			response["active_definition"] = map[string]interface{}{
				"schema_version": 1,
				"scope":          state["scope"],
				"service_id":     state["service_id"],
				"rules":          rules,
			}
			writeJSON(t, w, response)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/event-orchestrations/901":
			payload := decodeJSONBody(t, r)
			for key, value := range payload {
				state[key] = value
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/event-orchestrations/901":
			writeJSON(t, w, map[string]interface{}{"deleted": true, "id": 901})

		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	resource := resourceEventOrchestration()
	if !resource.Schema["group_id"].ForceNew {
		t.Fatal("group_id is not ForceNew")
	}
	if _, errors := resource.Schema["rules_json"].ValidateFunc(`{"not":"an array"}`, "rules_json"); len(errors) == 0 {
		t.Fatal("rules_json accepted a JSON object")
	}

	rulesJSON := `[{"name":"Critical production","enabled":true,"condition_tree":{"field":"event.severity","operator":"equals","value":"critical"},"actions":[{"type":"set_priority","value":"P1"}],"processing_mode":"continue","children":[]}]`
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"group_id":               10,
		"name":                   "Production routing",
		"description":            "Managed by Terraform",
		"scope":                  "global",
		"compatibility_mode":     "hybrid",
		"rules_json":             rulesJSON,
		"publish_comment":        "Terraform publication",
		"confirm_catch_all_drop": false,
		"mode":                   "disabled",
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Id(), "901"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("active_version_number"), 2; got != want {
		t.Fatalf("active_version_number = %v, want %v", got, want)
	}
	if got, want := data.Get("rules_json"), normalizeEventOrchestrationRulesState(rulesJSON); got != want {
		t.Fatalf("rules_json after create = %v, want %v", got, want)
	}

	updatedRulesJSON := `[{"name":"Send production to platform","condition_tree":{},"actions":[{"type":"set_team","team_id":42}],"processing_mode":"stop","children":[]}]`
	for field, value := range map[string]interface{}{
		"name":               "Production event routing",
		"rules_json":         updatedRulesJSON,
		"mode":               "active",
		"compatibility_mode": "orchestration",
	} {
		if err := data.Set(field, value); err != nil {
			t.Fatalf("set %s: %v", field, err)
		}
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("mode"), "active"; got != want {
		t.Fatalf("mode after update = %v, want %v", got, want)
	}
	if got, want := data.Get("enabled"), true; got != want {
		t.Fatalf("enabled after update = %v, want %v", got, want)
	}
	if got, want := data.Get("rules_json"), normalizeEventOrchestrationRulesState(updatedRulesJSON); got != want {
		t.Fatalf("rules_json after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/event-orchestrations",
		"POST /api/event-orchestrations/901/draft",
		"PUT /api/event-orchestrations/901/draft",
		"POST /api/event-orchestrations/901/publish",
		"PATCH /api/event-orchestrations/901/runtime",
		"GET /api/event-orchestrations/901",
		"PATCH /api/event-orchestrations/901",
		"POST /api/event-orchestrations/901/draft",
		"PUT /api/event-orchestrations/901/draft",
		"POST /api/event-orchestrations/901/publish",
		"PATCH /api/event-orchestrations/901/runtime",
		"GET /api/event-orchestrations/901",
		"DELETE /api/event-orchestrations/901",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestOrchestrationWebhookActionResourceUsesPatchAndPreservesSecrets(t *testing.T) {
	state := map[string]interface{}{
		"id":                     801,
		"uid":                    "22e17cd5-6d7f-4e8b-8be1-34736ff1466e",
		"group_id":               10,
		"name":                   "Automation receiver",
		"description":            "Notify the automation service",
		"url":                    "https://automation.example.com/hook?token=***REDACTED***",
		"method":                 "POST",
		"has_headers":            true,
		"body_template":          `{"title":"{{ event.title }}"}`,
		"timeout_seconds":        15,
		"retry_count":            3,
		"private_network_policy": "deny",
		"enabled":                true,
		"created_at":             "2026-08-18T09:00:00Z",
		"updated_at":             "2026-08-18T09:00:00Z",
	}
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/orchestration-webhook-actions":
			payload := decodeJSONBody(t, r)
			if got, want := payload["headers"], map[string]interface{}{"Authorization": "Bearer secret"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("create headers = %#v, want %#v", got, want)
			}
			writeJSON(t, w, state)
		case r.Method == http.MethodGet && r.URL.Path == "/api/orchestration-webhook-actions":
			if got, want := r.URL.Query().Get("group_id"), "10"; got != want {
				t.Fatalf("group_id = %q, want %q", got, want)
			}
			writeJSON(t, w, map[string]interface{}{"items": []map[string]interface{}{state}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/orchestration-webhook-actions/801":
			payload := decodeJSONBody(t, r)
			state["name"] = payload["name"]
			writeJSON(t, w, state)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/orchestration-webhook-actions/801":
			writeJSON(t, w, map[string]interface{}{"deleted": true, "id": 801})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	resource := resourceOrchestrationWebhookAction()
	if !resource.Schema["headers_json"].Sensitive {
		t.Fatal("headers_json is not sensitive")
	}
	if !resource.Schema["url"].Sensitive {
		t.Fatal("url is not sensitive")
	}
	url := "https://automation.example.com/hook?token=secret"
	headersJSON := `{"Authorization":"Bearer secret"}`
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"group_id":               10,
		"name":                   "Automation receiver",
		"description":            "Notify the automation service",
		"url":                    url,
		"method":                 "POST",
		"headers_json":           headersJSON,
		"body_template":          `{"title":"{{ event.title }}"}`,
		"timeout_seconds":        15,
		"retry_count":            3,
		"private_network_policy": "deny",
		"enabled":                true,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Get("url"), url; got != want {
		t.Fatalf("url after refresh = %v, want %v", got, want)
	}
	if got, want := data.Get("headers_json"), normalizeJSONStringState(headersJSON); got != want {
		t.Fatalf("headers_json after refresh = %v, want %v", got, want)
	}

	if err := data.Set("name", "Automation receiver v2"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("url"), url; got != want {
		t.Fatalf("url after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/orchestration-webhook-actions",
		"GET /api/orchestration-webhook-actions?group_id=10",
		"PATCH /api/orchestration-webhook-actions/801",
		"GET /api/orchestration-webhook-actions?group_id=10",
		"DELETE /api/orchestration-webhook-actions/801",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestSilenceResourceUsesLifecycleEndpointsAndOmitsEnabledPayload(t *testing.T) {
	state := map[string]interface{}{
		"id":                701,
		"team_id":           42,
		"name":              "Deploy silence",
		"reason":            "Production deployment",
		"matcher_preset_id": 51,
		"matchers":          map[string]interface{}{},
		"starts_at":         "2026-08-18T20:00:00Z",
		"ends_at":           "2026-08-18T22:00:00Z",
		"apply_to_existing": true,
		"reactivate_on_end": false,
		"enabled":           true,
	}
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/silences":
			payload := decodeJSONBody(t, r)
			if _, exists := payload["enabled"]; exists {
				t.Fatalf("create payload unexpectedly contains enabled: %#v", payload)
			}
			if got, want := payload["matcher_preset_id"], float64(51); got != want {
				t.Fatalf("matcher_preset_id = %v, want %v", got, want)
			}
			writeJSON(t, w, state)
		case r.Method == http.MethodPut && r.URL.Path == "/api/silences/701":
			payload := decodeJSONBody(t, r)
			if _, exists := payload["enabled"]; exists {
				t.Fatalf("update payload unexpectedly contains enabled: %#v", payload)
			}
			state["reason"] = payload["reason"]
			writeJSON(t, w, state)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/silences/701":
			state["enabled"] = false
			writeJSON(t, w, state)
		case r.Method == http.MethodPost && r.URL.Path == "/api/silences/701/enable":
			state["enabled"] = true
			writeJSON(t, w, state)
		case r.Method == http.MethodGet && r.URL.Path == "/api/silences/701":
			writeJSON(t, w, state)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	resource := resourceSilence()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"team_id":           42,
		"name":              "Deploy silence",
		"reason":            "Production deployment",
		"matcher_preset_id": 51,
		"matchers_json":     `{}`,
		"starts_at":         "2026-08-18T20:00:00Z",
		"ends_at":           "2026-08-18T22:00:00Z",
		"apply_to_existing": true,
		"reactivate_on_end": false,
		"enabled":           true,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if err := data.Set("enabled", false); err != nil {
		t.Fatalf("disable silence: %v", err)
	}
	if err := data.Set("reason", "Deployment delayed"); err != nil {
		t.Fatalf("set reason: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("disable diagnostics: %v", diags)
	}
	if got, want := data.Get("enabled"), false; got != want {
		t.Fatalf("enabled after disable = %v, want %v", got, want)
	}

	if err := data.Set("enabled", true); err != nil {
		t.Fatalf("enable silence: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("enable diagnostics: %v", diags)
	}
	if got, want := data.Get("enabled"), true; got != want {
		t.Fatalf("enabled after enable = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/silences",
		"GET /api/silences/701",
		"PUT /api/silences/701",
		"DELETE /api/silences/701",
		"GET /api/silences/701",
		"PUT /api/silences/701",
		"POST /api/silences/701/enable",
		"GET /api/silences/701",
		"DELETE /api/silences/701",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func cloneMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestGroupResourceCRUD(t *testing.T) {
	state := map[string]interface{}{
		"id":          101,
		"slug":        "platform",
		"name":        "Platform",
		"description": "Primary team",
		"active":      true,
	}
	var requests []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/groups":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			want := map[string]interface{}{
				"slug":        "platform",
				"name":        "Platform",
				"description": "Primary team",
				"active":      true,
			}
			if !reflect.DeepEqual(payload, want) {
				t.Fatalf("create payload = %#v, want %#v", payload, want)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/groups":
			writeJSON(t, w, map[string]interface{}{"items": []map[string]interface{}{state}})

		case r.Method == http.MethodPut && r.URL.Path == "/api/groups/101":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode update payload: %v", err)
			}
			if got, want := payload["name"], "Platform Ops"; got != want {
				t.Fatalf("update payload[name] = %v, want %v", got, want)
			}
			for key, value := range payload {
				state[key] = value
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/groups/101":
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
	config := &Config{Client: client}
	resource := resourceGroup()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"slug":        "platform",
		"name":        "Platform",
		"description": "Primary team",
		"active":      true,
	})

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Id(), "101"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("name"), "Platform"; got != want {
		t.Fatalf("name after create = %v, want %v", got, want)
	}

	if err := data.Set("name", "Platform Ops"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("name"), "Platform Ops"; got != want {
		t.Fatalf("name after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}
	if got := data.Id(); got != "" {
		t.Fatalf("id after delete = %q, want empty", got)
	}

	wantRequests := []string{
		"POST /api/groups",
		"GET /api/groups",
		"PUT /api/groups/101",
		"GET /api/groups",
		"DELETE /api/groups/101",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestChannelResourcePreservesIncidentRelay12MaskedSecrets(t *testing.T) {
	configJSON := `{
		"mode": "bot_api",
		"connection_mode": "socket_mode",
		"bot_token": "xoxb-terraform-test",
		"app_token": "xapp-terraform-test",
		"channel_id": "C0123456789"
	}`
	state := map[string]interface{}{
		"id":           202,
		"team_id":      42,
		"name":         "Slack Socket Mode",
		"channel_type": "slack",
		"config": map[string]interface{}{
			"mode":            "bot_api",
			"connection_mode": "socket_mode",
			"bot_token":       "xoxb-terraform-test",
			"app_token":       "xapp-terraform-test",
			"channel_id":      "C0123456789",
		},
		"enabled": true,
	}

	maskedResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"id":           state["id"],
			"team_id":      state["team_id"],
			"name":         state["name"],
			"channel_type": state["channel_type"],
			"config": map[string]interface{}{
				"mode":            "bot_api",
				"connection_mode": "socket_mode",
				"bot_token":       incidentRelaySecretPlaceholder,
				"app_token":       incidentRelaySecretPlaceholder,
				"channel_id":      "C0123456789",
			},
			"enabled": state["enabled"],
		}
	}

	assertPlaintextSecrets := func(payload map[string]interface{}) {
		t.Helper()
		config, ok := payload["config"].(map[string]interface{})
		if !ok {
			t.Fatalf("payload config = %T, want object", payload["config"])
		}
		if got, want := config["bot_token"], "xoxb-terraform-test"; got != want {
			t.Fatalf("payload bot_token = %v, want %v", got, want)
		}
		if got, want := config["app_token"], "xapp-terraform-test"; got != want {
			t.Fatalf("payload app_token = %v, want %v", got, want)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/channels":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			assertPlaintextSecrets(payload)
			writeJSON(t, w, maskedResponse())

		case r.Method == http.MethodGet && r.URL.Path == "/api/channels/202":
			writeJSON(t, w, maskedResponse())

		case r.Method == http.MethodPut && r.URL.Path == "/api/channels/202":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode update payload: %v", err)
			}
			assertPlaintextSecrets(payload)
			state["name"] = payload["name"]
			writeJSON(t, w, maskedResponse())

		case r.Method == http.MethodDelete && r.URL.Path == "/api/channels/202":
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
	config := &Config{Client: client}
	resource := resourceChannel()
	if !resource.Schema["config_json"].Sensitive {
		t.Fatal("channel config_json is not marked sensitive")
	}
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"team_id":      42,
		"name":         "Slack Socket Mode",
		"channel_type": "slack",
		"config_json":  configJSON,
		"enabled":      true,
	})

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Get("config_json"), normalizeJSONStringState(configJSON); got != want {
		t.Fatalf("config_json after create = %v, want %v", got, want)
	}

	if err := data.Set("name", "Slack Socket Mode Updated"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("config_json"), normalizeJSONStringState(configJSON); got != want {
		t.Fatalf("config_json after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}
}

func TestChannelResourcePreservesIncidentRelay23LarkMaskedSecrets(t *testing.T) {
	configJSON := `{
		"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		"signing_secret": "lark-signing-secret"
	}`
	maskedResponse := func(name string) map[string]interface{} {
		return map[string]interface{}{
			"id":           203,
			"team_id":      42,
			"name":         name,
			"channel_type": "lark",
			"config": map[string]interface{}{
				"webhook_url":    incidentRelaySecretPlaceholder,
				"signing_secret": incidentRelaySecretPlaceholder,
			},
			"enabled": true,
		}
	}

	name := "Lark"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/channels":
			payload := decodeJSONBody(t, r)
			config := payload["config"].(map[string]interface{})
			if got, want := config["signing_secret"], "lark-signing-secret"; got != want {
				t.Fatalf("create signing_secret = %v, want %v", got, want)
			}
			writeJSON(t, w, maskedResponse(name))
		case r.Method == http.MethodGet && r.URL.Path == "/api/channels/203":
			writeJSON(t, w, maskedResponse(name))
		case r.Method == http.MethodPut && r.URL.Path == "/api/channels/203":
			payload := decodeJSONBody(t, r)
			config := payload["config"].(map[string]interface{})
			if got, want := config["webhook_url"], "https://open.feishu.cn/open-apis/bot/v2/hook/test"; got != want {
				t.Fatalf("update webhook_url = %v, want %v", got, want)
			}
			name = payload["name"].(string)
			writeJSON(t, w, maskedResponse(name))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/channels/203":
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
	resource := resourceChannel()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"team_id": 42, "name": name, "channel_type": "lark", "config_json": configJSON, "enabled": true,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Get("config_json"), normalizeJSONStringState(configJSON); got != want {
		t.Fatalf("config_json after create = %v, want %v", got, want)
	}
	if err := data.Set("name", "Lark Operations"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("config_json"), normalizeJSONStringState(configJSON); got != want {
		t.Fatalf("config_json after update = %v, want %v", got, want)
	}
}

func TestRouteResourceIncidentRelay23IntegrationConfigRoundTrip(t *testing.T) {
	resource := resourceRoute()
	if !resource.Schema["integration_config_json"].Sensitive {
		t.Fatal("integration_config_json must be sensitive")
	}
	if !resource.Schema["matcher_preset_id"].Optional {
		t.Fatal("matcher_preset_id must be optional")
	}

	allowedSources := []string{"new_relic", "nagios", "azure_monitor", "cloud_ru"}
	for _, source := range allowedSources {
		if _, errors := resource.Schema["source"].ValidateFunc(source, "source"); len(errors) != 0 {
			t.Fatalf("source %q returned validation errors: %v", source, errors)
		}
	}

	cloudData := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"team_id":                 42,
		"name":                    "Cloud.ru",
		"source":                  "cloud_ru",
		"integration_config_json": `{"cloud_ru":{"topic_urn":"urn:smn:ru-a:project:incidentrelay"}}`,
	})
	cloudResponse := map[string]interface{}{
		"source": "cloud_ru",
		"integration_config": map[string]interface{}{
			"cloud_ru": map[string]interface{}{
				"topic_urn":    "urn:smn:ru-a:project:incidentrelay",
				"webhook_path": "/api/integrations/cloud-ru/42",
			},
		},
	}
	if err := routeResponseHook(cloudData, cloudResponse); err != nil {
		t.Fatalf("cloud_ru response hook: %v", err)
	}
	wantCloud := map[string]interface{}{
		"cloud_ru": map[string]interface{}{"topic_urn": "urn:smn:ru-a:project:incidentrelay"},
	}
	if got := cloudResponse["integration_config"]; !reflect.DeepEqual(got, wantCloud) {
		t.Fatalf("cloud_ru managed config = %#v, want %#v", got, wantCloud)
	}

	sentryData := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"team_id":                 42,
		"name":                    "Sentry",
		"source":                  "sentry",
		"integration_config_json": `{"sentry":{"webhook_secret":"secret","base_url":"https://sentry.io"}}`,
	})
	sentryResponse := map[string]interface{}{
		"source": "sentry",
		"integration_config": map[string]interface{}{
			"sentry": map[string]interface{}{
				"has_webhook_secret": true,
				"webhook_path":       "/api/integrations/sentry/42",
				"base_url":           "https://sentry.io",
				"organization_slug":  nil,
			},
		},
	}
	if err := routeResponseHook(sentryData, sentryResponse); err != nil {
		t.Fatalf("sentry response hook: %v", err)
	}
	wantSentry := map[string]interface{}{
		"sentry": map[string]interface{}{
			"webhook_secret": "secret",
			"base_url":       "https://sentry.io",
		},
	}
	if got := sentryResponse["integration_config"]; !reflect.DeepEqual(got, wantSentry) {
		t.Fatalf("sentry managed config = %#v, want %#v", got, wantSentry)
	}
}

func TestIncidentRelay23MatcherPresetsForNotificationRulesAndRunbooks(t *testing.T) {
	notificationState := map[string]interface{}{
		"id": 601, "name": "Critical", "description": nil, "position": 1,
		"event_types": []interface{}{"notification"}, "matchers": map[string]interface{}{},
		"matcher_preset_id": 91, "channel_ids": []interface{}{12},
		"continue_matching": false, "enabled": true,
	}
	runbookState := map[string]interface{}{
		"id": 701, "service_id": 77, "service_name": "API", "service_slug": "api",
		"title": "API incidents", "description": nil, "url": "https://runbooks.example.com/api",
		"severity": nil, "matchers": map[string]interface{}{}, "matcher_preset_id": 93,
		"priority": 100, "enabled": true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/notification-policies/55/rules":
			payload := decodeJSONBody(t, r)
			if got, want := payload["matcher_preset_id"], float64(91); got != want {
				t.Fatalf("notification rule create matcher_preset_id = %v, want %v", got, want)
			}
			writeJSON(t, w, notificationState)
		case r.Method == http.MethodPut && r.URL.Path == "/api/notification-policies/55/rules/601":
			payload := decodeJSONBody(t, r)
			if got, want := payload["matcher_preset_id"], float64(92); got != want {
				t.Fatalf("notification rule update matcher_preset_id = %v, want %v", got, want)
			}
			notificationState["matcher_preset_id"] = 92
			writeJSON(t, w, notificationState)
		case r.Method == http.MethodGet && r.URL.Path == "/api/notification-policies/55":
			writeJSON(t, w, map[string]interface{}{"rules": []interface{}{notificationState}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/services/77/runbooks":
			payload := decodeJSONBody(t, r)
			if got, want := payload["matcher_preset_id"], float64(93); got != want {
				t.Fatalf("runbook create matcher_preset_id = %v, want %v", got, want)
			}
			writeJSON(t, w, runbookState)
		case r.Method == http.MethodPut && r.URL.Path == "/api/services/runbooks/701":
			payload := decodeJSONBody(t, r)
			if got, want := payload["matcher_preset_id"], float64(94); got != want {
				t.Fatalf("runbook update matcher_preset_id = %v, want %v", got, want)
			}
			runbookState["matcher_preset_id"] = 94
			writeJSON(t, w, runbookState)
		case r.Method == http.MethodGet && r.URL.Path == "/api/services/runbooks":
			writeJSON(t, w, []interface{}{runbookState})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	config := &Config{Client: client}

	notificationResource := resourceNotificationPolicyRule()
	if _, errors := notificationResource.Schema["matcher_preset_id"].ValidateFunc(0, "matcher_preset_id"); len(errors) == 0 {
		t.Fatal("notification rule accepted matcher_preset_id=0")
	}
	notificationData := schema.TestResourceDataRaw(t, notificationResource.Schema, map[string]interface{}{
		"policy_id": 55, "name": "Critical", "event_types": []interface{}{"notification"},
		"matcher_preset_id": 91, "channel_ids": []interface{}{12},
	})
	if diags := notificationResource.CreateWithoutTimeout(context.Background(), notificationData, config); diags.HasError() {
		t.Fatalf("notification rule create diagnostics: %v", diags)
	}
	if err := notificationData.Set("matcher_preset_id", 92); err != nil {
		t.Fatalf("set notification matcher_preset_id: %v", err)
	}
	if diags := notificationResource.UpdateWithoutTimeout(context.Background(), notificationData, config); diags.HasError() {
		t.Fatalf("notification rule update diagnostics: %v", diags)
	}

	runbookResource := resourceServiceRunbook()
	if _, errors := runbookResource.Schema["matcher_preset_id"].ValidateFunc(0, "matcher_preset_id"); len(errors) == 0 {
		t.Fatal("service runbook accepted matcher_preset_id=0")
	}
	runbookData := schema.TestResourceDataRaw(t, runbookResource.Schema, map[string]interface{}{
		"service_id": 77, "title": "API incidents", "url": "https://runbooks.example.com/api",
		"matcher_preset_id": 93,
	})
	if diags := runbookResource.CreateWithoutTimeout(context.Background(), runbookData, config); diags.HasError() {
		t.Fatalf("runbook create diagnostics: %v", diags)
	}
	if err := runbookData.Set("matcher_preset_id", 94); err != nil {
		t.Fatalf("set runbook matcher_preset_id: %v", err)
	}
	if diags := runbookResource.UpdateWithoutTimeout(context.Background(), runbookData, config); diags.HasError() {
		t.Fatalf("runbook update diagnostics: %v", diags)
	}
}

func TestReadListDatasourceFindsSingleMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/api/groups"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		writeJSON(t, w, map[string]interface{}{
			"items": []map[string]interface{}{
				{"id": 1, "slug": "default", "name": "Default", "description": "", "active": true},
				{"id": 2, "slug": "platform", "name": "Platform", "description": "Primary", "active": true},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	dataSource := datasourceGroup()
	data := schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{
		"slug": "platform",
	})

	if diags := dataSource.ReadContext(context.Background(), data, &Config{Client: client}); diags.HasError() {
		t.Fatalf("read diagnostics: %v", diags)
	}
	if got, want := data.Id(), "2"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("name"), "Platform"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if got, want := data.Get("description"), "Primary"; got != want {
		t.Fatalf("description = %v, want %v", got, want)
	}
}

func TestReadListDatasourceRequiresCriteriaAndRejectsAmbiguousMatches(t *testing.T) {
	dataSource := datasourceGroup()
	data := schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{})

	diags := dataSource.ReadContext(context.Background(), data, &Config{Client: &Client{}})
	if !diags.HasError() {
		t.Fatal("read without criteria returned no diagnostics")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"id": 1, "slug": "platform", "name": "Platform", "active": true},
			{"id": 2, "slug": "platform", "name": "Platform Copy", "active": true},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	data = schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{
		"slug": "platform",
	})

	diags = dataSource.ReadContext(context.Background(), data, &Config{Client: client})
	if !diags.HasError() {
		t.Fatal("ambiguous read returned no diagnostics")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestFieldIDPath(t *testing.T) {
	fields := []fieldDef{
		reqInt("policy_id", "Policy id."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"policy_id": 55,
	})

	got := fieldIDPath("/api/notification-policies/%d/rules/%s", "policy_id")("99", data)
	if want := "/api/notification-policies/55/rules/99"; got != want {
		t.Fatalf("fieldIDPath = %q, want %q", got, want)
	}
}

func TestReadItemFromListNestedField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"rules": []map[string]interface{}{
				{"id": 1, "name": "first"},
				{"id": 2, "name": "second"},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	item, err := readItemFromList(context.Background(), client, "/policy", "rules", "2")
	if err != nil {
		t.Fatalf("readItemFromList returned error: %v", err)
	}
	if got, want := item["name"], "second"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}

	if _, err := readItemFromList(context.Background(), client, "/policy", "rules", "3"); !isNotFound(err) {
		t.Fatalf("missing item error = %v, want not found", err)
	}
}

func Example_stringID() {
	fmt.Println(stringID(42))
	// Output: 42
}

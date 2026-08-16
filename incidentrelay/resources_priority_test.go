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

func TestPriorityPolicyResourceCRUD(t *testing.T) {
	state := map[string]interface{}{
		"id":                   501,
		"team_id":              42,
		"team_name":            "Platform",
		"team_slug":            "platform",
		"name":                 "Production priority",
		"description":          "Production incident priority policy.",
		"enabled":              true,
		"default_for_team":     true,
		"update_mode":          "raise_only",
		"source_priority_mode": "ignore",
		"fallback_mode":        "fixed_priority",
		"fallback_priority_id": 2,
		"rules_count":          0,
		"services_count":       0,
	}

	var requests []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/priority-policies":
			payload := decodeJSONBody(t, r)
			if got, want := payload["team_id"], float64(42); got != want {
				t.Fatalf("create team_id = %v, want %v", got, want)
			}
			if got, want := payload["fallback_mode"], "fixed_priority"; got != want {
				t.Fatalf("create fallback_mode = %v, want %v", got, want)
			}
			if got, want := payload["fallback_priority_id"], float64(2); got != want {
				t.Fatalf("create fallback_priority_id = %v, want %v", got, want)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/priority-policies/501":
			writeJSON(t, w, state)

		case r.Method == http.MethodPut && r.URL.Path == "/api/priority-policies/501":
			payload := decodeJSONBody(t, r)
			if _, exists := payload["team_id"]; exists {
				t.Fatalf("update payload unexpectedly contains team_id: %#v", payload)
			}
			if got := payload["fallback_priority_id"]; got != nil {
				t.Fatalf("update fallback_priority_id = %#v, want nil", got)
			}
			for key, value := range payload {
				state[key] = value
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/priority-policies/501":
			writeJSON(t, w, map[string]interface{}{"deleted": true, "id": 501})

		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	resource := resourcePriorityPolicy()
	if !resource.Schema["team_id"].ForceNew {
		t.Fatal("team_id is not ForceNew")
	}
	if _, errors := resource.Schema["update_mode"].ValidateFunc("raise", "update_mode"); len(errors) == 0 {
		t.Fatal("invalid update_mode returned no errors")
	}

	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"team_id":              42,
		"name":                 "Production priority",
		"description":          "Production incident priority policy.",
		"enabled":              true,
		"default_for_team":     true,
		"update_mode":          "raise_only",
		"source_priority_mode": "ignore",
		"fallback_mode":        "fixed_priority",
		"fallback_priority_id": 2,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Id(), "501"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("team_slug"), "platform"; got != want {
		t.Fatalf("team_slug = %v, want %v", got, want)
	}

	for field, value := range map[string]interface{}{
		"default_for_team":     false,
		"update_mode":          "recalculate",
		"source_priority_mode": "prefer",
		"fallback_mode":        "severity_mapping",
		"fallback_priority_id": nil,
	} {
		if err := data.Set(field, value); err != nil {
			t.Fatalf("set %s: %v", field, err)
		}
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("update_mode"), "recalculate"; got != want {
		t.Fatalf("update_mode after update = %v, want %v", got, want)
	}
	if got, want := data.Get("fallback_priority_id"), 0; got != want {
		t.Fatalf("fallback_priority_id after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/priority-policies",
		"GET /api/priority-policies/501",
		"PUT /api/priority-policies/501",
		"GET /api/priority-policies/501",
		"DELETE /api/priority-policies/501",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestPriorityPolicyRuleResourceCRUD(t *testing.T) {
	state := map[string]interface{}{
		"id":                601,
		"policy_id":         501,
		"name":              "Critical production",
		"description":       "Critical production alerts.",
		"position":          1,
		"matchers":          map[string]interface{}{"severity": "critical"},
		"matcher_preset_id": nil,
		"priority_id":       1,
		"enabled":           true,
	}

	var requests []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/priority-policies/501/rules":
			payload := decodeJSONBody(t, r)
			if _, exists := payload["policy_id"]; exists {
				t.Fatalf("create payload unexpectedly contains policy_id: %#v", payload)
			}
			if _, exists := payload["position"]; exists {
				t.Fatalf("create payload unexpectedly contains omitted position: %#v", payload)
			}
			if got, want := payload["matchers"], map[string]interface{}{"severity": "critical"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("create matchers = %#v, want %#v", got, want)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/priority-policies/501":
			writeJSON(t, w, map[string]interface{}{"id": 501, "rules": []map[string]interface{}{state}})

		case r.Method == http.MethodPut && r.URL.Path == "/api/priority-policies/501/rules/601":
			payload := decodeJSONBody(t, r)
			state["name"] = payload["name"]
			state["position"] = payload["position"]
			state["matchers"] = payload["matchers"]
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/priority-policies/501/rules/601":
			writeJSON(t, w, map[string]interface{}{"deleted": true, "id": 601})

		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	resource := resourcePriorityPolicyRule()
	if !resource.Schema["policy_id"].ForceNew {
		t.Fatal("policy_id is not ForceNew")
	}
	if !resource.Schema["position"].Optional || !resource.Schema["position"].Computed {
		t.Fatal("position must be optional and computed")
	}
	importData := schema.TestResourceDataRaw(t, resource.Schema, nil)
	importData.SetId("501/601")
	imported, err := resource.Importer.StateContext(context.Background(), importData, nil)
	if err != nil {
		t.Fatalf("import returned error: %v", err)
	}
	if got, want := imported[0].Id(), "601"; got != want {
		t.Fatalf("imported rule id = %q, want %q", got, want)
	}
	if got, want := imported[0].Get("policy_id"), 501; got != want {
		t.Fatalf("imported policy_id = %v, want %v", got, want)
	}

	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"policy_id":     501,
		"name":          "Critical production",
		"description":   "Critical production alerts.",
		"matchers_json": `{"severity":"critical"}`,
		"priority_id":   1,
		"enabled":       true,
	})
	config := &Config{Client: client}

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Get("position"), 1; got != want {
		t.Fatalf("position after create = %v, want %v", got, want)
	}

	if err := data.Set("name", "Critical services"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if err := data.Set("position", 2); err != nil {
		t.Fatalf("set position: %v", err)
	}
	if err := data.Set("matchers_json", `{"fields":{"service.environment":"production"}}`); err != nil {
		t.Fatalf("set matchers_json: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("name"), "Critical services"; got != want {
		t.Fatalf("name after update = %v, want %v", got, want)
	}
	if got, want := data.Get("position"), 2; got != want {
		t.Fatalf("position after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}

	wantRequests := []string{
		"POST /api/priority-policies/501/rules",
		"GET /api/priority-policies/501",
		"PUT /api/priority-policies/501/rules/601",
		"GET /api/priority-policies/501",
		"DELETE /api/priority-policies/501/rules/601",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestLookupDataSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/rotations":
			writeJSON(t, w, []map[string]interface{}{
				{"id": 700, "team_id": 40, "team_slug": "other", "team_name": "Other", "name": "Primary", "enabled": true, "timezone": "UTC"},
				{"id": 701, "team_id": 42, "team_slug": "platform", "team_name": "Platform", "name": "Primary", "description": "Platform primary", "start_at": "2026-07-13T09:00:00", "duration_seconds": nil, "reminder_interval_seconds": 300, "rotation_type": "weekly", "interval_value": 1, "interval_unit": "weeks", "handoff_time": "09:00", "handoff_weekday": 0, "timezone": "Europe/Moscow", "enabled": true, "current_oncall": "alice"},
			})
		case "/api/incidents/priorities":
			if got, want := r.URL.Query().Get("include_disabled"), "1"; got != want {
				t.Fatalf("include_disabled = %q, want %q", got, want)
			}
			writeJSON(t, w, []map[string]interface{}{
				{"id": 1, "slug": "p1", "name": "Critical", "description": "Critical priority", "level": 1, "color": "#dc3545", "enabled": true, "default": false},
				{"id": 3, "slug": "p3", "name": "Medium", "description": "Default priority", "level": 3, "color": "#ffc107", "enabled": true, "default": true},
			})
		case "/api/channels":
			writeJSON(t, w, []map[string]interface{}{
				{"id": 901, "group_id": 9, "group_slug": "other", "group_name": "Other", "team_id": 40, "team_slug": "other", "team_name": "Other", "name": "Email", "channel_type": "email", "enabled": true},
				{"id": 902, "group_id": 10, "group_slug": "infra", "group_name": "Infrastructure", "team_id": 42, "team_slug": "platform", "team_name": "Platform", "name": "Email", "channel_type": "email", "enabled": true},
			})
		case "/api/escalation-policies":
			writeJSON(t, w, []map[string]interface{}{
				{"id": 1001, "group_id": 9, "group_slug": "other", "team_id": 40, "team_slug": "other", "team_name": "Other", "name": "Critical escalation", "description": "Other escalation", "enabled": true, "repeat_count": 0, "rules": []interface{}{}},
				{"id": 1002, "group_id": 10, "group_slug": "infra", "team_id": 42, "team_slug": "platform", "team_name": "Platform", "name": "Critical escalation", "description": "Production escalation", "enabled": true, "repeat_count": 2, "rules": []interface{}{}},
			})
		case "/api/notification-policies":
			writeJSON(t, w, []map[string]interface{}{
				{"id": 1101, "team_id": 40, "team_slug": "other", "team_name": "Other", "name": "Production notifications", "description": "Other notifications", "enabled": true, "rules_count": 1, "services_count": 0},
				{"id": 1102, "team_id": 42, "team_slug": "platform", "team_name": "Platform", "name": "Production notifications", "description": "Platform notifications", "enabled": true, "rules_count": 3, "services_count": 2},
			})
		case "/api/services":
			if got, want := r.URL.Query().Get("include_disabled"), "1"; got != want {
				t.Fatalf("include_disabled = %q, want %q", got, want)
			}
			writeJSON(t, w, []map[string]interface{}{
				{"id": 801, "team_id": 42, "team_slug": "platform", "team_name": "Platform", "group_id": 10, "slug": "platform-api", "name": "Platform API", "description": "Production API", "default_rotation_id": 701, "default_rotation_name": "Primary", "default_escalation_policy_id": 1002, "default_escalation_policy_name": "Critical escalation", "notification_policy_id": 1102, "notification_policy_name": "Production notifications", "priority_policy_id": 501, "priority_policy_name": "Production priority"},
			})
		case "/api/services/match-rules":
			if got, want := r.URL.Query().Get("service_id"), "801"; got != want {
				t.Fatalf("service_id query = %q, want %q", got, want)
			}
			writeJSON(t, w, []map[string]interface{}{
				{"id": 1201, "team_id": 42, "team_slug": "platform", "team_name": "Platform", "route_id": nil, "route_name": nil, "service_id": 801, "service_slug": "platform-api", "service_name": "Platform API", "position": 10, "name": "API labels", "description": "Match API alerts", "matcher_preset_id": nil, "matchers": map[string]interface{}{"labels": map[string]interface{}{"service": "api"}}, "enabled": true},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	config := &Config{Client: client}

	rotation := datasourceRotation()
	rotationData := schema.TestResourceDataRaw(t, rotation.Schema, map[string]interface{}{
		"team_slug": "platform",
		"name":      "Primary",
	})
	if diags := rotation.ReadContext(context.Background(), rotationData, config); diags.HasError() {
		t.Fatalf("rotation read diagnostics: %v", diags)
	}
	if got, want := rotationData.Id(), "701"; got != want {
		t.Fatalf("rotation id = %q, want %q", got, want)
	}
	if got, want := rotationData.Get("rotation_id"), 701; got != want {
		t.Fatalf("rotation_id = %v, want %v", got, want)
	}
	if got, want := rotationData.Get("timezone"), "Europe/Moscow"; got != want {
		t.Fatalf("timezone = %v, want %v", got, want)
	}

	priority := datasourceIncidentPriority()
	priorityData := schema.TestResourceDataRaw(t, priority.Schema, map[string]interface{}{
		"slug": "p1",
	})
	if diags := priority.ReadContext(context.Background(), priorityData, config); diags.HasError() {
		t.Fatalf("priority read diagnostics: %v", diags)
	}
	if got, want := priorityData.Id(), "1"; got != want {
		t.Fatalf("priority id = %q, want %q", got, want)
	}
	if got, want := priorityData.Get("priority_id"), 1; got != want {
		t.Fatalf("priority_id = %v, want %v", got, want)
	}
	if got, want := priorityData.Get("level"), 1; got != want {
		t.Fatalf("level = %v, want %v", got, want)
	}

	channel := datasourceChannel()
	if _, exists := channel.Schema["config_json"]; exists {
		t.Fatal("channel data source must not expose masked config_json")
	}
	channelData := schema.TestResourceDataRaw(t, channel.Schema, map[string]interface{}{
		"group_slug":   "infra",
		"team_slug":    "platform",
		"name":         "Email",
		"channel_type": "email",
	})
	if diags := channel.ReadContext(context.Background(), channelData, config); diags.HasError() {
		t.Fatalf("channel read diagnostics: %v", diags)
	}
	if got, want := channelData.Id(), "902"; got != want {
		t.Fatalf("channel id = %q, want %q", got, want)
	}
	if got, want := channelData.Get("channel_id"), 902; got != want {
		t.Fatalf("channel_id = %v, want %v", got, want)
	}
	if got, want := channelData.Get("group_name"), "Infrastructure"; got != want {
		t.Fatalf("group_name = %v, want %v", got, want)
	}

	escalationPolicy := datasourceEscalationPolicy()
	escalationPolicyData := schema.TestResourceDataRaw(t, escalationPolicy.Schema, map[string]interface{}{
		"group_slug": "infra",
		"team_slug":  "platform",
		"name":       "Critical escalation",
	})
	if diags := escalationPolicy.ReadContext(context.Background(), escalationPolicyData, config); diags.HasError() {
		t.Fatalf("escalation policy read diagnostics: %v", diags)
	}
	if got, want := escalationPolicyData.Id(), "1002"; got != want {
		t.Fatalf("escalation policy id = %q, want %q", got, want)
	}
	if got, want := escalationPolicyData.Get("repeat_count"), 2; got != want {
		t.Fatalf("escalation policy repeat_count = %v, want %v", got, want)
	}

	notificationPolicy := datasourceNotificationPolicy()
	notificationPolicyData := schema.TestResourceDataRaw(t, notificationPolicy.Schema, map[string]interface{}{
		"team_id": 42,
		"name":    "Production notifications",
	})
	if diags := notificationPolicy.ReadContext(context.Background(), notificationPolicyData, config); diags.HasError() {
		t.Fatalf("notification policy read diagnostics: %v", diags)
	}
	if got, want := notificationPolicyData.Id(), "1102"; got != want {
		t.Fatalf("notification policy id = %q, want %q", got, want)
	}
	if got, want := notificationPolicyData.Get("rules_count"), 3; got != want {
		t.Fatalf("notification policy rules_count = %v, want %v", got, want)
	}
	if got, want := notificationPolicyData.Get("services_count"), 2; got != want {
		t.Fatalf("notification policy services_count = %v, want %v", got, want)
	}

	service := datasourceService()
	serviceData := schema.TestResourceDataRaw(t, service.Schema, map[string]interface{}{
		"team_id": 42,
		"slug":    "platform-api",
	})
	if diags := service.ReadContext(context.Background(), serviceData, config); diags.HasError() {
		t.Fatalf("service read diagnostics: %v", diags)
	}
	if got, want := serviceData.Get("priority_policy_id"), 501; got != want {
		t.Fatalf("priority_policy_id = %v, want %v", got, want)
	}
	if got, want := serviceData.Get("priority_policy_name"), "Production priority"; got != want {
		t.Fatalf("priority_policy_name = %v, want %v", got, want)
	}
	if got, want := serviceData.Get("default_escalation_policy_id"), 1002; got != want {
		t.Fatalf("default_escalation_policy_id = %v, want %v", got, want)
	}
	if got, want := serviceData.Get("notification_policy_id"), 1102; got != want {
		t.Fatalf("notification_policy_id = %v, want %v", got, want)
	}

	matchRuleResource := resourceServiceMatchRule()
	if !matchRuleResource.Schema["matcher_preset_id"].Optional {
		t.Fatal("service match rule matcher_preset_id is not optional")
	}
	if !matchRuleResource.Schema["matchers_json"].Optional {
		t.Fatal("service match rule matchers_json is not optional")
	}

	matchRule := datasourceServiceMatchRule()
	unscopedMatchRuleData := schema.TestResourceDataRaw(t, matchRule.Schema, map[string]interface{}{
		"name": "API labels",
	})
	if diags := matchRule.ReadContext(context.Background(), unscopedMatchRuleData, config); !diags.HasError() {
		t.Fatal("unscoped service match rule lookup returned no error")
	}
	matchRuleData := schema.TestResourceDataRaw(t, matchRule.Schema, map[string]interface{}{
		"service_id": 801,
		"name":       "API labels",
	})
	if diags := matchRule.ReadContext(context.Background(), matchRuleData, config); diags.HasError() {
		t.Fatalf("service match rule read diagnostics: %v", diags)
	}
	if got, want := matchRuleData.Id(), "1201"; got != want {
		t.Fatalf("service match rule id = %q, want %q", got, want)
	}
	if got, want := matchRuleData.Get("match_rule_id"), 1201; got != want {
		t.Fatalf("match_rule_id = %v, want %v", got, want)
	}
	if got, want := matchRuleData.Get("matchers_json"), `{"labels":{"service":"api"}}`; got != want {
		t.Fatalf("matchers_json = %v, want %v", got, want)
	}
}

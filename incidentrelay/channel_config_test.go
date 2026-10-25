package incidentrelay

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestEmailConfigRefreshAndPlan(t *testing.T) {
	cases := []struct {
		name, configured, remote string
		wantDiff                 bool
	}{
		{"legacy", `{}`, `{}`, false},
		{"explicit_assignee", `{"recipient_mode":"assignee"}`, `{}`, false},
		{"stale_assignee_address", `{"recipient_mode":"assignee","recipient_email":"old@example.com"}`, `{}`, false},
		{"implicit_assignee_address", `{"recipient_email":"old@example.com"}`, `{}`, false},
		{"fixed", `{"recipient_mode":"fixed","recipient_email":"  Backup@EXAMPLE.COM  "}`, `{"recipient_mode":"fixed","recipient_email":"Backup@example.com"}`, false},
		{"fallback", `{"recipient_mode":"fallback","recipient_email":"Backup@example.com"}`, `{"recipient_mode":"fallback","recipient_email":"Backup@example.com"}`, false},
		{"unicode_local", `{"recipient_mode":"fixed","recipient_email":"e\u0301@example.com"}`, `{"recipient_mode":"fixed","recipient_email":"é@example.com"}`, false},
		{"idna_domain", `{"recipient_mode":"fixed","recipient_email":"Team@xn--bcher-kva.de"}`, `{"recipient_mode":"fixed","recipient_email":"Team@bücher.de"}`, false},
		{"nontransitional_domain", `{"recipient_mode":"fixed","recipient_email":"Team@FAẞ.DE"}`, `{"recipient_mode":"fixed","recipient_email":"Team@faß.de"}`, false},
		{"role_mailbox", `{"recipient_mode":"fixed","recipient_email":"SUPPORT@EXAMPLE.COM"}`, `{"recipient_mode":"fixed","recipient_email":"support@example.com"}`, false},
		{"mode_change", `{"recipient_mode":"fixed","recipient_email":"Team@example.com"}`, `{"recipient_mode":"fallback","recipient_email":"Team@example.com"}`, true},
		{"switch_to_assignee", `{"recipient_mode":"assignee"}`, `{"recipient_mode":"fixed","recipient_email":"Team@example.com"}`, true},
		{"address_change", `{"recipient_mode":"fixed","recipient_email":"other@example.com"}`, `{"recipient_mode":"fixed","recipient_email":"Team@example.com"}`, true},
		{"local_case_change", `{"recipient_mode":"fixed","recipient_email":"team@example.com"}`, `{"recipient_mode":"fixed","recipient_email":"Team@example.com"}`, true},
		{"template_change", `{"recipient_mode":"assignee","html_template":"New"}`, `{"html_template":"Old"}`, true},
		{"severity_change", `{"recipient_mode":"assignee","notify_on_severities":["critical"]}`, `{"notify_on_severities":["warning"]}`, true},
		{"remote_extra_key", `{"recipient_mode":"assignee"}`, `{"custom":"changed"}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource := resourceChannel()
			if !resource.Schema["config_json"].DiffSuppressOnRefresh {
				t.Fatal("equivalence must apply on refresh as well as planning")
			}
			configured := map[string]interface{}{"team_id": 42, "name": "Email", "channel_type": "email", "enabled": true, "config_json": tc.configured}
			data := schema.TestResourceDataRaw(t, resource.Schema, configured)
			data.SetId("101")
			var remote map[string]interface{}
			if err := json.Unmarshal([]byte(tc.remote), &remote); err != nil {
				t.Fatal(err)
			}
			response := map[string]interface{}{"channel_type": "email", "config": remote}
			if err := channelResponseHook(data, response); err != nil {
				t.Fatal(err)
			}
			fields := []fieldDef{reqSensitiveJSON("config_json", "config", "Email config")}
			if err := setFieldsFromResponse(data, fields, []string{"config_json"}, response); err != nil {
				t.Fatal(err)
			}
			assertConfigPlan(t, resource, data, configured, "config_json", tc.wantDiff)
		})
	}
}

func TestEmailConfigDoesNotSuppressInvalidOrOtherChannelValues(t *testing.T) {
	resource := resourceChannel()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"channel_type": "email"})
	for _, raw := range []string{`null`, `[]`, `{`, `{"recipient_mode":null}`, `{"recipient_mode":[]}`, `{"recipient_mode":"unknown"}`, `{"recipient_mode":"fixed"}`, `{"recipient_mode":"fixed","recipient_email":42}`, `{"recipient_mode":"fixed","recipient_email":"invalid"}`, `{"recipient_mode":"fixed","recipient_email":"a@b@c"}`, `{"recipient_mode":"fixed","recipient_email":"a@bad domain"}`} {
		if suppressEquivalentEmailConfig("config_json", raw, `{}`, data) {
			t.Error("invalid config was suppressed")
		}
	}
	data = schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"channel_type": "webhook"})
	if suppressEquivalentEmailConfig("config_json", `{"recipient_mode":"assignee"}`, `{}`, data) {
		t.Fatal("email normalization applied to a webhook channel")
	}
}

func TestIncidentRelay24RouteSources(t *testing.T) {
	validate := resourceRoute().Schema["source"].ValidateFunc
	for _, source := range []string{"opensearch", "kibana", "alertmanager", "cloud_ru", "heartbeat"} {
		if _, errs := validate(source, "source"); len(errs) != 0 {
			t.Fatalf("source %s: %v", source, errs)
		}
	}
	if _, errs := validate("not_a_source", "source"); len(errs) == 0 {
		t.Fatal("unknown source accepted")
	}
}

func TestEmailConfigDoesNotHideChannelTypeChange(t *testing.T) {
	resource := resourceChannel()
	previous := map[string]interface{}{
		"team_id": 42, "name": "Channel", "channel_type": "webhook",
		"config_json": `{"recipient_mode":"assignee"}`,
	}
	data := schema.TestResourceDataRaw(t, resource.Schema, previous)
	data.SetId("101")
	desired := map[string]interface{}{
		"team_id": 42, "name": "Channel", "channel_type": "email", "config_json": `{}`,
	}
	assertConfigPlan(t, resource, data, desired, "config_json", true)
}

func assertConfigPlan(t *testing.T, resource *schema.Resource, data *schema.ResourceData, configured map[string]interface{}, field string, wantDiff bool) {
	t.Helper()
	diff, err := resource.SimpleDiff(context.Background(), data.State(), terraform.NewResourceConfigRaw(configured), nil)
	if err != nil {
		t.Fatal(err)
	}
	attr := diff.Attributes[field]
	gotDiff := attr != nil && attr.Old != attr.New
	if gotDiff != wantDiff {
		t.Fatalf("%s plan changed=%v, want %v", field, gotDiff, wantDiff)
	}
}

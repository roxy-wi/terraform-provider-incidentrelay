package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const cmdbTestConfig = `{"source_type":"jira_assets","base_url":"https://jira.example.com/",
 "api_path":"rest/insight/1.0/iql/objects","query_parameter":"iql",
 "services":{"query":"objectType = System","owner_team_attribute":"SRE"},
 "business_services":{"query":"objectType = Product"}}`

func TestSync252DefaultsDoNotCauseDrift(t *testing.T) {
	server := strings.Replace(directoryTestConfig, `"base_url"`, `"current_enabled":true,"retained_source_id":null,"base_url"`, 1)
	if !suppressEquivalentSyncConfig("", directoryTestConfig, server, nil) {
		t.Fatal("IR 2.5.2 default fields caused drift")
	}
	for _, changed := range []string{
		strings.Replace(server, `"current_enabled":true`, `"current_enabled":false`, 1),
		strings.Replace(server, `"retained_source_id":null`, `"retained_source_id":8`, 1),
		strings.Replace(server, `https://directory.example.com/`, `https://other.example.com`, 1),
	} {
		if suppressEquivalentSyncConfig("", directoryTestConfig, changed, nil) {
			t.Fatal("real configuration change suppressed")
		}
	}
	normalized, err := normalizeDirectoryConfig(directoryTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(normalized, "current_enabled") || strings.Contains(normalized, "retained_source_id") {
		t.Fatal("new defaults must not be sent to IR 2.5.1")
	}
	for _, value := range []string{`"current_enabled":"false"`, `"current_enabled":null`, `"retained_source_id":0`, `"retained_source_id":1.5`} {
		invalid := strings.Replace(directoryTestConfig, `"base_url"`, value+`,"base_url"`, 1)
		if _, err := normalizeDirectoryConfig(invalid); err == nil {
			t.Fatal("accepted invalid 2.5.2 option", value)
		}
	}
}

func TestSync252LegacyStateDefaultsToDisable(t *testing.T) {
	resource := resourceDirectorySyncSource()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "Directory", "group_id": 2, "config_json": directoryTestConfig,
	})
	data.SetId("8")
	if err := data.Set("destroy_behavior", ""); err != nil {
		t.Fatal(err)
	}
	enabled := false
	if err := saveDirectorySync(data, directorySyncDetail{
		ID: 8, Name: "Directory", GroupID: 2, Revision: 1, Enabled: &enabled, Config: json.RawMessage(directoryTestConfig),
	}); err != nil {
		t.Fatal(err)
	}
	if data.Get("destroy_behavior") != "disable" {
		t.Fatal("imported or old state must retain disable behavior")
	}
	if err := data.Set("destroy_behavior", "delete_keep_data"); err != nil {
		t.Fatal(err)
	}
	if err := saveDirectorySync(data, directorySyncDetail{
		ID: 8, Name: "Directory", GroupID: 2, Revision: 1, Enabled: &enabled, Config: json.RawMessage(directoryTestConfig),
	}); err != nil {
		t.Fatal(err)
	}
	if data.Get("destroy_behavior") != "delete_keep_data" {
		t.Fatal("refresh overwrote an explicit lifecycle policy")
	}
}

func TestCMDBConfigNormalization(t *testing.T) {
	normalized, err := normalizeDirectoryConfig(cmdbTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	again, err := normalizeDirectoryConfig(normalized)
	if err != nil || normalized != again {
		t.Fatal("CMDB normalization is not idempotent", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(normalized), &config); err != nil {
		t.Fatal(err)
	}
	if config["api_path"] != "/rest/insight/1.0/iql/objects" || config["query_parameter"] != "iql" || config["timeout_seconds"] != float64(30) {
		t.Fatal("CMDB defaults or API path differ from the server")
	}
	if config["current_enabled"] != nil || config["operations"] != nil || config["members"] != nil {
		t.Fatal("directory defaults leaked into CMDB config")
	}
	selection := config["business_services"].(map[string]interface{})
	if value, exists := selection["owner_team_attribute"]; !exists || value != nil {
		t.Fatal("missing optional selection defaults")
	}
	for _, raw := range []string{
		strings.Replace(cmdbTestConfig, "jira_assets", "unknown", 1),
		strings.Replace(cmdbTestConfig, `{"query":"objectType = Product"}`, "[]", 1),
		strings.Replace(cmdbTestConfig, "objectType = Product", " ", 1),
		strings.Replace(cmdbTestConfig, "rest/insight/1.0/iql/objects", "https://other.example.com/api", 1),
		strings.Replace(cmdbTestConfig, "rest/insight/1.0/iql/objects", "//other.example.com/api", 1),
		strings.Replace(cmdbTestConfig, "rest/insight/1.0/iql/objects", "/api?query=example", 1),
		strings.Replace(cmdbTestConfig, "rest/insight/1.0/iql/objects", "/api#fragment", 1),
	} {
		if _, err := normalizeDirectoryConfig(raw); err == nil {
			t.Fatal("invalid CMDB selection accepted")
		}
	}
}

func TestSync252ReadStatusAndPendingApproval(t *testing.T) {
	normalized, err := normalizeDirectoryConfig(cmdbTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	client := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if body["enabled"] != true {
			t.Error("requested schedule was lost")
		}
		_, _ = fmt.Fprintf(w, `{"id":8,"group_id":2,"name":"Jira","config":%s,"revision":1,"enabled":true,
   "approved":false,"schedule_state":"pending_approval","current_error":"refresh failed",
   "current_running":true,"next_run_at":null,"next_current_at":"2026-11-21T10:00:00Z"}`, normalized)
	})
	resource := resourceDirectorySyncSource()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "Jira", "group_id": 2, "config_json": cmdbTestConfig, "enabled": true,
	})
	if diags := resource.CreateContext(context.Background(), data, client); diags.HasError() {
		t.Fatal(diags)
	}
	if data.Id() != "8" || data.Get("source_type") != "jira_assets" || data.Get("schedule_state") != "pending_approval" ||
		data.Get("approved") != false || data.Get("current_running") != true || data.Get("current_error") != "refresh failed" ||
		data.Get("next_run_at") != "" || data.Get("next_current_at") != "2026-11-21T10:00:00Z" {
		t.Fatal("source status was not saved correctly")
	}
}

func TestSync252DeleteContract(t *testing.T) {
	for _, mode := range []string{"delete_keep_data", "delete_with_data"} {
		for _, status := range []int{200, 400, 401, 403, 404, 405, 409, 500} {
			t.Run(fmt.Sprintf("%s/%d", mode, status), func(t *testing.T) {
				writes := 0
				client := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					writes++
					if r.Method != http.MethodDelete || r.URL.Path != directorySyncPath+"/8" {
						t.Error("unexpected deletion request")
					}
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					action := "keep"
					if mode == "delete_with_data" {
						action = "delete"
					}
					if body["revision"] != float64(4) || body["data_action"] != action || len(body) != 2 {
						t.Error("delete must send only planned revision and explicit data choice")
					}
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"id":8,"deleted":true}`))
				})
				resource := resourceDirectorySyncSource()
				data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
					"name": "Directory", "group_id": 2, "config_json": directoryTestConfig, "destroy_behavior": mode,
				})
				data.SetId("8")
				if err := data.Set("revision", 4); err != nil {
					t.Fatal(err)
				}
				diags := resource.DeleteContext(context.Background(), data, client)
				success := status == 200 || status == 404
				if diags.HasError() == success || (data.Id() == "") != success || writes != 1 {
					t.Fatal("delete lost its binding on failure or retried with another revision", diags)
				}
			})
		}
	}
}

func TestSync252DeleteRequiresConfirmation(t *testing.T) {
	for _, response := range []string{`{}`, `{"id":9,"deleted":true}`, `{"id":8,"deleted":false}`} {
		client := changeTestClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) })
		resource := resourceDirectorySyncSource()
		data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
			"name": "Directory", "group_id": 2, "config_json": directoryTestConfig, "destroy_behavior": "delete_keep_data",
		})
		data.SetId("8")
		if !resource.DeleteContext(context.Background(), data, client).HasError() || data.Id() != "8" {
			t.Fatal("unconfirmed delete removed the Terraform binding")
		}
	}
}

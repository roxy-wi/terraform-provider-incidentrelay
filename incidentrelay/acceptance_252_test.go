package incidentrelay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAcc252CLI(t *testing.T, config *Config, dir string, env []string, tf string) {
	ctx := context.Background()
	spec := testAccReadAPIObject(t, ctx, config, "/api/openapi.json")
	components, _ := spec["components"].(map[string]interface{})
	schemas, _ := components["schemas"].(map[string]interface{})
	if _, exists := schemas["DirectorySyncCmdbConfig"]; !exists {
		t.Skip("requires IR 2.5.2 CMDB API")
	}
	run := func(args ...string) string {
		t.Helper()
		return strings.TrimSpace(testAccRunCommand(t, dir, env, tf, args...))
	}
	apply := func() { t.Helper(); run("apply", "-input=false", "-auto-approve", "-no-color") }
	noDiff := func() { t.Helper(); run("plan", "-input=false", "-no-color", "-detailed-exitcode") }
	save := func(hcl string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "ir252.tf"), []byte(hcl), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fixture := `
resource "incidentrelay_directory_sync_source" "duty252" {
 name = "Duty"
 group_id = incidentrelay_group.cli.id
 config_json = DIRECTORY_CONFIG
 enabled = true
}
resource "incidentrelay_directory_sync_source" "jira252" {
 name = "Jira"
 group_id = incidentrelay_group.cli.id
 enabled = true
 destroy_behavior = "delete_keep_data"
 headers_json = jsonencode({Authorization = "Bearer fixture-secret"})
 config_json = jsonencode({
  source_type = "jira_assets"
  base_url = "https://jira.example.com/"
  api_path = "rest/insight/1.0/iql/objects"
  query_parameter = "iql"
  services = {query = "objectType = System", owner_team_attribute = "SRE"}
  business_services = {query = "objectType = Product"}
  team_matches = {"42" = tonumber(incidentrelay_team.cli.id)}
 })
}
resource "incidentrelay_directory_sync_source" "discard252" {
 name = "Disposable source"
 group_id = incidentrelay_group.cli.id
 config_json = DISCARD_CONFIG
 destroy_behavior = "delete_with_data"
}
output "duty252_id" { value = incidentrelay_directory_sync_source.duty252.id }
output "jira252_id" { value = incidentrelay_directory_sync_source.jira252.id }
output "discard252_id" { value = incidentrelay_directory_sync_source.discard252.id }
output "jira252_type" { value = incidentrelay_directory_sync_source.jira252.source_type }
`
	fixture = strings.Replace(fixture, "DIRECTORY_CONFIG", fmt.Sprintf("%q", directoryTestConfig), 1)
	fixture = strings.Replace(fixture, "DISCARD_CONFIG", fmt.Sprintf("%q", directoryTestConfig), 1)
	save(fixture)
	apply()
	noDiff()
	dutyID := run("output", "-raw", "duty252_id")
	jiraID := run("output", "-raw", "jira252_id")
	discardID := run("output", "-raw", "discard252_id")
	get := func(id string) directorySyncDetail {
		t.Helper()
		result, err := getDirectorySync(ctx, config.Client, id)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	for _, id := range []string{dutyID, jiraID} {
		live := get(id)
		if !*live.Enabled || live.Approved || live.ScheduleState != "pending_approval" || live.NextRunAt != "" {
			t.Fatal("requested schedule must remain pending UX approval")
		}
	}
	if run("output", "-raw", "jira252_type") != "jira_assets" {
		t.Fatal("CMDB source type missing")
	}
	// Settings added by 2.5.2 must cause neither refresh drift nor invalid writes.
	updatedConfig := strings.Replace(directoryTestConfig, `"base_url"`,
		`"current_enabled":false,"current_interval_seconds":90,"retained_source_id":null,"base_url"`, 1)
	fixture = strings.Replace(fixture, fmt.Sprintf("%q", directoryTestConfig), fmt.Sprintf("%q", updatedConfig), 1)
	save(fixture)
	apply()
	noDiff()
	var dutyConfig map[string]interface{}
	if err := json.Unmarshal(get(dutyID).Config, &dutyConfig); err != nil {
		t.Fatal(err)
	}
	if dutyConfig["current_enabled"] != false || dutyConfig["current_interval_seconds"] != float64(90) {
		t.Fatal("current-duty settings were not persisted")
	}
	// Switching source families is a replacement, not an unsupported in-place PUT.
	changedType := strings.Replace(fixture, fmt.Sprintf("%q", updatedConfig), fmt.Sprintf("%q", cmdbTestConfig), 1)
	save(changedType)
	run("plan", "-input=false", "-out=sync-kind.plan")
	var plan struct {
		Changes []struct {
			Address string `json:"address"`
			Change  struct {
				Actions []string `json:"actions"`
			} `json:"change"`
		} `json:"resource_changes"`
	}
	if err := json.Unmarshal([]byte(run("show", "-json", "sync-kind.plan")), &plan); err != nil {
		t.Fatal(err)
	}
	replaced := false
	for _, change := range plan.Changes {
		if change.Address == "incidentrelay_directory_sync_source.duty252" {
			replaced = strings.Join(change.Change.Actions, ",") == "delete,create"
		}
	}
	if !replaced {
		t.Fatal("changing sync source type must require replacement")
	}
	save(fixture)
	// Import cannot read headers. A rename after import must preserve them.
	fixture = strings.Replace(fixture, ` headers_json = jsonencode({Authorization = "Bearer fixture-secret"})`, "", 1)
	save(fixture)
	run("state", "rm", "incidentrelay_directory_sync_source.jira252")
	run("import", "-input=false", "incidentrelay_directory_sync_source.jira252", jiraID)
	// Import defaults the local deletion policy to disable. Apply the explicit
	// keep-data choice without issuing an otherwise unnecessary server update.
	revisionBeforePolicy := get(jiraID).Revision
	apply()
	if get(jiraID).Revision != revisionBeforePolicy {
		t.Fatal("local destroy policy triggered a remote write")
	}
	noDiff()
	fixture = strings.Replace(fixture, `name = "Jira"`, `name = "Jira renamed"`, 1)
	save(fixture)
	apply()
	noDiff()
	if !get(jiraID).HasHeaders {
		t.Fatal("CMDB rename lost stored credentials")
	}
	// Destruction uses the planned revision; it cannot delete a later UX edit.
	run("plan", "-destroy", "-target=incidentrelay_directory_sync_source.jira252", "-input=false", "-out=cmdb-delete.plan")
	live := get(jiraID)
	if err := config.Client.Do(ctx, http.MethodPut, directorySyncPath+"/"+jiraID, map[string]interface{}{
		"name": "UX rename", "group_id": live.GroupID, "config": live.Config, "enabled": true, "revision": live.Revision,
	}, nil); err != nil {
		t.Fatal(err)
	}
	result := testAccRunCommandResult(dir, env, tf, "apply", "-input=false", "-no-color", "cmdb-delete.plan")
	if result.ExitCode == 0 || !strings.Contains(result.Output, "409") {
		t.Fatalf("expected stale destroy to fail: %s", result.Output)
	}
	if get(jiraID).Name != "UX rename" {
		t.Fatal("stale destroy changed the source")
	}
	run("state", "show", "incidentrelay_directory_sync_source.jira252")
	apply()
	noDiff()
	if err := os.Remove(filepath.Join(dir, "ir252.tf")); err != nil {
		t.Fatal(err)
	}
	apply()
	if *get(dutyID).Enabled {
		t.Fatal("default destroy must disable the retained source")
	}
	for _, id := range []string{jiraID, discardID} {
		if _, err := getDirectorySync(ctx, config.Client, id); !errors.Is(err, ErrNotFound) {
			t.Fatal("explicit destroy did not archive the source", err)
		}
	}
	// An archived Directory source can be referenced for a reviewed reconnection.
	live = get(dutyID)
	if err := config.Client.Do(ctx, http.MethodDelete, directorySyncPath+"/"+dutyID,
		map[string]interface{}{"revision": live.Revision, "data_action": "keep"}, nil); err != nil {
		t.Fatal(err)
	}
	reconnectConfig := strings.Replace(updatedConfig, `"retained_source_id":null`, `"retained_source_id":`+dutyID, 1)
	save(fmt.Sprintf(`
resource "incidentrelay_directory_sync_source" "reconnect252" {
 name = "Reconnect"
 group_id = incidentrelay_group.cli.id
 config_json = %q
 destroy_behavior = "delete_keep_data"
}
`, reconnectConfig))
	apply()
	noDiff()
	if err := os.Remove(filepath.Join(dir, "ir252.tf")); err != nil {
		t.Fatal(err)
	}
	apply()
}

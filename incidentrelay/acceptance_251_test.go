package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAccPersonalConfig(t *testing.T, login *Config) *Config {
	t.Helper()
	var issued struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	err := login.Client.Do(context.Background(), http.MethodPost, "/api/profile/tokens", map[string]interface{}{
		"name": "terraform-acceptance", "days": 1,
		"scopes": []string{
			"resources:read", "resources:write", "alerts:read", "alerts:write", "profile:read",
			"change-integrations:read", "change-integrations:write",
			"system-settings:read", "system-settings:write", "directory-sync:read", "directory-sync:write",
		},
	}, &issued)
	if err != nil {
		t.Fatal(err)
	}
	if issued.ID < 1 || issued.Token == "" {
		t.Fatal("personal token issuance returned incomplete metadata")
	}
	t.Cleanup(func() {
		if err := login.Client.Do(context.Background(), http.MethodDelete, fmt.Sprintf("/api/profile/tokens/%d", issued.ID), nil, nil); err != nil {
			t.Errorf("revoke acceptance token: %v", err)
		}
	})
	client, err := NewClient(ClientConfig{BaseURL: login.Client.baseURL, Token: issued.Token})
	if err != nil {
		t.Fatal(err)
	}
	return &Config{Client: client}
}

func testAcc251CLI(t *testing.T, config *Config, dir string, env []string, tf string) {
	ctx := context.Background()
	spec := testAccReadAPIObject(t, ctx, config, "/api/openapi.json")
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI paths missing")
	}
	if _, exists := paths[directorySyncPath]; !exists {
		t.Skip("requires IR 2.5.1 Directory Sync API")
	}
	detailPath, _ := paths[directorySyncPath+"/{source_id}"].(map[string]interface{})
	_, supports252 := detailPath["delete"]
	run := func(args ...string) string {
		t.Helper()
		return strings.TrimSpace(testAccRunCommand(t, dir, env, tf, args...))
	}
	apply := func() { t.Helper(); run("apply", "-input=false", "-auto-approve", "-no-color") }
	noDiff := func() { t.Helper(); run("plan", "-input=false", "-no-color", "-detailed-exitcode") }
	save := func(hcl string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "ir251.tf"), []byte(hcl), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fixture := `
resource "incidentrelay_change_integration" "gitlab" {
 name = "GitLab"
 owner_group_id = incidentrelay_group.cli.id
 owner_team_id = incidentrelay_team.cli.id
}
resource "incidentrelay_change_source" "gitlab" {
 integration_id = incidentrelay_change_integration.gitlab.id
 owner_team_id = incidentrelay_team.cli.id
 source = "gitlab-deployments"
 ordering_mode = "timestamp"
 gitlab_mappings_json = jsonencode([{project_id = 42, environment = "production", service_ids = [tonumber(incidentrelay_service.cli.id)]}])
}
resource "incidentrelay_change_grant" "gitlab" {
 integration_id = incidentrelay_change_integration.gitlab.id
 source_id = incidentrelay_change_source.gitlab.id
 capability = "changed"
 target_team_id = incidentrelay_team.cli.id
 target_service_id = incidentrelay_service.cli.id
}
resource "incidentrelay_directory_sync_source" "directory" {
 name = "Directory"
 group_id = incidentrelay_group.cli.id
 config_json = CONFIG_JSON
 headers_json = jsonencode({Authorization = "Bearer fixture-secret"})
}
output "gitlab_integration_id" { value = incidentrelay_change_integration.gitlab.id }
output "gitlab_source_id" { value = incidentrelay_change_source.gitlab.id }
output "gitlab_webhook_path" { value = incidentrelay_change_source.gitlab.gitlab_webhook_path }
output "directory_source_id" { value = incidentrelay_directory_sync_source.directory.id }
`
	fixture = strings.Replace(fixture, "CONFIG_JSON", fmt.Sprintf("%q", directoryTestConfig), 1)
	save(fixture)
	apply()
	noDiff()
	integration := run("output", "-raw", "gitlab_integration_id")
	source := run("output", "-raw", "gitlab_source_id")
	directory := run("output", "-raw", "directory_source_id")
	if run("output", "-raw", "gitlab_webhook_path") != "/api/change-events/gitlab/"+source {
		t.Fatal("wrong webhook path")
	}
	getDirectory := func() directorySyncDetail {
		t.Helper()
		result, err := getDirectorySync(ctx, config.Client, directory)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	if result := getDirectory(); *result.Enabled || result.Approved || !result.HasHeaders {
		t.Fatal("new directory source must be disabled, unapproved, with credentials stored")
	}
	// Headers are write-only. After import, omitting them must preserve the secret.
	fixture = strings.Replace(fixture, ` headers_json = jsonencode({Authorization = "Bearer fixture-secret"})`, "", 1)
	save(fixture)
	for _, item := range []struct{ address, id string }{
		{"incidentrelay_change_source.gitlab", integration + "/" + source},
		{"incidentrelay_directory_sync_source.directory", directory},
	} {
		run("state", "rm", item.address)
		run("import", "-input=false", item.address, item.id)
	}
	noDiff()
	fixture = strings.Replace(fixture, `name = "Directory"`, `name = "Directory renamed"`, 1)
	save(fixture)
	apply()
	if result := getDirectory(); result.Name != "Directory renamed" || !result.HasHeaders {
		t.Fatal("rename lost headers")
	}
	noDiff()
	// IR 2.5.1 rejects an unapproved enable and must retain its binding.
	// IR 2.5.2 records the request but keeps the schedule pending UX approval.
	invalid := strings.Replace(fixture, `name = "Directory renamed"`, `name = "Directory renamed"
 enabled = true`, 1)
	save(invalid)
	result := testAccRunCommandResult(dir, env, tf, "apply", "-input=false", "-auto-approve", "-no-color")
	if supports252 {
		if result.ExitCode != 0 {
			t.Fatalf("pending approval schedule failed: %s", result.Output)
		}
		if live := getDirectory(); live.ScheduleState != "pending_approval" || live.Approved {
			t.Fatal("IR 2.5.2 must defer synchronization until UX approval")
		}
	} else if result.ExitCode == 0 || !strings.Contains(result.Output, "409") {
		t.Fatalf("expected IR 2.5.1 unapproved enable to fail: %s", result.Output)
	}
	if run("output", "-raw", "directory_source_id") != directory {
		t.Fatal("failed enable lost identity")
	}
	save(fixture)
	if supports252 {
		apply()
	}
	noDiff()
	// A changed revision after plan must not be overwritten by Terraform.
	planned := strings.Replace(fixture, `name = "Directory renamed"`, `name = "Planned rename"`, 1)
	save(planned)
	run("plan", "-input=false", "-out=directory.plan")
	live := getDirectory()
	err := config.Client.Do(ctx, http.MethodPut, directorySyncPath+"/"+directory, map[string]interface{}{
		"name": "UX rename", "group_id": live.GroupID, "config": live.Config, "enabled": false, "revision": live.Revision,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result = testAccRunCommandResult(dir, env, tf, "apply", "-input=false", "-no-color", "directory.plan")
	if result.ExitCode == 0 || !strings.Contains(result.Output, "409") {
		t.Fatalf("expected stale revision error: %s", result.Output)
	}
	if getDirectory().Name != "UX rename" {
		t.Fatal("stale plan overwrote UX edit")
	}
	apply()
	noDiff()
	fixture = planned
	// Omit mappings and update another source field after a saved plan. A UX
	// mapping edit between plan and apply must survive the source PATCH.
	mappingLine := ` gitlab_mappings_json = jsonencode([{project_id = 42, environment = "production", service_ids = [tonumber(incidentrelay_service.cli.id)]}])`
	fixture = strings.Replace(fixture, mappingLine, "", 1)
	fixture = strings.Replace(fixture, `source = "gitlab-deployments"`, `source = "gitlab-updated"`, 1)
	save(fixture)
	run("plan", "-input=false", "-out=gitlab.plan")
	detail, err := changeRequest(ctx, config.Client, http.MethodGet, changeAdminPath+"/"+integration, nil, testAccIDAsInt(t, integration))
	if err != nil {
		t.Fatal(err)
	}
	err = config.Client.Do(ctx, http.MethodPatch, changeAdminPath+"/"+integration+"/sources/"+source, map[string]interface{}{
		"expected_revision": detail.Revision,
		"gitlab_mappings":   []map[string]interface{}{{"project_id": 43, "environment": "staging", "service_ids": []int{testAccIDAsInt(t, run("output", "-raw", "service_id"))}}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	run("apply", "-input=false", "gitlab.plan")
	detail, err = changeRequest(ctx, config.Client, http.MethodGet, changeAdminPath+"/"+integration, nil, testAccIDAsInt(t, integration))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range detail.Sources {
		if fmt.Sprint(item.ID) == source {
			mappings, err := parseGitLabMappings(string(item.GitLabMappings))
			if err != nil || len(mappings) != 1 || mappings[0].ProjectID != 43 {
				t.Fatal("unmanaged GitLab mapping lost", err)
			}
		}
	}
	noDiff()
	// Explicit [] and {} clear the two independently managed configurations.
	fixture = strings.Replace(fixture, `source = "gitlab-updated"`, `source = "gitlab-updated"
 gitlab_mappings_json = "[]"`, 1)
	fixture = strings.Replace(fixture, `name = "Planned rename"`, `name = "Planned rename"
 headers_json = "{}"`, 1)
	save(fixture)
	apply()
	noDiff()
	// IR encrypts even an empty header object, so has_headers can stay true.
	if err := os.Remove(filepath.Join(dir, "ir251.tf")); err != nil {
		t.Fatal(err)
	}
	apply()
	if *getDirectory().Enabled {
		t.Fatal("destroy left periodic sync enabled")
	}
}

// Decode helper used by permission tests without printing token material.
func testAccTokenClient(t *testing.T, login *Config, scopes []string) *Client {
	t.Helper()
	var response map[string]json.RawMessage
	if err := login.Client.Do(context.Background(), http.MethodPost, "/api/profile/tokens",
		map[string]interface{}{"name": "scope-check", "days": 1, "scopes": scopes}, &response); err != nil {
		t.Fatal(err)
	}
	var token string
	var id int
	if err := json.Unmarshal(response["token"], &token); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(response["id"], &id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := login.Client.Do(context.Background(), http.MethodDelete, fmt.Sprintf("/api/profile/tokens/%d", id), nil, nil); err != nil {
			t.Error(err)
		}
	})
	client, err := NewClient(ClientConfig{BaseURL: login.Client.baseURL, Token: token})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

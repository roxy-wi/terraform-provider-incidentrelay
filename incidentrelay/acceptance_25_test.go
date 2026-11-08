package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const changeCLIFixture = `
resource "incidentrelay_service" "correlation25" {
  team_id = incidentrelay_team.cli.id
  slug = "change-audit"
  name = "Change audit"
  description = "Before"
  %s
}
resource "incidentrelay_change_integration" "ci" {
  name = "Deployments"
  owner_group_id = incidentrelay_group.cli.id
  owner_team_id = incidentrelay_team.cli.id
}
resource "incidentrelay_change_source" "releases" {
  integration_id = incidentrelay_change_integration.ci.id
  owner_team_id = incidentrelay_team.cli.id
  source = "release-events"
  ordering_mode = "sequence"
}
resource "incidentrelay_change_grant" "deploy" {
  integration_id = incidentrelay_change_integration.ci.id
  source_id = incidentrelay_change_source.releases.id
  capability = "changed"
  target_team_id = incidentrelay_team.cli.id
  target_service_id = incidentrelay_service.correlation25.id
}
output "integration25" { value = incidentrelay_change_integration.ci.id }
output "source25" { value = incidentrelay_change_source.releases.id }
output "grant25" { value = incidentrelay_change_grant.deploy.id }
output "service25" { value = incidentrelay_service.correlation25.id }
`

func testAccChangeCLI(t *testing.T, config *Config, dir string, env []string, tf string) {
	t.Helper()
	run := func(args ...string) string { return strings.TrimSpace(testAccRunCommand(t, dir, env, tf, args...)) }
	apply := func() { run("apply", "-auto-approve", "-input=false", "-no-color") }
	noDiff := func() { run("plan", "-input=false", "-no-color", "-detailed-exitcode") }
	save := func(s string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "changes.tf"), []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	const policy = `change_correlation_enabled = true
  change_correlation_exclusions_json = jsonencode([{ source = "heartbeat" }])`
	original := fmt.Sprintf(changeCLIFixture, policy)
	save(original)
	apply()
	id := func(out string) string { return run("output", "-raw", out) }
	integration, source, grant, service := id("integration25"), id("source25"), id("grant25"), id("service25")
	for _, obj := range []struct{ address, id string }{
		{"incidentrelay_service.correlation25", service},
		{"incidentrelay_change_integration.ci", integration},
		{"incidentrelay_change_source.releases", integration + "/" + source},
		{"incidentrelay_change_grant.deploy", integration + "/" + source + "/" + grant},
	} {
		run("state", "rm", obj.address)
		run("import", "-input=false", "-no-color", obj.address, obj.id)
	}
	noDiff()
	checkPolicy := func(enabled bool, rules []interface{}) {
		t.Helper()
		s := testAccReadAPIObject(t, context.Background(), config, "/api/services/"+service)
		if s["change_correlation_enabled"] != enabled || !reflect.DeepEqual(s["change_correlation_exclusions"], rules) {
			t.Fatalf("unexpected correlation settings: %v %v", s["change_correlation_enabled"], s["change_correlation_exclusions"])
		}
		lookup := datasourceService()
		data := schema.TestResourceDataRaw(t, lookup.Schema, map[string]interface{}{"service_id": testAccIDAsInt(t, service)})
		testAccRequireNoDiags(t, lookup.ReadContext(context.Background(), data, config))
		raw, err := json.Marshal(rules)
		if err != nil {
			t.Fatal(err)
		}
		if data.Get("change_correlation_enabled") != enabled || data.Get("change_correlation_exclusions_json") != string(raw) {
			t.Fatal("service lookup did not report the effective correlation settings")
		}
	}
	expected := []interface{}{map[string]interface{}{"source": "heartbeat"}}
	checkPolicy(true, expected)
	// Stop managing the policy, then force an ordinary Service PUT. It must
	// preserve both effective values even though Terraform has them in state.
	updated := strings.Replace(original, policy, "", 1)
	updated = strings.Replace(updated, `description = "Before"`, `description = "After"`, 1)
	updated = strings.Replace(updated, "target_service_id = incidentrelay_service.correlation25.id", "active = false", 1)
	updated = strings.Replace(updated, "source_id = incidentrelay_change_source.releases.id", "source_id = incidentrelay_change_integration.ci.initial_source_id", 1)
	updated = strings.Replace(updated, `ordering_mode = "sequence"`, `ordering_mode = "timestamp"`, 1)
	updated = strings.Replace(updated, `name = "Deployments"`, `name = "Production deployments"`, 1)
	save(updated)
	run("plan", "-input=false", "-out=legacy.plan")
	// Simulate an administrator editing exclusions after planning. Sending a
	// value copied from state would overwrite the new rule during apply.
	externalRules := []interface{}{map[string]interface{}{"labels": map[string]interface{}{"site": "test"}}}
	teamID := testAccIDAsInt(t, id("team_id"))
	err := config.Client.Do(context.Background(), "PUT", "/api/services/"+service, map[string]interface{}{
		"team_id": teamID, "name": "Change audit", "slug": "change-audit", "description": "Before",
		"change_correlation_exclusions": externalRules,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	run("apply", "-input=false", "-no-color", "legacy.plan")
	checkPolicy(true, externalRules)
	noDiff()
	// Explicitly clear the policy; deleting the attributes above must not clear it.
	save(strings.Replace(updated, `description = "After"`, `description = "After"
  change_correlation_enabled = false
  change_correlation_exclusions_json = "[]"`, 1))
	apply()
	checkPolicy(false, []interface{}{})
	noDiff()
	var detail integrationDetail
	read := func() {
		t.Helper()
		detail = integrationDetail{}
		if err := config.Client.Do(context.Background(), "GET", changeAdminPath+"/"+integration, nil, &detail); err != nil {
			t.Fatal(err)
		}
		if !detail.complete(testAccIDAsInt(t, integration)) {
			t.Fatal("incomplete Change integration details")
		}
	}
	read()
	if detail.Name != "Production deployments" {
		t.Fatal("integration rename was not persisted")
	}
	found := false
	for _, s := range detail.Sources {
		for _, g := range s.Grants {
			if fmt.Sprint(g.ID) == grant {
				found = true
				if *g.Active || g.Service != 0 || s.ID != detail.Sources[0].ID {
					t.Fatal("grant was not moved, widened and disabled in place")
				}
			}
		}
	}
	if !found {
		t.Fatal("updated grant lost its identity")
	}
	// Re-enable the grant at Service scope so destroy must actively revoke it.
	finalConfig := strings.Replace(updated, "active = false", "target_service_id = incidentrelay_service.correlation25.id", 1)
	save(finalConfig)
	apply()
	// Remove the 2.5 config while its owner group/team still exist; the API
	// must retain the historical rows but stop granting authority.
	if err := os.Remove(filepath.Join(dir, "changes.tf")); err != nil {
		t.Fatal(err)
	}
	apply()
	read()
	if *detail.Active {
		t.Fatal("destroy left integration active")
	}
	for _, s := range detail.Sources {
		if fmt.Sprint(s.ID) == source && *s.Active {
			t.Fatal("destroy left managed source active")
		}
		for _, g := range s.Grants {
			if fmt.Sprint(g.ID) == grant && *g.Active {
				t.Fatal("destroy left grant active")
			}
		}
	}
}

// The pinned 2.5 candidate predates the Google endpoint. Only skip this live
// case when OpenAPI actually lacks it; schema validation is tested everywhere.
func TestAccGoogleCloudMonitoringRoute(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	if !testAccSupportsVersion(t, config, 2, 5) {
		t.Skip("requires IR 2.5+")
	}
	spec := testAccReadAPIObject(t, ctx, config, "/api/openapi.json")
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI response has no paths object")
	}
	if _, ok := paths["/api/integrations/google-cloud-monitoring"]; !ok {
		t.Skip("server has no Google Cloud Monitoring endpoint")
	}
	suffix := testAccSuffix()
	group := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{"name": "GCM", "slug": "gcm-g-" + suffix})
	team := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{"group_id": testAccIDAsInt(t, group.Id()), "name": "GCM", "slug": "gcm-t-" + suffix})
	r := resourceRoute()
	v := map[string]interface{}{"name": "Google Cloud", "team_id": testAccIDAsInt(t, team.Id()), "source": "google_cloud_monitoring", "group_by": []interface{}{"gcp_project_id", "gcp_incident_id"}}
	d := testAccCreateResource(t, ctx, r, config, v)
	imported := testAccImportReadResource(t, ctx, r, config, d, nil, map[string]interface{}{"source": "google_cloud_monitoring", "integration_config_json": "{}"})
	assertConfigPlan(t, r, imported, v, "source", false)
	testAccUpdateResource(t, ctx, r, d, config, map[string]interface{}{"name": "Google updated"}, map[string]interface{}{"source": "google_cloud_monitoring"})
}

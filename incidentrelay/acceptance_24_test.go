package incidentrelay

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func testAccSupports24(t *testing.T, config *Config) bool {
	t.Helper()
	version := testAccReadAPIObject(t, context.Background(), config, "/api/version")
	parts := strings.Split(fmt.Sprint(version["service_version"]), ".")
	if len(parts) < 2 {
		t.Fatal("API did not return a major.minor service version")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		t.Fatalf("parse service major version: %v", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("parse service minor version: %v", err)
	}
	return major > 2 || (major == 2 && minor >= 4)
}

func TestAccIncidentRelay24Configuration(t *testing.T) {
	config := testAccProviderConfig(t)
	if !testAccSupports24(t, config) {
		t.Skip("requires IncidentRelay 2.4+")
	}
	ctx := context.Background()
	suffix := testAccSuffix()
	group := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug": "tf24-g-" + suffix, "name": "IR24 group " + suffix,
	})
	team := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id": testAccIDAsInt(t, group.Id()), "slug": "tf24-t-" + suffix, "name": "IR24 team " + suffix,
	})
	teamID := testAccIDAsInt(t, team.Id())

	for _, source := range []string{"opensearch", "kibana"} {
		t.Run(source, func(t *testing.T) {
			resource := resourceRoute()
			values := map[string]interface{}{"team_id": teamID, "name": source + " " + suffix, "source": source, "group_by": []interface{}{"alertname"}}
			data := testAccCreateResource(t, ctx, resource, config, values)
			testAccRequireNoDiags(t, resource.ReadWithoutTimeout(ctx, data, config))
			assertConfigPlan(t, resource, data, values, "source", false)
			imported := testAccImportReadResource(t, ctx, resource, config, data, nil, map[string]interface{}{"source": source, "integration_config_json": "{}"})
			assertConfigPlan(t, resource, imported, values, "source", false)
			values["name"] = source + " updated " + suffix
			testAccUpdateResource(t, ctx, resource, data, config, map[string]interface{}{"name": values["name"]}, map[string]interface{}{"name": values["name"], "source": source})
			assertConfigPlan(t, resource, data, values, "name", false)
		})
	}

	t.Run("email", func(t *testing.T) {
		resource := resourceChannel()
		values := map[string]interface{}{"team_id": teamID, "name": "IR24 email " + suffix, "channel_type": "email", "config_json": `{}`}
		data := testAccCreateResource(t, ctx, resource, config, values)
		for _, raw := range []string{
			`{}`,
			`{"recipient_mode":"assignee"}`,
			`{"recipient_mode":"fallback","recipient_email":"  Backup@EXAMPLE.COM  "}`,
			`{"recipient_mode":"fixed","recipient_email":"Other@example.com"}`,
			`{"recipient_mode":"fixed","recipient_email":"e\u0301@xn--bcher-kva.de"}`,
			`{"recipient_mode":"fixed","recipient_email":"SUPPORT@EXAMPLE.COM"}`,
			`{"recipient_mode":"fixed","recipient_email":"Team@FAẞ.DE"}`,
			`{"recipient_mode":"assignee","recipient_email":"old@example.com"}`,
		} {
			values["config_json"] = raw
			testAccUpdateResource(t, ctx, resource, data, config, map[string]interface{}{"config_json": raw}, nil)
			testAccRequireNoDiags(t, resource.ReadWithoutTimeout(ctx, data, config))
			assertConfigPlan(t, resource, data, values, "config_json", false)
			imported := testAccImportReadResource(t, ctx, resource, config, data, nil, nil)
			assertConfigPlan(t, resource, imported, values, "config_json", false)
		}
	})

	t.Run("reopen_rules", func(t *testing.T) {
		resource := resourceEventOrchestration()
		rules := func(seconds int) string {
			return fmt.Sprintf(`[{"name":"Reopen","actions":[{"type":"set_grouping","reopen_window_seconds":%d}]}]`, seconds)
		}
		values := map[string]interface{}{"group_id": testAccIDAsInt(t, group.Id()), "name": "IR24 reopen " + suffix, "compatibility_mode": "hybrid", "mode": "active", "rules_json": rules(300)}
		data := testAccCreateResource(t, ctx, resource, config, values)
		for _, seconds := range []int{300, 600, 0} {
			values["rules_json"] = rules(seconds)
			testAccUpdateResource(t, ctx, resource, data, config, map[string]interface{}{"rules_json": values["rules_json"]}, map[string]interface{}{"rules_json": normalizeEventOrchestrationRulesState(values["rules_json"])})
			testAccRequireNoDiags(t, resource.ReadWithoutTimeout(ctx, data, config))
			assertConfigPlan(t, resource, data, values, "rules_json", false)
			imported := testAccImportReadResource(t, ctx, resource, config, data, map[string]interface{}{"group_id": values["group_id"]}, map[string]interface{}{"rules_json": normalizeEventOrchestrationRulesState(values["rules_json"])})
			assertConfigPlan(t, resource, imported, values, "rules_json", false)
		}
	})
}

// Keep the CLI fixture in one place so the same cases exercise real Terraform
// validation, apply, refresh, import, mode changes, and destroy.
const testAcc24CLIConfig = `
variable "email_recipient_mode" { default = "fixed" }

resource "incidentrelay_channel" "email24" {
  team_id = incidentrelay_team.cli.id
  name = "CLI email 2.4"
  channel_type = "email"
  config_json = jsonencode({
    recipient_mode = var.email_recipient_mode
    recipient_email = "  Backup@EXAMPLE.COM  "
  })
}

resource "incidentrelay_route" "search24" {
  for_each = toset(["opensearch", "kibana"])
  team_id = incidentrelay_team.cli.id
  name = "CLI ${each.key} 2.4"
  source = each.key
}

resource "incidentrelay_event_orchestration" "reopen24" {
  group_id = incidentrelay_group.cli.id
  name = "CLI reopen 2.4"
  compatibility_mode = "hybrid"
  mode = "active"
  rules_json = jsonencode([{
    name = "Reopen"
    actions = [{ type = "set_grouping", reopen_window_seconds = 300 }]
  }])
}

output "email24_id" { value = incidentrelay_channel.email24.id }
output "opensearch24_id" { value = incidentrelay_route.search24["opensearch"].id }
output "reopen24_id" { value = incidentrelay_event_orchestration.reopen24.id }
`

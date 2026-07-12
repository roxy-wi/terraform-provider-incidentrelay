package incidentrelay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAccIncidentRelayVersionDataSource(t *testing.T) {
	config := testAccProviderConfig(t)
	dataSource := datasourceVersion()
	data := schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{})

	testAccRequireNoDiags(t, dataSource.ReadContext(context.Background(), data, config))

	if got := data.Id(); got != "version" {
		t.Fatalf("id = %q, want version", got)
	}
	if got := data.Get("service_version").(string); strings.TrimSpace(got) == "" {
		t.Fatal("service_version is empty")
	}
}

func TestAccIncidentRelayCoreResources(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupResource := resourceGroup()
	groupData := schema.TestResourceDataRaw(t, groupResource.Schema, map[string]interface{}{
		"slug":        "tfacc-" + suffix,
		"name":        "TF acc group " + suffix,
		"description": "Created by Terraform provider acceptance tests.",
		"active":      true,
	})
	testAccRequireNoDiags(t, groupResource.CreateWithoutTimeout(ctx, groupData, config))
	testAccCleanupResource(t, groupResource, groupData, config)

	teamResource := resourceTeam()
	teamData := schema.TestResourceDataRaw(t, teamResource.Schema, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfacc-team-" + suffix,
		"name":                       "TF acc team " + suffix,
		"description":                "Created by Terraform provider acceptance tests.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	testAccRequireNoDiags(t, teamResource.CreateWithoutTimeout(ctx, teamData, config))
	testAccCleanupResource(t, teamResource, teamData, config)

	serviceResource := resourceService()
	serviceData := schema.TestResourceDataRaw(t, serviceResource.Schema, map[string]interface{}{
		"team_id":      testAccIDAsInt(t, teamData.Id()),
		"slug":         "tfacc-service-" + suffix,
		"name":         "TF acc service " + suffix,
		"description":  "Created by Terraform provider acceptance tests.",
		"service_type": "api",
		"environment":  "testing",
		"criticality":  "medium",
		"tier":         "tier_3",
		"labels_json":  `{"managed_by":"terraform","test":"acceptance"}`,
		"metadata_json": `{
			"purpose": "terraform-provider-incidentrelay acceptance test"
		}`,
		"enabled": true,
	})
	testAccRequireNoDiags(t, serviceResource.CreateWithoutTimeout(ctx, serviceData, config))
	testAccCleanupResource(t, serviceResource, serviceData, config)

	groupLookup := datasourceGroup()
	groupLookupData := schema.TestResourceDataRaw(t, groupLookup.Schema, map[string]interface{}{
		"slug": groupData.Get("slug"),
	})
	testAccRequireNoDiags(t, groupLookup.ReadContext(ctx, groupLookupData, config))
	if groupLookupData.Id() != groupData.Id() {
		t.Fatalf("group lookup id = %q, want %q", groupLookupData.Id(), groupData.Id())
	}

	teamLookup := datasourceTeam()
	teamLookupData := schema.TestResourceDataRaw(t, teamLookup.Schema, map[string]interface{}{
		"group_id": testAccIDAsInt(t, groupData.Id()),
		"slug":     teamData.Get("slug"),
	})
	testAccRequireNoDiags(t, teamLookup.ReadContext(ctx, teamLookupData, config))
	if teamLookupData.Id() != teamData.Id() {
		t.Fatalf("team lookup id = %q, want %q", teamLookupData.Id(), teamData.Id())
	}

	serviceLookup := datasourceService()
	serviceLookupData := schema.TestResourceDataRaw(t, serviceLookup.Schema, map[string]interface{}{
		"team_id": testAccIDAsInt(t, teamData.Id()),
		"slug":    serviceData.Get("slug"),
	})
	testAccRequireNoDiags(t, serviceLookup.ReadContext(ctx, serviceLookupData, config))
	if serviceLookupData.Id() != serviceData.Id() {
		t.Fatalf("service lookup id = %q, want %q", serviceLookupData.Id(), serviceData.Id())
	}
}

func TestAccIncidentRelayOnCallRoutingResources(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        "tfacc-" + suffix,
		"name":        "Acc group " + suffix,
		"description": "Created by Terraform provider acceptance tests.",
		"active":      true,
	})

	teamData := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfacc-team-" + suffix,
		"name":                       "Acc team " + suffix,
		"description":                "Created by Terraform provider acceptance tests.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	teamID := testAccIDAsInt(t, teamData.Id())

	userData := testAccCreateResource(t, ctx, resourceAdminUser(), config, map[string]interface{}{
		"username":     "tfacc-" + suffix,
		"display_name": "Acc user " + suffix,
		"email":        "tfacc-" + suffix + "@example.com",
		"password":     "Change-me-123!",
		"active":       true,
		"is_admin":     false,
		"group_id":     testAccIDAsInt(t, groupData.Id()),
		"group_role":   "viewer",
	})
	userID := testAccIDAsInt(t, userData.Id())

	teamMembershipData := testAccCreateResource(t, ctx, resourceTeamMembership(), config, map[string]interface{}{
		"team_id": teamID,
		"user_id": userID,
		"role":    "responder",
		"active":  true,
	})
	testAccImportReadResource(t, ctx, resourceTeamMembership(), config, teamMembershipData, map[string]interface{}{
		"team_id": teamID,
	}, map[string]interface{}{
		"user_id": userID,
		"role":    "responder",
		"active":  true,
	})

	channelData := testAccCreateResource(t, ctx, resourceChannel(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Acc email " + suffix,
		"channel_type": "email",
		"config_json":  `{"notify_on_severities":["critical","high"]}`,
		"enabled":      true,
	})
	channelID := testAccIDAsInt(t, channelData.Id())

	rotationData := testAccCreateResource(t, ctx, resourceRotation(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Acc rotation " + suffix,
		"description":               "Acceptance rotation.",
		"start_at":                  "2026-07-13T09:00:00",
		"rotation_type":             "weekly",
		"interval_value":            1,
		"interval_unit":             "weeks",
		"handoff_time":              "09:00",
		"handoff_weekday":           0,
		"timezone":                  "UTC",
		"reminder_interval_seconds": 300,
		"enabled":                   true,
		"add_team_members":          false,
	})
	rotationID := testAccIDAsInt(t, rotationData.Id())

	layerData := testAccCreateResource(t, ctx, resourceRotationLayer(), config, map[string]interface{}{
		"rotation_id":     rotationID,
		"name":            "Acc layer " + suffix,
		"description":     "Acceptance layer.",
		"priority":        100,
		"start_at":        "2026-07-13T09:00:00",
		"rotation_type":   "weekly",
		"interval_value":  1,
		"interval_unit":   "weeks",
		"handoff_time":    "09:00",
		"handoff_weekday": 0,
		"timezone":        "UTC",
		"enabled":         true,
	})
	layerID := testAccIDAsInt(t, layerData.Id())
	testAccImportReadResource(t, ctx, resourceRotationLayer(), config, layerData, map[string]interface{}{
		"rotation_id": rotationID,
	}, map[string]interface{}{
		"name":     "Acc layer " + suffix,
		"priority": 100,
		"enabled":  true,
	})
	testAccUpdateResource(t, ctx, resourceRotationLayer(), layerData, config, map[string]interface{}{
		"description": "Updated acceptance layer.",
		"priority":    200,
	}, map[string]interface{}{
		"description": "Updated acceptance layer.",
		"priority":    200,
		"enabled":     true,
	})

	layerMemberData := testAccCreateResource(t, ctx, resourceRotationLayerMember(), config, map[string]interface{}{
		"layer_id":  layerID,
		"user_id":   userID,
		"position":  0,
		"active":    true,
		"starts_at": "2026-07-13T09:00:00",
	})
	testAccImportReadResource(t, ctx, resourceRotationLayerMember(), config, layerMemberData, map[string]interface{}{
		"layer_id": layerID,
	}, map[string]interface{}{
		"user_id":  userID,
		"position": 0,
		"active":   true,
	})

	overrideData := testAccCreateResource(t, ctx, resourceRotationOverride(), config, map[string]interface{}{
		"rotation_id": rotationID,
		"user_id":     userID,
		"starts_at":   "2026-07-14T09:00:00",
		"ends_at":     "2026-07-14T11:00:00",
		"reason":      "Terraform acceptance override.",
	})
	if got := overrideData.Get("username").(string); got != userData.Get("username").(string) {
		t.Fatalf("override username = %q, want %q", got, userData.Get("username").(string))
	}
	testAccImportReadResource(t, ctx, resourceRotationOverride(), config, overrideData, map[string]interface{}{
		"rotation_id": rotationID,
	}, map[string]interface{}{
		"user_id": userID,
		"reason":  "Terraform acceptance override.",
	})

	escalationPolicyData := testAccCreateResource(t, ctx, resourceEscalationPolicy(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Acc escalation " + suffix,
		"description":  "Acceptance escalation policy.",
		"enabled":      true,
		"repeat_count": 1,
	})
	escalationPolicyID := testAccIDAsInt(t, escalationPolicyData.Id())

	escalationPolicyRuleData := testAccCreateResource(t, ctx, resourceEscalationPolicyRule(), config, map[string]interface{}{
		"policy_id":     escalationPolicyID,
		"position":      1,
		"delay_seconds": 300,
		"target_type":   "rotation",
		"target_id":     rotationID,
		"enabled":       true,
	})
	testAccImportReadResource(t, ctx, resourceEscalationPolicyRule(), config, escalationPolicyRuleData, map[string]interface{}{
		"policy_id": escalationPolicyID,
	}, map[string]interface{}{
		"position":      1,
		"delay_seconds": 300,
		"target_type":   "rotation",
		"target_id":     rotationID,
		"enabled":       true,
	})
	testAccUpdateResource(t, ctx, resourceEscalationPolicyRule(), escalationPolicyRuleData, config, map[string]interface{}{
		"delay_seconds": 600,
		"enabled":       false,
	}, map[string]interface{}{
		"position":      1,
		"delay_seconds": 600,
		"target_type":   "rotation",
		"target_id":     rotationID,
		"enabled":       false,
	})

	notificationPolicyData := testAccCreateResource(t, ctx, resourceNotificationPolicy(), config, map[string]interface{}{
		"team_id":     teamID,
		"name":        "Acc notify " + suffix,
		"description": "Acceptance notification policy.",
		"enabled":     true,
	})
	notificationPolicyID := testAccIDAsInt(t, notificationPolicyData.Id())

	notificationPolicyRuleData := testAccCreateResource(t, ctx, resourceNotificationPolicyRule(), config, map[string]interface{}{
		"policy_id":         notificationPolicyID,
		"name":              "Acc notif rule " + suffix,
		"description":       "Acceptance notification rule.",
		"position":          1,
		"event_types":       testAccStringSet("notification", "reminder", "escalation"),
		"matchers_json":     `{"labels":{"severity":"critical","service":"api"}}`,
		"channel_ids":       testAccIntSet(channelID),
		"continue_matching": true,
		"enabled":           true,
	})
	testAccImportReadResource(t, ctx, resourceNotificationPolicyRule(), config, notificationPolicyRuleData, map[string]interface{}{
		"policy_id": notificationPolicyID,
	}, map[string]interface{}{
		"name":              "Acc notif rule " + suffix,
		"position":          1,
		"continue_matching": true,
		"enabled":           true,
	})
	testAccUpdateResource(t, ctx, resourceNotificationPolicyRule(), notificationPolicyRuleData, config, map[string]interface{}{
		"description":       "Updated notification rule.",
		"continue_matching": false,
		"enabled":           false,
	}, map[string]interface{}{
		"description":       "Updated notification rule.",
		"continue_matching": false,
		"enabled":           false,
	})

	serviceData := testAccCreateResource(t, ctx, resourceService(), config, map[string]interface{}{
		"team_id":                      teamID,
		"slug":                         "tfacc-svc-" + suffix,
		"name":                         "Acc service " + suffix,
		"description":                  "Acceptance service.",
		"service_type":                 "api",
		"environment":                  "testing",
		"criticality":                  "high",
		"tier":                         "tier_2",
		"status":                       "operational",
		"status_source":                "manual",
		"default_rotation_id":          rotationID,
		"default_escalation_policy_id": escalationPolicyID,
		"notification_policy_id":       notificationPolicyID,
		"labels_json":                  `{"managed_by":"terraform","tier":"api"}`,
		"tags":                         testAccStringSet("terraform", "acceptance"),
		"metadata_json":                `{"purpose":"acceptance"}`,
		"enabled":                      true,
		"public":                       false,
	})
	serviceID := testAccIDAsInt(t, serviceData.Id())
	testAccUpdateResource(t, ctx, resourceService(), serviceData, config, map[string]interface{}{
		"description":    "Updated acceptance service.",
		"criticality":    "critical",
		"status":         "degraded",
		"status_message": "Updated by acceptance test.",
		"public_name":    "Acc public " + suffix,
	}, map[string]interface{}{
		"description":    "Updated acceptance service.",
		"criticality":    "critical",
		"status":         "degraded",
		"status_message": "Updated by acceptance test.",
		"public_name":    "Acc public " + suffix,
	})

	routeData := testAccCreateResource(t, ctx, resourceRoute(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Acc route " + suffix,
		"source":                    "alertmanager",
		"service_id":                serviceID,
		"escalation_policy_id":      escalationPolicyID,
		"channel_ids":               testAccIntSet(channelID),
		"notification_channel_mode": "service_policy_plus_route",
		"matchers_json":             `{"labels":{"team":"platform","severity":"critical"}}`,
		"integration_config_json":   `{}`,
		"group_by":                  testAccStringSet("alertname", "instance"),
		"enabled":                   true,
	})
	if got := routeData.Get("escalation_mode").(string); got != "policy" {
		t.Fatalf("route escalation_mode = %q, want policy", got)
	}
	routeID := testAccIDAsInt(t, routeData.Id())
	testAccUpdateResource(t, ctx, resourceRoute(), routeData, config, map[string]interface{}{
		"name":                      "Acc route upd " + suffix,
		"rotation_id":               rotationID,
		"notification_channel_mode": "service_policy",
		"enabled":                   true,
	}, map[string]interface{}{
		"name":                      "Acc route upd " + suffix,
		"rotation_id":               rotationID,
		"notification_channel_mode": "service_policy",
		"enabled":                   true,
		"escalation_mode":           "policy",
	})

	serviceMatchRuleData := testAccCreateResource(t, ctx, resourceServiceMatchRule(), config, map[string]interface{}{
		"team_id":       teamID,
		"service_id":    serviceID,
		"route_id":      routeID,
		"position":      10,
		"name":          "Acc match " + suffix,
		"description":   "Acceptance service match rule.",
		"matchers_json": `{"labels":{"service":"api","environment":"testing"}}`,
		"enabled":       true,
	})
	testAccImportReadResource(t, ctx, resourceServiceMatchRule(), config, serviceMatchRuleData, map[string]interface{}{
		"service_id": serviceID,
	}, map[string]interface{}{
		"team_id":    teamID,
		"service_id": serviceID,
		"route_id":   routeID,
		"name":       "Acc match " + suffix,
		"enabled":    true,
	})
}

func TestAccIncidentRelayTerraformCLI(t *testing.T) {
	config := testAccProviderConfig(t)

	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform CLI is required for this acceptance test")
	}

	ctx := context.Background()
	suffix := testAccSuffix()

	groupSlug := "tfcli-g-" + suffix
	groupName := "CLI group " + suffix
	groupDescription := "Terraform CLI acceptance group."
	groupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        groupSlug,
		"name":        groupName,
		"description": groupDescription,
		"active":      true,
	})

	workDir := t.TempDir()
	repoRoot := testAccRepoRoot(t)
	providerDir := filepath.Join(t.TempDir(), "provider")
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		t.Fatalf("create provider override directory: %v", err)
	}

	providerBinary := filepath.Join(providerDir, "terraform-provider-incidentrelay")
	if runtime.GOOS == "windows" {
		providerBinary += ".exe"
	}
	testAccRunCommand(t, repoRoot, nil, "go", "build", "-o", providerBinary, ".")

	cliConfigPath := filepath.Join(workDir, "terraform.rc")
	cliConfig := fmt.Sprintf(`provider_installation {
  dev_overrides {
    "registry.terraform.io/roxy-wi/incidentrelay" = %s
  }
  direct {}
}
`, strconv.Quote(providerDir))
	if err := os.WriteFile(cliConfigPath, []byte(cliConfig), 0o600); err != nil {
		t.Fatalf("write Terraform CLI config: %v", err)
	}

	mainTF := fmt.Sprintf(`terraform {
  required_version = ">= 1.4.0"

  required_providers {
    incidentrelay = {
      source = "roxy-wi/incidentrelay"
    }
  }
}

provider "incidentrelay" {}

resource "incidentrelay_group" "cli" {
  slug        = %s
  name        = %s
  description = %s
  active      = true
}

resource "incidentrelay_team" "cli" {
  group_id                   = incidentrelay_group.cli.id
  slug                       = %s
  name                       = %s
  description                = "Terraform CLI acceptance team."
  escalation_enabled         = true
  escalation_after_reminders = 2
  active                     = true
}

resource "incidentrelay_service" "cli" {
  team_id        = incidentrelay_team.cli.id
  slug           = %s
  name           = %s
  description    = "Terraform CLI acceptance service."
  service_type   = "api"
  environment    = "testing"
  criticality    = "high"
  tier           = "tier_2"
  status         = "operational"
  status_source  = "manual"
  status_message = "Managed by Terraform CLI acceptance."
  labels_json    = jsonencode({ managed_by = "terraform", test = "cli" })
  tags           = ["terraform", "cli"]
  metadata_json  = jsonencode({ purpose = "cli-acceptance" })
  enabled        = true
  public         = false
  public_name    = %s
  public_description = "Terraform CLI acceptance public service."
}

output "team_id" {
  value = incidentrelay_team.cli.id
}

output "service_id" {
  value = incidentrelay_service.cli.id
}
`,
		strconv.Quote(groupSlug),
		strconv.Quote(groupName),
		strconv.Quote(groupDescription),
		strconv.Quote("tfcli-t-"+suffix),
		strconv.Quote("CLI team "+suffix),
		strconv.Quote("tfcli-svc-"+suffix),
		strconv.Quote("CLI service "+suffix),
		strconv.Quote("CLI public "+suffix),
	)
	if err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte(mainTF), 0o600); err != nil {
		t.Fatalf("write Terraform config: %v", err)
	}

	terraformEnv := append(os.Environ(),
		"TF_CLI_CONFIG_FILE="+cliConfigPath,
		"TF_IN_AUTOMATION=1",
		"TF_INPUT=0",
	)

	t.Cleanup(func() {
		result := testAccRunCommandResult(workDir, terraformEnv, terraformPath, "destroy", "-auto-approve", "-input=false", "-no-color")
		if result.ExitCode != 0 {
			t.Errorf("terraform destroy failed with exit code %d:\n%s", result.ExitCode, result.Output)
		}
	})

	testAccRunCommand(t, workDir, terraformEnv, terraformPath, "validate", "-no-color")
	testAccRunCommand(t, workDir, terraformEnv, terraformPath, "import", "-input=false", "-no-color", "incidentrelay_group.cli", groupData.Id())
	testAccRunCommand(t, workDir, terraformEnv, terraformPath, "apply", "-auto-approve", "-input=false", "-no-color")

	stateList := testAccRunCommand(t, workDir, terraformEnv, terraformPath, "state", "list")
	for _, resource := range []string{
		"incidentrelay_group.cli",
		"incidentrelay_team.cli",
		"incidentrelay_service.cli",
	} {
		if !strings.Contains(stateList, resource) {
			t.Fatalf("terraform state does not contain %s:\n%s", resource, stateList)
		}
	}

	for _, outputName := range []string{"team_id", "service_id"} {
		output := strings.TrimSpace(testAccRunCommand(t, workDir, terraformEnv, terraformPath, "output", "-raw", outputName))
		if output == "" {
			t.Fatalf("terraform output %s is empty", outputName)
		}
	}

	plan := testAccRunCommandResult(workDir, terraformEnv, terraformPath, "plan", "-input=false", "-detailed-exitcode", "-no-color")
	if plan.ExitCode != 0 {
		t.Fatalf("terraform plan after apply returned exit code %d, want 0:\n%s", plan.ExitCode, plan.Output)
	}
}

func testAccProviderConfig(t *testing.T) *Config {
	t.Helper()

	if os.Getenv("INCIDENTRELAY_ACC") != "1" {
		t.Skip("set INCIDENTRELAY_ACC=1 to run acceptance tests")
	}

	baseURL := os.Getenv("INCIDENTRELAY_BASE_URL")
	if strings.TrimSpace(baseURL) == "" {
		t.Fatal("INCIDENTRELAY_BASE_URL is required for acceptance tests")
	}

	token := os.Getenv("INCIDENTRELAY_TOKEN")
	username := os.Getenv("INCIDENTRELAY_USERNAME")
	password := os.Getenv("INCIDENTRELAY_PASSWORD")
	if strings.TrimSpace(token) == "" && (strings.TrimSpace(username) == "" || password == "") {
		t.Fatal("set INCIDENTRELAY_TOKEN or INCIDENTRELAY_USERNAME/INCIDENTRELAY_PASSWORD for acceptance tests")
	}

	provider := Provider()
	data := schema.TestResourceDataRaw(t, provider.Schema, map[string]interface{}{
		ProviderBaseURL:  baseURL,
		ProviderToken:    token,
		ProviderUsername: username,
		ProviderPassword: password,
	})

	config, diags := providerConfigure(context.Background(), data, "acceptance")
	testAccRequireNoDiags(t, diags)

	providerConfig, ok := config.(*Config)
	if !ok {
		t.Fatalf("provider config type = %T, want *Config", config)
	}
	return providerConfig
}

func testAccCreateResource(t *testing.T, ctx context.Context, resource *schema.Resource, config *Config, values map[string]interface{}) *schema.ResourceData {
	t.Helper()

	data := schema.TestResourceDataRaw(t, resource.Schema, values)
	testAccRequireNoDiags(t, resource.CreateWithoutTimeout(ctx, data, config))
	testAccCleanupResource(t, resource, data, config)
	return data
}

func testAccImportReadResource(t *testing.T, ctx context.Context, resource *schema.Resource, config *Config, source *schema.ResourceData, configValues map[string]interface{}, checks map[string]interface{}) *schema.ResourceData {
	t.Helper()

	data := schema.TestResourceDataRaw(t, resource.Schema, configValues)
	data.SetId(source.Id())

	if resource.Importer == nil || resource.Importer.StateContext == nil {
		t.Fatal("resource has no importer")
	}

	imported, err := resource.Importer.StateContext(ctx, data, config)
	if err != nil {
		t.Fatalf("import state failed: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("import returned %d states, want 1", len(imported))
	}

	data = imported[0]
	testAccRequireNoDiags(t, resource.ReadWithoutTimeout(ctx, data, config))

	if got, want := data.Id(), source.Id(); got != want {
		t.Fatalf("imported id = %q, want %q", got, want)
	}

	for field, want := range checks {
		if got := data.Get(field); !reflect.DeepEqual(got, want) {
			t.Fatalf("imported %s = %#v, want %#v", field, got, want)
		}
	}

	return data
}

func testAccUpdateResource(t *testing.T, ctx context.Context, resource *schema.Resource, data *schema.ResourceData, config *Config, updates map[string]interface{}, checks map[string]interface{}) {
	t.Helper()

	id := data.Id()
	for field, value := range updates {
		if err := data.Set(field, value); err != nil {
			t.Fatalf("set %s: %v", field, err)
		}
	}

	testAccRequireNoDiags(t, resource.UpdateWithoutTimeout(ctx, data, config))

	if got := data.Id(); got != id {
		t.Fatalf("id after update = %q, want %q", got, id)
	}

	for field, want := range checks {
		if got := data.Get(field); !reflect.DeepEqual(got, want) {
			t.Fatalf("updated %s = %#v, want %#v", field, got, want)
		}
	}
}

func testAccCleanupResource(t *testing.T, resource *schema.Resource, data *schema.ResourceData, config *Config) {
	t.Helper()

	t.Cleanup(func() {
		if data.Id() == "" {
			return
		}
		if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
			t.Errorf("cleanup %s failed: %v", data.Id(), diags)
		}
	})
}

func testAccRequireNoDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func testAccIDAsInt(t *testing.T, id string) int {
	t.Helper()

	value, ok := intFromInterface(id)
	if !ok {
		t.Fatalf("cannot convert id %q to int", id)
	}
	return value
}

func testAccSuffix() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func testAccIntSet(values ...int) []interface{} {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return items
}

func testAccStringSet(values ...string) []interface{} {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return items
}

func testAccRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	return filepath.Dir(filepath.Dir(file))
}

type testAccCommandResult struct {
	Output   string
	ExitCode int
}

func testAccRunCommand(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()

	result := testAccRunCommandResult(dir, env, name, args...)
	if result.ExitCode != 0 {
		t.Fatalf("%s %s failed with exit code %d:\n%s", name, strings.Join(args, " "), result.ExitCode, result.Output)
	}
	return result.Output
}

func testAccRunCommandResult(dir string, env []string, name string, args ...string) testAccCommandResult {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	if err == nil {
		return testAccCommandResult{Output: output.String(), ExitCode: 0}
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return testAccCommandResult{Output: output.String(), ExitCode: exitErr.ExitCode()}
	}

	return testAccCommandResult{Output: output.String() + err.Error(), ExitCode: -1}
}

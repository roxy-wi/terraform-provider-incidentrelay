package incidentrelay

import (
	"context"
	"fmt"
	"os"
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

	testAccCreateResource(t, ctx, resourceTeamMembership(), config, map[string]interface{}{
		"team_id": teamID,
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

	testAccCreateResource(t, ctx, resourceRotationLayerMember(), config, map[string]interface{}{
		"layer_id":  layerID,
		"user_id":   userID,
		"position":  0,
		"active":    true,
		"starts_at": "2026-07-13T09:00:00",
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

	escalationPolicyData := testAccCreateResource(t, ctx, resourceEscalationPolicy(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Acc escalation " + suffix,
		"description":  "Acceptance escalation policy.",
		"enabled":      true,
		"repeat_count": 1,
	})
	escalationPolicyID := testAccIDAsInt(t, escalationPolicyData.Id())

	testAccCreateResource(t, ctx, resourceEscalationPolicyRule(), config, map[string]interface{}{
		"policy_id":     escalationPolicyID,
		"position":      1,
		"delay_seconds": 300,
		"target_type":   "rotation",
		"target_id":     rotationID,
		"enabled":       true,
	})

	notificationPolicyData := testAccCreateResource(t, ctx, resourceNotificationPolicy(), config, map[string]interface{}{
		"team_id":     teamID,
		"name":        "Acc notify " + suffix,
		"description": "Acceptance notification policy.",
		"enabled":     true,
	})
	notificationPolicyID := testAccIDAsInt(t, notificationPolicyData.Id())

	testAccCreateResource(t, ctx, resourceNotificationPolicyRule(), config, map[string]interface{}{
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

	testAccCreateResource(t, ctx, resourceServiceMatchRule(), config, map[string]interface{}{
		"team_id":       teamID,
		"service_id":    serviceID,
		"route_id":      routeID,
		"position":      10,
		"name":          "Acc match " + suffix,
		"description":   "Acceptance service match rule.",
		"matchers_json": `{"labels":{"service":"api","environment":"testing"}}`,
		"enabled":       true,
	})
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

func testAccIntSet(values ...int) *schema.Set {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return schema.NewSet(schema.HashInt, items)
}

func testAccStringSet(values ...string) *schema.Set {
	items := make([]interface{}, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return schema.NewSet(schema.HashString, items)
}

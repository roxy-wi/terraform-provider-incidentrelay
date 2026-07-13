package incidentrelay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func TestAccIncidentRelayRouteBehavior(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        "tfroute-g-" + suffix,
		"name":        "Route group " + suffix,
		"description": "Route behavior acceptance group.",
		"active":      true,
	})
	teamData := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfroute-t-" + suffix,
		"name":                       "Route team " + suffix,
		"description":                "Route behavior acceptance team.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	teamID := testAccIDAsInt(t, teamData.Id())

	channelData := testAccCreateResource(t, ctx, resourceChannel(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Route email " + suffix,
		"channel_type": "email",
		"config_json":  `{"notify_on_severities":["critical","warning"]}`,
		"enabled":      true,
	})
	channelID := testAccIDAsInt(t, channelData.Id())

	rotationData := testAccCreateResource(t, ctx, resourceRotation(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Route rotation " + suffix,
		"description":               "Route behavior rotation.",
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

	escalationPolicyData := testAccCreateResource(t, ctx, resourceEscalationPolicy(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Route policy " + suffix,
		"description":  "Route behavior policy.",
		"enabled":      true,
		"repeat_count": 1,
	})
	escalationPolicyID := testAccIDAsInt(t, escalationPolicyData.Id())

	serviceData := testAccCreateResource(t, ctx, resourceService(), config, map[string]interface{}{
		"team_id":       teamID,
		"slug":          "tfroute-s-" + suffix,
		"name":          "Route svc " + suffix,
		"description":   "Route behavior service.",
		"service_type":  "api",
		"environment":   "testing",
		"criticality":   "high",
		"tier":          "tier_2",
		"status":        "operational",
		"status_source": "manual",
		"labels_json":   `{"service":"payments","managed_by":"terraform"}`,
		"tags":          testAccStringSet("terraform", "routes"),
		"metadata_json": `{"purpose":"route-behavior"}`,
		"enabled":       true,
		"public":        false,
	})
	serviceID := testAccIDAsInt(t, serviceData.Id())

	policyRouteMatchers := `{
		"fields": {"payload.custom.cluster": "core"},
		"labels": {
			"environment": "production",
			"severity": ["critical", "warning"]
		}
	}`
	policyRouteData := testAccCreateResource(t, ctx, resourceRoute(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Route policy " + suffix,
		"source":                    "alertmanager",
		"service_id":                serviceID,
		"escalation_policy_id":      escalationPolicyID,
		"channel_ids":               testAccIntSet(channelID),
		"notification_channel_mode": "service_policy_plus_route",
		"matchers_json":             policyRouteMatchers,
		"integration_config_json":   `{}`,
		"group_by":                  testAccStringSet("alertname", "labels.instance", "service"),
		"enabled":                   true,
	})
	policyRouteID := testAccIDAsInt(t, policyRouteData.Id())
	if got := policyRouteData.Get("escalation_mode").(string); got != "policy" {
		t.Fatalf("policy route escalation_mode = %q, want policy", got)
	}
	if got := policyRouteData.Get("service_name").(string); got != serviceData.Get("name").(string) {
		t.Fatalf("policy route service_name = %q, want %q", got, serviceData.Get("name").(string))
	}
	if got := policyRouteData.Get("has_intake_token").(bool); !got {
		t.Fatal("policy route has_intake_token = false, want true")
	}
	assertSetElements(t, policyRouteData.Get("group_by").(*schema.Set), testAccStringSet("alertname", "labels.instance", "service"))

	policyRouteAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/routes/%d", policyRouteID))
	testAccRequireAPIStringField(t, policyRouteAPI, "escalation_mode", "policy")
	testAccRequireAPIIntField(t, policyRouteAPI, "escalation_policy_id", escalationPolicyID)
	testAccRequireAPIIntField(t, policyRouteAPI, "service_id", serviceID)
	testAccRequireAPIStringField(t, policyRouteAPI, "notification_channel_mode", "service_policy_plus_route")
	testAccRequireAPIJSONField(t, policyRouteAPI, "matchers", policyRouteMatchers)
	testAccRequireAPIStringListField(t, policyRouteAPI, "group_by", "alertname", "labels.instance", "service")
	testAccRequireAPIChannelIDs(t, policyRouteAPI, channelID)

	rotationRouteMatchers := `{
		"labels": {
			"environment": "staging",
			"service": "payments"
		},
		"title_regex": "^Payment.*"
	}`
	rotationRouteData := testAccCreateResource(t, ctx, resourceRoute(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Route rotation " + suffix,
		"source":                    "alertmanager",
		"rotation_id":               rotationID,
		"service_id":                serviceID,
		"notification_channel_mode": "route_only",
		"matchers_json":             rotationRouteMatchers,
		"integration_config_json":   `{}`,
		"group_by":                  testAccStringSet("alertname", "route_id"),
		"enabled":                   true,
	})
	rotationRouteID := testAccIDAsInt(t, rotationRouteData.Id())
	if got := rotationRouteData.Get("escalation_mode").(string); got != "rotation" {
		t.Fatalf("rotation route escalation_mode = %q, want rotation", got)
	}

	rotationRouteAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/routes/%d", rotationRouteID))
	testAccRequireAPIStringField(t, rotationRouteAPI, "escalation_mode", "rotation")
	testAccRequireAPIIntField(t, rotationRouteAPI, "rotation_id", rotationID)
	testAccRequireAPINilField(t, rotationRouteAPI, "escalation_policy_id")
	testAccRequireAPIStringField(t, rotationRouteAPI, "notification_channel_mode", "route_only")
	testAccRequireAPIJSONField(t, rotationRouteAPI, "matchers", rotationRouteMatchers)
	testAccRequireAPIStringListField(t, rotationRouteAPI, "group_by", "alertname", "route_id")

	serviceMatchRuleData := testAccCreateResource(t, ctx, resourceServiceMatchRule(), config, map[string]interface{}{
		"team_id":       teamID,
		"service_id":    serviceID,
		"route_id":      policyRouteID,
		"position":      1,
		"name":          "Route rule " + suffix,
		"description":   "Route-scoped service match rule.",
		"matchers_json": `{"labels":{"service":"payments","environment":"production"}}`,
		"enabled":       true,
	})
	if got := serviceMatchRuleData.Get("route_name").(string); got != policyRouteData.Get("name").(string) {
		t.Fatalf("service match rule route_name = %q, want %q", got, policyRouteData.Get("name").(string))
	}
	testAccUpdateResource(t, ctx, resourceServiceMatchRule(), serviceMatchRuleData, config, map[string]interface{}{
		"route_id":      rotationRouteID,
		"position":      2,
		"matchers_json": `{"labels":{"service":"payments","environment":"staging"}}`,
	}, map[string]interface{}{
		"route_id": rotationRouteID,
		"position": 2,
		"enabled":  true,
	})
	if got := serviceMatchRuleData.Get("route_name").(string); got != rotationRouteData.Get("name").(string) {
		t.Fatalf("updated service match rule route_name = %q, want %q", got, rotationRouteData.Get("name").(string))
	}

	updatedRouteMatchers := `{
		"labels": {
			"environment": {"not": "production"},
			"service": "payments"
		}
	}`
	testAccUpdateResource(t, ctx, resourceRoute(), rotationRouteData, config, map[string]interface{}{
		"name":                      "Route rot upd " + suffix,
		"notification_channel_mode": "service_policy",
		"matchers_json":             updatedRouteMatchers,
		"group_by":                  testAccStringSet("route_id", "source"),
		"enabled":                   false,
	}, map[string]interface{}{
		"name":                      "Route rot upd " + suffix,
		"notification_channel_mode": "service_policy",
		"matchers_json":             `{"labels":{"environment":{"not":"production"},"service":"payments"}}`,
		"enabled":                   false,
		"escalation_mode":           "rotation",
	})
	assertSetElements(t, rotationRouteData.Get("group_by").(*schema.Set), testAccStringSet("route_id", "source"))

	updatedRouteAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/routes/%d", rotationRouteID))
	testAccRequireAPIStringField(t, updatedRouteAPI, "escalation_mode", "rotation")
	testAccRequireAPIIntField(t, updatedRouteAPI, "rotation_id", rotationID)
	testAccRequireAPINilField(t, updatedRouteAPI, "escalation_policy_id")
	testAccRequireAPIStringField(t, updatedRouteAPI, "notification_channel_mode", "service_policy")
	testAccRequireAPIBoolField(t, updatedRouteAPI, "enabled", false)
	testAccRequireAPIJSONField(t, updatedRouteAPI, "matchers", updatedRouteMatchers)
	testAccRequireAPIStringListField(t, updatedRouteAPI, "group_by", "route_id", "source")
}

func TestAccIncidentRelayOptionalLinkClearing(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        "tfclear-g-" + suffix,
		"name":        "Clear group " + suffix,
		"description": "Optional link clearing group.",
		"active":      true,
	})
	teamData := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfclear-t-" + suffix,
		"name":                       "Clear team " + suffix,
		"description":                "Optional link clearing team.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	teamID := testAccIDAsInt(t, teamData.Id())

	channelData := testAccCreateResource(t, ctx, resourceChannel(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Clear email " + suffix,
		"channel_type": "email",
		"config_json":  `{"notify_on_severities":["critical"]}`,
		"enabled":      true,
	})
	channelID := testAccIDAsInt(t, channelData.Id())

	rotationData := testAccCreateResource(t, ctx, resourceRotation(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Clear rotation " + suffix,
		"description":               "Optional link clearing rotation.",
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

	escalationPolicyData := testAccCreateResource(t, ctx, resourceEscalationPolicy(), config, map[string]interface{}{
		"team_id":      teamID,
		"name":         "Clear policy " + suffix,
		"description":  "Optional link clearing policy.",
		"enabled":      true,
		"repeat_count": 0,
	})
	escalationPolicyID := testAccIDAsInt(t, escalationPolicyData.Id())

	notificationPolicyData := testAccCreateResource(t, ctx, resourceNotificationPolicy(), config, map[string]interface{}{
		"team_id":     teamID,
		"name":        "Clear notif " + suffix,
		"description": "Optional link clearing notification policy.",
		"enabled":     true,
	})
	notificationPolicyID := testAccIDAsInt(t, notificationPolicyData.Id())

	serviceData := testAccCreateResource(t, ctx, resourceService(), config, map[string]interface{}{
		"team_id":                      teamID,
		"slug":                         "tfclear-s-" + suffix,
		"name":                         "Clear service " + suffix,
		"description":                  "Optional link clearing service.",
		"service_type":                 "api",
		"environment":                  "testing",
		"criticality":                  "high",
		"tier":                         "tier_2",
		"status":                       "operational",
		"status_source":                "manual",
		"default_rotation_id":          rotationID,
		"default_escalation_policy_id": escalationPolicyID,
		"notification_policy_id":       notificationPolicyID,
		"labels_json":                  `{"managed_by":"terraform","test":"clear"}`,
		"tags":                         testAccStringSet("terraform", "clear"),
		"metadata_json":                `{"purpose":"optional-link-clearing"}`,
		"enabled":                      true,
		"public":                       false,
	})
	serviceID := testAccIDAsInt(t, serviceData.Id())

	serviceAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/services/%d", serviceID))
	testAccRequireAPIIntField(t, serviceAPI, "default_rotation_id", rotationID)
	testAccRequireAPIIntField(t, serviceAPI, "default_escalation_policy_id", escalationPolicyID)
	testAccRequireAPIIntField(t, serviceAPI, "notification_policy_id", notificationPolicyID)

	testAccUpdateResource(t, ctx, resourceService(), serviceData, config, map[string]interface{}{
		"default_rotation_id":          nil,
		"default_escalation_policy_id": nil,
		"notification_policy_id":       nil,
	}, map[string]interface{}{
		"default_rotation_id":          0,
		"default_escalation_policy_id": 0,
		"notification_policy_id":       0,
	})

	clearedServiceAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/services/%d", serviceID))
	testAccRequireAPINilField(t, clearedServiceAPI, "default_rotation_id")
	testAccRequireAPINilField(t, clearedServiceAPI, "default_escalation_policy_id")
	testAccRequireAPINilField(t, clearedServiceAPI, "notification_policy_id")

	routeData := testAccCreateResource(t, ctx, resourceRoute(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Clear route " + suffix,
		"source":                    "alertmanager",
		"rotation_id":               rotationID,
		"service_id":                serviceID,
		"escalation_policy_id":      escalationPolicyID,
		"channel_ids":               testAccIntSet(channelID),
		"notification_channel_mode": "service_policy_plus_route",
		"matchers_json":             `{"labels":{"team":"terraform","severity":"critical"}}`,
		"integration_config_json":   `{}`,
		"group_by":                  testAccStringSet("alertname", "instance"),
		"enabled":                   true,
	})
	routeID := testAccIDAsInt(t, routeData.Id())

	routeAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/routes/%d", routeID))
	testAccRequireAPIIntField(t, routeAPI, "rotation_id", rotationID)
	testAccRequireAPIIntField(t, routeAPI, "service_id", serviceID)
	testAccRequireAPIIntField(t, routeAPI, "escalation_policy_id", escalationPolicyID)
	testAccRequireAPIChannelIDs(t, routeAPI, channelID)

	serviceMatchRuleData := testAccCreateResource(t, ctx, resourceServiceMatchRule(), config, map[string]interface{}{
		"team_id":       teamID,
		"service_id":    serviceID,
		"route_id":      routeID,
		"position":      1,
		"name":          "Clear match " + suffix,
		"description":   "Optional link clearing match rule.",
		"matchers_json": `{"labels":{"service":"api"}}`,
		"enabled":       true,
	})
	testAccUpdateResource(t, ctx, resourceServiceMatchRule(), serviceMatchRuleData, config, map[string]interface{}{
		"route_id": nil,
	}, map[string]interface{}{
		"route_id": 0,
		"enabled":  true,
	})

	testAccUpdateResource(t, ctx, resourceRoute(), routeData, config, map[string]interface{}{
		"rotation_id":               nil,
		"service_id":                nil,
		"escalation_policy_id":      nil,
		"channel_ids":               testAccIntSet(),
		"notification_channel_mode": "route_only",
	}, map[string]interface{}{
		"rotation_id":               0,
		"service_id":                0,
		"escalation_policy_id":      0,
		"notification_channel_mode": "route_only",
		"escalation_mode":           "rotation",
	})

	clearedRouteAPI := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/routes/%d", routeID))
	testAccRequireAPINilField(t, clearedRouteAPI, "rotation_id")
	testAccRequireAPINilField(t, clearedRouteAPI, "service_id")
	testAccRequireAPINilField(t, clearedRouteAPI, "escalation_policy_id")
	testAccRequireAPIChannelIDs(t, clearedRouteAPI)
	testAccRequireAPIStringField(t, clearedRouteAPI, "notification_channel_mode", "route_only")
	testAccRequireAPIStringField(t, clearedRouteAPI, "escalation_mode", "rotation")
}

func TestAccIncidentRelayNegativeValidation(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	invalidGroupData := schema.TestResourceDataRaw(t, resourceGroup().Schema, map[string]interface{}{
		"slug":        "BadSlug-" + suffix,
		"name":        "Invalid group " + suffix,
		"description": "This group should be rejected by API validation.",
		"active":      true,
	})
	testAccRequireDiagsContain(t, resourceGroup().CreateWithoutTimeout(ctx, invalidGroupData, config),
		"POST /api/groups returned 400",
		"validation_error",
		"slug",
	)
	if got := invalidGroupData.Id(); got != "" {
		t.Fatalf("invalid group id = %q, want empty", got)
	}

	groupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        "tfneg-g-" + suffix,
		"name":        "Neg group " + suffix,
		"description": "Negative validation acceptance group.",
		"active":      true,
	})
	teamData := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfneg-t-" + suffix,
		"name":                       "Neg team " + suffix,
		"description":                "Negative validation acceptance team.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	teamID := testAccIDAsInt(t, teamData.Id())

	invalidTeamData := schema.TestResourceDataRaw(t, resourceTeam().Schema, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfneg-bad-" + suffix,
		"name":                       "Neg bad team " + suffix,
		"description":                "This team should be rejected by API validation.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 101,
		"active":                     true,
	})
	testAccRequireDiagsContain(t, resourceTeam().CreateWithoutTimeout(ctx, invalidTeamData, config),
		"POST /api/teams returned 400",
		"validation_error",
		"escalation_after_reminders",
	)
	if got := invalidTeamData.Id(); got != "" {
		t.Fatalf("invalid team id = %q, want empty", got)
	}

	invalidServiceData := schema.TestResourceDataRaw(t, resourceService().Schema, map[string]interface{}{
		"team_id":       teamID,
		"slug":          "tfneg-s-" + suffix,
		"name":          "Neg service " + suffix,
		"description":   "This service should be rejected by API validation.",
		"service_type":  "api",
		"environment":   "test",
		"criticality":   "medium",
		"tier":          "tier_3",
		"status":        "operational",
		"status_source": "manual",
		"labels_json":   `{}`,
		"metadata_json": `{}`,
		"enabled":       true,
		"public":        false,
	})
	testAccRequireDiagsContain(t, resourceService().CreateWithoutTimeout(ctx, invalidServiceData, config),
		"POST /api/services returned 400",
		"validation_error",
		"environment",
	)
	if got := invalidServiceData.Id(); got != "" {
		t.Fatalf("invalid service id = %q, want empty", got)
	}

	invalidRouteData := schema.TestResourceDataRaw(t, resourceRoute().Schema, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Neg route " + suffix,
		"source":                    "prometheus",
		"notification_channel_mode": "route_only",
		"matchers_json":             `{}`,
		"integration_config_json":   `{}`,
		"enabled":                   true,
	})
	testAccRequireDiagsContain(t, resourceRoute().CreateWithoutTimeout(ctx, invalidRouteData, config),
		"POST /api/routes returned 400",
		"validation_error",
		"source",
	)
	if got := invalidRouteData.Id(); got != "" {
		t.Fatalf("invalid route id = %q, want empty", got)
	}
}

func TestAccIncidentRelayAlertIngestionSmoke(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        "tfing-g-" + suffix,
		"name":        "Ingest group " + suffix,
		"description": "Alert ingestion acceptance group.",
		"active":      true,
	})
	teamData := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfing-t-" + suffix,
		"name":                       "Ingest team " + suffix,
		"description":                "Alert ingestion acceptance team.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	teamID := testAccIDAsInt(t, teamData.Id())

	rotationData := testAccCreateResource(t, ctx, resourceRotation(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Ingest rotation " + suffix,
		"description":               "Alert ingestion rotation.",
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

	serviceData := testAccCreateResource(t, ctx, resourceService(), config, map[string]interface{}{
		"team_id":      teamID,
		"slug":         "tfing-s-" + suffix,
		"name":         "Ingest svc " + suffix,
		"description":  "Alert ingestion service.",
		"service_type": "api",
		"environment":  "testing",
		"criticality":  "high",
		"tier":         "tier_2",
		"labels_json":  `{"managed_by":"terraform","test":"ingestion"}`,
		"metadata_json": `{
			"purpose": "alert-ingestion-smoke"
		}`,
		"enabled": true,
		"public":  false,
	})
	serviceID := testAccIDAsInt(t, serviceData.Id())

	routeData := testAccCreateResource(t, ctx, resourceRoute(), config, map[string]interface{}{
		"team_id":                   teamID,
		"name":                      "Ingest route " + suffix,
		"source":                    "alertmanager",
		"rotation_id":               rotationID,
		"service_id":                serviceID,
		"notification_channel_mode": "route_only",
		"matchers_json":             `{"labels":{"severity":"critical","team":"terraform"}}`,
		"integration_config_json":   `{}`,
		"group_by":                  testAccStringSet("alertname", "instance"),
		"enabled":                   true,
	})
	routeID := testAccIDAsInt(t, routeData.Id())
	intakeToken := strings.TrimSpace(routeData.Get("intake_token").(string))
	if intakeToken == "" {
		t.Fatal("route intake_token is empty")
	}

	alertName := "TerraformIngestion" + suffix
	alertFingerprint := "tf-ingestion-" + suffix
	payload := map[string]interface{}{
		"status": "firing",
		"alerts": []map[string]interface{}{
			{
				"status": "firing",
				"labels": map[string]interface{}{
					"alertname": alertName,
					"severity":  "critical",
					"instance":  "host1",
					"team":      "terraform",
				},
				"annotations": map[string]interface{}{
					"summary":     "Terraform ingestion smoke",
					"description": "Created by Terraform provider acceptance tests.",
				},
				"fingerprint": alertFingerprint,
				"startsAt":    "2026-07-13T10:00:00Z",
			},
		},
	}

	var ingestResponse []map[string]interface{}
	testAccPostAPIWithBearer(t, ctx, config, "/api/integrations/alertmanager", intakeToken, payload, &ingestResponse)
	if len(ingestResponse) != 1 {
		t.Fatalf("ingest response length = %d, want 1: %#v", len(ingestResponse), ingestResponse)
	}
	item := ingestResponse[0]
	testAccRequireAPIBoolField(t, item, "created", true)
	testAccRequireAPIStringField(t, item, "outcome", "created")
	testAccRequireAPIStringField(t, item, "processing_status", "completed")
	testAccRequireAPIStringField(t, item, "status", "firing")
	testAccRequireAPIIntField(t, item, "team_id", teamID)
	testAccRequireAPIIntField(t, item, "route_id", routeID)
	testAccRequireAPIIntField(t, item, "rotation_id", rotationID)
	groupID, ok := intFromInterface(item["group_id"])
	if !ok || groupID <= 0 {
		t.Fatalf("group_id = %#v, want positive int", item["group_id"])
	}
	alertID, ok := intFromInterface(item["alert_id"])
	if !ok || alertID <= 0 {
		t.Fatalf("alert_id = %#v, want positive int", item["alert_id"])
	}
	traceID := strings.TrimSpace(fmt.Sprintf("%v", item["trace_id"]))
	if traceID == "" {
		t.Fatal("trace_id is empty")
	}

	alertGroup := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/alerts/%d", groupID))
	testAccRequireAPIIntField(t, alertGroup, "team_id", teamID)
	testAccRequireAPIIntField(t, alertGroup, "route_id", routeID)
	testAccRequireAPIIntField(t, alertGroup, "rotation_id", rotationID)
	testAccRequireAPIIntField(t, alertGroup, "service_id", serviceID)
	testAccRequireAPIStringField(t, alertGroup, "source", "alertmanager")
	testAccRequireAPIStringField(t, alertGroup, "status", "firing")
	testAccRequireAPIStringField(t, alertGroup, "severity", "critical")
	testAccRequireAPIStringField(t, alertGroup, "title", "Terraform ingestion smoke")
	testAccRequireAPINestedStringField(t, alertGroup, "labels", "alertname", alertName)

	trace := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/alerts/explain/%s", traceID))
	testAccRequireAPIStringField(t, trace, "trace_id", traceID)
	testAccRequireAPIStringField(t, trace, "status", "completed")
	testAccRequireAPIStringField(t, trace, "outcome", "created")
	testAccRequireAPIIntField(t, trace, "group_id", groupID)

	resolvedPayload := map[string]interface{}{
		"status": "resolved",
		"alerts": []map[string]interface{}{
			{
				"status": "resolved",
				"labels": map[string]interface{}{
					"alertname": alertName,
					"severity":  "critical",
					"instance":  "host1",
					"team":      "terraform",
				},
				"annotations": map[string]interface{}{
					"summary":     "Terraform ingestion smoke",
					"description": "Resolved by Terraform provider acceptance tests.",
				},
				"fingerprint": alertFingerprint,
				"startsAt":    "2026-07-13T10:00:00Z",
				"endsAt":      "2026-07-13T10:15:00Z",
			},
		},
	}

	var resolvedResponse []map[string]interface{}
	testAccPostAPIWithBearer(t, ctx, config, "/api/integrations/alertmanager", intakeToken, resolvedPayload, &resolvedResponse)
	if len(resolvedResponse) != 1 {
		t.Fatalf("resolved ingest response length = %d, want 1: %#v", len(resolvedResponse), resolvedResponse)
	}
	resolvedItem := resolvedResponse[0]
	testAccRequireAPIBoolField(t, resolvedItem, "created", false)
	testAccRequireAPIStringField(t, resolvedItem, "outcome", "updated")
	testAccRequireAPIStringField(t, resolvedItem, "processing_status", "completed")
	testAccRequireAPIStringField(t, resolvedItem, "status", "resolved")
	testAccRequireAPIIntField(t, resolvedItem, "team_id", teamID)
	testAccRequireAPIIntField(t, resolvedItem, "route_id", routeID)
	testAccRequireAPIIntField(t, resolvedItem, "rotation_id", rotationID)
	testAccRequireAPIIntField(t, resolvedItem, "group_id", groupID)
	testAccRequireAPIIntField(t, resolvedItem, "alert_id", alertID)
	resolvedTraceID := strings.TrimSpace(fmt.Sprintf("%v", resolvedItem["trace_id"]))
	if resolvedTraceID == "" {
		t.Fatal("resolved trace_id is empty")
	}

	resolvedAlertGroup := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/alerts/%d", groupID))
	testAccRequireAPIIntField(t, resolvedAlertGroup, "team_id", teamID)
	testAccRequireAPIIntField(t, resolvedAlertGroup, "route_id", routeID)
	testAccRequireAPIIntField(t, resolvedAlertGroup, "rotation_id", rotationID)
	testAccRequireAPIIntField(t, resolvedAlertGroup, "service_id", serviceID)
	testAccRequireAPIStringField(t, resolvedAlertGroup, "source", "alertmanager")
	testAccRequireAPIStringField(t, resolvedAlertGroup, "status", "resolved")
	testAccRequireAPIStringField(t, resolvedAlertGroup, "severity", "critical")
	testAccRequireAPIStringField(t, resolvedAlertGroup, "title", "Terraform ingestion smoke")
	testAccRequireAPINestedStringField(t, resolvedAlertGroup, "labels", "alertname", alertName)

	childAlerts, ok := resolvedAlertGroup["alerts"].([]interface{})
	if !ok {
		t.Fatalf("api alerts = %#v, want list", resolvedAlertGroup["alerts"])
	}
	if len(childAlerts) != 1 {
		t.Fatalf("child alert count = %d, want 1: %#v", len(childAlerts), childAlerts)
	}
	childAlert, ok := childAlerts[0].(map[string]interface{})
	if !ok {
		t.Fatalf("child alert = %#v, want object", childAlerts[0])
	}
	testAccRequireAPIIntField(t, childAlert, "id", alertID)
	testAccRequireAPIStringField(t, childAlert, "status", "resolved")

	resolvedTrace := testAccReadAPIObject(t, ctx, config, fmt.Sprintf("/api/alerts/explain/%s", resolvedTraceID))
	testAccRequireAPIStringField(t, resolvedTrace, "trace_id", resolvedTraceID)
	testAccRequireAPIStringField(t, resolvedTrace, "status", "completed")
	testAccRequireAPIStringField(t, resolvedTrace, "outcome", "updated")
	testAccRequireAPIIntField(t, resolvedTrace, "group_id", groupID)
	testAccRequireAPIIntField(t, resolvedTrace, "alert_id", alertID)
}

func TestAccIncidentRelayDriftAndReadAfterDelete(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupResource := resourceGroup()
	groupData := testAccCreateResource(t, ctx, groupResource, config, map[string]interface{}{
		"slug":        "tfdrift-g-" + suffix,
		"name":        "Drift group " + suffix,
		"description": "Created before external drift.",
		"active":      true,
	})
	groupID := groupData.Id()
	groupName := "Drift group upd " + suffix
	groupDescription := "Updated outside Terraform."
	if err := config.Client.Do(ctx, http.MethodPut, fmt.Sprintf("/api/groups/%s", groupID), map[string]interface{}{
		"slug":        groupData.Get("slug"),
		"name":        groupName,
		"description": groupDescription,
		"active":      true,
	}, nil); err != nil {
		t.Fatalf("external group update failed: %v", err)
	}

	testAccRequireNoDiags(t, groupResource.ReadWithoutTimeout(ctx, groupData, config))
	if got, want := groupData.Id(), groupID; got != want {
		t.Fatalf("group id after drift read = %q, want %q", got, want)
	}
	if got := groupData.Get("name"); got != groupName {
		t.Fatalf("drifted group name = %#v, want %#v", got, groupName)
	}
	if got := groupData.Get("description"); got != groupDescription {
		t.Fatalf("drifted group description = %#v, want %#v", got, groupDescription)
	}

	if err := config.Client.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/groups/%s", groupID), nil, nil); err != nil {
		t.Fatalf("external group delete failed: %v", err)
	}
	testAccRequireReadAfterExternalDelete(t, ctx, groupResource, groupData, config)

	parentGroupData := testAccCreateResource(t, ctx, resourceGroup(), config, map[string]interface{}{
		"slug":        "tfdrift-p-" + suffix,
		"name":        "Drift parent " + suffix,
		"description": "Parent for path-backed drift tests.",
		"active":      true,
	})
	teamData := testAccCreateResource(t, ctx, resourceTeam(), config, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, parentGroupData.Id()),
		"slug":                       "tfdrift-t-" + suffix,
		"name":                       "Drift team " + suffix,
		"description":                "Team for path-backed drift tests.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	teamID := testAccIDAsInt(t, teamData.Id())

	serviceResource := resourceService()
	serviceData := testAccCreateResource(t, ctx, serviceResource, config, map[string]interface{}{
		"team_id":       teamID,
		"slug":          "tfdrift-s-" + suffix,
		"name":          "Drift service " + suffix,
		"description":   "Created before external service drift.",
		"service_type":  "api",
		"environment":   "testing",
		"criticality":   "medium",
		"tier":          "tier_3",
		"status":        "operational",
		"status_source": "manual",
		"labels_json":   `{"managed_by":"terraform","test":"drift"}`,
		"tags":          testAccStringSet("terraform", "drift"),
		"metadata_json": `{"purpose":"drift"}`,
		"enabled":       true,
		"public":        false,
	})
	serviceID := serviceData.Id()
	serviceName := "Drift svc upd " + suffix
	serviceDescription := "Service updated outside Terraform."
	if err := config.Client.Do(ctx, http.MethodPut, fmt.Sprintf("/api/services/%s", serviceID), map[string]interface{}{
		"team_id":                      teamID,
		"slug":                         serviceData.Get("slug"),
		"name":                         serviceName,
		"description":                  serviceDescription,
		"service_type":                 "worker",
		"environment":                  "testing",
		"criticality":                  "critical",
		"tier":                         "tier_1",
		"status":                       "degraded",
		"status_source":                "manual",
		"status_message":               "Drifted outside Terraform.",
		"default_rotation_id":          nil,
		"default_escalation_policy_id": nil,
		"notification_policy_id":       nil,
		"priority_policy_id":           nil,
		"labels":                       map[string]interface{}{"managed_by": "terraform", "test": "drift", "source": "api"},
		"tags":                         []string{"terraform", "drift", "external"},
		"metadata":                     map[string]interface{}{"purpose": "drift", "source": "api"},
		"enabled":                      true,
		"public":                       false,
		"public_name":                  "Drift public " + suffix,
		"public_description":           "Public service updated outside Terraform.",
		"public_order":                 100,
	}, nil); err != nil {
		t.Fatalf("external service update failed: %v", err)
	}

	testAccRequireNoDiags(t, serviceResource.ReadWithoutTimeout(ctx, serviceData, config))
	if got, want := serviceData.Id(), serviceID; got != want {
		t.Fatalf("service id after drift read = %q, want %q", got, want)
	}
	for field, want := range map[string]interface{}{
		"name":           serviceName,
		"description":    serviceDescription,
		"service_type":   "worker",
		"criticality":    "critical",
		"tier":           "tier_1",
		"status":         "degraded",
		"status_message": "Drifted outside Terraform.",
	} {
		if got := serviceData.Get(field); got != want {
			t.Fatalf("drifted service %s = %#v, want %#v", field, got, want)
		}
	}

	if err := config.Client.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/services/%s", serviceID), nil, nil); err != nil {
		t.Fatalf("external service delete failed: %v", err)
	}
	testAccRequireReadAfterExternalDelete(t, ctx, serviceResource, serviceData, config)
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

func testAccRequireDiagsContain(t *testing.T, diags diag.Diagnostics, fragments ...string) {
	t.Helper()

	if !diags.HasError() {
		t.Fatal("expected diagnostics, got none")
	}
	text := fmt.Sprint(diags)
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("diagnostics = %v, want fragment %q", diags, fragment)
		}
	}
}

func testAccRequireReadAfterExternalDelete(t *testing.T, ctx context.Context, resource *schema.Resource, data *schema.ResourceData, config *Config) {
	t.Helper()

	id := data.Id()
	testAccRequireNoDiags(t, resource.ReadWithoutTimeout(ctx, data, config))
	if got := data.Id(); got != "" {
		t.Fatalf("id after external delete read = %q, want empty; original id was %q", got, id)
	}
}

func testAccReadAPIObject(t *testing.T, ctx context.Context, config *Config, path string) map[string]interface{} {
	t.Helper()

	var response map[string]interface{}
	if err := config.Client.Do(ctx, http.MethodGet, path, nil, &response); err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return response
}

func testAccPostAPIWithBearer(t *testing.T, ctx context.Context, config *Config, path, token string, body interface{}, out interface{}) {
	t.Helper()

	requestURL, err := joinURL(config.Client.baseURL, path)
	if err != nil {
		t.Fatalf("join %s: %v", path, err)
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal %s body: %v", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(rawBody))
	if err != nil {
		t.Fatalf("create POST %s request: %v", path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.Client.userAgent)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := config.Client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read POST %s response: %v", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("POST %s returned %d: %s", path, resp.StatusCode, string(rawResponse))
	}
	if out == nil {
		return
	}
	if err := json.Unmarshal(rawResponse, out); err != nil {
		t.Fatalf("decode POST %s response: %v; body: %s", path, err, string(rawResponse))
	}
}

func testAccRequireAPIStringField(t *testing.T, response map[string]interface{}, field, want string) {
	t.Helper()

	if got, ok := response[field].(string); !ok || got != want {
		t.Fatalf("api %s = %#v, want %q", field, response[field], want)
	}
}

func testAccRequireAPINestedStringField(t *testing.T, response map[string]interface{}, field, nestedField, want string) {
	t.Helper()

	nested, ok := response[field].(map[string]interface{})
	if !ok {
		t.Fatalf("api %s = %#v, want object", field, response[field])
	}
	if got, ok := nested[nestedField].(string); !ok || got != want {
		t.Fatalf("api %s.%s = %#v, want %q", field, nestedField, nested[nestedField], want)
	}
}

func testAccRequireAPIBoolField(t *testing.T, response map[string]interface{}, field string, want bool) {
	t.Helper()

	if got, ok := response[field].(bool); !ok || got != want {
		t.Fatalf("api %s = %#v, want %v", field, response[field], want)
	}
}

func testAccRequireAPIIntField(t *testing.T, response map[string]interface{}, field string, want int) {
	t.Helper()

	got, ok := intFromInterface(response[field])
	if !ok || got != want {
		t.Fatalf("api %s = %#v, want %d", field, response[field], want)
	}
}

func testAccRequireAPINilField(t *testing.T, response map[string]interface{}, field string) {
	t.Helper()

	if got := response[field]; got != nil {
		t.Fatalf("api %s = %#v, want nil", field, got)
	}
}

func testAccRequireAPIJSONField(t *testing.T, response map[string]interface{}, field, wantRaw string) {
	t.Helper()

	got, err := valueToJSONString(response[field])
	if err != nil {
		t.Fatalf("encode api %s: %v", field, err)
	}
	want, err := normalizeJSONString(wantRaw)
	if err != nil {
		t.Fatalf("normalize expected api %s: %v", field, err)
	}
	if got != want {
		t.Fatalf("api %s = %q, want %q", field, got, want)
	}
}

func testAccRequireAPIStringListField(t *testing.T, response map[string]interface{}, field string, want ...string) {
	t.Helper()

	items, ok := response[field].([]interface{})
	if !ok {
		t.Fatalf("api %s = %#v, want list", field, response[field])
	}
	wantItems := make([]interface{}, 0, len(want))
	for _, item := range want {
		wantItems = append(wantItems, item)
	}
	if !sameElements(items, wantItems) {
		t.Fatalf("api %s = %#v, want %#v", field, items, wantItems)
	}
}

func testAccRequireAPIChannelIDs(t *testing.T, response map[string]interface{}, want ...int) {
	t.Helper()

	channels, ok := response["channels"].([]interface{})
	if !ok {
		t.Fatalf("api channels = %#v, want list", response["channels"])
	}

	gotIDs := make([]interface{}, 0, len(channels))
	for _, item := range channels {
		channel, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("api channel item = %#v, want object", item)
		}
		id, ok := intFromInterface(channel["id"])
		if !ok {
			t.Fatalf("api channel id = %#v, want int", channel["id"])
		}
		gotIDs = append(gotIDs, id)
	}

	if !sameElements(gotIDs, testAccIntSet(want...)) {
		t.Fatalf("api channel ids = %#v, want %#v", gotIDs, want)
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

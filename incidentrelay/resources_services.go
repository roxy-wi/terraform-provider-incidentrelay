package incidentrelay

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceService() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("slug", "Stable service slug."),
		reqString("name", "Service name."),
		optString("description", "Optional service description."),
		optStringDefault("service_type", "other", "Service type."),
		optStringDefault("environment", "production", "Service environment."),
		optStringDefault("criticality", "medium", "Service criticality."),
		optStringDefault("tier", "tier_3", "Service tier."),
		optStringDefault("status", "operational", "Manual service status."),
		optStringDefault("status_source", "manual", "Service status source."),
		optString("status_message", "Optional status message."),
		optInt("default_rotation_id", "Default rotation id."),
		optInt("default_escalation_policy_id", "Default escalation policy id."),
		optInt("notification_policy_id", "Notification policy id."),
		optInt("priority_policy_id", "Incident priority policy id."),
		optJSONDefault("labels_json", "labels", "{}", "Service labels JSON."),
		optStringSet("tags", "Service tags."),
		optJSONDefault("metadata_json", "metadata", "{}", "Service metadata JSON."),
		optBoolDefault("enabled", true, "Whether the service is enabled."),
		optBoolDefault("public", false, "Whether the service can be exposed publicly."),
		optString("public_name", "Public service name."),
		optString("public_description", "Public service description."),
		optIntDefault("public_order", 100, "Public ordering value."),
		computedInt("group_id", "Owner group id."),
		computedString("team_name", "Team name."),
		computedString("team_slug", "Team slug."),
		computedString("default_rotation_name", "Default rotation name."),
		computedString("notification_policy_name", "Notification policy name."),
		computedString("priority_policy_name", "Incident priority policy name."),
		computedString("default_escalation_policy_name", "Default escalation policy name."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay service catalog service.",
		Fields:       fields,
		CreatePath:   createPath("/api/services"),
		ReadPath:     idPath("/api/services/%s"),
		UpdatePath:   idPath("/api/services/%s"),
		DeletePath:   idPath("/api/services/%s"),
		CreateFields: []string{"team_id", "slug", "name", "description", "service_type", "environment", "criticality", "tier", "status", "status_source", "status_message", "default_rotation_id", "default_escalation_policy_id", "notification_policy_id", "priority_policy_id", "labels_json", "tags", "metadata_json", "enabled", "public", "public_name", "public_description", "public_order"},
		UpdateFields: []string{"team_id", "slug", "name", "description", "service_type", "environment", "criticality", "tier", "status", "status_source", "status_message", "default_rotation_id", "default_escalation_policy_id", "notification_policy_id", "priority_policy_id", "labels_json", "tags", "metadata_json", "enabled", "public", "public_name", "public_description", "public_order"},
	})
}

func resourceServiceMatchRule() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqIntForceNew("service_id", "Target service id."),
		optInt("route_id", "Optional route scope."),
		optIntDefault("position", 0, "Lower position is evaluated first."),
		reqString("name", "Rule name."),
		optString("description", "Optional rule description."),
		optInt("matcher_preset_id", "Optional matcher preset id."),
		{Name: "matchers_json", APIName: "matchers", Kind: kindJSON, Optional: true, Description: "Matcher JSON evaluated against alerts."},
		optBoolDefault("enabled", true, "Whether the rule is enabled."),
		computedString("route_name", "Route name."),
		computedString("service_name", "Service name."),
	}
	resource := crudResource(resourceSpec{
		Description:  "IncidentRelay service match rule.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/services/%d/match-rules", "service_id"),
		ReadListPath: fieldListPath("/api/services/%d/match-rules", "service_id"),
		UpdatePath:   idPath("/api/services/match-rules/%s"),
		DeletePath:   idPath("/api/services/match-rules/%s"),
		CreateFields: []string{"team_id", "service_id", "route_id", "position", "name", "description", "matcher_preset_id", "matchers_json", "enabled"},
		UpdateFields: []string{"team_id", "service_id", "route_id", "position", "name", "description", "matcher_preset_id", "matchers_json", "enabled"},
	})
	resource.Schema["matcher_preset_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.CustomizeDiff = func(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
		if !diff.NewValueKnown("matcher_preset_id") || !diff.NewValueKnown("matchers_json") {
			return nil
		}
		if _, ok := diff.GetOk("matcher_preset_id"); ok {
			return nil
		}

		matchers, err := jsonStringToValue(diff.Get("matchers_json").(string))
		if err != nil {
			return fmt.Errorf("parse matchers_json: %w", err)
		}
		matcherObject, ok := matchers.(map[string]interface{})
		if !ok || len(matcherObject) == 0 {
			return fmt.Errorf("at least one of matcher_preset_id or a non-empty matchers_json object is required")
		}
		return nil
	}
	return resource
}

func resourceServiceLink() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("service_id", "Service id."),
		optStringDefault("link_type", "other", "Link type."),
		reqString("label", "Link label."),
		reqString("url", "Link URL."),
		optString("description", "Optional link description."),
		optIntDefault("priority", 100, "Display priority."),
		optBoolDefault("enabled", true, "Whether the link is enabled."),
		computedString("service_name", "Service name."),
		computedString("service_slug", "Service slug."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay service link.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/services/%d/links", "service_id"),
		ReadListPath: createPath("/api/services/links"),
		UpdatePath:   idPath("/api/services/links/%s"),
		DeletePath:   idPath("/api/services/links/%s"),
		CreateFields: []string{"link_type", "label", "url", "description", "priority", "enabled"},
		UpdateFields: []string{"link_type", "label", "url", "description", "priority", "enabled"},
	})
}

func resourceServiceRunbook() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("service_id", "Service id."),
		reqString("title", "Runbook title."),
		optString("description", "Optional runbook description."),
		reqString("url", "Runbook URL."),
		optString("severity", "Optional matching severity."),
		optJSONDefault("matchers_json", "matchers", "{}", "Runbook matcher JSON."),
		optIntDefault("priority", 100, "Display priority."),
		optBoolDefault("enabled", true, "Whether the runbook is enabled."),
		computedString("service_name", "Service name."),
		computedString("service_slug", "Service slug."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay service runbook.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/services/%d/runbooks", "service_id"),
		ReadListPath: createPath("/api/services/runbooks"),
		UpdatePath:   idPath("/api/services/runbooks/%s"),
		DeletePath:   idPath("/api/services/runbooks/%s"),
		CreateFields: []string{"title", "description", "url", "severity", "matchers_json", "priority", "enabled"},
		UpdateFields: []string{"title", "description", "url", "severity", "matchers_json", "priority", "enabled"},
	})
}

func resourceServiceDependency() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("service_id", "Service id."),
		reqInt("depends_on_service_id", "Upstream service id."),
		optStringDefault("dependency_type", "hard", "Dependency type."),
		optStringDefault("criticality", "important", "Dependency criticality."),
		optBoolDefault("correlation_enabled", true, "Whether the dependency can be used for alert correlation."),
		optIntDefault("propagation_delay_seconds", 300, "Maximum expected propagation delay between related alerts."),
		optString("description", "Optional dependency description."),
		optBoolDefault("enabled", true, "Whether the dependency is enabled."),
		computedString("service_name", "Service name."),
		computedString("service_slug", "Service slug."),
		computedString("depends_on_service_name", "Upstream service name."),
		computedString("depends_on_service_slug", "Upstream service slug."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay service dependency.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/services/%d/dependencies", "service_id"),
		ReadListPath: createPath("/api/services/dependencies"),
		UpdatePath:   idPath("/api/services/dependencies/%s"),
		DeletePath:   idPath("/api/services/dependencies/%s"),
		CreateFields: []string{"depends_on_service_id", "dependency_type", "criticality", "correlation_enabled", "propagation_delay_seconds", "description", "enabled"},
		UpdateFields: []string{"depends_on_service_id", "dependency_type", "criticality", "correlation_enabled", "propagation_delay_seconds", "description", "enabled"},
	})
}

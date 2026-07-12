package incidentrelay

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func resourceEscalationPolicy() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("name", "Escalation policy name."),
		optString("description", "Optional policy description."),
		optBoolDefault("enabled", true, "Whether the policy is enabled."),
		optIntDefault("repeat_count", 0, "Number of additional full rule-chain repeats after the first pass."),
		computedString("team_name", "Team name."),
		computedString("team_slug", "Team slug."),
		computedInt("group_id", "Owner group id."),
		computedString("group_slug", "Owner group slug."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay escalation policy.",
		Fields:       fields,
		CreatePath:   createPath("/api/escalation-policies"),
		ReadPath:     idPath("/api/escalation-policies/%s"),
		UpdatePath:   idPath("/api/escalation-policies/%s"),
		DeletePath:   idPath("/api/escalation-policies/%s"),
		CreateFields: []string{"team_id", "name", "description", "enabled", "repeat_count"},
		UpdateFields: []string{"team_id", "name", "description", "enabled", "repeat_count"},
	})
}

func resourceEscalationPolicyRule() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("policy_id", "Parent escalation policy id."),
		reqInt("position", "Rule order inside the policy."),
		reqInt("delay_seconds", "Delay before moving to the next rule."),
		reqString("target_type", "Escalation target type: rotation or user."),
		reqInt("target_id", "Rotation id when target_type=rotation, user id when target_type=user."),
		optBoolDefault("enabled", true, "Whether the rule is enabled."),
		computedString("target_name", "Human-readable target name."),
	}
	return crudResource(resourceSpec{
		Description:   "IncidentRelay escalation policy rule.",
		Fields:        fields,
		CreatePath:    fieldCreatePath("/api/escalation-policies/%d/rules", "policy_id"),
		ReadListPath:  fieldListPath("/api/escalation-policies/%d", "policy_id"),
		ReadListField: "rules",
		UpdatePath:    idPath("/api/escalation-policies/rules/%s"),
		DeletePath:    idPath("/api/escalation-policies/rules/%s"),
		CreateFields:  []string{"position", "delay_seconds", "target_type", "target_id", "enabled"},
		UpdateFields:  []string{"position", "delay_seconds", "target_type", "target_id", "enabled"},
	})
}

func resourceNotificationPolicy() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("name", "Notification policy name."),
		optString("description", "Optional policy description."),
		optBoolDefault("enabled", true, "Whether the policy is enabled."),
		computedString("team_name", "Team name."),
		computedString("team_slug", "Team slug."),
		computedInt("rules_count", "Number of active rules."),
		computedInt("services_count", "Number of services using this policy."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay service notification policy.",
		Fields:       fields,
		CreatePath:   createPath("/api/notification-policies"),
		ReadPath:     idPath("/api/notification-policies/%s"),
		UpdatePath:   idPath("/api/notification-policies/%s"),
		DeletePath:   idPath("/api/notification-policies/%s"),
		CreateFields: []string{"team_id", "name", "description", "enabled"},
		UpdateFields: []string{"team_id", "name", "description", "enabled"},
	})
}

func resourceNotificationPolicyRule() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("policy_id", "Parent notification policy id."),
		reqString("name", "Rule name."),
		optString("description", "Optional rule description."),
		optIntDefault("position", 1, "Rule evaluation order."),
		optStringSet("event_types", "Notification event types: notification, reminder, escalation."),
		optJSONDefault("matchers_json", "matchers", "{}", "Alert matcher JSON."),
		optIntSet("channel_ids", "Notification channel ids selected by this rule."),
		optBoolDefault("continue_matching", false, "Continue evaluating following rules after this rule matches."),
		optBoolDefault("enabled", true, "Whether the rule is enabled."),
	}
	return crudResource(resourceSpec{
		Description:   "IncidentRelay notification policy rule.",
		Fields:        fields,
		CreatePath:    fieldCreatePath("/api/notification-policies/%d/rules", "policy_id"),
		ReadListPath:  fieldListPath("/api/notification-policies/%d", "policy_id"),
		ReadListField: "rules",
		UpdatePath:    fieldIDPath("/api/notification-policies/%d/rules/%s", "policy_id"),
		DeletePath:    fieldIDPath("/api/notification-policies/%d/rules/%s", "policy_id"),
		CreateFields:  []string{"name", "description", "position", "event_types", "matchers_json", "channel_ids", "continue_matching", "enabled"},
		UpdateFields:  []string{"name", "description", "position", "event_types", "matchers_json", "channel_ids", "continue_matching", "enabled"},
	})
}

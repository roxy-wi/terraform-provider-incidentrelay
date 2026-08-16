package incidentrelay

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePriorityPolicy() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("team_id", "Owner team id."),
		reqString("name", "Priority policy name."),
		optString("description", "Optional priority policy description."),
		optBoolDefault("enabled", true, "Whether the priority policy is enabled."),
		optBoolDefault("default_for_team", false, "Whether this is the default priority policy for the team."),
		optStringDefault("update_mode", "raise_only", "Priority update mode: raise_only, recalculate, or initial_only."),
		optStringDefault("source_priority_mode", "ignore", "Source priority mode: ignore or prefer."),
		optStringDefault("fallback_mode", "severity_mapping", "Fallback mode: severity_mapping or fixed_priority."),
		optInt("fallback_priority_id", "Incident priority id required when fallback_mode is fixed_priority."),
		computedString("team_name", "Owner team name."),
		computedString("team_slug", "Owner team slug."),
		computedInt("rules_count", "Number of active priority policy rules."),
		computedInt("services_count", "Number of services using this priority policy."),
	}

	resource := crudResource(resourceSpec{
		Description:  "IncidentRelay incident priority policy.",
		Fields:       fields,
		CreatePath:   createPath("/api/priority-policies"),
		ReadPath:     idPath("/api/priority-policies/%s"),
		UpdatePath:   idPath("/api/priority-policies/%s"),
		DeletePath:   idPath("/api/priority-policies/%s"),
		CreateFields: []string{"team_id", "name", "description", "enabled", "default_for_team", "update_mode", "source_priority_mode", "fallback_mode", "fallback_priority_id"},
		UpdateFields: []string{"name", "description", "enabled", "default_for_team", "update_mode", "source_priority_mode", "fallback_mode", "fallback_priority_id"},
	})

	resource.Schema["team_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["update_mode"].ValidateFunc = validation.StringInSlice(
		[]string{"raise_only", "recalculate", "initial_only"},
		false,
	)
	resource.Schema["source_priority_mode"].ValidateFunc = validation.StringInSlice(
		[]string{"ignore", "prefer"},
		false,
	)
	resource.Schema["fallback_mode"].ValidateFunc = validation.StringInSlice(
		[]string{"severity_mapping", "fixed_priority"},
		false,
	)
	resource.Schema["fallback_priority_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.CustomizeDiff = func(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
		if diff.NewValueKnown("default_for_team") && diff.NewValueKnown("enabled") &&
			diff.Get("default_for_team").(bool) && !diff.Get("enabled").(bool) {
			return fmt.Errorf("default_for_team requires enabled to be true")
		}

		if diff.NewValueKnown("fallback_mode") && diff.NewValueKnown("fallback_priority_id") &&
			diff.Get("fallback_mode").(string) == "fixed_priority" {
			if priorityID, ok := diff.GetOk("fallback_priority_id"); !ok || priorityID.(int) < 1 {
				return fmt.Errorf("fallback_priority_id is required when fallback_mode is fixed_priority")
			}
		}

		return nil
	}

	return resource
}

func resourcePriorityPolicyRule() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("policy_id", "Parent priority policy id."),
		reqString("name", "Priority policy rule name."),
		optString("description", "Optional priority policy rule description."),
		{
			Name:        "position",
			Kind:        kindInt,
			Optional:    true,
			Computed:    true,
			Description: "Rule evaluation order. When omitted, IncidentRelay appends the rule.",
		},
		optJSONDefault("matchers_json", "matchers", "{}", "Alert matcher JSON."),
		optInt("matcher_preset_id", "Optional matcher preset id."),
		reqInt("priority_id", "Incident priority id assigned by this rule."),
		optBoolDefault("enabled", true, "Whether the priority policy rule is enabled."),
	}

	resource := crudResource(resourceSpec{
		Description:   "Rule inside an IncidentRelay incident priority policy.",
		Fields:        fields,
		CreatePath:    fieldCreatePath("/api/priority-policies/%d/rules", "policy_id"),
		ReadListPath:  fieldListPath("/api/priority-policies/%d", "policy_id"),
		ReadListField: "rules",
		UpdatePath:    fieldIDPath("/api/priority-policies/%d/rules/%s", "policy_id"),
		DeletePath:    fieldIDPath("/api/priority-policies/%d/rules/%s", "policy_id"),
		CreateFields:  []string{"name", "description", "position", "matchers_json", "matcher_preset_id", "priority_id", "enabled"},
		UpdateFields:  []string{"name", "description", "position", "matchers_json", "matcher_preset_id", "priority_id", "enabled"},
	})

	resource.Schema["policy_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["position"].ValidateFunc = validation.IntBetween(1, 1000)
	resource.Schema["matcher_preset_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["priority_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Importer = &schema.ResourceImporter{
		StateContext: func(_ context.Context, data *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
			rawID := strings.TrimSpace(data.Id())
			parts := strings.Split(rawID, "/")

			if len(parts) == 1 {
				if policyID, ok := intFromInterface(data.Get("policy_id")); ok && policyID > 0 {
					return []*schema.ResourceData{data}, nil
				}
				return nil, fmt.Errorf("priority policy rule import id must use policy_id/rule_id format")
			}
			if len(parts) != 2 {
				return nil, fmt.Errorf("priority policy rule import id must use policy_id/rule_id format")
			}

			policyID, err := strconv.Atoi(parts[0])
			if err != nil || policyID < 1 {
				return nil, fmt.Errorf("invalid priority policy id %q", parts[0])
			}
			ruleID, err := strconv.Atoi(parts[1])
			if err != nil || ruleID < 1 {
				return nil, fmt.Errorf("invalid priority policy rule id %q", parts[1])
			}

			if err := data.Set("policy_id", policyID); err != nil {
				return nil, err
			}
			data.SetId(strconv.Itoa(ruleID))
			return []*schema.ResourceData{data}, nil
		},
	}

	return resource
}

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

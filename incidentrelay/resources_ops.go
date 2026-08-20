package incidentrelay

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSilence() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("name", "Silence name."),
		optString("reason", "Optional silence reason."),
		optInt("matcher_preset_id", "Optional matcher preset id."),
		optJSONDefault("matchers_json", "matchers", "{}", "Alert matcher JSON."),
		reqString("starts_at", "Silence start datetime."),
		reqString("ends_at", "Silence end datetime."),
		optBoolDefault("apply_to_existing", false, "Whether the silence is applied retroactively to matching unresolved alerts."),
		optBoolDefault("reactivate_on_end", true, "Whether alerts suppressed by this silence are reactivated when it ends."),
		optBoolDefault("enabled", true, "Whether the silence is enabled."),
	}
	spec := resourceSpec{
		Description:  "IncidentRelay alert silence.",
		Fields:       fields,
		CreatePath:   createPath("/api/silences"),
		ReadPath:     idPath("/api/silences/%s"),
		UpdatePath:   idPath("/api/silences/%s"),
		DeletePath:   idPath("/api/silences/%s"),
		CreateFields: []string{"team_id", "name", "reason", "matcher_preset_id", "matchers_json", "starts_at", "ends_at", "apply_to_existing", "reactivate_on_end"},
		UpdateFields: []string{"team_id", "name", "reason", "matcher_preset_id", "matchers_json", "starts_at", "ends_at", "apply_to_existing", "reactivate_on_end"},
	}
	resource := crudResource(spec)
	resource.CreateWithoutTimeout = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		return silenceCreate(ctx, d, m, spec)
	}
	resource.UpdateWithoutTimeout = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		return silenceUpdate(ctx, d, m, spec)
	}
	resource.Schema["matcher_preset_id"].ValidateFunc = validation.IntAtLeast(1)
	return resource
}

func silenceCreate(ctx context.Context, d *schema.ResourceData, m interface{}, spec resourceSpec) diag.Diagnostics {
	client := m.(*Config).Client
	desiredEnabled := d.Get("enabled").(bool)
	payload, err := buildPayload(d, spec.Fields, spec.CreateFields, false)
	if err != nil {
		return diag.FromErr(err)
	}

	var response map[string]interface{}
	if err := client.Do(ctx, http.MethodPost, spec.CreatePath(d), payload, &response); err != nil {
		return diag.FromErr(err)
	}
	if err := setIDFromResponse(d, response); err != nil {
		return diag.FromErr(err)
	}
	if !desiredEnabled {
		if err := client.Do(ctx, http.MethodDelete, spec.DeletePath(d.Id(), d), nil, nil); err != nil {
			return diag.FromErr(err)
		}
	}
	return crudRead(ctx, d, m, spec)
}

func silenceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, spec resourceSpec) diag.Diagnostics {
	client := m.(*Config).Client
	desiredEnabled := d.Get("enabled").(bool)
	enabledChanged := d.HasChange("enabled")
	payload, err := buildPayload(d, spec.Fields, spec.UpdateFields, true)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := client.Do(ctx, http.MethodPut, spec.UpdatePath(d.Id(), d), payload, nil); err != nil {
		return diag.FromErr(err)
	}

	if enabledChanged && desiredEnabled {
		if err := client.Do(ctx, http.MethodPost, fmt.Sprintf("/api/silences/%s/enable", d.Id()), nil, nil); err != nil {
			return diag.FromErr(err)
		}
	} else if enabledChanged && !desiredEnabled {
		if err := client.Do(ctx, http.MethodDelete, spec.DeletePath(d.Id(), d), nil, nil); err != nil {
			return diag.FromErr(err)
		}
	}
	return crudRead(ctx, d, m, spec)
}

func resourceMaintenanceWindow() *schema.Resource {
	fields := []fieldDef{
		reqString("name", "Maintenance window name."),
		optString("description", "Optional maintenance description."),
		optStringDefault("behavior", "suppress_notifications", "Maintenance behavior."),
		optStringDefault("timezone", "UTC", "Maintenance timezone."),
		optString("rrule", "Optional RFC5545 RRULE string."),
		reqString("starts_at", "Maintenance start as wall-clock time in the selected timezone."),
		reqString("ends_at", "Maintenance end as wall-clock time in the selected timezone."),
		optBoolDefault("enabled", true, "Whether the maintenance window is enabled."),
		optBoolDefault("apply_to_existing", false, "Whether the window is applied retroactively to matching unresolved alerts."),
		optBoolDefault("reactivate_on_end", true, "Whether alerts suppressed by this window are reactivated when it ends."),
		reqJSON("scopes_json", "scopes", "Maintenance scopes JSON array."),
		computedString("status", "Effective maintenance status."),
		computedBool("deleted", "Whether the window is deleted."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay maintenance window.",
		Fields:       fields,
		CreatePath:   createPath("/api/maintenance-windows"),
		ReadPath:     idPath("/api/maintenance-windows/%s"),
		UpdatePath:   idPath("/api/maintenance-windows/%s"),
		DeletePath:   idPath("/api/maintenance-windows/%s"),
		CreateFields: []string{"name", "description", "behavior", "timezone", "rrule", "starts_at", "ends_at", "enabled", "apply_to_existing", "reactivate_on_end", "scopes_json"},
		UpdateFields: []string{"name", "description", "behavior", "timezone", "rrule", "starts_at", "ends_at", "enabled", "apply_to_existing", "reactivate_on_end", "scopes_json"},
	})
}

func resourceHeartbeat() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqInt("route_id", "Route with source=heartbeat."),
		optInt("service_id", "Optional associated service id."),
		reqString("name", "Heartbeat name."),
		reqString("slug", "Heartbeat slug."),
		optString("description", "Optional heartbeat description."),
		optStringDefault("mode", "interval", "Heartbeat mode: interval or scheduled."),
		optInt("expected_interval_seconds", "Expected interval in seconds for interval heartbeats. Defaults to 300 in the API."),
		optIntDefault("grace_period_seconds", 300, "Grace period in seconds."),
		optString("schedule_kind", "Schedule kind: daily, weekly, or monthly."),
		optString("schedule_time", "Scheduled heartbeat time in HH:MM."),
		optInt("schedule_weekday", "Weekly schedule weekday, Monday is 0."),
		optInt("schedule_monthday", "Monthly schedule day."),
		optStringDefault("timezone", "UTC", "Heartbeat timezone."),
		optStringDefault("severity", "critical", "Alert severity when heartbeat is overdue."),
		optStringDefault("priority_slug", "p2", "Alert priority slug."),
		optBoolDefault("enabled", true, "Whether the heartbeat is enabled."),
		optBoolDefault("auto_resolve", true, "Whether recovery pings auto-resolve the alert."),
		optBoolDefault("instance_tracking_enabled", false, "Whether instance-level tracking is enabled."),
		optStringDefault("instance_key", "instance", "Payload field used as instance key."),
		optStringDefault("expected_instances_mode", "none", "Expected instances mode: none, static, or auto."),
		optStringSet("expected_instances", "Expected static instance keys."),
		optInt("auto_discovery_ttl_days", "Auto-discovery TTL in days. Defaults to 30 in the API when auto discovery is enabled."),
		optJSONDefault("labels_json", "labels", "{}", "Heartbeat alert labels JSON."),
		optJSONDefault("metadata_json", "metadata", "{}", "Heartbeat metadata JSON."),
		computedString("uid", "Heartbeat UUID."),
		computedString("status", "Heartbeat status."),
		computedString("token_prefix", "Heartbeat token prefix."),
		computedSensitiveString("token", "One-time heartbeat token returned on create/regeneration."),
		computedSensitiveString("ping_url", "One-time heartbeat ping URL returned on create/regeneration."),
		computedString("ping_url_hint", "Redacted ping URL hint returned on refresh."),
		computedString("last_seen_at", "Last seen timestamp."),
		computedString("next_expected_at", "Next expected timestamp."),
		computedString("deadline_at", "Deadline timestamp."),
		computedInt("current_alert_group_id", "Current alert group id."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay heartbeat dead-man-switch check.",
		Fields:       fields,
		CreatePath:   createPath("/api/heartbeats"),
		ReadPath:     idPath("/api/heartbeats/%s"),
		UpdatePath:   idPath("/api/heartbeats/%s"),
		DeletePath:   idPath("/api/heartbeats/%s"),
		CreateFields: []string{"team_id", "route_id", "service_id", "name", "slug", "description", "mode", "expected_interval_seconds", "grace_period_seconds", "schedule_kind", "schedule_time", "schedule_weekday", "schedule_monthday", "timezone", "severity", "priority_slug", "enabled", "auto_resolve", "instance_tracking_enabled", "instance_key", "expected_instances_mode", "expected_instances", "auto_discovery_ttl_days", "labels_json", "metadata_json"},
		UpdateFields: []string{"team_id", "route_id", "service_id", "name", "slug", "description", "mode", "expected_interval_seconds", "grace_period_seconds", "schedule_kind", "schedule_time", "schedule_weekday", "schedule_monthday", "timezone", "severity", "priority_slug", "enabled", "auto_resolve", "instance_tracking_enabled", "instance_key", "expected_instances_mode", "expected_instances", "auto_discovery_ttl_days", "labels_json", "metadata_json"},
	})
}

func resourceBusinessService() *schema.Resource {
	fields := []fieldDef{
		reqInt("group_id", "Owner group id."),
		optInt("owner_team_id", "Optional owner team id."),
		reqString("slug", "Business service slug."),
		reqString("name", "Business service name."),
		optString("description", "Optional business service description."),
		optStringDefault("criticality", "important", "Business service criticality."),
		optStringDefault("tier", "tier_2", "Business service tier."),
		optBoolDefault("public", true, "Whether the business service is public."),
		optString("public_name", "Public name."),
		optString("public_description", "Public description."),
		optIntDefault("public_order", 100, "Public ordering value."),
		optJSONDefault("labels_json", "labels", "{}", "Business service labels JSON."),
		optJSONDefault("metadata_json", "metadata", "{}", "Business service metadata JSON."),
		optBoolDefault("enabled", true, "Whether the business service is enabled."),
		computedString("status", "Calculated business service status."),
		computedInt("components_count", "Number of active components."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay business service.",
		Fields:       fields,
		CreatePath:   createPath("/api/business-services"),
		ReadPath:     idPath("/api/business-services/%s"),
		UpdatePath:   idPath("/api/business-services/%s"),
		DeletePath:   idPath("/api/business-services/%s"),
		CreateFields: []string{"group_id", "owner_team_id", "slug", "name", "description", "criticality", "tier", "public", "public_name", "public_description", "public_order", "labels_json", "metadata_json", "enabled"},
		UpdateFields: []string{"group_id", "owner_team_id", "slug", "name", "description", "criticality", "tier", "public", "public_name", "public_description", "public_order", "labels_json", "metadata_json", "enabled"},
	})
}

func resourceBusinessServiceComponent() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("business_service_id", "Parent business service id."),
		reqInt("service_id", "Technical service id."),
		optStringDefault("component_type", "technical_service", "Component type."),
		optStringDefault("criticality", "required", "Component criticality."),
		optIntDefault("impact_weight", 100, "Impact weight from 0 to 100."),
		optIntDefault("position", 0, "Component display position."),
		optStringDefault("status_rule", "inherit", "Component status rule."),
		optString("description", "Optional component description."),
		optBoolDefault("enabled", true, "Whether the component is enabled."),
		computedString("service_name", "Service name."),
		computedString("service_slug", "Service slug."),
		computedString("service_status", "Service status."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay business service component.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/business-services/%d/components", "business_service_id"),
		ReadListPath: fieldListPath("/api/business-services/%d/components", "business_service_id"),
		UpdatePath:   idPath("/api/business-services/components/%s"),
		DeletePath:   idPath("/api/business-services/components/%s"),
		CreateFields: []string{"service_id", "component_type", "criticality", "impact_weight", "position", "status_rule", "description", "enabled"},
		UpdateFields: []string{"service_id", "component_type", "criticality", "impact_weight", "position", "status_rule", "description", "enabled"},
	})
}

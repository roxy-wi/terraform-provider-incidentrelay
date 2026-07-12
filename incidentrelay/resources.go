package incidentrelay

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func resourceGroup() *schema.Resource {
	fields := []fieldDef{
		reqString("slug", "Stable group slug."),
		reqString("name", "Human-readable group name."),
		optString("description", "Optional group description."),
		optBoolDefault("active", true, "Whether the group is active."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay access group.",
		Fields:       fields,
		CreatePath:   createPath("/api/groups"),
		ReadListPath: createPath("/api/groups"),
		UpdatePath:   idPath("/api/groups/%s"),
		DeletePath:   idPath("/api/groups/%s"),
		CreateFields: []string{"slug", "name", "description", "active"},
		UpdateFields: []string{"slug", "name", "description", "active"},
	})
}

func resourceAdminUser() *schema.Resource {
	fields := []fieldDef{
		reqString("username", "Login username."),
		optString("display_name", "Human-readable display name."),
		optString("email", "Email address."),
		optString("phone", "Phone number for voice integrations."),
		optString("telegram_user_id", "Telegram user ID."),
		optString("slack_user_id", "Slack user ID."),
		optString("mattermost_user_id", "Mattermost user ID."),
		optSensitiveString("password", "Initial or replacement password. Omit on update to keep the current password."),
		optBoolDefault("active", true, "Whether the user is active."),
		optBoolDefault("is_admin", false, "Whether the user is a global admin."),
		optInt("group_id", "Optional active group id. On create, the user is added to this group."),
		optStringDefault("group_role", "viewer", "Role assigned when group_id is provided."),
		computedInt("active_group_id", "Current active group id."),
		computedString("active_group_slug", "Current active group slug."),
		computedString("active_group_role", "Current active group role."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay local admin-managed user.",
		Fields:       fields,
		CreatePath:   createPath("/api/admin/users"),
		ReadPath:     idPath("/api/admin/users/%s"),
		UpdatePath:   idPath("/api/admin/users/%s"),
		DeletePath:   idPath("/api/admin/users/%s"),
		CreateFields: []string{"username", "display_name", "email", "phone", "telegram_user_id", "slack_user_id", "mattermost_user_id", "password", "active", "is_admin", "group_id", "group_role"},
		UpdateFields: []string{"username", "display_name", "email", "phone", "telegram_user_id", "slack_user_id", "mattermost_user_id", "password", "active", "is_admin", "group_id", "group_role"},
	})
}

func resourceGroupMembership() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("group_id", "Group id."),
		reqIntForceNew("user_id", "User id."),
		optStringDefault("role", "viewer", "Group role: viewer, editor, or user_admin."),
		optBoolDefault("active", true, "Whether the membership is active."),
		computedString("username", "Username."),
		computedString("display_name", "Display name."),
	}
	return crudResource(resourceSpec{
		Description:  "Membership of an existing user in an IncidentRelay group.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/groups/%d/users", "group_id"),
		ReadListPath: fieldListPath("/api/groups/%d/users", "group_id"),
		UpdatePath:   idPath("/api/groups/users/%s"),
		DeletePath:   idPath("/api/groups/users/%s"),
		CreateFields: []string{"user_id", "role"},
		UpdateFields: []string{"role", "active"},
	})
}

func resourceTeam() *schema.Resource {
	fields := []fieldDef{
		reqInt("group_id", "Owner group id."),
		reqString("slug", "Stable team slug."),
		reqString("name", "Human-readable team name."),
		optString("description", "Optional team description."),
		optBoolDefault("escalation_enabled", true, "Enable simple team escalation."),
		optIntDefault("escalation_after_reminders", 2, "Reminder count before simple team escalation."),
		optBoolDefault("active", true, "Whether the team is active."),
		computedString("group_slug", "Owner group slug."),
		computedString("group_name", "Owner group name."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay on-call team.",
		Fields:       fields,
		CreatePath:   createPath("/api/teams"),
		ReadPath:     idPath("/api/teams/%s"),
		UpdatePath:   idPath("/api/teams/%s"),
		DeletePath:   idPath("/api/teams/%s"),
		CreateFields: []string{"group_id", "slug", "name", "description", "escalation_enabled", "escalation_after_reminders", "active"},
		UpdateFields: []string{"group_id", "slug", "name", "description", "escalation_enabled", "escalation_after_reminders", "active"},
	})
}

func resourceTeamMembership() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("team_id", "Team id."),
		reqIntForceNew("user_id", "User id."),
		optStringDefault("role", "viewer", "Team role: viewer, responder, or manager."),
		optBoolDefault("active", true, "Whether the membership is active."),
		computedString("username", "Username."),
		computedString("display_name", "Display name."),
	}
	return crudResource(resourceSpec{
		Description:  "Membership of an existing group user in an IncidentRelay team.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/teams/%d/users", "team_id"),
		ReadListPath: fieldListPath("/api/teams/%d/users", "team_id"),
		UpdatePath:   idPath("/api/teams/users/%s"),
		DeletePath:   idPath("/api/teams/users/%s"),
		CreateFields: []string{"user_id", "role"},
		UpdateFields: []string{"role", "active"},
	})
}

func resourceChannel() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("name", "Channel name."),
		reqString("channel_type", "Channel type, for example email, slack, mattermost, telegram, discord, msteams, webhook."),
		reqJSON("config_json", "config", "Channel-specific JSON configuration."),
		optBoolDefault("enabled", true, "Whether the channel is enabled."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay route-attached outbound notification channel.",
		Fields:       fields,
		CreatePath:   createPath("/api/channels"),
		ReadPath:     idPath("/api/channels/%s"),
		UpdatePath:   idPath("/api/channels/%s"),
		DeletePath:   idPath("/api/channels/%s"),
		CreateFields: []string{"team_id", "name", "channel_type", "config_json", "enabled"},
		UpdateFields: []string{"team_id", "name", "channel_type", "config_json", "enabled"},
	})
}

func resourceRoute() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("name", "Route name."),
		reqString("source", "Incoming alert source: alertmanager, grafana, rmon, zabbix, webhook, sentry, librenms, aws_sns, or heartbeat."),
		optInt("rotation_id", "Rotation id used by this route."),
		optInt("service_id", "Default service id for this route."),
		optInt("escalation_policy_id", "Escalation policy id used by this route."),
		optIntSet("channel_ids", "Channel ids attached directly to this route."),
		optStringDefault("notification_channel_mode", "route_only", "Notification channel mode."),
		optJSONDefault("matchers_json", "matchers", "{}", "Route matcher JSON."),
		optJSONDefault("integration_config_json", "integration_config", "{}", "Provider-specific route integration JSON."),
		optStringSet("group_by", "Alert grouping label names."),
		optBoolDefault("enabled", true, "Whether the route is enabled."),
		computedSensitiveString("intake_token", "One-time route intake token returned on create/regeneration."),
		computedString("intake_token_prefix", "Route intake token prefix."),
		computedBool("has_intake_token", "Whether the route has an intake token."),
		computedString("service_name", "Default service name."),
		computedString("service_slug", "Default service slug."),
		computedString("escalation_mode", "Escalation mode."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay alert route.",
		Fields:       fields,
		CreatePath:   createPath("/api/routes"),
		ReadPath:     idPath("/api/routes/%s"),
		UpdatePath:   idPath("/api/routes/%s"),
		DeletePath:   idPath("/api/routes/%s"),
		CreateFields: []string{"team_id", "name", "source", "rotation_id", "service_id", "escalation_policy_id", "channel_ids", "notification_channel_mode", "matchers_json", "integration_config_json", "group_by", "enabled"},
		UpdateFields: []string{"team_id", "name", "source", "rotation_id", "service_id", "escalation_policy_id", "channel_ids", "notification_channel_mode", "matchers_json", "integration_config_json", "group_by", "enabled"},
	})
}

func resourceRotation() *schema.Resource {
	fields := []fieldDef{
		reqInt("team_id", "Owner team id."),
		reqString("name", "Rotation name."),
		optString("description", "Optional rotation description."),
		reqString("start_at", "Rotation start datetime."),
		optStringDefault("rotation_type", "daily", "Rotation type: daily, weekly, or custom."),
		optIntDefault("interval_value", 1, "Rotation interval value."),
		optStringDefault("interval_unit", "days", "Rotation interval unit."),
		optStringDefault("handoff_time", "09:00", "Local handoff time in HH:MM."),
		optInt("handoff_weekday", "Weekly handoff weekday, Monday is 0."),
		optStringDefault("timezone", "UTC", "Rotation timezone."),
		optInt("duration_seconds", "Custom slot duration in seconds."),
		optIntDefault("reminder_interval_seconds", 300, "Unacknowledged alert reminder interval in seconds. 0 disables reminders."),
		optBoolDefault("enabled", true, "Whether the rotation is enabled."),
		optBoolDefault("add_team_members", true, "On create, add all active team members to the default layer."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay on-call rotation.",
		Fields:       fields,
		CreatePath:   createPath("/api/rotations"),
		ReadPath:     idPath("/api/rotations/%s"),
		UpdatePath:   idPath("/api/rotations/%s"),
		DeletePath:   idPath("/api/rotations/%s"),
		CreateFields: []string{"team_id", "name", "description", "start_at", "rotation_type", "interval_value", "interval_unit", "handoff_time", "handoff_weekday", "timezone", "duration_seconds", "reminder_interval_seconds", "enabled", "add_team_members"},
		UpdateFields: []string{"team_id", "name", "description", "start_at", "rotation_type", "interval_value", "interval_unit", "handoff_time", "handoff_weekday", "timezone", "duration_seconds", "reminder_interval_seconds", "enabled"},
	})
}

func resourceRotationLayer() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("rotation_id", "Parent rotation id."),
		reqString("name", "Layer name."),
		optString("description", "Optional layer description."),
		optIntDefault("priority", 0, "Higher priority active layer wins."),
		optString("start_at", "Layer start datetime."),
		optString("rotation_type", "Layer rotation type."),
		optInt("interval_value", "Layer interval value."),
		optString("interval_unit", "Layer interval unit."),
		optString("handoff_time", "Local handoff time in HH:MM."),
		optInt("handoff_weekday", "Weekly handoff weekday, Monday is 0."),
		optString("timezone", "Layer timezone."),
		optInt("duration_seconds", "Layer slot duration in seconds."),
		optBoolDefault("enabled", true, "Whether the layer is enabled."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay rotation schedule layer.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/rotations/%d/layers", "rotation_id"),
		ReadListPath: fieldListPath("/api/rotations/%d/layers", "rotation_id"),
		UpdatePath:   idPath("/api/rotations/layers/%s"),
		DeletePath:   idPath("/api/rotations/layers/%s"),
		CreateFields: []string{"name", "description", "priority", "start_at", "rotation_type", "interval_value", "interval_unit", "handoff_time", "handoff_weekday", "timezone", "duration_seconds", "enabled"},
		UpdateFields: []string{"name", "description", "priority", "start_at", "rotation_type", "interval_value", "interval_unit", "handoff_time", "handoff_weekday", "timezone", "duration_seconds", "enabled"},
	})
}

func resourceRotationLayerMember() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("layer_id", "Rotation layer id."),
		reqIntForceNew("user_id", "User id."),
		reqInt("position", "Position in the layer rotation order."),
		optBoolDefault("active", true, "Whether the membership period is active."),
		optString("starts_at", "Optional membership period start."),
		computedString("username", "Username."),
		computedString("display_name", "Display name."),
		computedString("ends_at", "Membership period end."),
	}
	return crudResource(resourceSpec{
		Description:  "IncidentRelay rotation layer member period.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/rotations/layers/%d/members", "layer_id"),
		ReadListPath: fieldListPath("/api/rotations/layers/%d/members", "layer_id"),
		UpdatePath:   idPath("/api/rotations/layers/members/%s"),
		DeletePath:   idPath("/api/rotations/layers/members/%s"),
		CreateFields: []string{"user_id", "position", "starts_at"},
		UpdateFields: []string{"position", "active"},
	})
}

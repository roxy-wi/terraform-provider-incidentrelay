package incidentrelay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type datasourceSpec struct {
	Description  string
	Endpoint     string
	EndpointFunc func(*schema.ResourceData) string
	Fields       []fieldDef
	SearchFields []string
	ReadFields   []string
}

func datasourceVersion() *schema.Resource {
	fields := []fieldDef{
		computedString("service_version", "IncidentRelay service version."),
		computedJSON("migrations_json", "migrations", "Migration status JSON."),
	}
	return &schema.Resource{
		Description: "IncidentRelay version information.",
		ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			client := m.(*Config).Client
			var response map[string]interface{}
			if err := client.Do(ctx, http.MethodGet, "/api/version", nil, &response); err != nil {
				return diag.FromErr(err)
			}
			d.SetId("version")
			if err := setFieldsFromResponse(d, fields, []string{"service_version", "migrations_json"}, response); err != nil {
				return diag.FromErr(err)
			}
			return nil
		},
		Schema: schemaFromFields(fields),
	}
}

func datasourceGroup() *schema.Resource {
	fields := []fieldDef{
		optInt("group_id", "Group id."),
		optString("slug", "Group slug."),
		optString("name", "Group name."),
		optString("description", "Group description."),
		optBoolDefault("active", true, "Whether the group is active."),
	}
	return listDatasource(datasourceSpec{
		Description:  "Look up an IncidentRelay group.",
		Endpoint:     "/api/groups",
		Fields:       fields,
		SearchFields: []string{"group_id", "slug", "name"},
		ReadFields:   []string{"slug", "name", "description", "active"},
	})
}

func datasourceTeam() *schema.Resource {
	fields := []fieldDef{
		optInt("team_id", "Team id."),
		optInt("group_id", "Owner group id."),
		optString("slug", "Team slug."),
		optString("name", "Team name."),
		optString("description", "Team description."),
		computedString("group_slug", "Owner group slug."),
		computedString("group_name", "Owner group name."),
	}
	return listDatasource(datasourceSpec{
		Description:  "Look up an IncidentRelay team.",
		Endpoint:     "/api/teams?include_inactive=1",
		Fields:       fields,
		SearchFields: []string{"team_id", "group_id", "slug", "name"},
		ReadFields:   []string{"group_id", "slug", "name", "description", "group_slug", "group_name"},
	})
}

func datasourceUser() *schema.Resource {
	fields := []fieldDef{
		optInt("user_id", "User id."),
		optString("username", "Username."),
		optString("display_name", "Display name."),
		optString("email", "Email address."),
		optString("phone", "Phone number."),
		optString("telegram_user_id", "Telegram user ID."),
		optString("slack_user_id", "Slack user ID."),
		optString("mattermost_user_id", "Mattermost user ID."),
	}
	return listDatasource(datasourceSpec{
		Description:  "Look up an IncidentRelay user visible to the current principal.",
		Endpoint:     "/api/users?all=1",
		Fields:       fields,
		SearchFields: []string{"user_id", "username", "email"},
		ReadFields:   []string{"username", "display_name", "email", "phone", "telegram_user_id", "slack_user_id", "mattermost_user_id"},
	})
}

func datasourceChannel() *schema.Resource {
	fields := []fieldDef{
		{Name: "channel_id", APIName: "id", Kind: kindInt, Optional: true, Description: "Notification channel id."},
		optInt("group_id", "Owner group id."),
		optString("group_slug", "Owner group slug."),
		optInt("team_id", "Owner team id."),
		optString("team_slug", "Owner team slug."),
		optString("name", "Notification channel name."),
		optString("channel_type", "Notification channel type."),
		computedString("group_name", "Owner group name."),
		computedString("team_name", "Owner team name."),
		computedBool("enabled", "Whether the notification channel is enabled."),
	}

	dataSource := listDatasource(datasourceSpec{
		Description:  "Look up an existing IncidentRelay notification channel.",
		Endpoint:     "/api/channels",
		Fields:       fields,
		SearchFields: []string{"channel_id", "group_id", "group_slug", "team_id", "team_slug", "name", "channel_type"},
		ReadFields:   []string{"channel_id", "group_id", "group_slug", "group_name", "team_id", "team_slug", "team_name", "name", "channel_type", "enabled"},
	})

	dataSource.Schema["channel_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["group_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["team_id"].ValidateFunc = validation.IntAtLeast(1)
	return dataSource
}

func datasourceService() *schema.Resource {
	fields := []fieldDef{
		optInt("service_id", "Service id."),
		optInt("team_id", "Owner team id."),
		optString("slug", "Service slug."),
		optString("name", "Service name."),
		optString("description", "Service description."),
		computedString("team_name", "Team name."),
		computedString("team_slug", "Team slug."),
		computedInt("group_id", "Owner group id."),
		computedInt("default_rotation_id", "Default rotation id."),
		computedString("default_rotation_name", "Default rotation name."),
		computedInt("default_escalation_policy_id", "Default escalation policy id."),
		computedString("default_escalation_policy_name", "Default escalation policy name."),
		computedInt("notification_policy_id", "Assigned notification policy id."),
		computedString("notification_policy_name", "Assigned notification policy name."),
		computedInt("priority_policy_id", "Assigned incident priority policy id."),
		computedString("priority_policy_name", "Assigned incident priority policy name."),
	}
	return listDatasource(datasourceSpec{
		Description:  "Look up an IncidentRelay service.",
		Endpoint:     "/api/services?include_disabled=1",
		Fields:       fields,
		SearchFields: []string{"service_id", "team_id", "slug", "name"},
		ReadFields:   []string{"team_id", "slug", "name", "description", "team_name", "team_slug", "group_id", "default_rotation_id", "default_rotation_name", "default_escalation_policy_id", "default_escalation_policy_name", "notification_policy_id", "notification_policy_name", "priority_policy_id", "priority_policy_name"},
	})
}

func datasourceRotation() *schema.Resource {
	fields := []fieldDef{
		{Name: "rotation_id", APIName: "id", Kind: kindInt, Optional: true, Description: "Rotation id."},
		optInt("team_id", "Owner team id."),
		optString("team_slug", "Owner team slug."),
		optString("name", "Rotation name."),
		computedString("team_name", "Owner team name."),
		computedString("description", "Rotation description."),
		computedString("start_at", "Rotation start datetime."),
		computedInt("duration_seconds", "Custom slot duration in seconds."),
		computedInt("reminder_interval_seconds", "Unacknowledged alert reminder interval in seconds."),
		computedString("rotation_type", "Rotation type."),
		computedInt("interval_value", "Rotation interval value."),
		computedString("interval_unit", "Rotation interval unit."),
		computedString("handoff_time", "Local handoff time."),
		computedInt("handoff_weekday", "Weekly handoff weekday."),
		computedString("timezone", "Rotation timezone."),
		computedBool("enabled", "Whether the rotation is enabled."),
		computedString("current_oncall", "Current on-call username."),
	}

	dataSource := listDatasource(datasourceSpec{
		Description:  "Look up an existing IncidentRelay on-call rotation.",
		Endpoint:     "/api/rotations",
		Fields:       fields,
		SearchFields: []string{"rotation_id", "team_id", "team_slug", "name"},
		ReadFields:   []string{"rotation_id", "team_id", "team_slug", "team_name", "name", "description", "start_at", "duration_seconds", "reminder_interval_seconds", "rotation_type", "interval_value", "interval_unit", "handoff_time", "handoff_weekday", "timezone", "enabled", "current_oncall"},
	})

	dataSource.Schema["rotation_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["team_id"].ValidateFunc = validation.IntAtLeast(1)
	return dataSource
}

func datasourceIncidentPriority() *schema.Resource {
	fields := []fieldDef{
		{Name: "priority_id", APIName: "id", Kind: kindInt, Optional: true, Description: "Incident priority id."},
		optString("slug", "Incident priority slug, such as p1."),
		optString("name", "Incident priority name."),
		optInt("level", "Incident priority numeric level."),
		computedString("description", "Incident priority description."),
		computedString("color", "Incident priority display color."),
		computedBool("enabled", "Whether the incident priority is enabled."),
		computedBool("default", "Whether this is the default incident priority."),
	}

	dataSource := listDatasource(datasourceSpec{
		Description:  "Look up an IncidentRelay incident priority.",
		Endpoint:     "/api/incidents/priorities?include_disabled=1",
		Fields:       fields,
		SearchFields: []string{"priority_id", "slug", "name", "level"},
		ReadFields:   []string{"priority_id", "slug", "name", "description", "level", "color", "enabled", "default"},
	})

	dataSource.Schema["priority_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["level"].ValidateFunc = validation.IntAtLeast(1)
	return dataSource
}

func datasourceEscalationPolicy() *schema.Resource {
	fields := []fieldDef{
		{Name: "policy_id", APIName: "id", Kind: kindInt, Optional: true, Description: "Escalation policy id."},
		optInt("group_id", "Owner group id."),
		optString("group_slug", "Owner group slug."),
		optInt("team_id", "Owner team id."),
		optString("team_slug", "Owner team slug."),
		optString("name", "Escalation policy name."),
		computedString("team_name", "Owner team name."),
		computedString("description", "Escalation policy description."),
		computedBool("enabled", "Whether the escalation policy is enabled."),
		computedInt("repeat_count", "Number of additional full rule-chain repeats."),
	}

	dataSource := listDatasource(datasourceSpec{
		Description:  "Look up an existing IncidentRelay escalation policy.",
		Endpoint:     "/api/escalation-policies",
		Fields:       fields,
		SearchFields: []string{"policy_id", "group_id", "group_slug", "team_id", "team_slug", "name"},
		ReadFields:   []string{"policy_id", "group_id", "group_slug", "team_id", "team_slug", "team_name", "name", "description", "enabled", "repeat_count"},
	})

	dataSource.Schema["policy_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["group_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["team_id"].ValidateFunc = validation.IntAtLeast(1)
	return dataSource
}

func datasourceNotificationPolicy() *schema.Resource {
	fields := []fieldDef{
		{Name: "policy_id", APIName: "id", Kind: kindInt, Optional: true, Description: "Notification policy id."},
		optInt("team_id", "Owner team id."),
		optString("team_slug", "Owner team slug."),
		optString("name", "Notification policy name."),
		computedString("team_name", "Owner team name."),
		computedString("description", "Notification policy description."),
		computedBool("enabled", "Whether the notification policy is enabled."),
		computedInt("rules_count", "Number of active notification policy rules."),
		computedInt("services_count", "Number of services using this notification policy."),
	}

	dataSource := listDatasource(datasourceSpec{
		Description:  "Look up an existing IncidentRelay notification policy.",
		Endpoint:     "/api/notification-policies",
		Fields:       fields,
		SearchFields: []string{"policy_id", "team_id", "team_slug", "name"},
		ReadFields:   []string{"policy_id", "team_id", "team_slug", "team_name", "name", "description", "enabled", "rules_count", "services_count"},
	})

	dataSource.Schema["policy_id"].ValidateFunc = validation.IntAtLeast(1)
	dataSource.Schema["team_id"].ValidateFunc = validation.IntAtLeast(1)
	return dataSource
}

func datasourceServiceMatchRule() *schema.Resource {
	fields := []fieldDef{
		{Name: "match_rule_id", APIName: "id", Kind: kindInt, Optional: true, Description: "Service match rule id."},
		optInt("team_id", "Owner team id used to scope the API lookup."),
		optInt("route_id", "Optional route id used to scope the API lookup."),
		optInt("service_id", "Target service id used to scope the API lookup."),
		optString("name", "Service match rule name."),
		computedString("team_slug", "Owner team slug."),
		computedString("team_name", "Owner team name."),
		computedString("route_name", "Route name."),
		computedString("service_slug", "Target service slug."),
		computedString("service_name", "Target service name."),
		computedInt("position", "Rule evaluation position."),
		computedString("description", "Service match rule description."),
		computedInt("matcher_preset_id", "Assigned matcher preset id."),
		computedJSON("matchers_json", "matchers", "Matcher JSON evaluated against alerts."),
		computedBool("enabled", "Whether the service match rule is enabled."),
	}

	dataSource := listDatasource(datasourceSpec{
		Description: "Look up an existing IncidentRelay service match rule.",
		EndpointFunc: func(data *schema.ResourceData) string {
			return queryPath("/api/services/match-rules", map[string]string{
				"team_id":    "team_id",
				"route_id":   "route_id",
				"service_id": "service_id",
			}, data)
		},
		Fields:       fields,
		SearchFields: []string{"match_rule_id", "team_id", "route_id", "service_id", "name"},
		ReadFields:   []string{"match_rule_id", "team_id", "team_slug", "team_name", "route_id", "route_name", "service_id", "service_slug", "service_name", "position", "name", "description", "matcher_preset_id", "matchers_json", "enabled"},
	})

	scopeFields := []string{"team_id", "route_id", "service_id"}
	for _, fieldName := range scopeFields {
		dataSource.Schema[fieldName].AtLeastOneOf = scopeFields
		dataSource.Schema[fieldName].ValidateFunc = validation.IntAtLeast(1)
	}
	dataSource.Schema["match_rule_id"].ValidateFunc = validation.IntAtLeast(1)
	read := dataSource.ReadContext
	dataSource.ReadContext = func(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
		for _, fieldName := range scopeFields {
			if value, ok := data.GetOkExists(fieldName); ok && !isZeroValue(value) {
				return read(ctx, data, meta)
			}
		}
		return diag.Errorf("at least one API scope field is required: team_id, route_id, or service_id")
	}
	return dataSource
}

func listDatasource(spec datasourceSpec) *schema.Resource {
	return &schema.Resource{
		Description: spec.Description,
		ReadContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return readListDatasource(ctx, d, m, spec)
		},
		Schema: schemaFromFields(spec.Fields),
	}
}

func readListDatasource(ctx context.Context, d *schema.ResourceData, m interface{}, spec datasourceSpec) diag.Diagnostics {
	client := m.(*Config).Client
	criteria := make(map[string]interface{})

	for _, field := range spec.SearchFields {
		value, ok := d.GetOkExists(field)
		if ok && !isZeroValue(value) {
			criteria[field] = value
		}
	}
	if len(criteria) == 0 {
		return diag.Errorf("at least one lookup field is required")
	}

	var response interface{}
	endpoint := spec.Endpoint
	if spec.EndpointFunc != nil {
		endpoint = spec.EndpointFunc(d)
	}
	if err := client.Do(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return diag.FromErr(err)
	}

	items, err := extractItems(response)
	if err != nil {
		return diag.FromErr(err)
	}

	fieldByName := make(map[string]fieldDef, len(spec.Fields))
	for _, field := range spec.Fields {
		fieldByName[field.Name] = field
	}

	matches := make([]map[string]interface{}, 0)
	for _, item := range items {
		if itemMatchesCriteria(item, criteria, fieldByName) {
			matches = append(matches, item)
		}
	}

	if len(matches) == 0 {
		return diag.FromErr(ErrNotFound)
	}
	if len(matches) > 1 {
		return diag.Errorf("lookup returned %d matches; add more filters", len(matches))
	}

	item := matches[0]
	id, err := interfaceToID(item["id"])
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	if err := setFieldsFromResponse(d, spec.Fields, spec.ReadFields, item); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func itemMatchesCriteria(item map[string]interface{}, criteria map[string]interface{}, fields map[string]fieldDef) bool {
	for fieldName, expected := range criteria {
		apiName := fieldName
		if fieldName == "group_id" || fieldName == "team_id" || fieldName == "user_id" || fieldName == "service_id" {
			apiName = "id"
			if _, ok := item[fieldName]; ok {
				apiName = fieldName
			}
		}
		if field, ok := fields[fieldName]; ok && field.APIName != "" {
			apiName = field.APIName
		}

		actual, ok := item[apiName]
		if !ok {
			return false
		}
		if !valuesEqual(actual, expected) {
			return false
		}
	}
	return true
}

func isZeroValue(value interface{}) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case int:
		return v == 0
	case bool:
		return !v
	case *schema.Set:
		return v.Len() == 0
	default:
		return false
	}
}

func dataSourceNotFound(name string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, name)
}

func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func stringID(id int) string {
	return strconv.Itoa(id)
}

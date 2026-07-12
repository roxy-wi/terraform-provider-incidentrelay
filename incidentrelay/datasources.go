package incidentrelay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type datasourceSpec struct {
	Description  string
	Endpoint     string
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
	}
	return listDatasource(datasourceSpec{
		Description:  "Look up an IncidentRelay service.",
		Endpoint:     "/api/services?include_disabled=1",
		Fields:       fields,
		SearchFields: []string{"service_id", "team_id", "slug", "name"},
		ReadFields:   []string{"team_id", "slug", "name", "description", "team_name", "team_slug", "group_id"},
	})
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
	if err := client.Do(ctx, http.MethodGet, spec.Endpoint, nil, &response); err != nil {
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
			if fieldName == "group_id" {
				if _, ok := item["group_id"]; ok {
					apiName = "group_id"
				}
			}
			if fieldName == "team_id" {
				if _, ok := item["team_id"]; ok {
					apiName = "team_id"
				}
			}
			if fieldName == "user_id" {
				apiName = "id"
			}
			if fieldName == "service_id" {
				apiName = "id"
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

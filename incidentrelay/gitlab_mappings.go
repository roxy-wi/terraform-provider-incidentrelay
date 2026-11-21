package incidentrelay

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type gitlabMapping struct {
	ProjectID   int64   `json:"project_id"`
	Environment string  `json:"environment"`
	ServiceIDs  []int64 `json:"service_ids"`
}

func parseGitLabMappings(raw string) ([]gitlabMapping, error) {
	var mappings []gitlabMapping
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&mappings); err != nil {
		return nil, fmt.Errorf("invalid GitLab mappings JSON: %w", err)
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("GitLab mappings must contain a single JSON array")
	}
	if mappings == nil || len(mappings) > 100 {
		return nil, fmt.Errorf("GitLab mappings must be an array of at most 100 rules")
	}
	seen := make(map[string]bool, len(mappings))
	for _, mapping := range mappings {
		if mapping.ProjectID < 1 || mapping.ProjectID > 9007199254740991 {
			return nil, fmt.Errorf("GitLab project_id must be an integer between 1 and 9007199254740991")
		}
		if !validTrimmedText(mapping.Environment, 1, 255) {
			return nil, fmt.Errorf("GitLab environment must be 1–255 characters with no surrounding whitespace")
		}
		if len(mapping.ServiceIDs) < 1 || len(mapping.ServiceIDs) > 100 {
			return nil, fmt.Errorf("GitLab service_ids must contain 1–100 positive integer IDs")
		}
		for _, id := range mapping.ServiceIDs {
			if id < 1 {
				return nil, fmt.Errorf("GitLab service_ids must contain positive integers")
			}
		}
		key := fmt.Sprintf("%d/%s", mapping.ProjectID, mapping.Environment)
		if seen[key] {
			return nil, fmt.Errorf("duplicate GitLab project/environment mapping")
		}
		seen[key] = true
	}
	return mappings, nil
}

func validateGitLabMappings(value interface{}, key string) ([]string, []error) {
	if _, err := parseGitLabMappings(value.(string)); err != nil {
		return nil, []error{fmt.Errorf("%s: %w", key, err)}
	}
	return nil, nil
}

func gitlabMappingsState(value interface{}) string {
	raw := value.(string)
	var mappings interface{}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&mappings); err != nil {
		return raw
	}
	encoded, err := json.Marshal(mappings)
	if err != nil {
		return raw
	}
	return string(encoded)
}

// Use raw configuration so an omitted field stays under UX control after refresh.
func addGitLabMappings(d *schema.ResourceData, body map[string]interface{}) error {
	raw := d.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() || raw.GetAttr("gitlab_mappings_json").IsNull() {
		return nil
	}
	if d.Id() != "" && !d.HasChange("gitlab_mappings_json") {
		return nil
	}
	mappings, err := parseGitLabMappings(d.Get("gitlab_mappings_json").(string))
	if err != nil {
		return err
	}
	if len(mappings) > 0 && d.Get("ordering_mode") != "timestamp" {
		return fmt.Errorf("GitLab mappings require timestamp ordering")
	}
	body["gitlab_mappings"] = mappings
	return nil
}

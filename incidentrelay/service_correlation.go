package incidentrelay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The effective values are readable after import, but only configured values
// belong in a PUT. This also keeps IR versions before 2.5 usable.
func serviceCorrelationPayload(d *schema.ResourceData, payload map[string]interface{}) {
	raw := d.GetRawConfig()
	for _, field := range []string{"change_correlation_enabled", "change_correlation_exclusions_json"} {
		if raw.IsNull() || !raw.IsKnown() || raw.GetAttr(field).IsNull() {
			delete(payload, strings.TrimSuffix(field, "_json"))
		} else if field == "change_correlation_enabled" {
			payload[field] = d.Get(field).(bool)
		}
	}
}

func validateCorrelationExclusions(input interface{}, key string) ([]string, []error) {
	var rules []map[string]interface{}
	if err := json.Unmarshal([]byte(input.(string)), &rules); err != nil || rules == nil || len(rules) > 50 {
		return nil, []error{fmt.Errorf("%s must be a JSON array of at most 50 rules", key)}
	}
	for i, rule := range rules {
		var matches bool
		for field, value := range rule {
			switch field {
			case "source":
				if value == nil {
					continue
				}
				source, ok := value.(string)
				if !ok || !validTrimmedText(source, 1, 255) {
					return nil, []error{fmt.Errorf("%s[%d].source must be a nonempty source of at most 255 characters", key, i)}
				}
				matches = true
			case "labels":
				labels, ok := value.(map[string]interface{})
				if !ok || len(labels) > 50 {
					return nil, []error{fmt.Errorf("%s[%d].labels must be an object with at most 50 labels", key, i)}
				}
				for name, v := range labels {
					s, ok := v.(string)
					if !ok || !validTrimmedText(name, 1, 255) || !validTrimmedText(s, 0, 1024) {
						return nil, []error{fmt.Errorf("%s[%d] contains an invalid label name or value", key, i)}
					}
				}
				matches = matches || len(labels) > 0
			default:
				return nil, []error{fmt.Errorf("%s[%d]: unknown exclusion field %q", key, i, field)}
			}
		}
		if !matches {
			return nil, []error{fmt.Errorf("%s[%d] requires a source or at least one label", key, i)}
		}
	}
	return nil, nil
}

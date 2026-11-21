package incidentrelay

import (
	"strings"
	"testing"
)

func TestGitLabMappingsValidation(t *testing.T) {
	valid := `[{"project_id":9007199254740991,"environment":"production","service_ids":[1,2]}]`
	for _, value := range []string{"[]", valid} {
		if _, err := parseGitLabMappings(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []string{
		"null", "{}", valid + "[]",
		strings.Replace(valid, "9007199254740991", "9007199254740992", 1),
		strings.Replace(valid, "9007199254740991", "1.5", 1),
		strings.Replace(valid, "[1,2]", "[]", 1),
		strings.Replace(valid, "[1,2]", "[0]", 1),
		strings.Replace(valid, "production", " production", 1),
		strings.Replace(valid, `"project_id"`, `"unknown"`, 1),
		"[" + strings.Trim(valid, "[]") + "," + strings.Trim(valid, "[]") + "]",
	} {
		if _, err := parseGitLabMappings(value); err == nil {
			t.Errorf("accepted %s", value)
		}
	}
}

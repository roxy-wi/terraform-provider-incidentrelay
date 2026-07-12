package incidentrelay

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestNormalizeJSONString(t *testing.T) {
	got, err := normalizeJSONString(` { "b": 2, "a": 1 } `)
	if err != nil {
		t.Fatalf("normalizeJSONString returned error: %v", err)
	}
	if want := `{"a":1,"b":2}`; got != want {
		t.Fatalf("normalizeJSONString = %q, want %q", got, want)
	}

	if got, err := normalizeJSONString("   "); err != nil || got != "" {
		t.Fatalf("empty normalizeJSONString = %q, %v; want empty nil", got, err)
	}

	if got, err := normalizeJSONString("{broken"); err == nil || got != "{broken" {
		t.Fatalf("invalid normalizeJSONString = %q, %v; want original with error", got, err)
	}
}

func TestInterfaceToID(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    string
		wantErr bool
	}{
		{name: "string", value: "abc", want: "abc"},
		{name: "int", value: 42, want: "42"},
		{name: "int64", value: int64(43), want: "43"},
		{name: "float64", value: float64(44), want: "44"},
		{name: "json number", value: json.Number("45"), want: "45"},
		{name: "empty string", value: " ", wantErr: true},
		{name: "unsupported", value: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interfaceToID(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("interfaceToID returned nil error")
				}
				return
			}
			if err != nil {
				t.Fatalf("interfaceToID returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("interfaceToID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPayloadConvertsFieldKinds(t *testing.T) {
	fields := []fieldDef{
		reqString("name", "Name."),
		optJSONDefault("labels_json", "labels", "{}", "Labels."),
		optStringSet("tags", "Tags."),
		optIntSet("numbers", "Numbers."),
		optBoolDefault("enabled", true, "Enabled."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"name":        "service",
		"labels_json": `{"b":2,"a":1}`,
		"tags":        []interface{}{"prod", "api"},
		"numbers":     []interface{}{2, 1},
		"enabled":     false,
	})

	payload, err := buildPayload(data, fields, []string{"name", "labels_json", "tags", "numbers", "enabled"}, false)
	if err != nil {
		t.Fatalf("buildPayload returned error: %v", err)
	}

	if got, want := payload["name"], "service"; got != want {
		t.Fatalf("payload[name] = %v, want %v", got, want)
	}
	if got, want := payload["enabled"], false; got != want {
		t.Fatalf("payload[enabled] = %v, want %v", got, want)
	}
	if got, want := payload["labels"], map[string]interface{}{"a": float64(1), "b": float64(2)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("payload[labels] = %#v, want %#v", got, want)
	}
	if !sameElements(payload["tags"].([]interface{}), []interface{}{"prod", "api"}) {
		t.Fatalf("payload[tags] = %#v, want prod/api", payload["tags"])
	}
	if !sameElements(payload["numbers"].([]interface{}), []interface{}{1, 2}) {
		t.Fatalf("payload[numbers] = %#v, want 1/2", payload["numbers"])
	}
}

func TestBuildPayloadRejectsInvalidJSON(t *testing.T) {
	fields := []fieldDef{
		reqJSON("config_json", "config", "Config."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"config_json": "{broken",
	})

	if _, err := buildPayload(data, fields, []string{"config_json"}, false); err == nil {
		t.Fatal("buildPayload invalid JSON returned nil error")
	}
}

func TestSetFieldsFromResponseConvertsFieldKinds(t *testing.T) {
	fields := []fieldDef{
		reqString("name", "Name."),
		computedInt("count", "Count."),
		computedBool("enabled", "Enabled."),
		computedJSON("labels_json", "labels", "Labels."),
		optStringSet("tags", "Tags."),
		optIntSet("numbers", "Numbers."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{})

	err := setFieldsFromResponse(data, fields, []string{"name", "count", "enabled", "labels_json", "tags", "numbers"}, map[string]interface{}{
		"name":    "service",
		"count":   "3",
		"enabled": "true",
		"labels":  map[string]interface{}{"b": float64(2), "a": float64(1)},
		"tags":    []interface{}{"prod", "api"},
		"numbers": []interface{}{float64(2), "1"},
	})
	if err != nil {
		t.Fatalf("setFieldsFromResponse returned error: %v", err)
	}

	if got, want := data.Get("name"), "service"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if got, want := data.Get("count"), 3; got != want {
		t.Fatalf("count = %v, want %v", got, want)
	}
	if got, want := data.Get("enabled"), true; got != want {
		t.Fatalf("enabled = %v, want %v", got, want)
	}
	if got, want := data.Get("labels_json"), `{"a":1,"b":2}`; got != want {
		t.Fatalf("labels_json = %v, want %v", got, want)
	}
	assertSetElements(t, data.Get("tags").(*schema.Set), []interface{}{"api", "prod"})
	assertSetElements(t, data.Get("numbers").(*schema.Set), []interface{}{1, 2})
}

func sameElements(got []interface{}, want []interface{}) bool {
	if len(got) != len(want) {
		return false
	}
	remaining := make(map[interface{}]int, len(want))
	for _, item := range want {
		remaining[item]++
	}
	for _, item := range got {
		if remaining[item] == 0 {
			return false
		}
		remaining[item]--
	}
	return true
}

func assertSetElements(t *testing.T, got *schema.Set, want []interface{}) {
	t.Helper()
	if !sameElements(got.List(), want) {
		t.Fatalf("set = %#v, want %#v", got.List(), want)
	}
}

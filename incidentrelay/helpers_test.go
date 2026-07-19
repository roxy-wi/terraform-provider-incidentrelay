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

func TestNormalizeJSONStringCanonicalForms(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "nested objects and arrays",
			raw: `{
				"z": [3, {"b": true, "a": false}],
				"a": {"b": 2, "a": 1}
			}`,
			want: `{"a":{"a":1,"b":2},"z":[3,{"a":false,"b":true}]}`,
		},
		{
			name: "array order is preserved",
			raw:  `[{"b":2,"a":1},{"d":4,"c":3}]`,
			want: `[{"a":1,"b":2},{"c":3,"d":4}]`,
		},
		{
			name: "scalar json is compacted",
			raw:  ` " keep spacing inside string " `,
			want: `" keep spacing inside string "`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeJSONString(tt.raw)
			if err != nil {
				t.Fatalf("normalizeJSONString returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeJSONString = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeJSONStringState(t *testing.T) {
	if got, want := normalizeJSONStringState(` { "b": 2, "a": 1 } `), `{"a":1,"b":2}`; got != want {
		t.Fatalf("normalizeJSONStringState valid = %q, want %q", got, want)
	}
	if got, want := normalizeJSONStringState("{broken"), "{broken"; got != want {
		t.Fatalf("normalizeJSONStringState invalid = %q, want original %q", got, want)
	}
	if got := normalizeJSONStringState(map[string]interface{}{"a": 1}); got != "" {
		t.Fatalf("normalizeJSONStringState non-string = %q, want empty", got)
	}
}

func TestJSONStringToValue(t *testing.T) {
	if got, err := jsonStringToValue(" \n\t "); err != nil || got != nil {
		t.Fatalf("jsonStringToValue empty = %#v, %v; want nil nil", got, err)
	}

	got, err := jsonStringToValue(` { "items": [ {"b": 2, "a": 1 } ], "enabled": true } `)
	if err != nil {
		t.Fatalf("jsonStringToValue object returned error: %v", err)
	}
	want := map[string]interface{}{
		"enabled": true,
		"items": []interface{}{
			map[string]interface{}{"a": float64(1), "b": float64(2)},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("jsonStringToValue object = %#v, want %#v", got, want)
	}

	if _, err := jsonStringToValue("{broken"); err == nil {
		t.Fatal("jsonStringToValue invalid returned nil error")
	}
}

func TestRestoreMaskedJSONValues(t *testing.T) {
	current := `{
		"mode": "bot_api",
		"connection_mode": "socket_mode",
		"bot_token": "xoxb-current",
		"app_token": "xapp-current",
		"signing_secret": "signing-current",
		"channel_id": "C123",
		"nested": [{"secret": "keep-me"}]
	}`
	remote := map[string]interface{}{
		"mode":            "bot_api",
		"connection_mode": "socket_mode",
		"bot_token":       incidentRelaySecretPlaceholder,
		"app_token":       incidentRelaySecretPlaceholder,
		"signing_secret":  incidentRelaySecretPlaceholder,
		"channel_id":      "C456",
		"nested": []interface{}{
			map[string]interface{}{"secret": incidentRelaySecretPlaceholder},
		},
	}

	got, err := restoreMaskedJSONValues(current, remote, incidentRelaySecretPlaceholder)
	if err != nil {
		t.Fatalf("restoreMaskedJSONValues returned error: %v", err)
	}

	want := map[string]interface{}{
		"mode":            "bot_api",
		"connection_mode": "socket_mode",
		"bot_token":       "xoxb-current",
		"app_token":       "xapp-current",
		"signing_secret":  "signing-current",
		"channel_id":      "C456",
		"nested": []interface{}{
			map[string]interface{}{"secret": "keep-me"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restored JSON = %#v, want %#v", got, want)
	}
}

func TestRestoreMaskedJSONValuesWithoutCurrentState(t *testing.T) {
	remote := map[string]interface{}{
		"bot_token": incidentRelaySecretPlaceholder,
	}

	got, err := restoreMaskedJSONValues("", remote, incidentRelaySecretPlaceholder)
	if err != nil {
		t.Fatalf("restoreMaskedJSONValues returned error: %v", err)
	}
	if !reflect.DeepEqual(got, remote) {
		t.Fatalf("restored JSON = %#v, want unchanged %#v", got, remote)
	}
}

func TestValueToJSONStringCanonicalizesAPIValues(t *testing.T) {
	if got, err := valueToJSONString(nil); err != nil || got != "" {
		t.Fatalf("valueToJSONString nil = %q, %v; want empty nil", got, err)
	}

	got, err := valueToJSONString(map[string]interface{}{
		"z": []interface{}{
			map[string]interface{}{"b": false, "a": true},
		},
		"a": map[string]interface{}{"b": float64(2), "a": float64(1)},
	})
	if err != nil {
		t.Fatalf("valueToJSONString returned error: %v", err)
	}
	if want := `{"a":{"a":1,"b":2},"z":[{"a":true,"b":false}]}`; got != want {
		t.Fatalf("valueToJSONString = %q, want %q", got, want)
	}
}

func TestSchemaJSONStateFuncNormalizesConfig(t *testing.T) {
	fields := []fieldDef{
		reqJSON("config_json", "config", "Config."),
		computedJSON("computed_json", "computed", "Computed."),
	}
	schemaMap := schemaFromFields(fields)

	stateFunc := schemaMap["config_json"].StateFunc
	if stateFunc == nil {
		t.Fatal("config_json StateFunc is nil")
	}
	if got, want := stateFunc(` { "b": 2, "a": 1 } `), `{"a":1,"b":2}`; got != want {
		t.Fatalf("config_json StateFunc = %q, want %q", got, want)
	}
	if got, want := stateFunc("{broken"), "{broken"; got != want {
		t.Fatalf("config_json StateFunc invalid = %q, want original %q", got, want)
	}
	if schemaMap["computed_json"].StateFunc != nil {
		t.Fatal("computed_json StateFunc is set")
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

func TestBuildPayloadOmitsUnsetOptionalInt(t *testing.T) {
	fields := []fieldDef{
		reqString("name", "Name."),
		optInt("rotation_id", "Rotation id."),
		optIntDefault("public_order", 100, "Public order."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"name": "route",
	})

	payload, err := buildPayload(data, fields, []string{"name", "rotation_id", "public_order"}, false)
	if err != nil {
		t.Fatalf("buildPayload returned error: %v", err)
	}

	if _, ok := payload["rotation_id"]; ok {
		t.Fatalf("payload[rotation_id] = %#v, want omitted", payload["rotation_id"])
	}
	if got, want := payload["public_order"], 100; got != want {
		t.Fatalf("payload[public_order] = %#v, want %#v", got, want)
	}
}

func TestBuildPayloadClearsChangedOptionalIntOnUpdate(t *testing.T) {
	fields := []fieldDef{
		reqString("name", "Name."),
		optInt("rotation_id", "Rotation id."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"name":        "route",
		"rotation_id": 42,
	})
	if err := data.Set("rotation_id", nil); err != nil {
		t.Fatalf("set rotation_id: %v", err)
	}

	payload, err := buildPayload(data, fields, []string{"name", "rotation_id"}, true)
	if err != nil {
		t.Fatalf("buildPayload returned error: %v", err)
	}

	value, ok := payload["rotation_id"]
	if !ok {
		t.Fatal("payload[rotation_id] missing, want explicit nil")
	}
	if value != nil {
		t.Fatalf("payload[rotation_id] = %#v, want nil", value)
	}
}

func TestBuildPayloadOmitsUnsetOptionalString(t *testing.T) {
	fields := []fieldDef{
		reqString("name", "Name."),
		optString("public_name", "Public name."),
		optStringDefault("status", "operational", "Status."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"name": "service",
	})

	payload, err := buildPayload(data, fields, []string{"name", "public_name", "status"}, false)
	if err != nil {
		t.Fatalf("buildPayload returned error: %v", err)
	}

	if _, ok := payload["public_name"]; ok {
		t.Fatalf("payload[public_name] = %#v, want omitted", payload["public_name"])
	}
	if got, want := payload["status"], "operational"; got != want {
		t.Fatalf("payload[status] = %#v, want %#v", got, want)
	}
}

func TestBuildPayloadClearsChangedOptionalStringOnUpdate(t *testing.T) {
	fields := []fieldDef{
		reqString("name", "Name."),
		optString("public_name", "Public name."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"name":        "service",
		"public_name": "Public service",
	})
	if err := data.Set("public_name", ""); err != nil {
		t.Fatalf("set public_name: %v", err)
	}

	payload, err := buildPayload(data, fields, []string{"name", "public_name"}, true)
	if err != nil {
		t.Fatalf("buildPayload returned error: %v", err)
	}

	value, ok := payload["public_name"]
	if !ok {
		t.Fatal("payload[public_name] missing, want explicit nil")
	}
	if value != nil {
		t.Fatalf("payload[public_name] = %#v, want nil", value)
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

func TestSetFieldsFromResponseUsesJSONDefaultForNil(t *testing.T) {
	fields := []fieldDef{
		optJSONDefault("labels_json", "labels", "{}", "Labels."),
		computedJSON("metadata_json", "metadata", "Metadata."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{})

	err := setFieldsFromResponse(data, fields, []string{"labels_json", "metadata_json"}, map[string]interface{}{
		"labels":   nil,
		"metadata": nil,
	})
	if err != nil {
		t.Fatalf("setFieldsFromResponse returned error: %v", err)
	}

	if got, want := data.Get("labels_json"), "{}"; got != want {
		t.Fatalf("labels_json = %v, want %v", got, want)
	}
	if got, want := data.Get("metadata_json"), ""; got != want {
		t.Fatalf("metadata_json = %v, want empty %v", got, want)
	}
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

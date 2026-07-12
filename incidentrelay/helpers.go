package incidentrelay

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type fieldKind string

const (
	kindString    fieldKind = "string"
	kindInt       fieldKind = "int"
	kindBool      fieldKind = "bool"
	kindJSON      fieldKind = "json"
	kindStringSet fieldKind = "string_set"
	kindIntSet    fieldKind = "int_set"
)

type fieldDef struct {
	Name        string
	APIName     string
	Kind        fieldKind
	Required    bool
	Optional    bool
	Computed    bool
	Sensitive   bool
	ForceNew    bool
	Default     interface{}
	Description string
}

func (f fieldDef) apiName() string {
	if f.APIName != "" {
		return f.APIName
	}
	return f.Name
}

func schemaFromFields(fields []fieldDef) map[string]*schema.Schema {
	result := make(map[string]*schema.Schema, len(fields))
	for _, field := range fields {
		item := &schema.Schema{
			Description: field.Description,
			Required:    field.Required,
			Optional:    field.Optional,
			Computed:    field.Computed,
			Sensitive:   field.Sensitive,
			ForceNew:    field.ForceNew,
		}
		if field.Default != nil {
			item.Default = field.Default
		}

		switch field.Kind {
		case kindString:
			item.Type = schema.TypeString
		case kindInt:
			item.Type = schema.TypeInt
		case kindBool:
			item.Type = schema.TypeBool
		case kindJSON:
			item.Type = schema.TypeString
			if !field.Computed || field.Optional || field.Required {
				item.StateFunc = normalizeJSONStringState
				item.ValidateFunc = validation.StringIsJSON
			}
		case kindStringSet:
			item.Type = schema.TypeSet
			item.Elem = &schema.Schema{Type: schema.TypeString}
		case kindIntSet:
			item.Type = schema.TypeSet
			item.Elem = &schema.Schema{Type: schema.TypeInt}
		default:
			panic(fmt.Sprintf("unsupported field kind %q for %s", field.Kind, field.Name))
		}

		result[field.Name] = item
	}

	return result
}

func normalizeJSONStringState(value interface{}) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	normalized, err := normalizeJSONString(raw)
	if err != nil {
		return raw
	}
	return normalized
}

func normalizeJSONString(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw, err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw, err
	}
	return string(encoded), nil
}

func jsonStringToValue(raw string) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func valueToJSONString(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return normalizeJSONString(string(encoded))
}

func setIDFromResponse(d *schema.ResourceData, data map[string]interface{}) error {
	id, ok := data["id"]
	if !ok {
		return fmt.Errorf("response does not contain id: %v", data)
	}

	idString, err := interfaceToID(id)
	if err != nil {
		return err
	}
	d.SetId(idString)
	return nil
}

func interfaceToID(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("empty id")
		}
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.Itoa(int(v)), nil
	case json.Number:
		return v.String(), nil
	default:
		return "", fmt.Errorf("unsupported id value %T", value)
	}
}

func intFromInterface(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		parsed, err := v.Int64()
		return int(parsed), err == nil
	case string:
		parsed, err := strconv.Atoi(v)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func boolFromInterface(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(v)
		return parsed, err == nil
	default:
		return false, false
	}
}

func stringSliceFromInterface(value interface{}) []interface{} {
	items := make([]interface{}, 0)
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
	case []string:
		for _, item := range v {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return fmt.Sprintf("%v", items[i]) < fmt.Sprintf("%v", items[j])
	})
	return items
}

func intSliceFromInterface(value interface{}) []interface{} {
	items := make([]interface{}, 0)
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			if parsed, ok := intFromInterface(item); ok {
				items = append(items, parsed)
			}
		}
	case []int:
		for _, item := range v {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].(int) < items[j].(int)
	})
	return items
}

func setToSlice(set *schema.Set) []interface{} {
	if set == nil {
		return []interface{}{}
	}
	return set.List()
}

func mapHasID(item map[string]interface{}, id string) bool {
	itemID, ok := item["id"]
	if !ok {
		return false
	}
	itemIDString, err := interfaceToID(itemID)
	return err == nil && itemIDString == id
}

func extractItems(value interface{}) ([]map[string]interface{}, error) {
	switch data := value.(type) {
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
		return items, nil
	case map[string]interface{}:
		if rawItems, ok := data["items"]; ok {
			return extractItems(rawItems)
		}
		return []map[string]interface{}{data}, nil
	default:
		return nil, fmt.Errorf("unexpected list response type %T", value)
	}
}

func valuesEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b) || fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

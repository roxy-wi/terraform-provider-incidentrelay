package incidentrelay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type pathFunc func(*schema.ResourceData) string
type idPathFunc func(string, *schema.ResourceData) string

type resourceSpec struct {
	Description   string
	Fields        []fieldDef
	CreatePath    pathFunc
	ReadPath      idPathFunc
	ReadListPath  pathFunc
	ReadListField string
	UpdatePath    idPathFunc
	DeletePath    idPathFunc
	CreateFields  []string
	UpdateFields  []string
	ReadFields    []string
	PayloadHook   func(*schema.ResourceData, map[string]interface{})
	ResponseHook  func(*schema.ResourceData, map[string]interface{}) error
}

func crudResource(spec resourceSpec) *schema.Resource {
	resource := &schema.Resource{
		Description: spec.Description,
		CreateWithoutTimeout: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return crudCreate(ctx, d, m, spec)
		},
		ReadWithoutTimeout: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return crudRead(ctx, d, m, spec)
		},
		DeleteWithoutTimeout: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return crudDelete(ctx, d, m, spec)
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Read:   schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},
		Schema: schemaFromFields(spec.Fields),
	}

	if spec.UpdatePath != nil && len(spec.UpdateFields) > 0 {
		resource.UpdateWithoutTimeout = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return crudUpdate(ctx, d, m, spec)
		}
	}

	return resource
}

func crudCreate(ctx context.Context, d *schema.ResourceData, m interface{}, spec resourceSpec) diag.Diagnostics {
	client := m.(*Config).Client
	payload, err := buildPayload(d, spec.Fields, spec.CreateFields, false)
	if err != nil {
		return diag.FromErr(err)
	}
	if spec.PayloadHook != nil {
		spec.PayloadHook(d, payload)
	}

	var response map[string]interface{}
	if err := client.Do(ctx, http.MethodPost, spec.CreatePath(d), payload, &response); err != nil {
		return diag.FromErr(err)
	}
	if err := setIDFromResponse(d, response); err != nil {
		return diag.FromErr(err)
	}
	if spec.ResponseHook != nil {
		if err := spec.ResponseHook(d, response); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := setFieldsFromResponse(d, spec.Fields, spec.readFields(), response); err != nil {
		return diag.FromErr(err)
	}

	return crudRead(ctx, d, m, spec)
}

func crudRead(ctx context.Context, d *schema.ResourceData, m interface{}, spec resourceSpec) diag.Diagnostics {
	client := m.(*Config).Client

	var response map[string]interface{}
	var err error

	if spec.ReadPath != nil {
		err = client.Do(ctx, http.MethodGet, spec.ReadPath(d.Id(), d), nil, &response)
	} else if spec.ReadListPath != nil {
		response, err = readItemFromList(ctx, client, spec.ReadListPath(d), spec.ReadListField, d.Id())
	} else {
		return diag.Errorf("resource has neither ReadPath nor ReadListPath")
	}

	if errors.Is(err, ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if spec.ResponseHook != nil {
		if err := spec.ResponseHook(d, response); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := setFieldsFromResponse(d, spec.Fields, spec.readFields(), response); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func crudUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, spec resourceSpec) diag.Diagnostics {
	client := m.(*Config).Client
	payload, err := buildPayload(d, spec.Fields, spec.UpdateFields, true)
	if err != nil {
		return diag.FromErr(err)
	}
	if spec.PayloadHook != nil {
		spec.PayloadHook(d, payload)
	}

	var response map[string]interface{}
	if err := client.Do(ctx, http.MethodPut, spec.UpdatePath(d.Id(), d), payload, &response); err != nil {
		return diag.FromErr(err)
	}
	if len(response) > 0 {
		if spec.ResponseHook != nil {
			if err := spec.ResponseHook(d, response); err != nil {
				return diag.FromErr(err)
			}
		}
		if err := setFieldsFromResponse(d, spec.Fields, spec.readFields(), response); err != nil {
			return diag.FromErr(err)
		}
	}

	return crudRead(ctx, d, m, spec)
}

func crudDelete(ctx context.Context, d *schema.ResourceData, m interface{}, spec resourceSpec) diag.Diagnostics {
	client := m.(*Config).Client
	if err := client.Do(ctx, http.MethodDelete, spec.DeletePath(d.Id(), d), nil, nil); err != nil {
		if errors.Is(err, ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func (s resourceSpec) readFields() []string {
	if len(s.ReadFields) > 0 {
		return s.ReadFields
	}
	result := make([]string, 0, len(s.Fields))
	for _, field := range s.Fields {
		if field.Name != "password" {
			result = append(result, field.Name)
		}
	}
	return result
}

func buildPayload(d *schema.ResourceData, fields []fieldDef, fieldNames []string, update bool) (map[string]interface{}, error) {
	fieldByName := make(map[string]fieldDef, len(fields))
	for _, field := range fields {
		fieldByName[field.Name] = field
	}

	payload := make(map[string]interface{}, len(fieldNames))
	for _, name := range fieldNames {
		field, ok := fieldByName[name]
		if !ok {
			return nil, fmt.Errorf("unknown payload field %q", name)
		}

		value, exists := d.GetOkExists(name)
		if !exists {
			if update && d.HasChange(name) {
				payload[field.apiName()] = nil
			}
			continue
		}
		if omitUnsetOptionalInt(field, value) {
			if update && d.HasChange(name) {
				payload[field.apiName()] = nil
			}
			continue
		}
		if omitUnsetOptionalString(field, value) {
			if update && d.HasChange(name) {
				payload[field.apiName()] = nil
			}
			continue
		}

		switch field.Kind {
		case kindString:
			payload[field.apiName()] = value.(string)
		case kindInt:
			payload[field.apiName()] = value.(int)
		case kindBool:
			payload[field.apiName()] = value.(bool)
		case kindJSON:
			parsed, err := jsonStringToValue(value.(string))
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", name, err)
			}
			payload[field.apiName()] = parsed
		case kindStringSet, kindIntSet:
			payload[field.apiName()] = setToSlice(value.(*schema.Set))
		default:
			return nil, fmt.Errorf("unsupported payload field kind %q", field.Kind)
		}
	}

	return payload, nil
}

func omitUnsetOptionalInt(field fieldDef, value interface{}) bool {
	if field.Kind != kindInt || !field.Optional || field.Default != nil {
		return false
	}
	parsed, ok := intFromInterface(value)
	return ok && parsed == 0
}

func omitUnsetOptionalString(field fieldDef, value interface{}) bool {
	if field.Kind != kindString || !field.Optional || field.Default != nil {
		return false
	}
	parsed, ok := value.(string)
	return ok && parsed == ""
}

func setFieldsFromResponse(d *schema.ResourceData, fields []fieldDef, fieldNames []string, response map[string]interface{}) error {
	fieldByName := make(map[string]fieldDef, len(fields))
	for _, field := range fields {
		fieldByName[field.Name] = field
	}

	for _, name := range fieldNames {
		field, ok := fieldByName[name]
		if !ok {
			return fmt.Errorf("unknown read field %q", name)
		}

		value, exists := response[field.apiName()]
		if !exists {
			continue
		}

		var setValue interface{}
		switch field.Kind {
		case kindString:
			if value == nil {
				setValue = ""
			} else {
				setValue = fmt.Sprintf("%v", value)
			}
		case kindInt:
			if value == nil {
				setValue = 0
			} else if parsed, ok := intFromInterface(value); ok {
				setValue = parsed
			} else {
				return fmt.Errorf("cannot convert %s=%v to int", name, value)
			}
		case kindBool:
			if value == nil {
				setValue = false
			} else if parsed, ok := boolFromInterface(value); ok {
				setValue = parsed
			} else {
				return fmt.Errorf("cannot convert %s=%v to bool", name, value)
			}
		case kindJSON:
			jsonValue, err := valueToJSONString(value)
			if err != nil {
				return fmt.Errorf("encode %s: %w", name, err)
			}
			if jsonValue == "" && field.Default != nil {
				jsonValue = fmt.Sprintf("%v", field.Default)
			}
			setValue = jsonValue
		case kindStringSet:
			setValue = schema.NewSet(schema.HashString, stringSliceFromInterface(value))
		case kindIntSet:
			setValue = schema.NewSet(schema.HashInt, intSliceFromInterface(value))
		default:
			return fmt.Errorf("unsupported read field kind %q", field.Kind)
		}

		if err := d.Set(name, setValue); err != nil {
			return err
		}
	}

	return nil
}

func readItemFromList(ctx context.Context, client *Client, endpoint, listField, id string) (map[string]interface{}, error) {
	var response interface{}
	if err := client.Do(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}

	if listField != "" {
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected object response while reading nested field %s", listField)
		}
		nested, ok := responseMap[listField]
		if !ok {
			return nil, ErrNotFound
		}
		response = nested
	}

	items, err := extractItems(response)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if mapHasID(item, id) {
			return item, nil
		}
	}

	return nil, ErrNotFound
}

func createPath(endpoint string) pathFunc {
	return func(_ *schema.ResourceData) string {
		return endpoint
	}
}

func idPath(format string) idPathFunc {
	return func(id string, _ *schema.ResourceData) string {
		return fmt.Sprintf(format, id)
	}
}

func fieldCreatePath(format string, fields ...string) pathFunc {
	return func(d *schema.ResourceData) string {
		values := make([]interface{}, 0, len(fields))
		for _, field := range fields {
			values = append(values, d.Get(field))
		}
		return fmt.Sprintf(format, values...)
	}
}

func fieldListPath(format string, fields ...string) pathFunc {
	return fieldCreatePath(format, fields...)
}

func fieldIDPath(format string, fields ...string) idPathFunc {
	return func(id string, d *schema.ResourceData) string {
		values := make([]interface{}, 0, len(fields)+1)
		for _, field := range fields {
			values = append(values, d.Get(field))
		}
		values = append(values, id)
		return fmt.Sprintf(format, values...)
	}
}

func queryPath(endpoint string, params map[string]string, d *schema.ResourceData) string {
	pairs := make([]string, 0, len(params))
	for queryName, fieldName := range params {
		value, exists := d.GetOkExists(fieldName)
		if !exists {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", queryName, value))
	}
	if len(pairs) == 0 {
		return endpoint
	}
	return endpoint + "?" + strings.Join(pairs, "&")
}

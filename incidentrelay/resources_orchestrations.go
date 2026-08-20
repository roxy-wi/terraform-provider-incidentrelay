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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const orchestrationRedactedValue = "***REDACTED***"

func resourceEventOrchestration() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("group_id", "Owner group id."),
		reqString("name", "Event orchestration name."),
		optString("description", "Optional event orchestration description."),
		optStringDefault("scope", "global", "Orchestration scope: global or service."),
		optInt("service_id", "Service id for a service-scoped orchestration."),
		optStringDefault("compatibility_mode", "legacy", "Runtime compatibility mode: legacy, hybrid, or orchestration."),
		optJSONDefault("rules_json", "rules", "[]", "Ordered event orchestration rule tree JSON array."),
		optString("publish_comment", "Optional comment recorded on draft and published versions."),
		optBoolDefault("confirm_catch_all_drop", false, "Confirm publication of a catch-all drop rule."),
		optStringDefault("mode", "disabled", "Runtime mode: disabled, shadow, or active."),
		computedBool("enabled", "Whether the orchestration runtime is enabled."),
		computedString("uid", "Event orchestration UUID."),
		computedInt("active_version_id", "Published version id used by the runtime."),
		computedInt("active_version_number", "Published version number used by the runtime."),
		computedString("created_at", "Creation timestamp."),
		computedString("updated_at", "Last update timestamp."),
	}

	resource := &schema.Resource{
		Description: "IncidentRelay 2.0 Event Orchestration. Terraform saves and publishes a new immutable version whenever rules_json changes.",
		CreateWithoutTimeout: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return eventOrchestrationCreate(ctx, d, m, fields)
		},
		ReadWithoutTimeout: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return eventOrchestrationRead(ctx, d, m, fields)
		},
		UpdateWithoutTimeout: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return eventOrchestrationUpdate(ctx, d, m, fields)
		},
		DeleteWithoutTimeout: eventOrchestrationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Read:   schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},
		Schema: schemaFromFields(fields),
	}

	resource.Schema["group_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["name"].ValidateFunc = validation.StringLenBetween(1, 255)
	resource.Schema["description"].ValidateFunc = validation.StringLenBetween(0, 8192)
	resource.Schema["scope"].ValidateFunc = validation.StringInSlice([]string{"global", "service"}, false)
	resource.Schema["service_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["compatibility_mode"].ValidateFunc = validation.StringInSlice([]string{"legacy", "hybrid", "orchestration"}, false)
	resource.Schema["rules_json"].ValidateFunc = validateJSONArray
	resource.Schema["rules_json"].StateFunc = normalizeEventOrchestrationRulesState
	resource.Schema["publish_comment"].ValidateFunc = validation.StringLenBetween(0, 8192)
	resource.Schema["mode"].ValidateFunc = validation.StringInSlice([]string{"disabled", "shadow", "active"}, false)
	resource.CustomizeDiff = validateEventOrchestrationScope

	return resource
}

func validateEventOrchestrationScope(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.NewValueKnown("scope") || !diff.NewValueKnown("service_id") {
		return nil
	}
	scope := diff.Get("scope").(string)
	_, hasService := diff.GetOk("service_id")
	if scope == "global" && hasService {
		return fmt.Errorf("service_id must not be set when scope is global")
	}
	if scope == "service" && !hasService {
		return fmt.Errorf("service_id is required when scope is service")
	}
	return nil
}

func validateJSONArray(value interface{}, key string) ([]string, []error) {
	parsed, err := jsonStringToValue(value.(string))
	if err != nil {
		return nil, []error{fmt.Errorf("%s must contain valid JSON: %w", key, err)}
	}
	if _, ok := parsed.([]interface{}); !ok {
		return nil, []error{fmt.Errorf("%s must contain a JSON array", key)}
	}
	return nil, nil
}

func validateJSONObject(value interface{}, key string) ([]string, []error) {
	parsed, err := jsonStringToValue(value.(string))
	if err != nil {
		return nil, []error{fmt.Errorf("%s must contain valid JSON: %w", key, err)}
	}
	if _, ok := parsed.(map[string]interface{}); !ok {
		return nil, []error{fmt.Errorf("%s must contain a JSON object", key)}
	}
	return nil, nil
}

func normalizeEventOrchestrationRulesState(value interface{}) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	parsed, err := jsonStringToValue(raw)
	if err != nil {
		return raw
	}
	rules, ok := parsed.([]interface{})
	if !ok {
		return normalizeJSONStringState(raw)
	}
	normalized, err := valueToJSONString(normalizeEventOrchestrationRules(rules))
	if err != nil {
		return raw
	}
	return normalized
}

func normalizeEventOrchestrationRules(rules []interface{}) []interface{} {
	result := make([]interface{}, 0, len(rules))
	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			result = append(result, rawRule)
			continue
		}

		name := strings.TrimSpace(fmt.Sprintf("%v", rule["name"]))
		if name == "" || name == "<nil>" {
			name = fmt.Sprintf("Rule %d", index+1)
		}
		description := rule["description"]
		enabled, ok := rule["enabled"].(bool)
		if !ok {
			enabled = true
		}
		conditionTree := rule["condition_tree"]
		if conditionTree == nil {
			conditionTree = map[string]interface{}{}
		}
		actions, ok := rule["actions"].([]interface{})
		if !ok {
			actions = []interface{}{}
		}
		processingMode := strings.TrimSpace(fmt.Sprintf("%v", rule["processing_mode"]))
		if processingMode == "" || processingMode == "<nil>" {
			processingMode = "continue"
		}
		children, ok := rule["children"].([]interface{})
		if !ok {
			children = []interface{}{}
		}

		result = append(result, map[string]interface{}{
			"name":            name,
			"description":     description,
			"enabled":         enabled,
			"condition_tree":  conditionTree,
			"actions":         actions,
			"processing_mode": processingMode,
			"children":        normalizeEventOrchestrationRules(children),
		})
	}
	return result
}

func eventOrchestrationCreate(ctx context.Context, d *schema.ResourceData, m interface{}, fields []fieldDef) diag.Diagnostics {
	client := m.(*Config).Client
	payload, err := buildPayload(d, fields, []string{"group_id", "name", "description", "scope", "service_id", "compatibility_mode"}, false)
	if err != nil {
		return diag.FromErr(err)
	}

	var response map[string]interface{}
	if err := client.Do(ctx, http.MethodPost, "/api/event-orchestrations", payload, &response); err != nil {
		return diag.FromErr(err)
	}
	if err := setIDFromResponse(d, response); err != nil {
		return diag.FromErr(err)
	}
	if err := saveAndPublishEventOrchestration(ctx, client, d); err != nil {
		return diag.FromErr(err)
	}
	if err := setEventOrchestrationRuntime(ctx, client, d); err != nil {
		return diag.FromErr(err)
	}
	return eventOrchestrationRead(ctx, d, m, fields)
}

func eventOrchestrationRead(ctx context.Context, d *schema.ResourceData, m interface{}, fields []fieldDef) diag.Diagnostics {
	client := m.(*Config).Client
	var response map[string]interface{}
	if err := client.Do(ctx, http.MethodGet, fmt.Sprintf("/api/event-orchestrations/%s", d.Id()), nil, &response); err != nil {
		if errors.Is(err, ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	definition, _ := response["active_definition"].(map[string]interface{})
	if definition == nil {
		if draft, ok := response["draft"].(map[string]interface{}); ok {
			definition, _ = draft["definition"].(map[string]interface{})
		}
	}
	if definition != nil {
		if rules, ok := definition["rules"]; ok {
			response["rules"] = rules
		}
	}
	if activeVersion, ok := response["active_version"].(map[string]interface{}); ok {
		response["active_version_number"] = activeVersion["version_number"]
	}

	readFields := []string{
		"group_id", "name", "description", "scope", "service_id",
		"compatibility_mode", "rules_json", "mode", "enabled", "uid",
		"active_version_id", "active_version_number", "created_at", "updated_at",
	}
	if err := setFieldsFromResponse(d, fields, readFields, response); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func eventOrchestrationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}, fields []fieldDef) diag.Diagnostics {
	client := m.(*Config).Client
	metadataChanged := d.HasChanges("name", "description", "scope", "service_id")
	publishChanged := d.HasChanges("rules_json", "publish_comment", "confirm_catch_all_drop")

	if metadataChanged {
		payload, err := buildPayload(d, fields, []string{"name", "description", "scope", "service_id"}, true)
		if err != nil {
			return diag.FromErr(err)
		}
		var response map[string]interface{}
		if err := client.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/event-orchestrations/%s", d.Id()), payload, &response); err != nil {
			return diag.FromErr(err)
		}
	}

	if publishChanged {
		if err := saveAndPublishEventOrchestration(ctx, client, d); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := setEventOrchestrationRuntime(ctx, client, d); err != nil {
		return diag.FromErr(err)
	}
	return eventOrchestrationRead(ctx, d, m, fields)
}

func saveAndPublishEventOrchestration(ctx context.Context, client *Client, d *schema.ResourceData) error {
	basePath := fmt.Sprintf("/api/event-orchestrations/%s", d.Id())
	if err := client.Do(ctx, http.MethodPost, basePath+"/draft", nil, nil); err != nil {
		return err
	}

	normalizedRules := normalizeEventOrchestrationRulesState(d.Get("rules_json"))
	rules, err := jsonStringToValue(normalizedRules)
	if err != nil {
		return fmt.Errorf("parse rules_json: %w", err)
	}
	draftPayload := map[string]interface{}{"rules": rules}
	if comment := strings.TrimSpace(d.Get("publish_comment").(string)); comment != "" {
		draftPayload["comment"] = comment
	}
	if err := client.Do(ctx, http.MethodPut, basePath+"/draft", draftPayload, nil); err != nil {
		return err
	}

	publishPayload := map[string]interface{}{
		"confirm_catch_all_drop": d.Get("confirm_catch_all_drop").(bool),
	}
	if comment := strings.TrimSpace(d.Get("publish_comment").(string)); comment != "" {
		publishPayload["comment"] = comment
	}
	return client.Do(ctx, http.MethodPost, basePath+"/publish", publishPayload, nil)
}

func setEventOrchestrationRuntime(ctx context.Context, client *Client, d *schema.ResourceData) error {
	payload := map[string]interface{}{
		"mode":               d.Get("mode").(string),
		"compatibility_mode": d.Get("compatibility_mode").(string),
	}
	return client.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/event-orchestrations/%s/runtime", d.Id()), payload, nil)
}

func eventOrchestrationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Config).Client
	if err := client.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/event-orchestrations/%s", d.Id()), nil, nil); err != nil && !errors.Is(err, ErrNotFound) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func resourceOrchestrationWebhookAction() *schema.Resource {
	headersField := optJSONDefault("headers_json", "headers", "{}", "Sensitive HTTP headers JSON object. Headers are write-only in the IncidentRelay API.")
	headersField.Sensitive = true
	urlField := reqString("url", "Sensitive webhook destination URL.")
	urlField.Sensitive = true
	fields := []fieldDef{
		reqIntForceNew("group_id", "Owner group id."),
		reqString("name", "Webhook action name."),
		optString("description", "Optional webhook action description."),
		urlField,
		optStringDefault("method", "POST", "HTTP method: GET, POST, PUT, PATCH, or DELETE."),
		headersField,
		optString("body_template", "Optional safe orchestration template used as the request body."),
		optIntDefault("timeout_seconds", 10, "Request timeout from 1 to 60 seconds."),
		optIntDefault("retry_count", 2, "Retry count from 0 to 10."),
		optStringDefault("private_network_policy", "deny", "Private-network policy: deny or allowlist."),
		optBoolDefault("enabled", true, "Whether the webhook action is enabled."),
		computedString("uid", "Webhook action UUID."),
		computedBool("has_headers", "Whether encrypted headers are configured."),
		computedString("created_at", "Creation timestamp."),
		computedString("updated_at", "Last update timestamp."),
	}

	resource := crudResource(resourceSpec{
		Description: "Reusable IncidentRelay Event Orchestration webhook action.",
		Fields:      fields,
		CreatePath:  createPath("/api/orchestration-webhook-actions"),
		ReadListPath: func(d *schema.ResourceData) string {
			return queryPath("/api/orchestration-webhook-actions", map[string]string{"group_id": "group_id"}, d)
		},
		UpdatePath:   idPath("/api/orchestration-webhook-actions/%s"),
		UpdateMethod: http.MethodPatch,
		DeletePath:   idPath("/api/orchestration-webhook-actions/%s"),
		CreateFields: []string{"group_id", "name", "description", "url", "method", "headers_json", "body_template", "timeout_seconds", "retry_count", "private_network_policy", "enabled"},
		UpdateFields: []string{"name", "description", "url", "method", "headers_json", "body_template", "timeout_seconds", "retry_count", "private_network_policy", "enabled"},
		ReadFields:   []string{"group_id", "name", "description", "url", "method", "body_template", "timeout_seconds", "retry_count", "private_network_policy", "enabled", "uid", "has_headers", "created_at", "updated_at"},
		ResponseHook: orchestrationWebhookActionResponseHook,
	})

	resource.Schema["group_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["name"].ValidateFunc = validation.StringLenBetween(1, 255)
	resource.Schema["description"].ValidateFunc = validation.StringLenBetween(0, 8192)
	resource.Schema["url"].ValidateFunc = validation.StringLenBetween(1, 4096)
	resource.Schema["method"].ValidateFunc = validation.StringInSlice([]string{"GET", "POST", "PUT", "PATCH", "DELETE"}, false)
	resource.Schema["headers_json"].ValidateFunc = validateJSONObject
	resource.Schema["body_template"].ValidateFunc = validation.StringLenBetween(0, 65536)
	resource.Schema["timeout_seconds"].ValidateFunc = validation.IntBetween(1, 60)
	resource.Schema["retry_count"].ValidateFunc = validation.IntBetween(0, 10)
	resource.Schema["private_network_policy"].ValidateFunc = validation.StringInSlice([]string{"deny", "allowlist"}, false)

	return resource
}

func orchestrationWebhookActionResponseHook(d *schema.ResourceData, response map[string]interface{}) error {
	remoteURL, _ := response["url"].(string)
	currentURL, _ := d.Get("url").(string)
	if currentURL != "" && strings.Contains(remoteURL, orchestrationRedactedValue) {
		delete(response, "url")
	}
	return nil
}

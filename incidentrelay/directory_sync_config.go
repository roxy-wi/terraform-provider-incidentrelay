package incidentrelay

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Defaults mirror IR 2.5.1 SyncConfig, Operation and Pagination. Normalizing
// both sides prevents server-added defaults from causing permanent plan drift.
func normalizeDirectoryConfig(raw string) (string, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &config); err != nil || config == nil {
		return "", fmt.Errorf("config_json must be a JSON object")
	}
	if sourceType, exists := config["source_type"]; exists {
		if sourceType != "jira_assets" {
			return "", fmt.Errorf("config_json source_type must be jira_assets, or omitted for Directory Sync")
		}
		return normalizeCMDBConfig(config)
	}
	if value, exists := config["current_enabled"]; exists {
		if _, ok := value.(bool); !ok {
			return "", fmt.Errorf("current_enabled must be a JSON boolean")
		}
	}
	if value := config["retained_source_id"]; value != nil {
		id, ok := value.(float64)
		if !ok || id < 1 || math.Trunc(id) != id {
			return "", fmt.Errorf("retained_source_id must be a positive integer or null")
		}
	}
	defaults := map[string]interface{}{
		"private_network_policy": "deny", "verify_tls": true,
		"interval_seconds": 3600, "current_interval_seconds": 60, "current_max_age_seconds": 120,
		"horizon_months": 2, "timeout_seconds": 15, "editable": false,
		"overwrite_existing": false, "delete_missing": false,
		"membership_authority": "sync", "member_role": "responder", "responsible_role": "manager",
		"group_role": "viewer", "user_fetch": "per_user", "current_fetch": "batch",
		"detail_fields": map[string]interface{}{}, "team_filter": nil, "responsible": nil,
		"current_fields": map[string]interface{}{}, "team_matches": map[string]interface{}{},
	}
	directoryDefaults(config, defaults)
	for _, key := range []string{"base_url", "private_network_policy", "membership_authority", "member_role", "responsible_role", "group_role", "user_fetch", "current_fetch", "team_filter"} {
		if value, ok := config[key].(string); ok {
			config[key] = strings.TrimSpace(value)
		}
	}
	baseURL, ok := config["base_url"].(string)
	if !ok || baseURL == "" {
		return "", fmt.Errorf("config_json requires base_url")
	}
	config["base_url"] = strings.TrimRight(baseURL, "/")
	operations, ok := config["operations"].(map[string]interface{})
	if !ok || operations["teams"] == nil || operations["shifts"] == nil {
		return "", fmt.Errorf("config_json requires operations.teams and operations.shifts")
	}
	for name, value := range operations {
		operation, ok := value.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("operation %s must be an object", name)
		}
		directoryDefaults(operation, map[string]interface{}{
			"method": "GET", "params": map[string]interface{}{}, "body": nil, "items": "",
			"pagination": map[string]interface{}{},
		})
		for _, key := range []string{"method", "path", "items"} {
			if value, ok := operation[key].(string); ok {
				operation[key] = strings.TrimSpace(value)
			}
		}
		pagination, ok := operation["pagination"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("operation %s pagination must be an object", name)
		}
		directoryDefaults(pagination, map[string]interface{}{
			"mode": "none", "parameter": "pageToken", "next_path": "nextPageToken",
			"size_parameter": "pageSize", "size": 100, "start": 0, "max_pages": 1000,
		})
		for _, key := range []string{"mode", "parameter", "next_path", "size_parameter"} {
			if value, ok := pagination[key].(string); ok {
				pagination[key] = strings.TrimSpace(value)
			}
		}
	}
	for _, key := range []string{"members", "schedules", "replacements", "responsible"} {
		collection, ok := config[key].(map[string]interface{})
		if !ok && key != "responsible" {
			return "", fmt.Errorf("config_json requires %s as an object", key)
		}
		if path, ok := collection["path"].(string); ok {
			collection["path"] = strings.TrimSpace(path)
		}
	}
	for _, key := range []string{"team_fields", "shift_fields", "user_fields"} {
		if _, ok := config[key].(map[string]interface{}); !ok {
			return "", fmt.Errorf("config_json requires %s as an object", key)
		}
	}
	result, err := json.Marshal(config)
	return string(result), err
}

func syncConfigType(raw string) string {
	var config struct {
		SourceType string `json:"source_type"`
	}
	if json.Unmarshal([]byte(raw), &config) == nil && config.SourceType == "jira_assets" {
		return "jira_assets"
	}
	return "directory"
}

func suppressEquivalentSyncConfig(_ string, old, next string, _ *schema.ResourceData) bool {
	normalize := func(raw string) (string, error) {
		normalized, err := normalizeDirectoryConfig(raw)
		if err != nil {
			return "", err
		}
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(normalized), &config); err != nil {
			return "", err
		}
		if syncConfigType(normalized) == "directory" {
			// Compare server-added 2.5.2 defaults without sending unsupported keys
			// in payloads to IR 2.5.1. Explicit non-default values still cause drift.
			directoryDefaults(config, map[string]interface{}{"current_enabled": true, "retained_source_id": nil})
		}
		result, err := json.Marshal(config)
		return string(result), err
	}
	oldConfig, oldErr := normalize(old)
	newConfig, newErr := normalize(next)
	return oldErr == nil && newErr == nil && oldConfig == newConfig
}

func directoryDefaults(object, defaults map[string]interface{}) {
	for name, value := range defaults {
		if _, exists := object[name]; !exists {
			object[name] = value
		}
	}
}

func directoryConfigState(value interface{}) string {
	raw := value.(string)
	normalized, err := normalizeDirectoryConfig(raw)
	if err != nil {
		return raw
	}
	return normalized
}

func validateDirectoryConfig(value interface{}, key string) ([]string, []error) {
	if _, err := normalizeDirectoryConfig(value.(string)); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func directoryHeaders(raw string) (map[string]string, error) {
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil || headers == nil {
		return nil, fmt.Errorf("headers_json must be an object of string header values")
	}
	// Validation diagnostics deliberately omit credential names and values.
	for name, value := range headers {
		if name == "" || strings.TrimSpace(name) != name || strings.ContainsAny(name+value, "\r\n") {
			return nil, fmt.Errorf("headers_json contains an invalid HTTP header")
		}
	}
	return headers, nil
}

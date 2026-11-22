package incidentrelay

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Match CmdbConfig and CatalogSelection defaults from IR 2.5.2. Directory
// defaults must never leak into this separate API configuration contract.
func normalizeCMDBConfig(config map[string]interface{}) (string, error) {
	directoryDefaults(config, map[string]interface{}{
		"private_network_policy": "deny", "verify_tls": true,
		"timeout_seconds": 30, "interval_seconds": 3600,
		"api_path": "/rest/assets/1.0/aql/objects", "query_parameter": "qlQuery",
		"page_size": 25, "max_pages": 1000, "relationship_side": "services",
		"relationship_attribute": "Product", "team_matches": map[string]interface{}{},
		"environment_values":          map[string]interface{}{"Prod": "production", "Stage": "staging", "Dev": "development", "Test": "testing"},
		"criticality_values":          map[string]interface{}{"Business Critical": "critical"},
		"business_criticality_values": map[string]interface{}{},
	})
	for _, key := range []string{"base_url", "private_network_policy", "api_path", "query_parameter", "relationship_side", "relationship_attribute"} {
		if value, ok := config[key].(string); ok {
			config[key] = strings.TrimSpace(value)
		}
	}
	baseURL, ok := config["base_url"].(string)
	if !ok || baseURL == "" {
		return "", fmt.Errorf("CMDB config_json requires base_url")
	}
	config["base_url"] = strings.TrimRight(baseURL, "/")
	path, ok := config["api_path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("CMDB api_path must be a non-empty relative API path")
	}
	parsedPath, err := url.Parse(path)
	if err != nil || parsedPath.Scheme != "" || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "\\?#") {
		return "", fmt.Errorf("CMDB api_path must be relative to base_url, without a query, fragment or backslash")
	}
	if !strings.HasPrefix(path, "/") {
		config["api_path"] = "/" + path
	}
	for _, kind := range []string{"services", "business_services"} {
		selection, ok := config[kind].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("CMDB config_json requires %s as an object", kind)
		}
		query, ok := selection["query"].(string)
		if !ok || strings.TrimSpace(query) == "" {
			return "", fmt.Errorf("CMDB %s requires an AQL query", kind)
		}
		selection["query"] = strings.TrimSpace(query)
		for _, key := range []string{"owner_team_attribute", "name_attribute", "description_attribute", "fallback_description_attribute", "environment_attribute", "criticality_attribute", "status_attribute"} {
			if value, exists := selection[key]; !exists {
				selection[key] = nil
			} else if value, ok := value.(string); ok {
				selection[key] = strings.TrimSpace(value)
			}
		}
	}
	result, err := json.Marshal(config)
	return string(result), err
}

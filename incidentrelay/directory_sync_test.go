package incidentrelay

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const directoryTestConfig = `{
 "base_url":"https://directory.example.com/",
 "operations":{"teams":{"path":"/teams"},"shifts":{"path":"/shifts"}},
 "team_fields":{"id":"id","name":"name"},
 "members":{"path":"members","fields":{}},
 "schedules":{"path":"schedules","fields":{}},
 "shift_fields":{},"replacements":{"path":"replacements","fields":{}},"user_fields":{}
}`

func TestDirectoryConfigDefaultsAndExplicitValues(t *testing.T) {
	normalized, err := normalizeDirectoryConfig(directoryTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	again, err := normalizeDirectoryConfig(normalized)
	if err != nil || again != normalized {
		t.Fatal("normalization must be idempotent", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(normalized), &config); err != nil {
		t.Fatal(err)
	}
	if config["verify_tls"] != true || config["base_url"] != "https://directory.example.com" {
		t.Fatal("server defaults or URL normalization missing")
	}
	config["verify_tls"] = false
	config["editable"] = true
	config["interval_seconds"] = 120
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := normalizeDirectoryConfig(string(raw))
	if err != nil || changed == normalized {
		t.Fatal("explicit values lost", err)
	}
	for _, invalid := range []string{"null", "[]", "{}", strings.Replace(directoryTestConfig, `"shifts":{"path":"/shifts"}`, `"shifts":[]`, 1)} {
		if _, err := normalizeDirectoryConfig(invalid); err == nil {
			t.Errorf("accepted %s", invalid)
		}
	}
}

func TestDirectorySyncCredentialsAndDestroy(t *testing.T) {
	ctx := context.Background()
	config, err := normalizeDirectoryConfig(directoryTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	enabled := false
	remote := directorySyncDetail{ID: 8, GroupID: 2, Name: "Directory", Config: json.RawMessage(config), Enabled: &enabled, Revision: 1}
	var writes []map[string]interface{}
	client := changeTestClient(t, func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			if request.Method != http.MethodPost && request.Method != http.MethodPut {
				t.Error("unexpected method", request.Method)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			writes = append(writes, body)
			if headers, supplied := body["headers"]; supplied {
				remote.HasHeaders = len(headers.(map[string]interface{})) > 0
			}
			enabled = body["enabled"].(bool)
			remote.Name = body["name"].(string)
			remote.Revision++
		}
		if err := json.NewEncoder(w).Encode(remote); err != nil {
			t.Error(err)
		}
	})
	resource := resourceDirectorySyncSource()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "Directory", "group_id": 2, "config_json": directoryTestConfig, "headers_json": `{"Authorization":"Bearer local-test"}`,
	})
	if diags := resource.CreateContext(ctx, data, client); diags.HasError() {
		t.Fatal(diags)
	}
	if data.Id() != "8" || writes[0]["headers"] == nil {
		t.Fatal("creation lost ID or credentials")
	}
	if diags := resource.ReadContext(ctx, data, client); diags.HasError() {
		t.Fatal(diags)
	}
	if data.Get("headers_json") != `{"Authorization":"Bearer local-test"}` {
		t.Fatal("read lost write-only credentials")
	}
	// An imported source without configured credentials must preserve its headers
	// even when a subsequent rename produces a full PUT.
	imported := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "Renamed", "group_id": 2, "config_json": directoryTestConfig,
	})
	imported.SetId("8")
	if err := imported.Set("revision", remote.Revision); err != nil {
		t.Fatal(err)
	}
	if diags := resource.UpdateContext(ctx, imported, client); diags.HasError() {
		t.Fatal(diags)
	}
	if _, ok := writes[1]["headers"]; ok {
		t.Fatal("rename sent unknown credentials")
	}
	// Destroy must use the latest remote configuration, including UX edits.
	remote.Name = "Changed through UX"
	remote.Config = json.RawMessage(strings.Replace(config, `"interval_seconds":3600`, `"interval_seconds":120`, 1))
	enabled = true
	// A destroy without refresh can still see state written by provider v0.10.0.
	if err := imported.Set("destroy_behavior", ""); err != nil {
		t.Fatal(err)
	}
	if diags := resource.DeleteContext(ctx, imported, client); diags.HasError() {
		t.Fatal(diags)
	}
	last := writes[len(writes)-1]
	if last["name"] != "Changed through UX" || last["enabled"] != false || last["config"].(map[string]interface{})["interval_seconds"] != float64(120) {
		t.Fatal("destroy overwrote configuration instead of disabling the live source")
	}
	if _, ok := last["headers"]; ok {
		t.Fatal("destroy changed credentials")
	}
	if imported.Id() != "" || !remote.HasHeaders {
		t.Fatal("destroy lost remote credentials or retained binding")
	}
}

func TestDirectorySyncFailedWritesRetainState(t *testing.T) {
	for _, status := range []int{400, 401, 403, 409, 500} {
		for _, action := range []string{"update", "destroy"} {
			t.Run(action+"/"+http.StatusText(status), func(t *testing.T) {
				config, err := normalizeDirectoryConfig(directoryTestConfig)
				if err != nil {
					t.Fatal(err)
				}
				puts := 0
				client := changeTestClient(t, func(w http.ResponseWriter, request *http.Request) {
					if request.Method == http.MethodGet {
						_, _ = w.Write([]byte(`{"id":8,"group_id":2,"name":"Directory","revision":3,"enabled":true,"config":` + config + `}`))
						return
					}
					puts++
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"error":"conflict"}`))
				})
				resource := resourceDirectorySyncSource()
				data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"name": "Directory", "group_id": 2, "config_json": directoryTestConfig})
				data.SetId("8")
				operation := resource.UpdateContext
				if action == "destroy" {
					operation = schema.UpdateContextFunc(resource.DeleteContext)
				}
				if !operation(context.Background(), data, client).HasError() || data.Id() != "8" || puts != 1 {
					t.Fatal("failed operation lost identity or retried a write")
				}
			})
		}
	}
}

func TestDirectorySyncApprovedEnablePreservesCredentials(t *testing.T) {
	config, err := normalizeDirectoryConfig(directoryTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	client := changeTestClient(t, func(w http.ResponseWriter, request *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if request.Method != http.MethodPut || body["enabled"] != true || body["revision"] != float64(4) {
			t.Error("enable did not send reviewed revision", body["enabled"], body["revision"])
		}
		if _, supplied := body["headers"]; supplied {
			t.Error("enable resent write-only headers and would invalidate approval")
		}
		_, _ = w.Write([]byte(`{"id":8,"group_id":2,"name":"Directory","revision":5,"enabled":true,"approved":true,"has_headers":true,"config":` + config + `}`))
	})
	resource := resourceDirectorySyncSource()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "Directory", "group_id": 2, "config_json": directoryTestConfig, "enabled": true,
	})
	data.SetId("8")
	if err := data.Set("revision", 4); err != nil {
		t.Fatal(err)
	}
	if diags := resource.UpdateContext(context.Background(), data, client); diags.HasError() {
		t.Fatal(diags)
	}
	if data.Get("enabled") != true || data.Get("approved") != true || data.Get("has_headers") != true {
		t.Fatal("approved source enable lost remote status")
	}
}

func TestDirectorySyncCreateRetainsReturnedIdentity(t *testing.T) {
	client := changeTestClient(t, func(w http.ResponseWriter, request *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":8,"group_id":2,"revision":1,"enabled":false,"config":null}`))
	})
	resource := resourceDirectorySyncSource()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "Directory", "group_id": 2, "config_json": directoryTestConfig,
	})
	if !resource.CreateContext(context.Background(), data, client).HasError() || data.Id() != "8" {
		t.Fatal("invalid creation response must return an error and retain the created identity")
	}
}

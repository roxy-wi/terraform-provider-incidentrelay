package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestGroupResourceCRUD(t *testing.T) {
	state := map[string]interface{}{
		"id":          101,
		"slug":        "platform",
		"name":        "Platform",
		"description": "Primary team",
		"active":      true,
	}
	var requests []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/groups":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			want := map[string]interface{}{
				"slug":        "platform",
				"name":        "Platform",
				"description": "Primary team",
				"active":      true,
			}
			if !reflect.DeepEqual(payload, want) {
				t.Fatalf("create payload = %#v, want %#v", payload, want)
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodGet && r.URL.Path == "/api/groups":
			writeJSON(t, w, map[string]interface{}{"items": []map[string]interface{}{state}})

		case r.Method == http.MethodPut && r.URL.Path == "/api/groups/101":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode update payload: %v", err)
			}
			if got, want := payload["name"], "Platform Ops"; got != want {
				t.Fatalf("update payload[name] = %v, want %v", got, want)
			}
			for key, value := range payload {
				state[key] = value
			}
			writeJSON(t, w, state)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/groups/101":
			w.WriteHeader(http.StatusNoContent)

		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	config := &Config{Client: client}
	resource := resourceGroup()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"slug":        "platform",
		"name":        "Platform",
		"description": "Primary team",
		"active":      true,
	})

	if diags := resource.CreateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("create diagnostics: %v", diags)
	}
	if got, want := data.Id(), "101"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("name"), "Platform"; got != want {
		t.Fatalf("name after create = %v, want %v", got, want)
	}

	if err := data.Set("name", "Platform Ops"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if diags := resource.UpdateWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("update diagnostics: %v", diags)
	}
	if got, want := data.Get("name"), "Platform Ops"; got != want {
		t.Fatalf("name after update = %v, want %v", got, want)
	}

	if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
		t.Fatalf("delete diagnostics: %v", diags)
	}
	if got := data.Id(); got != "" {
		t.Fatalf("id after delete = %q, want empty", got)
	}

	wantRequests := []string{
		"POST /api/groups",
		"GET /api/groups",
		"PUT /api/groups/101",
		"GET /api/groups",
		"DELETE /api/groups/101",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestReadListDatasourceFindsSingleMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/api/groups"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		writeJSON(t, w, map[string]interface{}{
			"items": []map[string]interface{}{
				{"id": 1, "slug": "default", "name": "Default", "description": "", "active": true},
				{"id": 2, "slug": "platform", "name": "Platform", "description": "Primary", "active": true},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	dataSource := datasourceGroup()
	data := schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{
		"slug": "platform",
	})

	if diags := dataSource.ReadContext(context.Background(), data, &Config{Client: client}); diags.HasError() {
		t.Fatalf("read diagnostics: %v", diags)
	}
	if got, want := data.Id(), "2"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := data.Get("name"), "Platform"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if got, want := data.Get("description"), "Primary"; got != want {
		t.Fatalf("description = %v, want %v", got, want)
	}
}

func TestReadListDatasourceRequiresCriteriaAndRejectsAmbiguousMatches(t *testing.T) {
	dataSource := datasourceGroup()
	data := schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{})

	diags := dataSource.ReadContext(context.Background(), data, &Config{Client: &Client{}})
	if !diags.HasError() {
		t.Fatal("read without criteria returned no diagnostics")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"id": 1, "slug": "platform", "name": "Platform", "active": true},
			{"id": 2, "slug": "platform", "name": "Platform Copy", "active": true},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	data = schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{
		"slug": "platform",
	})

	diags = dataSource.ReadContext(context.Background(), data, &Config{Client: client})
	if !diags.HasError() {
		t.Fatal("ambiguous read returned no diagnostics")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestFieldIDPath(t *testing.T) {
	fields := []fieldDef{
		reqInt("policy_id", "Policy id."),
	}
	data := schema.TestResourceDataRaw(t, schemaFromFields(fields), map[string]interface{}{
		"policy_id": 55,
	})

	got := fieldIDPath("/api/notification-policies/%d/rules/%s", "policy_id")("99", data)
	if want := "/api/notification-policies/55/rules/99"; got != want {
		t.Fatalf("fieldIDPath = %q, want %q", got, want)
	}
}

func TestReadItemFromListNestedField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"rules": []map[string]interface{}{
				{"id": 1, "name": "first"},
				{"id": 2, "name": "second"},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	item, err := readItemFromList(context.Background(), client, "/policy", "rules", "2")
	if err != nil {
		t.Fatalf("readItemFromList returned error: %v", err)
	}
	if got, want := item["name"], "second"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}

	if _, err := readItemFromList(context.Background(), client, "/policy", "rules", "3"); !isNotFound(err) {
		t.Fatalf("missing item error = %v, want not found", err)
	}
}

func Example_stringID() {
	fmt.Println(stringID(42))
	// Output: 42
}

package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProviderInternalValidate(t *testing.T) {
	provider := Provider()
	if err := provider.InternalValidate(); err != nil {
		t.Fatalf("provider failed internal validation: %v", err)
	}

	for _, name := range []string{
		"incidentrelay_group",
		"incidentrelay_admin_user",
		"incidentrelay_team",
		"incidentrelay_channel",
		"incidentrelay_route",
		"incidentrelay_rotation_override",
		"incidentrelay_service",
		"incidentrelay_heartbeat",
		"incidentrelay_business_service",
	} {
		if provider.ResourcesMap[name] == nil {
			t.Fatalf("provider missing resource %s", name)
		}
	}

	for _, name := range []string{
		"incidentrelay_version",
		"incidentrelay_group",
		"incidentrelay_team",
		"incidentrelay_user",
		"incidentrelay_service",
	} {
		if provider.DataSourcesMap[name] == nil {
			t.Fatalf("provider missing data source %s", name)
		}
	}
}

func TestCommonFieldLengthValidation(t *testing.T) {
	resource := resourceGroup()
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{
			name:  "name allows 40 characters",
			field: "name",
			value: strings.Repeat("n", nameSlugMaxLength),
		},
		{
			name:    "name rejects 41 characters",
			field:   "name",
			value:   strings.Repeat("n", nameSlugMaxLength+1),
			wantErr: true,
		},
		{
			name:  "slug allows 40 characters",
			field: "slug",
			value: strings.Repeat("s", nameSlugMaxLength),
		},
		{
			name:    "slug rejects 41 characters",
			field:   "slug",
			value:   strings.Repeat("s", nameSlugMaxLength+1),
			wantErr: true,
		},
		{
			name:  "description allows 120 characters",
			field: "description",
			value: strings.Repeat("d", descriptionMaxLength),
		},
		{
			name:    "description rejects 121 characters",
			field:   "description",
			value:   strings.Repeat("d", descriptionMaxLength+1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validate := resource.Schema[tt.field].ValidateFunc
			if validate == nil {
				t.Fatalf("%s has no ValidateFunc", tt.field)
			}

			_, errors := validate(tt.value, tt.field)
			if gotErr := len(errors) > 0; gotErr != tt.wantErr {
				t.Fatalf("validation errors = %v, wantErr %v", errors, tt.wantErr)
			}
		})
	}
}

func TestRoutePayloadHookSetsEscalationMode(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		want    string
	}{
		{
			name:    "policy when escalation policy id is present",
			payload: map[string]interface{}{"escalation_policy_id": 42},
			want:    "policy",
		},
		{
			name:    "rotation when escalation policy id is absent",
			payload: map[string]interface{}{},
			want:    "rotation",
		},
		{
			name:    "rotation when escalation policy id is zero",
			payload: map[string]interface{}{"escalation_policy_id": 0},
			want:    "rotation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routePayloadHook(nil, tt.payload)
			if got := tt.payload["escalation_mode"]; got != tt.want {
				t.Fatalf("escalation_mode = %v, want %s", got, tt.want)
			}
		})
	}
}

func TestProviderConfigureWithTokenDoesNotAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request during token configuration: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)

	provider := Provider()
	data := schema.TestResourceDataRaw(t, provider.Schema, map[string]interface{}{
		ProviderBaseURL: server.URL,
		ProviderToken:   "static-token",
	})

	config, diags := providerConfigure(context.Background(), data, "1.9.0")
	if diags.HasError() {
		t.Fatalf("providerConfigure returned diagnostics: %v", diags)
	}

	cfg, ok := config.(*Config)
	if !ok {
		t.Fatalf("config type = %T, want *Config", config)
	}
	if got, want := cfg.Client.token, "static-token"; got != want {
		t.Fatalf("client token = %q, want %q", got, want)
	}
}

func TestProviderConfigureAuthenticatesWithUsernamePassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/api/auth/login"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("User-Agent"), "terraform/1.9.0 terraform-provider-incidentrelay"; got != want {
			t.Fatalf("User-Agent = %q, want %q", got, want)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got, want := body["username"], "admin"; got != want {
			t.Fatalf("username = %q, want %q", got, want)
		}
		if got, want := body["password"], "secret"; got != want {
			t.Fatalf("password = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"access_token":"jwt-token"}`)
	}))
	t.Cleanup(server.Close)

	provider := Provider()
	data := schema.TestResourceDataRaw(t, provider.Schema, map[string]interface{}{
		ProviderBaseURL:  server.URL,
		ProviderUsername: "admin",
		ProviderPassword: "secret",
	})

	config, diags := providerConfigure(context.Background(), data, "1.9.0")
	if diags.HasError() {
		t.Fatalf("providerConfigure returned diagnostics: %v", diags)
	}

	cfg := config.(*Config)
	if got, want := cfg.Client.token, "jwt-token"; got != want {
		t.Fatalf("client token = %q, want %q", got, want)
	}
}

func TestProviderConfigureRequiresCredentials(t *testing.T) {
	provider := Provider()
	data := schema.TestResourceDataRaw(t, provider.Schema, map[string]interface{}{
		ProviderBaseURL: "https://incidentrelay.example.com",
	})

	_, diags := providerConfigure(context.Background(), data, "1.9.0")
	if !diags.HasError() {
		t.Fatal("providerConfigure without credentials returned no diagnostics")
	}
}

func TestBoolEnvDefaultFunc(t *testing.T) {
	t.Setenv("INCIDENTRELAY_TEST_BOOL", "true")
	value, err := boolEnvDefaultFunc("INCIDENTRELAY_TEST_BOOL", false)()
	if err != nil {
		t.Fatalf("boolEnvDefaultFunc returned error: %v", err)
	}
	if got, want := value, true; got != want {
		t.Fatalf("value = %v, want %v", got, want)
	}

	t.Setenv("INCIDENTRELAY_TEST_BOOL", "definitely-not-bool")
	if _, err := boolEnvDefaultFunc("INCIDENTRELAY_TEST_BOOL", false)(); err == nil {
		t.Fatal("boolEnvDefaultFunc invalid value returned nil error")
	}
}

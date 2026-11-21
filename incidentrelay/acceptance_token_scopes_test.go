package incidentrelay

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestAccAdministrativePersonalTokenScopes(t *testing.T) {
	configured := testAccProviderConfig(t)
	ctx := context.Background()
	spec := testAccReadAPIObject(t, ctx, configured, "/api/openapi.json")
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI paths missing")
	}
	if _, ok := paths[directorySyncPath]; !ok {
		t.Skip("requires IR 2.5.1")
	}
	if os.Getenv("INCIDENTRELAY_USERNAME") == "" {
		t.Skip("requires disposable administrator login to issue restricted test tokens")
	}
	bootstrap, err := NewClient(ClientConfig{
		BaseURL:  configured.Client.baseURL,
		Username: os.Getenv("INCIDENTRELAY_USERNAME"), Password: os.Getenv("INCIDENTRELAY_PASSWORD"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.Configure(ctx); err != nil {
		t.Fatal(err)
	}
	login := &Config{Client: bootstrap}
	testAccCreateResource(t, ctx, resourceGroup(), login, map[string]interface{}{
		"name": "Token scopes", "slug": "token-scopes-" + testAccSuffix(),
	})
	for _, item := range []struct{ path, scope string }{
		{systemSettingsPath, "system-settings:read"},
		{changeAdminPath, "change-integrations:read"},
		{directorySyncPath, "directory-sync:read"},
	} {
		t.Run(item.scope, func(t *testing.T) {
			client := testAccTokenClient(t, login, []string{item.scope})
			if err := client.Do(ctx, http.MethodGet, item.path, nil, nil); err != nil {
				t.Fatal(err)
			}
			method := http.MethodPost
			if item.path == systemSettingsPath {
				method = http.MethodPatch
			}
			err := client.Do(ctx, method, item.path, map[string]interface{}{}, nil)
			if err == nil || !strings.Contains(err.Error(), "returned 403") {
				t.Fatalf("read-only token write should return 403, got %v", err)
			}
		})
	}
	aggregate := testAccTokenClient(t, login, []string{"resources:read", "resources:write"})
	for _, path := range []string{systemSettingsPath, changeAdminPath, directorySyncPath} {
		err := aggregate.Do(ctx, http.MethodGet, path, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "returned 403") {
			t.Fatalf("legacy resource scopes unexpectedly authorized %s: %v", path, err)
		}
	}
}

package incidentrelay

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAccIncidentRelayVersionDataSource(t *testing.T) {
	config := testAccProviderConfig(t)
	dataSource := datasourceVersion()
	data := schema.TestResourceDataRaw(t, dataSource.Schema, map[string]interface{}{})

	testAccRequireNoDiags(t, dataSource.ReadContext(context.Background(), data, config))

	if got := data.Id(); got != "version" {
		t.Fatalf("id = %q, want version", got)
	}
	if got := data.Get("service_version").(string); strings.TrimSpace(got) == "" {
		t.Fatal("service_version is empty")
	}
}

func TestAccIncidentRelayCoreResources(t *testing.T) {
	config := testAccProviderConfig(t)
	ctx := context.Background()
	suffix := testAccSuffix()

	groupResource := resourceGroup()
	groupData := schema.TestResourceDataRaw(t, groupResource.Schema, map[string]interface{}{
		"slug":        "tfacc-" + suffix,
		"name":        "TF acc group " + suffix,
		"description": "Created by Terraform provider acceptance tests.",
		"active":      true,
	})
	testAccRequireNoDiags(t, groupResource.CreateWithoutTimeout(ctx, groupData, config))
	testAccCleanupResource(t, groupResource, groupData, config)

	teamResource := resourceTeam()
	teamData := schema.TestResourceDataRaw(t, teamResource.Schema, map[string]interface{}{
		"group_id":                   testAccIDAsInt(t, groupData.Id()),
		"slug":                       "tfacc-team-" + suffix,
		"name":                       "TF acc team " + suffix,
		"description":                "Created by Terraform provider acceptance tests.",
		"escalation_enabled":         true,
		"escalation_after_reminders": 2,
		"active":                     true,
	})
	testAccRequireNoDiags(t, teamResource.CreateWithoutTimeout(ctx, teamData, config))
	testAccCleanupResource(t, teamResource, teamData, config)

	serviceResource := resourceService()
	serviceData := schema.TestResourceDataRaw(t, serviceResource.Schema, map[string]interface{}{
		"team_id":      testAccIDAsInt(t, teamData.Id()),
		"slug":         "tfacc-service-" + suffix,
		"name":         "TF acc service " + suffix,
		"description":  "Created by Terraform provider acceptance tests.",
		"service_type": "api",
		"environment":  "test",
		"criticality":  "medium",
		"tier":         "tier_3",
		"labels_json":  `{"managed_by":"terraform","test":"acceptance"}`,
		"metadata_json": `{
			"purpose": "terraform-provider-incidentrelay acceptance test"
		}`,
		"enabled": true,
	})
	testAccRequireNoDiags(t, serviceResource.CreateWithoutTimeout(ctx, serviceData, config))
	testAccCleanupResource(t, serviceResource, serviceData, config)

	groupLookup := datasourceGroup()
	groupLookupData := schema.TestResourceDataRaw(t, groupLookup.Schema, map[string]interface{}{
		"slug": groupData.Get("slug"),
	})
	testAccRequireNoDiags(t, groupLookup.ReadContext(ctx, groupLookupData, config))
	if groupLookupData.Id() != groupData.Id() {
		t.Fatalf("group lookup id = %q, want %q", groupLookupData.Id(), groupData.Id())
	}

	teamLookup := datasourceTeam()
	teamLookupData := schema.TestResourceDataRaw(t, teamLookup.Schema, map[string]interface{}{
		"group_id": testAccIDAsInt(t, groupData.Id()),
		"slug":     teamData.Get("slug"),
	})
	testAccRequireNoDiags(t, teamLookup.ReadContext(ctx, teamLookupData, config))
	if teamLookupData.Id() != teamData.Id() {
		t.Fatalf("team lookup id = %q, want %q", teamLookupData.Id(), teamData.Id())
	}

	serviceLookup := datasourceService()
	serviceLookupData := schema.TestResourceDataRaw(t, serviceLookup.Schema, map[string]interface{}{
		"team_id": testAccIDAsInt(t, teamData.Id()),
		"slug":    serviceData.Get("slug"),
	})
	testAccRequireNoDiags(t, serviceLookup.ReadContext(ctx, serviceLookupData, config))
	if serviceLookupData.Id() != serviceData.Id() {
		t.Fatalf("service lookup id = %q, want %q", serviceLookupData.Id(), serviceData.Id())
	}
}

func testAccProviderConfig(t *testing.T) *Config {
	t.Helper()

	if os.Getenv("INCIDENTRELAY_ACC") != "1" {
		t.Skip("set INCIDENTRELAY_ACC=1 to run acceptance tests")
	}

	baseURL := os.Getenv("INCIDENTRELAY_BASE_URL")
	if strings.TrimSpace(baseURL) == "" {
		t.Fatal("INCIDENTRELAY_BASE_URL is required for acceptance tests")
	}

	token := os.Getenv("INCIDENTRELAY_TOKEN")
	username := os.Getenv("INCIDENTRELAY_USERNAME")
	password := os.Getenv("INCIDENTRELAY_PASSWORD")
	if strings.TrimSpace(token) == "" && (strings.TrimSpace(username) == "" || password == "") {
		t.Fatal("set INCIDENTRELAY_TOKEN or INCIDENTRELAY_USERNAME/INCIDENTRELAY_PASSWORD for acceptance tests")
	}

	provider := Provider()
	data := schema.TestResourceDataRaw(t, provider.Schema, map[string]interface{}{
		ProviderBaseURL:  baseURL,
		ProviderToken:    token,
		ProviderUsername: username,
		ProviderPassword: password,
	})

	config, diags := providerConfigure(context.Background(), data, "acceptance")
	testAccRequireNoDiags(t, diags)

	providerConfig, ok := config.(*Config)
	if !ok {
		t.Fatalf("provider config type = %T, want *Config", config)
	}
	return providerConfig
}

func testAccCleanupResource(t *testing.T, resource *schema.Resource, data *schema.ResourceData, config *Config) {
	t.Helper()

	t.Cleanup(func() {
		if data.Id() == "" {
			return
		}
		if diags := resource.DeleteWithoutTimeout(context.Background(), data, config); diags.HasError() {
			t.Errorf("cleanup %s failed: %v", data.Id(), diags)
		}
	})
}

func testAccRequireNoDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func testAccIDAsInt(t *testing.T, id string) int {
	t.Helper()

	value, ok := intFromInterface(id)
	if !ok {
		t.Fatalf("cannot convert id %q to int", id)
	}
	return value
}

func testAccSuffix() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

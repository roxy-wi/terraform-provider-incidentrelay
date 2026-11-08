package incidentrelay

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// System settings are installation-wide. Only run mutations on an explicitly
// opted-in disposable instance and restore precisely the settings we touched.
func testAccSystemSettingsCLI(t *testing.T, c *Config, dir string, env []string, tf string) {
	if os.Getenv("INCIDENTRELAY_ACC_SYSTEM_SETTINGS") != "1" {
		t.Skip("requires disposable system settings opt-in")
	}
	ctx := context.Background()
	spec := testAccReadAPIObject(t, ctx, c, "/api/openapi.json")
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI response has no paths object")
	}
	if _, ok := paths[systemSettingsPath]; !ok {
		t.Skip("server does not advertise system settings")
	}
	get := func() map[string]interface{} {
		t.Helper()
		return testAccReadAPIObject(t, ctx, c, systemSettingsPath)
	}
	patch := func(body map[string]interface{}) {
		t.Helper()
		if err := c.Client.Do(ctx, http.MethodPatch, systemSettingsPath, body, nil); err != nil {
			t.Fatal(err)
		}
	}
	original := get()
	t.Cleanup(func() {
		restore := map[string]interface{}{
			"change_intake_enabled":     original["change_intake_enabled"],
			"change_max_future_seconds": original["change_max_future_seconds"],
		}
		if err := c.Client.Do(ctx, http.MethodPatch, systemSettingsPath, restore, nil); err != nil {
			t.Errorf("restore settings: %v", err)
		}
	})
	patch(map[string]interface{}{"change_intake_enabled": true, "change_max_future_seconds": 1})
	run := func(args ...string) string {
		t.Helper()
		return strings.TrimSpace(testAccRunCommand(t, dir, env, tf, args...))
	}
	set := func(fields string) {
		t.Helper()
		s := `resource "incidentrelay_system_settings" "instance" {
` + fields + `
}
data "incidentrelay_system_settings" "current" {
 depends_on = [incidentrelay_system_settings.instance]
}
output "effective_future_window" { value = data.incidentrelay_system_settings.current.change_max_future_seconds }
`
		if err := os.WriteFile(filepath.Join(dir, "settings.tf"), []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	apply := func() { run("apply", "-input=false", "-auto-approve", "-no-color") }
	noDiff := func() { run("plan", "-input=false", "-no-color", "-detailed-exitcode") }
	set("change_worker_retry_base_seconds = 6\nchange_worker_retry_max_seconds = 5")
	invalid := testAccRunCommandResult(dir, env, tf, "plan", "-input=false", "-no-color")
	if invalid.ExitCode != 1 || !strings.Contains(invalid.Output, "must not exceed") {
		t.Fatalf("expected the plan to reject an invalid retry window, got %d:\n%s", invalid.ExitCode, invalid.Output)
	}
	set("change_intake_enabled = false\nchange_max_future_seconds = 0")
	apply()
	if got := get(); got["change_intake_enabled"] != false || got["change_max_future_seconds"] != float64(0) {
		t.Fatal("explicit false or zero not applied")
	}
	if run("output", "-raw", "effective_future_window") != "0" {
		t.Fatal("data source did not read current settings")
	}
	run("state", "rm", "incidentrelay_system_settings.instance")
	run("import", "-input=false", "incidentrelay_system_settings.instance", "1")
	noDiff()
	set("change_max_future_seconds = 2")
	run("plan", "-input=false", "-out=settings.plan")
	patch(map[string]interface{}{"change_intake_enabled": true})
	run("apply", "-input=false", "settings.plan")
	if got := get(); got["change_intake_enabled"] != true || got["change_max_future_seconds"] != float64(2) {
		t.Fatal("apply overwrote an unmanaged UX edit or lost the configured setting")
	}
	noDiff()
	patch(map[string]interface{}{"change_max_future_seconds": 3})
	apply()
	if get()["change_max_future_seconds"] != float64(2) {
		t.Fatal("managed drift was not corrected")
	}
	if err := os.Remove(filepath.Join(dir, "settings.tf")); err != nil {
		t.Fatal(err)
	}
	apply()
	if got := get(); got["change_intake_enabled"] != true || got["change_max_future_seconds"] != float64(2) {
		t.Fatal("destroy reset system settings")
	}
}

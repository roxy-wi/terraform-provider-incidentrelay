package incidentrelay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const systemSettingsPath = "/api/admin/system-settings"
const systemSettingsID = "1"

// IR stores these installation-wide settings in a single row.
type systemSetting struct {
	name, description string
	kind              schema.ValueType
	min, max          int
}

const maxSettingInt = 2147483647

var systemSettings = []systemSetting{
	{"change_intake_enabled", "Accept incoming Change Events.", schema.TypeBool, 0, 0},
	{"change_worker_enabled", "Process background Change jobs.", schema.TypeBool, 0, 0},
	{"change_max_future_seconds", "Maximum accepted future event timestamp offset, in seconds.", schema.TypeInt, 0, maxSettingInt},
	{"change_events_per_minute", "Maximum new Change Events per integration per minute.", schema.TypeInt, 1, maxSettingInt},
	{"change_events_per_change", "Maximum events per Change.", schema.TypeInt, 1, 10000},
	{"change_worker_interval_seconds", "Worker tick interval in seconds.", schema.TypeInt, 1, maxSettingInt},
	{"change_worker_jobs_per_tick", "Maximum jobs per worker tick.", schema.TypeInt, 1, 100},
	{"change_worker_batch_size", "Maximum items in a worker batch.", schema.TypeInt, 1, 1000},
	{"change_worker_lease_seconds", "Worker lease duration in seconds.", schema.TypeInt, 10, maxSettingInt},
	{"change_worker_max_attempts", "Maximum attempts per job.", schema.TypeInt, 1, maxSettingInt},
	{"change_worker_retry_base_seconds", "Base worker retry delay in seconds.", schema.TypeInt, 1, maxSettingInt},
	{"change_worker_retry_max_seconds", "Maximum worker retry delay in seconds.", schema.TypeInt, 1, maxSettingInt},
	{"change_pending_ttl_seconds", "Pending reference lifetime in seconds.", schema.TypeInt, 60, maxSettingInt},
	{"change_pending_poll_seconds", "Polling interval for pending references in seconds.", schema.TypeInt, 5, maxSettingInt},
}

func systemSettingsSchema(writable bool) map[string]*schema.Schema {
	fields := make(map[string]*schema.Schema, len(systemSettings))
	for _, f := range systemSettings {
		attr := &schema.Schema{Type: f.kind, Optional: writable, Computed: true, Description: f.description}
		if writable && f.kind == schema.TypeInt {
			attr.ValidateFunc = validation.IntBetween(f.min, f.max)
		}
		fields[f.name] = attr
	}
	return fields
}

func resourceSystemSettings() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage selected installation-wide Change Intelligence settings. Requires a global administrator JWT.",
		Schema:        systemSettingsSchema(true),
		CreateContext: writeSystemSettings,
		ReadContext:   readSystemSettings,
		UpdateContext: writeSystemSettings,
		DeleteContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
			d.SetId("")
			return nil
		},
		Importer: &schema.ResourceImporter{StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
			if d.Id() != systemSettingsID {
				return nil, fmt.Errorf("system settings import ID must be %s", systemSettingsID)
			}
			return []*schema.ResourceData{d}, nil
		}},
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			const base, max = "change_worker_retry_base_seconds", "change_worker_retry_max_seconds"
			raw := d.GetRawConfig()
			if raw.IsNull() || !raw.IsKnown() {
				return nil
			}
			baseValue, maxValue := raw.GetAttr(base), raw.GetAttr(max)
			if baseValue.IsNull() || maxValue.IsNull() || !baseValue.IsKnown() || !maxValue.IsKnown() {
				return nil
			}
			if baseValue.GreaterThan(maxValue).True() {
				return fmt.Errorf("%s must not exceed %s", base, max)
			}
			return nil
		},
	}
}

func datasourceSystemSettings() *schema.Resource {
	return &schema.Resource{
		Description: "Read installation-wide Change Intelligence settings with a global administrator JWT.",
		Schema:      systemSettingsSchema(false),
		ReadContext: readSystemSettings,
	}
}

func readSystemSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return fetchSystemSettings(ctx, d, meta.(*Config).Client, http.MethodGet, nil)
}

func writeSystemSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	raw := d.GetRawConfig()
	changes := map[string]interface{}{}
	if raw.IsKnown() && !raw.IsNull() {
		for _, f := range systemSettings {
			if !raw.GetAttr(f.name).IsNull() && (d.Id() == "" || d.HasChange(f.name)) {
				changes[f.name] = d.Get(f.name)
			}
		}
	}
	if len(changes) == 0 {
		return readSystemSettings(ctx, d, meta)
	}
	return fetchSystemSettings(ctx, d, meta.(*Config).Client, http.MethodPatch, changes)
}

func fetchSystemSettings(ctx context.Context, d *schema.ResourceData, client *Client, method string, body interface{}) diag.Diagnostics {
	var response map[string]json.RawMessage
	if err := client.Do(ctx, method, systemSettingsPath, body, &response); err != nil {
		return diag.Errorf("%s %s: %s", method, systemSettingsPath, err)
	}
	// Validate the full response before touching state. Absent/null values are
	// not equivalent to disabling a switch or setting a limit to zero.
	values := make(map[string]interface{}, len(systemSettings))
	for _, f := range systemSettings {
		raw := response[f.name]
		if len(raw) == 0 || string(raw) == "null" {
			return diag.Errorf("system settings response missing %s", f.name)
		}
		if f.kind == schema.TypeBool {
			var v bool
			if err := json.Unmarshal(raw, &v); err != nil {
				return diag.Errorf("invalid boolean for system setting %s", f.name)
			}
			values[f.name] = v
		} else {
			var v int
			if err := json.Unmarshal(raw, &v); err != nil || v < f.min || v > f.max {
				return diag.Errorf("invalid integer for system setting %s (expected %d..%d)", f.name, f.min, f.max)
			}
			values[f.name] = v
		}
	}
	for name, v := range values {
		if err := d.Set(name, v); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(systemSettingsID)
	return nil
}

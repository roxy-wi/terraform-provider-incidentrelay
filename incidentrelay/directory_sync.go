package incidentrelay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const directorySyncPath = "/api/admin/directory-sync"

type directorySyncDetail struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	GroupID        int             `json:"group_id"`
	Config         json.RawMessage `json:"config"`
	Enabled        *bool           `json:"enabled"`
	Revision       int             `json:"revision"`
	Approved       bool            `json:"approved"`
	HasHeaders     bool            `json:"has_headers"`
	Running        bool            `json:"running"`
	PendingMode    string          `json:"pending_mode"`
	LastError      string          `json:"last_error"`
	ScheduleState  string          `json:"schedule_state"`
	CurrentError   string          `json:"current_error"`
	CurrentRunning bool            `json:"current_running"`
	NextRunAt      string          `json:"next_run_at"`
	NextCurrentAt  string          `json:"next_current_at"`
}

func resourceDirectorySyncSource() *schema.Resource {
	fields := schemaFromFields([]fieldDef{
		reqString("name", "Directory synchronization source name."),
		reqIntForceNew("group_id", "Destination Group. Immutable after creation."),
		reqJSON("config_json", "config", "Directory Sync configuration JSON. IR adds defaults to omitted settings."),
		{Name: "headers_json", Kind: kindJSON, Optional: true, Computed: true, Sensitive: true, Description: "Write-only HTTP authentication headers. Omit to preserve; use {} to clear."},
		optBoolDefault("enabled", false, "Request periodic sync. IR 2.5.2+ waits for preview/apply approval before running."),
		optStringDefault("destroy_behavior", "disable", "disable (default), delete_keep_data, or delete_with_data. Deletion requires IR 2.5.2+."),
		computedString("source_type", "directory or jira_assets."),
		computedString("schedule_state", "disabled, pending_approval, preview, or enabled (IR 2.5.2+)."),
		computedString("current_error", "Last current-duty refresh error (IR 2.5.2+)."),
		computedBool("current_running", "Whether current-duty refresh is running (IR 2.5.2+)."),
		computedString("next_run_at", "Next scheduled import in UTC, or empty when unscheduled (IR 2.5.2+)."),
		computedString("next_current_at", "Next current-duty refresh in UTC, or empty when unscheduled (IR 2.5.2+)."),
		computedInt("revision", "Current configuration revision."),
		computedBool("approved", "Whether this revision has been approved through preview/apply."),
		computedBool("has_headers", "Whether the server stores encrypted headers."),
		computedBool("running", "Whether the main synchronization worker is running."),
		computedString("pending_mode", "Queued preview, apply or run action."),
		computedString("last_error", "Last synchronization error."),
	})
	fields["group_id"].ValidateFunc = validation.IntAtLeast(1)
	fields["name"].ValidateFunc = func(value interface{}, key string) ([]string, []error) {
		if !validTrimmedText(value.(string), 1, 255) {
			return nil, []error{fmt.Errorf("name must be 1–255 characters with no surrounding whitespace")}
		}
		return nil, nil
	}
	fields["config_json"].StateFunc = directoryConfigState
	fields["config_json"].ValidateFunc = validateDirectoryConfig
	fields["config_json"].DiffSuppressFunc = suppressEquivalentSyncConfig
	fields["config_json"].DiffSuppressOnRefresh = true
	fields["destroy_behavior"].ValidateFunc = validation.StringInSlice([]string{"disable", "delete_keep_data", "delete_with_data"}, false)
	fields["headers_json"].ValidateFunc = func(value interface{}, key string) ([]string, []error) {
		if _, err := directoryHeaders(value.(string)); err != nil {
			return nil, []error{err}
		}
		return nil, nil
	}
	return &schema.Resource{
		Description:   "Manage Directory Sync (IR 2.5.1+) or Jira CMDB (IR 2.5.2+). Preview/apply is performed in IR. Destroy disables periodic sync by default.",
		Schema:        fields,
		CreateContext: createDirectorySyncSource,
		ReadContext:   readDirectorySyncSource,
		UpdateContext: updateDirectorySyncSource,
		DeleteContext: deleteDirectorySyncSource,
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			if d.Id() != "" && d.NewValueKnown("config_json") {
				oldValue, newValue := d.GetChange("config_json")
				if syncConfigType(oldValue.(string)) != syncConfigType(newValue.(string)) {
					return d.ForceNew("config_json")
				}
			}
			return nil
		},
		Importer: &schema.ResourceImporter{StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
			id, err := strconv.Atoi(d.Id())
			if err != nil || id < 1 {
				return nil, fmt.Errorf("Directory Sync import requires a positive numeric source ID")
			}
			return []*schema.ResourceData{d}, nil
		}},
	}
}

func getDirectorySync(ctx context.Context, client *Client, id string) (directorySyncDetail, error) {
	return requestDirectorySync(ctx, client, http.MethodGet, directorySyncPath+"/"+id, nil)
}

func requestDirectorySync(ctx context.Context, client *Client, method, path string, body interface{}) (directorySyncDetail, error) {
	var result directorySyncDetail
	if err := client.Do(ctx, method, path, body, &result); err != nil {
		return result, err
	}
	if result.ID < 1 || result.GroupID < 1 || result.Revision < 1 || result.Enabled == nil {
		return result, fmt.Errorf("incomplete Directory Sync response")
	}
	if _, err := normalizeDirectoryConfig(string(result.Config)); err != nil {
		return result, fmt.Errorf("invalid Directory Sync configuration response: %w", err)
	}
	return result, nil
}

func saveDirectorySync(d *schema.ResourceData, result directorySyncDetail) error {
	config, err := normalizeDirectoryConfig(string(result.Config))
	if err != nil {
		return err
	}
	// This local lifecycle setting is absent in imported and v0.10.0 state.
	// Its safe default must not produce a plan change or fall through to DELETE.
	if d.Get("destroy_behavior").(string) == "" {
		if err := d.Set("destroy_behavior", "disable"); err != nil {
			return err
		}
	}
	return setChangeState(d, map[string]interface{}{
		"name": result.Name, "group_id": result.GroupID, "config_json": config,
		"enabled": *result.Enabled, "revision": result.Revision, "approved": result.Approved,
		"has_headers": result.HasHeaders, "running": result.Running,
		"pending_mode": result.PendingMode, "last_error": result.LastError,
		"source_type": syncConfigType(config), "schedule_state": result.ScheduleState,
		"current_error": result.CurrentError, "current_running": result.CurrentRunning,
		"next_run_at": result.NextRunAt, "next_current_at": result.NextCurrentAt,
	})
}

func readDirectorySyncSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	result, err := getDirectorySync(ctx, meta.(*Config).Client, d.Id())
	if errors.Is(err, ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if strconv.Itoa(result.ID) != d.Id() {
		return diag.Errorf("unexpected Directory Sync source ID")
	}
	return diag.FromErr(saveDirectorySync(d, result))
}

func directorySyncBody(d *schema.ResourceData, create bool) (map[string]interface{}, error) {
	config, err := normalizeDirectoryConfig(d.Get("config_json").(string))
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"name": d.Get("name"), "group_id": d.Get("group_id"),
		"config": json.RawMessage(config), "enabled": d.Get("enabled"),
	}
	if raw, ok := d.GetOk("headers_json"); ok && (create || d.HasChange("headers_json")) {
		headers, err := directoryHeaders(raw.(string))
		if err != nil {
			return nil, err
		}
		body["headers"] = headers
	}
	return body, nil
}

func createDirectorySyncSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	body, err := directorySyncBody(d, true)
	if err != nil {
		return diag.FromErr(err)
	}
	result, err := requestDirectorySync(ctx, meta.(*Config).Client, http.MethodPost, directorySyncPath, body)
	// Retain an identity returned by a successful write even if response
	// validation fails, so Terraform can refresh it instead of creating a duplicate.
	if result.ID > 0 {
		d.SetId(strconv.Itoa(result.ID))
	}
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(saveDirectorySync(d, result))
}

func updateDirectorySyncSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	remoteChanged := false
	for _, field := range []string{"name", "config_json", "headers_json", "enabled"} {
		remoteChanged = remoteChanged || d.HasChange(field)
	}
	if !remoteChanged {
		return nil
	}
	body, err := directorySyncBody(d, false)
	if err != nil {
		return diag.FromErr(err)
	}
	// Use the revision captured by refresh/plan. A concurrent edit must fail
	// with 409, rather than silently overwriting a reviewed configuration.
	body["revision"] = d.Get("revision")
	result, err := requestDirectorySync(ctx, meta.(*Config).Client, http.MethodPut, directorySyncPath+"/"+d.Id(), body)
	if err != nil {
		return diag.FromErr(err)
	}
	if strconv.Itoa(result.ID) != d.Id() {
		return diag.Errorf("unexpected Directory Sync source ID")
	}
	return diag.FromErr(saveDirectorySync(d, result))
}

func deleteDirectorySyncSource(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Config).Client
	if behavior := d.Get("destroy_behavior").(string); behavior != "" && behavior != "disable" {
		return removeDirectorySyncSource(ctx, d, client, behavior)
	}
	current, err := getDirectorySync(ctx, client, d.Id())
	if errors.Is(err, ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if strconv.Itoa(current.ID) != d.Id() {
		return diag.Errorf("unexpected Directory Sync source ID")
	}
	// PUT also checks worker leases and queued actions, including when already
	// disabled. Do not forget a source while it still has pending writes.
	body := map[string]interface{}{
		"name": current.Name, "group_id": current.GroupID, "config": current.Config,
		"revision": current.Revision, "enabled": false,
	}
	after, err := requestDirectorySync(ctx, client, http.MethodPut, directorySyncPath+"/"+d.Id(), body)
	if err != nil {
		return diag.FromErr(err)
	}
	if after.ID != current.ID || *after.Enabled {
		return diag.Errorf("IR did not disable the Directory Sync source")
	}
	d.SetId("")
	return nil
}

func removeDirectorySyncSource(ctx context.Context, d *schema.ResourceData, client *Client, behavior string) diag.Diagnostics {
	action := "keep"
	if behavior == "delete_with_data" {
		action = "delete"
	} else if behavior != "delete_keep_data" {
		return diag.Errorf("unsupported Directory Sync destroy_behavior")
	}
	var result struct {
		ID      int  `json:"id"`
		Deleted bool `json:"deleted"`
	}
	// Delete only the revision reviewed in the destroy plan. Never retry with a
	// newly fetched revision: it could describe different imported objects.
	err := client.Do(ctx, http.MethodDelete, directorySyncPath+"/"+d.Id(),
		map[string]interface{}{"revision": d.Get("revision"), "data_action": action}, &result)
	if errors.Is(err, ErrNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if strconv.Itoa(result.ID) != d.Id() || !result.Deleted {
		return diag.Errorf("IR did not confirm deletion of the Directory Sync source")
	}
	d.SetId("")
	return nil
}

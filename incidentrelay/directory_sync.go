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
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	GroupID     int             `json:"group_id"`
	Config      json.RawMessage `json:"config"`
	Enabled     *bool           `json:"enabled"`
	Revision    int             `json:"revision"`
	Approved    bool            `json:"approved"`
	HasHeaders  bool            `json:"has_headers"`
	Running     bool            `json:"running"`
	PendingMode string          `json:"pending_mode"`
	LastError   string          `json:"last_error"`
}

func resourceDirectorySyncSource() *schema.Resource {
	fields := schemaFromFields([]fieldDef{
		reqString("name", "Directory synchronization source name."),
		reqIntForceNew("group_id", "Destination Group. Immutable after creation."),
		reqJSON("config_json", "config", "Directory Sync configuration JSON. IR adds defaults to omitted settings."),
		{Name: "headers_json", Kind: kindJSON, Optional: true, Computed: true, Sensitive: true, Description: "Write-only HTTP authentication headers. Omit to preserve; use {} to clear."},
		optBoolDefault("enabled", false, "Enable periodic sync after applying a preview in IR. Create disabled first."),
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
	fields["headers_json"].ValidateFunc = func(value interface{}, key string) ([]string, []error) {
		if _, err := directoryHeaders(value.(string)); err != nil {
			return nil, []error{err}
		}
		return nil, nil
	}
	return &schema.Resource{
		Description:   "Manage an IR 2.5.1 Directory Sync source. Preview/apply is performed in IR. Destroy disables periodic sync.",
		Schema:        fields,
		CreateContext: createDirectorySyncSource,
		ReadContext:   readDirectorySyncSource,
		UpdateContext: updateDirectorySyncSource,
		DeleteContext: deleteDirectorySyncSource,
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			if d.Id() == "" && d.NewValueKnown("enabled") && d.Get("enabled").(bool) {
				return fmt.Errorf("create Directory Sync disabled, review and apply a preview in IR, then set enabled = true")
			}
			configChanged := false
			if d.Id() != "" && d.NewValueKnown("config_json") {
				oldValue, newValue := d.GetChange("config_json")
				oldConfig, oldError := normalizeDirectoryConfig(oldValue.(string))
				newConfig, newError := normalizeDirectoryConfig(newValue.(string))
				configChanged = oldError != nil || newError != nil || oldConfig != newConfig
			}
			raw := d.GetRawConfig()
			headersConfigured := !raw.IsNull() && raw.IsKnown() && !raw.GetAttr("headers_json").IsNull()
			headersChanged := headersConfigured && d.NewValueKnown("headers_json") && d.HasChange("headers_json")
			if d.Id() != "" && d.NewValueKnown("enabled") && d.Get("enabled").(bool) && (configChanged || headersChanged) {
				return fmt.Errorf("set enabled = false when changing Directory Sync configuration or headers; apply a new preview in IR before enabling")
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
	return setChangeState(d, map[string]interface{}{
		"name": result.Name, "group_id": result.GroupID, "config_json": config,
		"enabled": *result.Enabled, "revision": result.Revision, "approved": result.Approved,
		"has_headers": result.HasHeaders, "running": result.Running,
		"pending_mode": result.PendingMode, "last_error": result.LastError,
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
	if d.Get("enabled").(bool) {
		return diag.Errorf("create Directory Sync disabled, then apply a preview in IR before enabling")
	}
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

---
page_title: "incidentrelay_directory_sync_source Resource - IncidentRelay"
subcategory: "Directory Sync"
description: |-
  Manages a Group-owned directory and on-call synchronization source.
---

# incidentrelay_directory_sync_source

Requires IR 2.5.1+ and a global administrator or destination Group `user_admin`.
Personal API tokens need both `directory-sync:read` and `directory-sync:write`.
Group-bound tokens can manage their own Group. Use a token without a Team bound.

Terraform manages the source's configuration and periodic synchronization switch.
Review and apply the initial preview in **Administration → Directory Sync**.
IR 2.5.2 also accepts Jira Assets/Insight sources in this resource. See the
[IR 2.5.2 guide](../guides/incidentrelay-2.5.2.md) and
[CMDB example](../../examples/incidentrelay-2.5.2/main.tf).

```hcl
resource "incidentrelay_directory_sync_source" "directory" {
  name     = "Company directory"
  group_id = incidentrelay_group.platform.id
  enabled  = false

  headers_json = jsonencode({
    Authorization = "Bearer ${var.directory_api_token}"
  })

  config_json = jsonencode({
    base_url = "https://directory.example.com"
    operations = {
      teams  = { path = "/teams", items = "teams" }
      shifts = { path = "/teams/{team_id}/shifts", items = "shifts" }
    }
    team_fields = { id = "id", name = "name" }
    members = {
      path   = "members"
      fields = { id = "id", username = "username", email = "email", name = "name" }
    }
    schedules = {
      path   = "schedules"
      fields = { id = "id", timezone = "timezone" }
    }
    shift_fields = {
      id = "id", schedule_id = "schedule_id", user_id = "user_id"
      start = "start", end = "end"
    }
    replacements = {
      path   = "replacements"
      fields = { id = "id", user_id = "user_id", start = "start", end = "end" }
    }
    user_fields = { id = "id", username = "username", email = "email", name = "name" }
  })
}
```

Adapt the endpoint paths and field mappings to your external API. The IR
Directory Sync editor provides the full configuration schema and an editable
org-service preset.

## Arguments and attributes

- `name` (required) Source display name, 1–255 characters.
- `group_id` (required, replacement) Destination Group ID.
- `config_json` (required) Configuration object accepted by IR's `SyncConfig`,
  or `CmdbConfig` with `source_type: "jira_assets"` on IR 2.5.2.
  Includes operations, pagination, field mappings, timeouts, intervals, TLS and
  private-network policy, and imported-resource editing and deletion policies.
  Omitted settings use the corresponding server defaults. Terraform manages the
  whole object, including defaults. Changing source type requires replacement.
- `headers_json` (optional, sensitive) HTTP authentication headers. IR stores
  them encrypted and returns only `has_headers`. Terraform preserves configured
  headers in state. Omit to preserve the server's current headers, including
  after import. Set `"{}"` to clear them.
  IR encrypts even the empty object, so `has_headers` can stay true after clearing.
- `enabled` (optional, default `false`) Periodic synchronization switch.
- `destroy_behavior` (optional, default `"disable"`) See the deletion modes below.
- `id`, `revision`, `approved`, `has_headers`, `running`, `pending_mode`,
  `last_error` (computed) Source identity, configuration revision and status.
- `source_type` (computed) `directory` or `jira_assets`.
- `schedule_state`, `current_error`, `current_running`, `next_run_at`,
  `next_current_at` (computed, IR 2.5.2) Schedule state and current-duty status.
  Unscheduled timestamps are empty strings.

Put credentials in `headers_json` and sensitive variables. Restrict access to
Terraform state because it contains configured header values.

## Preview and activation

1. Apply the source with `enabled = false`.
2. In IR, fetch a preview, inspect the proposed changes to users, Teams and
   schedules, and apply the reviewed preview. Wait for completion.
3. Set `enabled = true` and run Terraform apply.

On IR 2.5.1, set `enabled = false` when changing configuration or headers and
repeat preview/apply before enabling. An unapproved enable returns a server
error and retains the Terraform binding.

On IR 2.5.2, `enabled = true` can be selected at creation and kept during edits.
It requests a schedule; `schedule_state = "pending_approval"` means no scheduled
import will run until a preview is reviewed and applied in IR. Import-affecting
changes clear approval. Operational changes (including current-duty settings,
timeouts, intervals and TLS verification) preserve approval. Terraform performs
no preview, apply or run requests. A name-only edit preserves approval.

Terraform sends the revision captured during refresh and planning. A concurrent
edit returns `409`. Refresh and review a new plan before retrying. A running or
queued synchronization also returns `409`, including during destroy.

Directory Sync owns imported records according to its configured policies.
For records protected by sync, use Terraform data sources to reference existing
Teams, users and rotations. Enable local editing in the sync configuration when
you intend to edit imported resources.

## Import and destroy

```sh
terraform import incidentrelay_directory_sync_source.directory 12
```

Import reads the effective configuration. Configure `enabled` to match the
source's current state and omit `headers_json` to retain its stored credentials.

The default `destroy_behavior = "disable"` preserves the v0.10.0 behavior on
all supported versions. Destroy disables synchronization using the latest remote
configuration, then forgets the Terraform binding. IR retains the source,
credentials, imported records and history. Import that source ID to manage it again.

IR 2.5.2 adds two explicit options:

| Behavior | Result |
|---|---|
| `delete_keep_data` | Archive the source and clear credentials, retaining imported records. |
| `delete_with_data` | Archive the source and delete its owned imported resources through IR's normal deletion rules, including dependent configuration. |

Deletion sends the revision recorded in the destroy plan. A concurrent edit or
running/queued worker causes an error and preserves the binding; refresh and
review a new plan before retrying. Older IR versions reject these deletion modes.
An archived source cannot be re-imported as an active source. To reconnect
retained Directory data, create a new source with `config.retained_source_id`.
Jira CMDB reconnects by external object ID for the same Group and Jira base URL
after a reviewed preview/apply.

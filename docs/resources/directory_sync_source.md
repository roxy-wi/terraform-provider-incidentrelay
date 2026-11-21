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
- `config_json` (required) Configuration object accepted by IR's `SyncConfig`.
  Includes operations, pagination, field mappings, timeouts, intervals, TLS and
  private-network policy, and imported-resource editing and deletion policies.
  Omitted settings use IR 2.5.1 defaults. Terraform manages the whole object,
  including those defaults.
- `headers_json` (optional, sensitive) HTTP authentication headers. IR stores
  them encrypted and returns only `has_headers`. Terraform preserves configured
  headers in state. Omit to preserve the server's current headers, including
  after import. Set `"{}"` to clear them.
  IR encrypts even the empty object, so `has_headers` can stay true after clearing.
- `enabled` (optional, default `false`) Periodic synchronization switch.
- `id`, `revision`, `approved`, `has_headers`, `running`, `pending_mode`,
  `last_error` (computed) Source identity, configuration revision and status.

Put credentials in `headers_json` and sensitive variables. Restrict access to
Terraform state because it contains configured header values.

## Preview and activation

1. Apply the source with `enabled = false`.
2. In IR, fetch a preview, inspect the proposed changes to users, Teams and
   schedules, and apply the reviewed preview. Wait for completion.
3. Set `enabled = true` and run Terraform apply.

When changing `config_json` or `headers_json`, set `enabled = false` in that same
configuration and repeat preview/apply before enabling again. A name-only edit
preserves an existing approval. An unapproved enable returns a server error and
retains the Terraform binding.

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

IR 2.5.1 exposes source creation and updates, but has no source DELETE endpoint.
Destroy disables periodic synchronization using the latest remote configuration,
then removes the Terraform binding. IR retains the source, credentials, imported
records and history. Import the retained source ID to manage it again.

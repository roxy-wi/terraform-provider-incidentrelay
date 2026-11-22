---
page_title: "IncidentRelay 2.5.2 - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  Configure Jira CMDB and updated Directory Sync behavior with provider v0.11.0.
---

# IncidentRelay 2.5.2

These additions require provider v0.11.0 or later.
Compatibility is verified against the final IR `v2.5.2` release.
IR 2.5.1 remains covered.

## Jira Assets and Insight

Use [incidentrelay_directory_sync_source](../resources/directory_sync_source.md)
with `config_json.source_type = "jira_assets"`. The
[complete example](../../examples/incidentrelay-2.5.2/main.tf) configures two AQL
selections, independently selected owner Team attributes, external Team mappings
and Service-to-Business-Service relationships.

The destination Group must already exist. Values in `team_matches` are active
IR Team IDs in that Group; keys are positive CMDB object IDs. Configure the
external Jira PAT in sensitive `headers_json`, for example an Authorization
Bearer header. Provider authentication continues to use an IR administrator JWT
or a personal token with `directory-sync:read` and `directory-sync:write`.

| Configuration | Purpose |
|---|---|
| `base_url` | Jira HTTP(S) URL without embedded credentials, query or fragment. |
| `api_path`, `query_parameter` | Assets defaults: `/rest/assets/1.0/aql/objects`, `qlQuery`. Older Insight installations can use `/rest/insight/1.0/iql/objects`, `iql`. |
| `services`, `business_services` | Each requires an AQL `query`; attribute selectors accept exact names or IDs represented as strings. |
| `owner_team_attribute` | Select independently inside each query's configuration. |
| `name_attribute`, `description_attribute`, `fallback_description_attribute` | Optional catalog field selectors inside each selection. |
| `environment_attribute`, `criticality_attribute`, `status_attribute` | Optional field selectors inside each selection. Lifecycle status remains metadata rather than operational health. |
| `team_matches` | External Team object ID to existing IR Team ID. |
| `relationship_side`, `relationship_attribute` | Which selection holds references and which attribute links Services and Business Services. Defaults: `services`, `Product`. |
| `environment_values`, `criticality_values`, `business_criticality_values` | Explicit translations from external values into IR enums. |
| `page_size`, `max_pages` | Pagination limits, defaults 25 and 1000. |
| `interval_seconds`, `timeout_seconds` | Defaults 3600 and 30 seconds. |
| `verify_tls`, `private_network_policy` | TLS verification and the existing server outbound network policy. |

Inspect fields and Teams, review a preview and apply it in IR UX. Terraform
does not fetch external catalog data or invoke inspect/preview/apply/run.
Changing the source family requires replacement. After an import, IR also
rejects changing the Jira base URL in place; create a separate source instead.

CMDB manages selected catalog fields. Avoid declaring those same fields in
separate Terraform Service resources. Existing local policies, manual status,
unrelated metadata and manual component links remain governed by IR's CMDB
ownership rules.

## Directory current-duty settings

The existing Directory configuration shape is preserved. Add these keys inside
`config_json` as needed:

| Setting | Default and behavior |
|---|---|
| `current_enabled` | `true`. Uses the configured `operations.current` API. Set `false` to use imported shifts and replacements. |
| `current_interval_seconds` | 60; range 10–3600. Time between current-duty checks, including after failure. |
| `current_max_age_seconds` | 120; range 10–3600. Freshness limit, independent of polling interval. |
| `retained_source_id` | `null`. ID of an archived Directory source in the same Group whose retained imports should reconnect after UX preview/apply. |

The provider treats omitted `current_enabled: true` and
`retained_source_id: null` as equivalent to server-added defaults. It does not
inject these keys into IR 2.5.1 payloads. Explicit non-default values still
produce a plan change. The whole configuration remains Terraform-managed:
copy intended UX edits into HCL to preserve them.
IR rejects changing `retained_source_id` after the first successful import.

## Requested schedules and approval

IR 2.5.2 accepts `enabled = true` before the first reviewed import. This records
the requested schedule. `schedule_state` shows `pending_approval` until a
preview is applied, `preview` while a new preview awaits review, `enabled`
when scheduled, or `disabled`.

Import-affecting edits invalidate approval. Operational edits such as intervals,
current-duty settings and TLS verification preserve approval. New
`current_error`, `current_running`, `next_run_at` and `next_current_at`
attributes expose scheduling diagnostics; unscheduled timestamps are empty.
IR 2.5.1 still requires creating and editing configuration disabled, then
approving in UX before enabling.

## Destroy and migration

Existing configurations keep `destroy_behavior = "disable"`. No source or
imported data is removed when upgrading the provider.

To explicitly archive a source on IR 2.5.2, choose `delete_keep_data` or
`delete_with_data`. The latter deletes owned imported records and related
configuration through IR's normal deletion rules. Review that choice before
applying. Deletion uses the planned revision and refuses stale plans or busy
sources; errors retain the Terraform binding.

The deletion policy is local Terraform state. Import starts with `disable`;
apply an explicit alternative policy after import. Changing only this policy
does not write to IR or increment its revision.

An archived source's ID cannot be imported as active. Directory reconnection
uses a new source's `retained_source_id`. Jira CMDB reconnects retained records
by external object ID for the same Group and Jira base URL after review.

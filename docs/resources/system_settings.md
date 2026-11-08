---
page_title: "incidentrelay_system_settings Resource - IncidentRelay"
subcategory: "Change Intelligence"
description: |-
  Manages installation-wide Change Intelligence settings from Administration → System settings.
---

# incidentrelay_system_settings

Manages the settings shown under **Administration → System settings**. Requires an
IR 2.5 build with `/api/admin/system-settings` and a global administrator JWT.
The values apply to every Group in the installation.

```hcl
resource "incidentrelay_system_settings" "installation" {
  change_intake_enabled       = true
  change_worker_enabled       = true
  change_worker_jobs_per_tick = 20
}
```

Use one resource per installation. Declare only the fields Terraform should own.
On the next plan, Terraform detects external edits to those fields and proposes
restoring their configured values. Omit a field or remove it from the
configuration to leave its current value under UX control. An explicit `false`
or `0` is applied where the field allows it. To inspect settings without
managing them, use the [system settings data source](../data-sources/system_settings.md).

## Settings

All arguments are optional. Both the resource and the data source export the
current value of every setting. Integer ranges are inclusive.

| Argument | Purpose | Accepted values |
| --- | --- | --- |
| `change_intake_enabled` | Accept incoming Change Events | `true`, `false` |
| `change_worker_enabled` | Run background Change jobs | `true`, `false` |
| `change_max_future_seconds` | Allowed future event timestamp offset, in seconds | 0–2147483647 |
| `change_events_per_minute` | New events per integration per minute, across all its sources | 1–2147483647 |
| `change_events_per_change` | Events per change | 1–10000 |
| `change_worker_interval_seconds` | Interval between worker ticks, in seconds | 1–2147483647 |
| `change_worker_jobs_per_tick` | Jobs processed per tick | 1–100 |
| `change_worker_batch_size` | Rows processed per batch | 1–1000 |
| `change_worker_lease_seconds` | Job lease duration, in seconds | 10–2147483647 |
| `change_worker_max_attempts` | Attempts per job | 1–2147483647 |
| `change_worker_retry_base_seconds` | Base retry delay, in seconds | 1–2147483647 |
| `change_worker_retry_max_seconds` | Maximum retry delay, in seconds | 1–2147483647 |
| `change_pending_ttl_seconds` | Pending reference lifetime, in seconds | 60–2147483647 |
| `change_pending_poll_seconds` | Interval between pending reference checks, in seconds | 5–2147483647 |

The base retry delay must be less than or equal to the maximum. Set both in
the same apply when a new base delay would exceed the current maximum.

## Import and destroy

The installation's settings always have ID `1`. Import reads their current values.

```sh
terraform import incidentrelay_system_settings.installation 1
```

Destroy removes the Terraform state binding and preserves the live settings.

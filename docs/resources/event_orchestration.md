---
page_title: "incidentrelay_event_orchestration Resource - IncidentRelay"
subcategory: "Event Orchestration"
description: |-
  Manages an IncidentRelay 2.0 Event Orchestration and its published rule version.
---

# incidentrelay_event_orchestration

The resource manages orchestration metadata, the rule tree, publication, and
runtime mode as one declarative object. On create and whenever `rules_json`
changes, the provider saves a draft and publishes a new immutable IncidentRelay
version before applying the requested runtime mode.

```hcl
resource "incidentrelay_event_orchestration" "production" {
  group_id          = incidentrelay_group.infrastructure.id
  name              = "Production routing"
  description       = "Classify and route production events"
  scope             = "global"
  compatibility_mode = "hybrid"
  mode              = "shadow"
  publish_comment   = "Managed by Terraform"

  rules_json = jsonencode([
    {
      name    = "Route critical production alerts"
      enabled = true
      condition_tree = {
        all = [
          {
            field    = "labels.environment"
            operator = "equals"
            value    = "production"
          },
          {
            field    = "event.severity"
            operator = "equals"
            value    = "critical"
          }
        ]
      }
      actions = [
        {
          type    = "set_team"
          team_id = incidentrelay_team.platform.id
        },
        {
          type  = "set_priority"
          value = "P1"
        }
      ]
      processing_mode = "continue"
      children        = []
    }
  ])
}
```

## Arguments

- `group_id` (Number, Required) Owner group ID. Changing it recreates the
  orchestration.
- `name` (String, Required) Orchestration name.
- `description` (String, Optional) Orchestration description.
- `scope` (String, Optional) `global` or `service`. Defaults to `global`.
- `service_id` (Number, Optional) Required for `service` scope and forbidden
  for `global` scope.
- `compatibility_mode` (String, Optional) `legacy`, `hybrid`, or
  `orchestration`. Defaults to `legacy`.
- `rules_json` (String, Optional) Ordered orchestration rule-tree JSON array.
  Defaults to `[]`.
- `publish_comment` (String, Optional) Comment recorded on the draft and
  published version.
- `confirm_catch_all_drop` (Boolean, Optional) Explicit safety confirmation
  required when publishing a catch-all drop rule. Defaults to `false`.
- `mode` (String, Optional) `disabled`, `shadow`, or `active`. Defaults to
  `disabled`. Start new rules in `shadow` before moving them to `active`.

## Read-Only Attributes

- `enabled` (Boolean) Whether the orchestration runtime is enabled.
- `uid` (String) Orchestration UUID.
- `active_version_id` (Number) Published version selected by the runtime.
- `active_version_number` (Number) Published version number.
- `created_at` (String) Creation timestamp.
- `updated_at` (String) Last update timestamp.

The API validates condition operators and action payloads during publication.
Terraform intentionally keeps the rule DSL in JSON so it follows the running
IncidentRelay API without flattening the nested rule and action model.

## Import

```sh
terraform import incidentrelay_event_orchestration.production 91
```

After import, configure `group_id` and review the imported published rules
before applying changes.

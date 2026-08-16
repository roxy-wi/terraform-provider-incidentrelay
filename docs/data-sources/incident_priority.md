---
page_title: "incidentrelay_incident_priority Data Source - IncidentRelay"
subcategory: "Policies"
description: |-
  Looks up an IncidentRelay incident priority such as P1 or P2.
---

# incidentrelay_incident_priority

```hcl
data "incidentrelay_incident_priority" "p1" {
  slug = "p1"
}
```

Use the resulting `id` for `fallback_priority_id` and priority policy rule
`priority_id` arguments.

## Lookup Arguments

- `priority_id` (Number) Incident priority ID.
- `slug` (String) Priority slug, such as `p1`.
- `name` (String) Priority name.
- `level` (Number) Numeric priority level.

At least one lookup argument is required. Ambiguous lookups return an error.

## Attributes

- `id` (String) Incident priority ID.
- `priority_id` (Number) Incident priority ID.
- `slug` (String) Priority slug.
- `name` (String) Priority name.
- `description` (String) Priority description.
- `level` (Number) Numeric priority level.
- `color` (String) Display color.
- `enabled` (Boolean) Whether the priority is enabled.
- `default` (Boolean) Whether this is the default priority.

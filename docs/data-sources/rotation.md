---
page_title: "incidentrelay_rotation Data Source - IncidentRelay"
subcategory: "On-call"
description: |-
  Looks up an existing IncidentRelay on-call rotation.
---

# incidentrelay_rotation

```hcl
data "incidentrelay_rotation" "primary" {
  team_slug = "platform"
  name      = "Primary"
}

resource "incidentrelay_escalation_policy_rule" "primary" {
  policy_id     = incidentrelay_escalation_policy.critical.id
  position      = 1
  delay_seconds = 300
  target_type   = "rotation"
  target_id     = data.incidentrelay_rotation.primary.id
}
```

## Lookup Arguments

- `rotation_id` (Number) Rotation ID.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `name` (String) Rotation name.

At least one lookup argument is required. Use `team_id` or `team_slug` with
`name` when different teams contain rotations with the same name. Ambiguous
lookups return an error.

## Attributes

- `id` (String) Rotation ID.
- `rotation_id` (Number) Rotation ID.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `team_name` (String) Owner team name.
- `name` (String) Rotation name.
- `description` (String) Rotation description.
- `start_at` (String) Rotation start datetime.
- `duration_seconds` (Number) Custom slot duration.
- `reminder_interval_seconds` (Number) Reminder interval.
- `rotation_type` (String) Rotation type.
- `interval_value` (Number) Rotation interval value.
- `interval_unit` (String) Rotation interval unit.
- `handoff_time` (String) Local handoff time.
- `handoff_weekday` (Number) Weekly handoff weekday.
- `timezone` (String) Rotation timezone.
- `enabled` (Boolean) Whether the rotation is enabled.
- `current_oncall` (String) Current on-call username, when available.

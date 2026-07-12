---
page_title: "incidentrelay_team Data Source - IncidentRelay"
subcategory: "Teams"
description: |-
  Looks up an IncidentRelay team by ID, group ID, slug, or name.
---

# incidentrelay_team

```hcl
data "incidentrelay_group" "infra" {
  slug = "infra"
}

data "incidentrelay_team" "platform" {
  group_id = data.incidentrelay_group.infra.id
  slug     = "platform"
}
```

## Lookup Arguments

- `team_id` (Number) Team ID.
- `group_id` (Number) Owner group ID.
- `slug` (String) Team slug.
- `name` (String) Team name.

At least one lookup argument is required. If more than one team matches, add
more filters.

## Attributes

- `id` (String) Team ID.
- `group_id` (Number) Owner group ID.
- `slug` (String) Team slug.
- `name` (String) Team name.
- `description` (String) Team description.
- `group_slug` (String) Owner group slug.
- `group_name` (String) Owner group name.

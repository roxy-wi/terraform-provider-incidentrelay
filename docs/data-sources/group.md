---
page_title: "incidentrelay_group Data Source - IncidentRelay"
subcategory: "Identity"
description: |-
  Looks up an IncidentRelay group by ID, slug, or name.
---

# incidentrelay_group

```hcl
data "incidentrelay_group" "infra" {
  slug = "infra"
}
```

## Lookup Arguments

- `group_id` (Number) Group ID.
- `slug` (String) Group slug.
- `name` (String) Group name.

At least one lookup argument is required. If more than one group matches, add
more filters.

## Attributes

- `id` (String) Group ID.
- `slug` (String) Group slug.
- `name` (String) Group name.
- `description` (String) Group description.
- `active` (Boolean) Whether the group is active.

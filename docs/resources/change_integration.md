---
page_title: "incidentrelay_change_integration Resource - IncidentRelay"
subcategory: "Change Intelligence"
description: |-
  Manages a Change Integration and exposes its initial source ID.
---

# incidentrelay_change_integration

Requires IR 2.5 and an [administrator JWT](../guides/authentication.md#change-integrations-ir-25). The integration identifies a producer of Change Events. IR creates its first source automatically, with no grants. Use `initial_source_id` when granting it permissions.

```hcl
resource "incidentrelay_change_integration" "pipeline" {
  name           = "Release pipeline"
  owner_group_id = incidentrelay_group.platform.id
  owner_team_id  = incidentrelay_team.platform.id
}
```

- `name` (required): display name, 1–255 characters. Renaming does not rename the initial source.
- `owner_team_id` (required, replacement): Team used for the first source at creation. On import this is initialized from the earliest source's current Team.
- `owner_group_id` (optional, replacement): immutable owning Group. Omit for a central integration; only a global administrator can manage central integrations.
- `active` (optional, default `true`): enable or disable an existing integration. IR only creates active objects.
- `id`, `uid`, `initial_source_id` (computed): numeric integration ID, stable UUID and numeric first source ID.

Destroy disables the integration; IR retains its identity, sources and history. To manage it again, import its ID. To manage the automatically created source as a separate resource, [import the source](change_source.md); declaring another source with the same namespace will conflict. Machine credential issuance and rotation are separate from configuration; this resource does not return tokens.

```sh
terraform import incidentrelay_change_integration.pipeline 12
```

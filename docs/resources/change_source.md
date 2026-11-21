---
page_title: "incidentrelay_change_source Resource - IncidentRelay"
subcategory: "Change Intelligence"
description: |-
  Manages a producer namespace on a Change Integration.
---

# incidentrelay_change_source

Requires IR 2.5 and administrator authentication. IR 2.5.1 also supports scoped personal API tokens. See the [authentication guide](../guides/authentication.md#change-integrations-ir-25).

```hcl
resource "incidentrelay_change_source" "deploy" {
  integration_id = incidentrelay_change_integration.pipeline.id
  owner_team_id  = incidentrelay_team.platform.id
  source         = "production-deploys"
  ordering_mode  = "sequence"
}
```

- `integration_id` (required, replacement): parent integration.
- `owner_team_id`, `source` (required): the Team and producer namespace. The name is 1–255 characters. IR allows changing either value only before the first Change is recorded.
- `ordering_mode` (optional): `timestamp` (default) or `sequence`.
- `active` (optional, default `true`): a newly created source is active.
- `id`, `identity_editable` (computed): numeric source ID and whether IR still permits editing its identity.

## GitLab deployments (IR 2.5.1+)

```hcl
gitlab_mappings_json = jsonencode([{
  project_id  = 42
  environment = "production"
  service_ids = [tonumber(incidentrelay_service.api.id)]
}])
ordering_mode = "timestamp"
```

`gitlab_mappings_json` is optional and exports the current mappings. Omit it to
manage mappings through UX, or set `"[]"` to clear them. Each of at most 100
rules contains a positive project ID up to 9007199254740991, an exact,
case-sensitive environment name of 1–255 characters, and 1–100 positive Service
IDs. Each project/environment pair must be unique. GitLab mappings require
timestamp ordering.

`gitlab_webhook_path` exports the native deployment endpoint, for example
`/api/change-events/gitlab/34`. Use this path with the externally reachable IR
base URL and enable **Deployment events** in GitLab. Set GitLab's Secret token
to a Change integration credential issued in IR. The source also needs
`changed` grants for its mapped Services. Administrator personal API tokens
authenticate Terraform configuration operations. Ingestion uses the separate
Change integration credential.

Destroy deactivates the source. Its namespace remains reserved, including after deactivation. Import it to reuse it; use a fresh name to create another source. To grant access to the integration's first source without managing that source separately, use the integration's `initial_source_id`.

```sh
# integration_id/source_id
terraform import incidentrelay_change_source.deploy 12/34
```

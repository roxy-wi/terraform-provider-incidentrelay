---
page_title: "incidentrelay_version Data Source - IncidentRelay"
subcategory: "System"
description: |-
  Reads IncidentRelay service version and migration status.
---

# incidentrelay_version

```hcl
data "incidentrelay_version" "current" {}

output "incidentrelay_version" {
  value = data.incidentrelay_version.current.service_version
}
```

## Attributes

- `id` (String) Always `version`.
- `service_version` (String) IncidentRelay service version.
- `migrations_json` (String) Migration status JSON returned by the API.

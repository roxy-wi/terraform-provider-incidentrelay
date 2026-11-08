---
page_title: "incidentrelay_system_settings Data Source - IncidentRelay"
subcategory: "Change Intelligence"
description: |-
  Reads the current installation-wide Change Intelligence settings.
---

# incidentrelay_system_settings

Reads the same values a global administrator sees under **Administration → System
settings**. Requires a global administrator JWT and an IR 2.5 build that exposes
`/api/admin/system-settings`. The data source takes no arguments.

```hcl
data "incidentrelay_system_settings" "current" {}

output "change_intake_enabled" {
  value = data.incidentrelay_system_settings.current.change_intake_enabled
}
```

The data source exports the singleton ID `1`, both switches, and all twelve
numeric limits and intervals. See the [settings table](../resources/system_settings.md#settings)
for the attribute names, meanings and ranges.

---
page_title: "incidentrelay_orchestration_webhook_action Resource - IncidentRelay"
subcategory: "Event Orchestration"
description: |-
  Manages a reusable IncidentRelay Event Orchestration webhook action.
---

# incidentrelay_orchestration_webhook_action

```hcl
resource "incidentrelay_orchestration_webhook_action" "automation" {
  group_id   = incidentrelay_group.infrastructure.id
  name       = "Automation receiver"
  description = "Notify the internal remediation service"
  url        = var.automation_webhook_url
  method     = "POST"

  headers_json = jsonencode({
    Authorization = "Bearer ${var.automation_webhook_token}"
    "Content-Type" = "application/json"
  })

  body_template = jsonencode({
    title    = "{{ event.title }}"
    severity = "{{ event.severity }}"
  })

  timeout_seconds        = 15
  retry_count            = 3
  private_network_policy = "deny"
  enabled                = true
}
```

## Arguments

- `group_id` (Number, Required) Owner group ID. Changing it recreates the
  action.
- `name` (String, Required) Webhook action name.
- `description` (String, Optional) Action description.
- `url` (String, Required, Sensitive) Destination URL.
- `method` (String, Optional) `GET`, `POST`, `PUT`, `PATCH`, or `DELETE`.
  Defaults to `POST`.
- `headers_json` (String, Optional, Sensitive) HTTP header JSON object.
  Defaults to `{}`.
- `body_template` (String, Optional) IncidentRelay safe template used as the
  request body.
- `timeout_seconds` (Number, Optional) Request timeout from 1 to 60 seconds.
  Defaults to `10`.
- `retry_count` (Number, Optional) Retry count from 0 to 10. Defaults to `2`.
- `private_network_policy` (String, Optional) `deny` or `allowlist`. Defaults
  to `deny`.
- `enabled` (Boolean, Optional) Whether the action is enabled. Defaults to
  `true`.

## Read-Only Attributes

- `uid` (String) Webhook action UUID.
- `has_headers` (Boolean) Whether IncidentRelay stores encrypted headers.
- `created_at` (String) Creation timestamp.
- `updated_at` (String) Last update timestamp.

IncidentRelay encrypts headers and never returns their values. The provider
therefore preserves configured `headers_json` during refresh. It also preserves
the configured URL if IncidentRelay redacts a credential embedded in it. Both
fields are sensitive because webhook URLs frequently contain credentials.
Sensitive values still exist in Terraform state; use an encrypted backend with
restricted access.

## Import

```sh
terraform import incidentrelay_orchestration_webhook_action.automation 81
```

Configure `group_id`, `url`, and any secret headers before applying an imported
action because write-only values cannot be recovered from the API.

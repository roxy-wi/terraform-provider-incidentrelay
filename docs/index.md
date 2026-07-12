# IncidentRelay Provider

Use the IncidentRelay provider to manage groups, users, teams, notification
channels, alert routes, rotations, escalation policies, service catalog objects,
maintenance windows, silences, heartbeats, and business services.

```hcl
provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

Nested API configuration is represented as JSON strings:

```hcl
config_json = jsonencode({
  webhook_url = var.webhook_url
  notify_on_severities = ["critical", "high"]
})
```


# incidentrelay_notification_policy_rule

Manages a rule inside a service notification policy.

```hcl
resource "incidentrelay_notification_policy_rule" "critical" {
  policy_id   = incidentrelay_notification_policy.production.id
  name        = "Critical alerts"
  event_types = ["notification", "reminder", "escalation"]
  channel_ids = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    severity = "critical"
  })
}
```


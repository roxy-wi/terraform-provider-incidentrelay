# incidentrelay_heartbeat

Manages a heartbeat dead-man-switch check.

The API returns `token` and `ping_url` only once on creation or token
regeneration. Terraform stores those computed values as sensitive.


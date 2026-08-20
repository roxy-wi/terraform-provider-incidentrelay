---
page_title: "IncidentRelay Provider"
description: |-
  The IncidentRelay provider manages IncidentRelay access, SSO, on-call, alert routing, Event Orchestration, service catalog, maintenance, heartbeat, and business-service configuration.
---

# IncidentRelay Provider

Use the IncidentRelay provider to manage IncidentRelay configuration as code:
groups, users, teams, notification channels, alert routes, rotations, escalation
policies, notification policies, service catalog objects, maintenance windows,
incident priority policies, silences, heartbeats, business services, OIDC/SAML
providers, SSO group mappings, and Event Orchestration.

The provider communicates with the IncidentRelay HTTP API. Use token
authentication for automation and username/password authentication for local
development or bootstrap workflows.

The current provider code is tested against IncidentRelay 2.0, including Event
Orchestration, reusable orchestration webhooks, Uptime Kuma and Datadog routes,
Slack Bot API channels using HTTP actions or Socket Mode, and SSO provider and
group mapping administration.

## Example Usage

```hcl
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.6"
    }
  }
}

variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

## Authentication

Token authentication:

```hcl
provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}
```

Username/password authentication:

```hcl
provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  username = var.incidentrelay_username
  password = var.incidentrelay_password
}
```

Environment variables:

- `INCIDENTRELAY_BASE_URL`
- `INCIDENTRELAY_TOKEN`
- `INCIDENTRELAY_USERNAME`
- `INCIDENTRELAY_PASSWORD`
- `INCIDENTRELAY_INSECURE_SKIP_TLS_VERIFY`

## Schema

### Required

- `base_url` (String) IncidentRelay base URL, for example
  `https://incidentrelay.example.com`.

### Optional

- `token` (String, Sensitive) Bearer API token or JWT token.
- `username` (String) IncidentRelay username for `/api/auth/login`.
- `password` (String, Sensitive) IncidentRelay password for `/api/auth/login`.
- `insecure_skip_tls_verify` (Boolean) Skip TLS certificate verification.
  Intended only for local development.

## JSON Fields

Nested API configuration is represented as JSON strings. Prefer `jsonencode(...)`
so Terraform can validate the expression before the provider sends it:

```hcl
config_json = jsonencode({
  webhook_url          = var.webhook_url
  notify_on_severities = ["critical", "high"]
})
```

See [JSON fields guide](guides/json-fields.md) for matchers, labels,
metadata, maintenance scopes, and integration config examples.

## Importing Existing Configuration

Resources support Terraform import using their IncidentRelay API identity.
Most use a numeric ID; nested resources may also require their parent ID in
configuration or use a documented `parent_id/resource_id` import identity.

See [Import guide](guides/importing.md) and [import examples](../examples/imports/main.tf).

## Guides

- [Authentication](guides/authentication.md)
- [Importing existing resources](guides/importing.md)
- [JSON fields](guides/json-fields.md)
- [Modeling IncidentRelay in Terraform](guides/modeling.md)
- [Resource reference](guides/resource-reference.md)
- [Testing and release](guides/testing-and-release.md)

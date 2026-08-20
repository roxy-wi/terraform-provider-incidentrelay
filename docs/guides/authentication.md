---
page_title: "Authentication - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  Authentication patterns for the IncidentRelay Terraform provider.
---

# Authentication

The provider supports two authentication modes:

- Bearer token authentication with `token` or `INCIDENTRELAY_TOKEN`.
- Username/password login against `/api/auth/login`.

Token authentication is recommended for CI, Terraform Cloud, and automation.
Username/password authentication is useful for local development or bootstrap
workflows.

## Token Authentication

```hcl
variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

For CI, prefer environment variables:

```sh
export INCIDENTRELAY_BASE_URL="https://incidentrelay.example.com"
export INCIDENTRELAY_TOKEN="..."
terraform plan
```

### IncidentRelay 2.0 token scopes

Personal API tokens in IncidentRelay 2.0 use granular read/write scopes. A
Terraform plan needs the matching `:read` scopes, and apply/destroy also need
the matching `:write` scopes:

| Terraform configuration | IncidentRelay scopes |
| --- | --- |
| Groups and memberships | `groups:read`, `groups:write` |
| Admin users | `users:read`, `users:write` |
| Teams and memberships | `teams:read`, `teams:write` |
| Channels | `channels:read`, `channels:write` |
| Routes | `routes:read`, `routes:write` |
| Rotations and overrides | `rotations:read`, `rotations:write` |
| Escalation, notification, priority, and matcher policies | `policies:read`, `policies:write` |
| Services and business services | `services:read`, `services:write` |
| Silences and maintenance windows | `maintenance:read`, `maintenance:write` |
| Heartbeats | `heartbeats:read`, `heartbeats:write` |
| Event Orchestration and webhook actions | `orchestrations:read`, `orchestrations:write` |
| SSO providers and mappings | `sso:read`, `sso:write` |

The legacy `resources:read` and `resources:write` aggregate scopes remain
compatible with these configuration domains. Prefer the granular scopes for a
least-privilege automation token.

## Username And Password Authentication

```hcl
variable "incidentrelay_username" {
  type = string
}

variable "incidentrelay_password" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  username = var.incidentrelay_username
  password = var.incidentrelay_password
}
```

The provider exchanges the credentials for an access token during provider
configuration. The password is marked sensitive in Terraform.

## SSO Configuration And Provider Authentication

`incidentrelay_sso_provider` configures how users sign in to IncidentRelay.
It does not authenticate Terraform itself. The Terraform provider still needs
an API token or a local IncidentRelay username and password with global
administrator access to manage SSO providers and group mappings.

## Local Development With Self-Signed TLS

```hcl
provider "incidentrelay" {
  base_url                 = "https://localhost:5000"
  token                    = var.incidentrelay_token
  insecure_skip_tls_verify = true
}
```

Do not use `insecure_skip_tls_verify` in production automation.

## GitHub Actions Example

```yaml
name: Terraform

on:
  pull_request:

jobs:
  plan:
    runs-on: ubuntu-latest
    env:
      INCIDENTRELAY_BASE_URL: ${{ secrets.INCIDENTRELAY_BASE_URL }}
      INCIDENTRELAY_TOKEN: ${{ secrets.INCIDENTRELAY_TOKEN }}
    steps:
      - uses: actions/checkout@v6
      - uses: hashicorp/setup-terraform@v3
      - run: terraform init
      - run: terraform validate
      - run: terraform plan -input=false
```

## Secret Handling

- Keep tokens in the secret store of your CI platform.
- Do not commit `.tfvars` files containing secrets.
- Use sensitive Terraform variables for tokens and passwords.
- Rotate tokens used by CI separately from human user credentials.
